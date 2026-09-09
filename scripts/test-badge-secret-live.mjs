#!/usr/bin/env node
import crypto from 'node:crypto';
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const base = process.env.API_BASE || 'http://127.0.0.1:3101/api/v2';
const run = process.env.BADGE_TEST_RUN_ID || `badge-secret-live-${Date.now()}`;
const resume = process.env.BADGE_RESUME_EXISTING === 'true';
const activityId = `activity-${run}`;
const seriesId = `series-${run}`;
const seed = `badge-secret-live-seed-${run}-2026`;
const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const outPath = process.env.BADGE_EVIDENCE_PATH || path.join(projectRoot, 'tmp', `badge-secret-live-${run}.json`);

async function call(path, token, { method = 'GET', body, idem } = {}) {
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
  if (idem) headers['Idempotency-Key'] = idem;
  const response = await fetch(`${base}${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await response.text();
  let data;
  try { data = JSON.parse(text); } catch { throw new Error(`${method} ${path} returned non-JSON ${response.status}`); }
  if (!response.ok || data.error) throw new Error(`${method} ${path} failed ${response.status}: ${data.message || data.code || 'unknown error'}`);
  return data.data;
}

async function login(studentID, password) {
  const response = await fetch(`${base}/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ studentID, password }) });
  const data = await response.json();
  if (!response.ok || data.error) throw new Error(`login failed for ${studentID}: ${data.message || data.code || response.status}`);
  return data.data;
}

const waitUntil = async (at) => {
  const delay = Math.max(0, at - Date.now());
  if (delay) await new Promise((resolve) => setTimeout(resolve, delay));
};
const digest = (value) => crypto.createHash('sha256').update(value).digest('hex');
const secret1 = `badge-secret-${run}-student-one-000000000000`;
const secret2 = `badge-secret-${run}-student-two-000000000000`;

const admin = await login(process.env.BADGE_ADMIN_ID || 'admin01', process.env.BADGE_ADMIN_PASSWORD || 'eventchain-admin-2026');
const organizer = await login(process.env.BADGE_ORGANIZER_ID || 'organizer01', process.env.BADGE_ORGANIZER_PASSWORD || 'eventchain-organizer-2026');
const verifier = await login(process.env.BADGE_VERIFIER_ID || 'verifier01', process.env.BADGE_VERIFIER_PASSWORD || 'eventchain-verifier-2026');
const operator = await login(process.env.BADGE_OPERATOR_ID || 'operator01', process.env.BADGE_OPERATOR_PASSWORD || 'eventchain-operator-2026');
const student1 = await login(process.env.BADGE_STUDENT1_ID || '3220100001', process.env.BADGE_STUDENT1_PASSWORD || 'eventchain-student-2026');
const student2 = await login(process.env.BADGE_STUDENT2_ID || '3220100002', process.env.BADGE_STUDENT2_PASSWORD || 'eventchain-student-2026');

const now = Date.now();
const applicationCloseAt = new Date(now + 30_000).toISOString();
const startsAt = new Date(now + 40_000).toISOString();
const endsAt = new Date(now + 600_000).toISOString();
const eligibilityOpenAt = new Date(now + 15_000).toISOString();
const eligibilityCloseAt = new Date(now + 86_400_000).toISOString();
const claimCloseAt = new Date(now + 2_592_000_000).toISOString();

if (!resume) await call('/activities', organizer.token, { method: 'POST', body: {
  id: activityId, categoryId: 'basketball', title: 'Live badge secret acceptance', capacity: 2,
  applicationCloseAt, startsAt, endsAt,
} });
if (!resume) await call('/badges/series', admin.token, { method: 'POST', body: {
  seriesId, seasonId: '2026-live', categoryId: 'basketball', title: 'Live badge secret acceptance',
  description: 'Live acceptance series', assetUri: '/badges/basketball-2026-demo/v1/badge.png',
  assetSha256: '382005d778fa4e12347b32ef555ff8d09f5a816f80c7ca38611a4422002abd93',
  metadataSha256: '8fa829a0dd92e46376c1af9ddb7f213bfedc1694d202ac11e7d3c068d239f145',
  eligibilityPolicyHash: 'd76226a58b5f6010210a7056e9d62d03bac33e9682932be9ae943561e27b3ba3',
  issuanceMode: 'CHECK_IN_GUARANTEED', maxSupply: 2,
  eligibilityOpenAt, eligibilityCloseAt, claimOpenAt: eligibilityOpenAt, claimCloseAt,
  supplyRationale: 'Supply equals activity capacity.',
} });
if (!resume) await call(`/badges/series/${seriesId}/activities/${activityId}`, admin.token, { method: 'POST', body: { eligibilityQuota: 2 } });
if (!resume) await call(`/badges/series/${seriesId}/activate`, admin.token, { method: 'POST', body: {} });
if (!resume) await call(`/activities/${activityId}/draw-commitment`, verifier.token, { method: 'POST', body: { seed } });
if (!resume) await call(`/activities/${activityId}/applications`, student1.token, { method: 'POST', body: {} });
if (!resume) await call(`/activities/${activityId}/applications`, student2.token, { method: 'POST', body: {} });

