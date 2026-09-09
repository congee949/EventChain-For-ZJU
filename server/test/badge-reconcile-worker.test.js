import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, test } from 'node:test';
import { createBadgeReconcileWorker } from '../src/services/badgeReconcileWorker.js';

const dirs = new Set();
afterEach(() => { for (const dir of dirs) fs.rmSync(dir, { recursive: true, force: true }); dirs.clear(); });
function journal() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'badge-reconcile-'));
  dirs.add(dir);
  return path.join(dir, 'pending.json');
}

test('pending journal is durable, opaque, and reconciles to a uniform award view', async () => {
  const file = journal();
  let receiptCalls = 0;
  const worker = createBadgeReconcileWorker({
    journalPath: file,
    getClaimReceipt: async () => { receiptCalls += 1; return { refId: 'badge-ref-001' }; },
    getMyBadge: async () => ({ award: { instanceId: 'instance-001' }, instance: { serialNumber: 1 } }),
    intervalMs: 10,
    logger: { error() {} },
  });
  const entry = worker.registerPending({ accountId: 'acct_abcdef0123456789', seriesId: 'series-2026', refId: 'badge-ref-001' });
  assert.equal(worker.registerPending(entry).createdAt, entry.createdAt);
  assert.equal(fs.statSync(file).mode & 0o777, 0o600);
  assert.doesNotMatch(fs.readFileSync(file, 'utf8'), /password|secret|ticket/i);
  assert.deepEqual(await worker.runOnce(), { checked: 1, confirmed: 1, invalid: 0, remaining: 0 });
  assert.equal(receiptCalls, 1);
  assert.equal(worker.getStatus(entry.accountId, entry.seriesId, entry.refId).status, 'CONFIRMED');
  assert.equal(worker.getStatus(entry.accountId, entry.seriesId, entry.refId).result.instance.serialNumber, 1);
});

test('transient errors remain UNKNOWN and stop awaits the in-flight run', async () => {
  const file = journal();
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const worker = createBadgeReconcileWorker({
    journalPath: file,
    getClaimReceipt: async () => { await gate; return null; },
    getMyBadge: async () => null,
    intervalMs: 10,
    logger: { error() {} },
  });
  worker.registerPending({ accountId: 'acct_abcdef0123456789', seriesId: 'series-2026', refId: 'badge-ref-002' });
  worker.start();
  await new Promise((resolve) => setTimeout(resolve, 20));
  let stopped = false;
  const stopping = worker.stop().then(() => { stopped = true; });
  await new Promise((resolve) => setTimeout(resolve, 5));
  assert.equal(stopped, false);
  release();
  await stopping;
  assert.equal(worker.listPending().length, 1);
});
