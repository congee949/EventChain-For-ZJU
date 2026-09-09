import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { afterEach, test } from 'node:test';
import express from 'express';
import jwt from 'jsonwebtoken';
import config from '../src/config/index.js';
import { errorHandler } from '../src/middleware/errorHandler.js';
import { createBadgesRouter } from '../src/routes/badges.js';
import { BadgeHttpError, createBadgeService } from '../src/services/badgeService.js';

const servers = new Set();
afterEach(async () => {
  await Promise.all([...servers].map((server) => new Promise((resolve, reject) => {
    server.close((error) => error ? reject(error) : resolve());
  })));
  servers.clear();
});

function token(role = 'student', userId = `acct_${role}`) {
  return jwt.sign({ sub: userId, userId, orgMSP: role === 'student' ? 'StudentMSP' : 'PlatformMSP', role }, config.jwt.secret);
}

function service(overrides = {}) {
  return {
    listSeries: async () => [], listMyAwards: async () => [],
    getPublicInstanceById: async (_id, instanceId) => ({ instanceId }),
    getSeries: async (_id, seriesId) => ({ seriesId }),
    listActivityLinks: async (_id, seriesId) => [{ seriesId, activityId: 'activity-1' }],
    getMyAward: async (_id, seriesId) => ({ series: { seriesId }, eligibility: null, award: null, instance: null }),
    getClaimStatus: async (_id, seriesId, refId) => ({ pending: true, data: { seriesId, refId, status: 'UNKNOWN' } }),
    claimMyBadge: async (_id, seriesId, refId) => ({ pending: false, data: { seriesId, refId, serialNumber: 1 } }),
    createSeries: async (_id, body) => body, linkActivity: async () => ({}),
    activateSeries: async (_id, seriesId) => ({ seriesId, status: 'ACTIVE' }),
    pauseSeries: async (_id, seriesId) => ({ seriesId, status: 'PAUSED' }),
    resumeSeries: async (_id, seriesId) => ({ seriesId, status: 'ACTIVE' }),
    closeSeries: async (_id, seriesId) => ({ seriesId, status: 'CLOSED' }),
    finalizeSeries: async (_id, seriesId) => ({ seriesId, status: 'FINALIZED' }),
    ...overrides,
  };
}

async function start(badgeService) {
  const testApp = express();
  testApp.use(express.json());
  testApp.use('/api/v2/badges', createBadgesRouter(badgeService));
  testApp.use(errorHandler);
  const server = createServer(testApp);
  servers.add(server);
  server.listen(0, '127.0.0.1');
  await new Promise((resolve) => server.once('listening', resolve));
  return `http://127.0.0.1:${server.address().port}/api/v2/badges`;
}

function auth(role = 'student') {
  return { authorization: `Bearer ${token(role)}`, 'content-type': 'application/json' };
}

test('badge routes require auth and claim derives a chaincode-safe stable ref', async () => {
  const calls = [];
  const base = await start(service({ claimMyBadge: async (...args) => {
    calls.push(args);
    return { pending: false, data: { serialNumber: 1 } };
  } }));
  assert.equal((await fetch(`${base}/series/s-1`)).status, 401);
  const headers = { ...auth(), 'Idempotency-Key': 'client.Key:001' };
  const first = await fetch(`${base}/series/s-1/claim`, { method: 'POST', headers, body: '{}' });
  const second = await fetch(`${base}/series/s-1/claim`, { method: 'POST', headers, body: '{}' });
  assert.equal(first.status, 201);
  assert.equal(second.status, 201);
  const a = (await first.json()).data;
  const b = (await second.json()).data;
  assert.equal(a.refId, b.refId);
  assert.match(a.refId, /^[a-z0-9][a-z0-9_-]{7,63}$/);
  assert.equal(calls[0][2], a.refId);
  assert.equal(a.serialNumber, 1);
});

