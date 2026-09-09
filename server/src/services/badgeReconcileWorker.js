import fs from 'node:fs';
import path from 'node:path';

const JOURNAL_VERSION = 1;
const KEY_PATTERN = /^[a-z0-9][a-z0-9_-]{7,63}$/;
const ACCOUNT_ID_PATTERN = /^[a-z0-9][a-z0-9_-]{0,127}$/;
const SERIES_ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/;
const TERMINAL_STATUSES = new Set(['CONFIRMED', 'INVALID']);

function defaultClock() {
  return new Date();
}

function timestamp(clock) {
  return clock().toISOString();
}

function clone(value) {
  return value == null ? value : JSON.parse(JSON.stringify(value));
}

function validateId(value, field, pattern) {
  if (typeof value !== 'string' || !pattern.test(value)) {
    throw new Error(`${field} is invalid`);
  }
  return value;
}

function validateRef(value) {
  if (typeof value !== 'string' || !KEY_PATTERN.test(value)) {
    throw new Error('refId is invalid');
  }
  return value;
}

function entryKey(accountId, seriesId, refId) {
  return `${accountId}\u0000${seriesId}\u0000${refId}`;
}

function emptyJournal() {
  return { version: JOURNAL_VERSION, entries: {} };
}

function normalizeJournal(raw) {
  if (!raw || raw.version !== JOURNAL_VERSION || !raw.entries || typeof raw.entries !== 'object' || Array.isArray(raw.entries)) {
    throw new Error('badge reconcile journal has an unsupported shape');
  }
  return raw;
}

export function createBadgeReconcileWorker({
  journalPath,
  getClaimReceipt,
  getMyBadge,
  classifyTerminalError = () => null,
  clock = defaultClock,
  intervalMs = 5_000,
  maxAttemptsPerRun = 100,
  logger = console,
  fsImpl = fs,
} = {}) {
  if (!journalPath || !path.isAbsolute(journalPath)) {
    throw new Error('journalPath must be absolute');
  }
  if (typeof getClaimReceipt !== 'function' || typeof getMyBadge !== 'function') {
    throw new Error('getClaimReceipt and getMyBadge are required');
  }
  if (!Number.isSafeInteger(intervalMs) || intervalMs < 10) throw new Error('intervalMs must be at least 10ms');

  let timer = null;
  let active = false;
  let running = null;
  let journal = loadJournal();

  function loadJournal() {
    try {
      return normalizeJournal(JSON.parse(fsImpl.readFileSync(journalPath, 'utf8')));
    } catch (error) {
      if (error?.code === 'ENOENT') return emptyJournal();
      throw error;
    }
  }

  function persist() {
    fsImpl.mkdirSync(path.dirname(journalPath), { recursive: true });
    const temporary = `${journalPath}.${process.pid}.${Date.now()}.tmp`;
    try {
      fsImpl.writeFileSync(temporary, `${JSON.stringify(journal, null, 2)}\n`, { encoding: 'utf8', mode: 0o600 });
      fsImpl.renameSync(temporary, journalPath);
    } finally {
      try { fsImpl.unlinkSync(temporary); } catch (error) { if (error?.code !== 'ENOENT') throw error; }
    }
  }

  function registerPending({ accountId, seriesId, refId }) {
    accountId = validateId(accountId, 'accountId', ACCOUNT_ID_PATTERN);
    seriesId = validateId(seriesId, 'seriesId', SERIES_ID_PATTERN);
    refId = validateRef(refId);
    const key = entryKey(accountId, seriesId, refId);
    const existing = journal.entries[key];
    if (existing) return clone(existing);
    const now = timestamp(clock);
    const entry = {
      accountId,
      seriesId,
      refId,
      status: 'UNKNOWN',
      attempts: 0,
      createdAt: now,
      updatedAt: now,
      lastCheckedAt: '',
      resolvedAt: '',
      result: null,
    };
    journal.entries[key] = entry;
    persist();
    return clone(entry);
  }

  function getStatus(accountId, seriesId, refId) {
    const entry = journal.entries[entryKey(
      validateId(accountId, 'accountId', ACCOUNT_ID_PATTERN),
      validateId(seriesId, 'seriesId', SERIES_ID_PATTERN),
      validateRef(refId),
    )];
    return entry ? clone(entry) : null;
  }

  function listPending() {
    return Object.values(journal.entries)
      .filter((entry) => !TERMINAL_STATUSES.has(entry.status))
      .map(clone);
  }

  function markChecked(entry, now) {
    entry.attempts += 1;
    entry.lastCheckedAt = now;
    entry.updatedAt = now;
  }

  function settle(entry, status, result, now) {
    entry.status = status;
    entry.result = result == null ? null : clone(result);
    entry.resolvedAt = now;
    entry.updatedAt = now;
  }

  async function reconcileEntry(entry) {
    const now = timestamp(clock);
    markChecked(entry, now);
    try {
      const receipt = await getClaimReceipt(entry.accountId, entry.seriesId, entry.refId);
      if (!receipt) return false;
      const awardView = await getMyBadge(entry.accountId, entry.seriesId);
      if (!awardView?.award || !awardView?.instance) return false;
      settle(entry, 'CONFIRMED', awardView, now);
      return true;
    } catch (error) {
      const terminal = classifyTerminalError(error);
      if (terminal?.code) {
        settle(entry, 'INVALID', { code: terminal.code }, now);
        return true;
      }
      return false;
    }
  }

  async function runOnce() {
    if (running) return running;
    running = (async () => {
      const pending = Object.values(journal.entries)
        .filter((entry) => !TERMINAL_STATUSES.has(entry.status))
        .slice(0, maxAttemptsPerRun);
      let changed = false;
      let confirmed = 0;
      let invalid = 0;
      for (const entry of pending) {
        changed = true;
        const settled = await reconcileEntry(entry);
        if (settled && entry.status === 'CONFIRMED') confirmed += 1;
        if (settled && entry.status === 'INVALID') invalid += 1;
      }
      if (changed) persist();
      return { checked: pending.length, confirmed, invalid, remaining: listPending().length };
    })();
    try {
      return await running;
    } finally {
      running = null;
    }
  }

  function start() {
    if (timer) return timer;
    active = true;
    const tick = () => {
      if (!active) return;
      runOnce().catch((error) => logger.error('[BadgeReconcileWorker]', error?.message || String(error)));
    };
    timer = setInterval(tick, intervalMs);
    timer.unref?.();
    queueMicrotask(tick);
    return timer;
  }

  async function stop() {
    active = false;
    if (timer) clearInterval(timer);
    timer = null;
    if (running) await running;
  }

  return Object.freeze({ registerPending, getStatus, listPending, runOnce, start, stop });
}