if (!resume) await waitUntil(now + 31_000);
if (!resume) await call(`/activities/${activityId}/draw`, organizer.token, { method: 'POST', body: { seed } });
const ticket1 = resume
  ? await call(`/activities/${activityId}/ticket`, student1.token)
  : (await call(`/activities/${activityId}/ticket`, student1.token, { method: 'POST', body: { secret: secret1 }, idem: `ticket-${run}-1` })).ticket;
const ticket2 = resume
  ? await call(`/activities/${activityId}/ticket`, student2.token)
  : (await call(`/activities/${activityId}/ticket`, student2.token, { method: 'POST', body: { secret: secret2 }, idem: `ticket-${run}-2` })).ticket;
const timeSlice = Math.floor(Date.now() / 30_000);
const checkinBody = (ticket, secret) => ({ ticketId: ticket.ticketId, secret, timeSlice });
const receipt1 = await call('/activities/check-in/verify', operator.token, { method: 'POST', body: checkinBody(ticket1, secret1) });
const receipt2 = await call('/activities/check-in/verify', operator.token, { method: 'POST', body: checkinBody(ticket2, secret2) });
if (!receipt1.receipt?.badgeEligibility?.some((entry) => entry.seriesId === seriesId && entry.status === 'CLAIMABLE')) throw new Error('student1 check-in did not return CLAIMABLE eligibility');
if (!receipt2.receipt?.badgeEligibility?.some((entry) => entry.seriesId === seriesId && entry.status === 'CLAIMABLE')) throw new Error('student2 check-in did not return CLAIMABLE eligibility');
const duplicate = await call('/activities/check-in/verify', operator.token, { method: 'POST', body: checkinBody(ticket1, secret1) });
if (duplicate.receipt?.replayed !== true) throw new Error('duplicate check-in was not marked replayed');
if (duplicate.receipt.refId !== receipt1.receipt.refId || duplicate.receipt.checkedInAt !== receipt1.receipt.checkedInAt) throw new Error('duplicate check-in changed the original receipt');
async function claimAndWait(student, suffix) {
  let result = await call(`/badges/series/${seriesId}/claim`, student.token, { method: 'POST', body: {}, idem: `badge-${run}-${suffix}` });
  const refId = result.refId;
  for (let attempt = 0; (!result.instance || !result.award) && attempt < 12; attempt += 1) {
    await new Promise((resolve) => setTimeout(resolve, 250));
    result = await call(`/badges/series/${seriesId}/claims/${refId}`, student.token);
  }
  if (!result.instance || !result.award) throw new Error(`badge claim ${suffix} remained pending`);
  return { ...result, refId };
}
const claim1 = await claimAndWait(student1, '1');
const claim2 = await claimAndWait(student2, '2');
const claimReplay = await claimAndWait(student1, '1');
if (claimReplay.instance?.instanceId !== claim1.instance?.instanceId) throw new Error('badge claim retry returned a different instance');
const claimReceipt = await call(`/badges/series/${seriesId}/claims/${claim1.refId}`, student1.token);
const series = await call(`/badges/series/${seriesId}`, student1.token);
if (series.issuedCount !== 2 || series.maxSupply !== 2) throw new Error('series supply count mismatch');
const transferProbe = await fetch(`${base}/badges/series/${seriesId}/transfer`, {
  method: 'POST', headers: { Authorization: `Bearer ${student1.token}`, 'Content-Type': 'application/json' }, body: '{}',
});
const transferProbeBody = await transferProbe.json();
if (transferProbe.status !== 404 || transferProbeBody.code !== 'NOT_FOUND') throw new Error('unexpected badge transfer API exists');
const evidence = {
  run, activityId, seriesId, status: 'PASS',
  checkins: [
    { account: student1.userId, ticketId: ticket1.ticketId, eligibility: receipt1.receipt.badgeEligibility, rewardPending: receipt1.rewardPending === true },
    { account: student2.userId, ticketId: ticket2.ticketId, eligibility: receipt2.receipt.badgeEligibility, rewardPending: receipt2.rewardPending === true },
  ],
  duplicateCheckIn: { replayed: duplicate.receipt.replayed === true, refId: duplicate.receipt.refId },
  claims: [claim1.instance?.instanceId, claim2.instance?.instanceId],
  claimRetryStable: claimReplay.instance?.instanceId === claim1.instance?.instanceId,
  claimReceiptLookup: { refId: claim1.refId, instanceId: claimReceipt.instance?.instanceId, serialNumber: claimReceipt.instance?.serialNumber },
  series: { issuedCount: series.issuedCount, maxSupply: series.maxSupply },
  badgeTransferEndpoint: { status: transferProbe.status, code: transferProbeBody.code },
  note: 'Tokens, secrets and claim hashes intentionally omitted.',
};
await fs.mkdir(path.dirname(outPath), { recursive: true });
await fs.writeFile(outPath, `${JSON.stringify(evidence, null, 2)}\n`);
console.log(JSON.stringify(evidence));