test('pending claims return 202 and expose the canonical ref for polling', async () => {
  const base = await start(service({ claimMyBadge: async (_id, seriesId, refId) => ({
    pending: true, code: 'BADGE_CLAIM_PENDING', data: { seriesId, refId, status: 'UNKNOWN' },
  }) }));
  const response = await fetch(`${base}/series/s-1/claim`, {
    method: 'POST', headers: { ...auth(), 'Idempotency-Key': 'client-key-002' }, body: '{}',
  });
  assert.equal(response.status, 202);
  const json = await response.json();
  assert.equal(json.code, 'BADGE_CLAIM_PENDING');
  assert.match(json.data.refId, /^[a-z0-9][a-z0-9_-]{7,63}$/);
  assert.equal(json.data.status, 'UNKNOWN');
});

test('students cannot call admin lifecycle routes and explicit badge errors preserve 503/409 semantics', async () => {
  let invoked = false;
  const base = await start(service({ activateSeries: async () => { invoked = true; } }));
  const forbidden = await fetch(`${base}/series/s-1/activate`, { method: 'POST', headers: auth(), body: '{}' });
  assert.equal(forbidden.status, 403);
  assert.equal(invoked, false);
  const unavailable = await start(service({ getSeries: async () => {
    throw new BadgeHttpError('BADGE_CHAINCODE_UNAVAILABLE', 503, '徽章链码未升级或当前不可用');
  } }));
  const response = await fetch(`${unavailable}/series/s-1`, { headers: auth() });
  assert.equal(response.status, 503);
  assert.equal((await response.json()).code, 'BADGE_CHAINCODE_UNAVAILABLE');
});

test('an empty claim receipt is not mistaken for an unavailable badge chaincode', async () => {
  const serviceInstance = createBadgeService({
    activityEvaluate: async (_accountId, transactionName) => {
      assert.equal(transactionName, 'GetMyBadgeClaimReceipt');
      throw new Error('chaincode response 500, badge claim receipt not found');
    },
  });
  assert.equal(await serviceInstance.findClaimReceipt('acct_student', 'series-2026', 'badge-ref-001'), null);
});

test('claim proceeds to submit after the wrapped missing-receipt response', async () => {
  const submits = [];
  const serviceInstance = createBadgeService({
    activityEvaluate: async (_accountId, transactionName) => {
      assert.equal(transactionName, 'GetMyBadgeClaimReceipt');
      throw { details: [{ message: 'chaincode response 500, badge claim receipt not found' }] };
    },
    activitySubmit: async (accountId, transactionName, args) => {
      submits.push({ accountId, transactionName, args });
      return { award: { instanceId: 'instance-1' }, instance: { serialNumber: 1 } };
    },
  });
  const result = await serviceInstance.claimMyBadge('acct_student', 'series-2026', 'badge-ref-001');
  assert.equal(result.pending, false);
  assert.deepEqual(submits, [{
    accountId: 'acct_student', transactionName: 'ClaimMyBadge', args: ['series-2026', 'badge-ref-001'],
  }]);
});

test('a missing badge transaction remains a clear 503', async () => {
  const serviceInstance = createBadgeService({
    activityEvaluate: async () => { throw new Error('unknown transaction GetBadgeSeries'); },
  });
  await assert.rejects(
    serviceInstance.getSeries('acct_student', 'series-2026'),
    (error) => error instanceof BadgeHttpError
      && error.code === 'BADGE_CHAINCODE_UNAVAILABLE'
      && error.status === 503,
  );
});

test('claim status does not hide badge chaincode outages behind a pending 202', async () => {
  const serviceInstance = createBadgeService({
    activityEvaluate: async () => { throw new Error('unknown transaction GetMyBadgeClaimReceipt'); },
  });
  await assert.rejects(
    serviceInstance.getClaimStatus('acct_student', 'series-2026', 'badge-ref-001'),
    (error) => error instanceof BadgeHttpError
      && error.code === 'BADGE_CHAINCODE_UNAVAILABLE'
      && error.status === 503,
  );
});
