import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { app } from '../src/app.js';
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

test('V2 finance routes require authentication', async () => {
  const response = await fetch(`${origin}/api/v2/finance/wallet`);
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
