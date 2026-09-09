import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import express from 'express';
import { app, createReadinessHandler } from '../src/app.js';
import { validateProductionConfig } from '../src/config/index.js';
import { asText, hashEvidence, idempotencyKey } from '../src/services/financeService.js';

let server;
let origin;

before(async () => {
  server = app.listen(0, '127.0.0.1');
  await new Promise((resolve) => server.once('listening', resolve));
  origin = `http://127.0.0.1:${server.address().port}`;
});

after(async () => {
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
});

test('V1 is retired and cannot bypass V2 controls', async () => {
  const response = await fetch(`${origin}/api/v1/predictions/bet`, { method: 'POST' });
  assert.equal(response.status, 410);
  assert.equal((await response.json()).code, 'V1_RETIRED');
});

test('health endpoint exposes security headers without framework fingerprinting', async () => {
  const response = await fetch(`${origin}/api/health`);
  assert.equal(response.status, 200);
  assert.equal(response.headers.get('x-content-type-options'), 'nosniff');
  assert.equal(response.headers.get('x-frame-options'), 'DENY');
  assert.equal(response.headers.get('x-powered-by'), null);
  assert.equal((await response.json()).status, 'ok');
});

test('unknown API endpoints return a stable 404 error', async () => {
  const response = await fetch(`${origin}/api/v2/not-a-route`);
  assert.equal(response.status, 404);
  assert.equal((await response.json()).code, 'NOT_FOUND');
});

test('malformed JSON is rejected as a client error', async () => {
  const response = await fetch(`${origin}/api/v2/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: '{bad json',
  });
  assert.equal(response.status, 400);
  assert.equal((await response.json()).code, 'INVALID_JSON');
});

test('V2 finance routes require authentication', async () => {
  const response = await fetch(`${origin}/api/v2/finance/wallet`);
  assert.equal(response.status, 401);
  assert.equal((await response.json()).code, 'UNAUTHORIZED');
});

test('secret-based CheckIn route remains protected and keeps its original endpoint', async () => {
  const response = await fetch(`${origin}/api/v2/activities/check-in/verify`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ ticketId: 'ticket-1', secret: 'a'.repeat(32), timeSlice: 1 }),
  });
  assert.equal(response.status, 401);
  assert.equal((await response.json()).code, 'UNAUTHORIZED');
});

test('public student registration is closed by default', async () => {
  const response = await fetch(`${origin}/api/v2/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ studentID: 'sybil-1', password: 'password123', name: 'Sybil' }),
  });
  assert.equal(response.status, 403);
  assert.equal((await response.json()).code, 'FORBIDDEN');
});

test('financial helpers reject weak idempotency keys and canonicalize evidence hashes', () => {
  assert.equal(asText(12.5, 'amount'), '12.5');
  assert.throws(() => idempotencyKey({ get: () => 'short' }), /8-128/);
  const hash = hashEvidence('observable evidence');
  assert.match(hash, /^[0-9a-f]{64}$/);
  assert.equal(hashEvidence(hash.toUpperCase()), hash);
});

test('production configuration requires explicit identity protection keys', () => {
  const original = {
    nodeEnv: process.env.NODE_ENV,
    lookupKey: process.env.IDENTITY_LOOKUP_KEY,
    encryptionKey: process.env.IDENTITY_ENCRYPTION_KEY,
  };
  const productionConfig = {
    jwt: { secret: 'a-production-jwt-secret-that-is-long-enough' },
    demoMode: false,
    identity: { demoBootstrapKey: '' },
  };

  try {
    process.env.NODE_ENV = 'production';
    delete process.env.IDENTITY_LOOKUP_KEY;
    delete process.env.IDENTITY_ENCRYPTION_KEY;
    assert.throws(
      () => validateProductionConfig(productionConfig),
      /IDENTITY_LOOKUP_KEY, IDENTITY_ENCRYPTION_KEY/,
    );

    process.env.IDENTITY_LOOKUP_KEY = 'a'.repeat(64);
    process.env.IDENTITY_ENCRYPTION_KEY = 'b'.repeat(64);
    assert.doesNotThrow(() => validateProductionConfig(productionConfig));
  } finally {
    if (original.nodeEnv === undefined) delete process.env.NODE_ENV;
    else process.env.NODE_ENV = original.nodeEnv;
    if (original.lookupKey === undefined) delete process.env.IDENTITY_LOOKUP_KEY;
    else process.env.IDENTITY_LOOKUP_KEY = original.lookupKey;
    if (original.encryptionKey === undefined) delete process.env.IDENTITY_ENCRYPTION_KEY;
    else process.env.IDENTITY_ENCRYPTION_KEY = original.encryptionKey;
  }
});

test('readiness requires the admin badge capability probe and reports clear 503s', async () => {
  const calls = [];
  const handler = createReadinessHandler({
    findProbeUser: () => ({ account_id: 'acct_admin' }),
    financeProbe: async (_id, fn) => { calls.push(fn); return {}; },
    activityProbe: async (_id, fn) => {
      calls.push(fn);
      if (fn === 'GetBadgeReadiness') throw new Error('unknown transaction GetBadgeReadiness');
      return [];
    },
  });
  const readinessApp = express();
  readinessApp.get('/api/readiness', handler);
  const readinessServer = readinessApp.listen(0, '127.0.0.1');
  await new Promise((resolve) => readinessServer.once('listening', resolve));
  const response = await fetch(`http://127.0.0.1:${readinessServer.address().port}/api/readiness`);
  await new Promise((resolve, reject) => readinessServer.close((error) => error ? reject(error) : resolve()));
  assert.equal(response.status, 503);
  const json = await response.json();
  assert.equal(json.code, 'BADGE_CHAINCODE_UNAVAILABLE');
  assert.equal(json.checks.activity, true);
  assert.equal(json.checks.badges, false);
  assert.deepEqual(calls, ['GetConfig', 'ListActivities', 'ListBadgeSeries', 'GetBadgeReadiness']);
});
