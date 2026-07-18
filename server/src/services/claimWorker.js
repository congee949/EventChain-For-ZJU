import config from '../config/index.js';
import { findFirstUserByRole } from './wallet.js';
import { financeEvaluate, financeSubmit } from './financeService.js';

const benignClaimErrors = [
  'account has no claimable position', 'claim not found', 'claim is still pending',
  'settlement has not been activated', 'settlement epoch was superseded',
];

function isMVCC(error) {
  const message = `${error?.message || ''} ${JSON.stringify(error?.details || '')}`;
  return /MVCC_READ_CONFLICT|PHANTOM_READ_CONFLICT|ENDORSEMENT_POLICY_FAILURE/i.test(message);
}

function isBenign(error) {
  const message = `${error?.message || ''} ${JSON.stringify(error?.details || '')}`;
  return benignClaimErrors.some((part) => message.includes(part));
}

async function withRetry(operation, retries = 5) {
  let lastError;
  for (let attempt = 0; attempt < retries; attempt++) {
    try {
      return await operation();
    } catch (error) {
      lastError = error;
      if (!isMVCC(error) || attempt === retries - 1) throw error;
      await new Promise((resolve) => setTimeout(resolve, 100 * (2 ** attempt) + Math.floor(Math.random() * 100)));
    }
  }
  throw lastError;
}

async function mapConcurrent(items, concurrency, handler) {
  let cursor = 0;
  const results = [];
  const workers = Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (cursor < items.length) {
      const index = cursor++;
      try {
        results[index] = { ok: true, value: await handler(items[index]) };
      } catch (error) {
        results[index] = { ok: false, error };
      }
    }
  });
  await Promise.all(workers);
  return results;
}

async function allParticipants(operatorId, marketId) {
  const accounts = [];
  let startAfter = '';
  for (;;) {
    const page = await financeEvaluate(operatorId, 'ListParticipants', [marketId, startAfter, config.claimWorker.pageSize]);
    if (!Array.isArray(page) || page.length === 0) break;
    accounts.push(...page);
    if (page.length < config.claimWorker.pageSize) break;
    const next = page.at(-1);
    if (next === startAfter) throw new Error('participant pagination did not advance');
    startAfter = next;
  }
  return [...new Set(accounts)];
}

async function processAccounts(operatorId, fn, marketId, epoch, accounts) {
  const results = await mapConcurrent(accounts, config.claimWorker.concurrency, async (accountId) =>
    withRetry(() => financeSubmit(operatorId, fn, [marketId, epoch, accountId])));
  const summary = { succeeded: 0, skipped: 0, failed: [] };
  results.forEach((result, index) => {
    if (result.ok) summary.succeeded++;
    else if (isBenign(result.error)) summary.skipped++;
    else summary.failed.push({ accountId: accounts[index], message: result.error?.message || String(result.error) });
  });
  return summary;
}

export async function processSettlement(operatorId, marketId, epoch) {
  const settlement = await financeEvaluate(operatorId, 'GetSettlement', [marketId, epoch]);
  const accounts = await allParticipants(operatorId, marketId);
  const prepared = await processAccounts(operatorId, 'ClaimOne', marketId, epoch, accounts);
  const result = { marketId, epoch: Number(epoch), participants: accounts.length, prepared, activated: false, matured: null, closed: false };
  if (prepared.failed.length) return result;
  if (Date.now() < Date.parse(settlement.availableAt)) return result;
  await withRetry(() => financeSubmit(operatorId, 'ActivateSettlement', [marketId, epoch]));
  result.activated = true;
  result.matured = await processAccounts(operatorId, 'MatureClaimOne', marketId, epoch, accounts);
  if (result.matured.failed.length) return result;
  await withRetry(() => financeSubmit(operatorId, 'CloseSettlement', [marketId, epoch]));
  result.closed = true;
  return result;
}

export async function runClaimCycle() {
  const operator = findFirstUserByRole('operator') || findFirstUserByRole('admin');
  if (!operator) return { skipped: true, reason: 'no platform operator identity' };
  const markets = await financeEvaluate(operator.account_id, 'ListMarkets');
  const pending = (markets || []).filter((market) => market.status === 'PROVISIONAL_FINALIZED' && market.settlementEpoch > 0);
  const results = [];
  for (const market of pending) {
    try {
      results.push(await processSettlement(operator.account_id, market.marketId, market.settlementEpoch));
    } catch (error) {
      results.push({ marketId: market.marketId, epoch: market.settlementEpoch, error: error?.message || String(error) });
    }
  }
  return { skipped: false, processed: results };
}

export function startClaimWorker() {
  if (!config.claimWorker.enabled) return null;
  let running = false;
  const tick = async () => {
    if (running) return;
    running = true;
    try {
      const result = await runClaimCycle();
      if (!result.skipped && result.processed.length) console.log('[ClaimWorker]', JSON.stringify(result));
    } catch (error) {
      console.error('[ClaimWorker]', error);
    } finally {
      running = false;
    }
  };
  const timer = setInterval(tick, config.claimWorker.intervalMs);
  timer.unref();
  setTimeout(tick, 5_000).unref();
  return timer;
}
