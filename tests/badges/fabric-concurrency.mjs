#!/usr/bin/env node

import crypto from 'node:crypto';
import process from 'node:process';

let StatusNames;
let StatusCode;
let config;
let createBadgeService;
let activityEvaluate;
let activitySubmit;
let registerAndEnrollUser;
let closeGateway;
let getContract;

async function loadRuntime() {
  const [fabricGateway, configModule, badgeModule, activityModule, caModule, gatewayModule] = await Promise.all([
    import('../../server/node_modules/@hyperledger/fabric-gateway/dist/status.js'),
    import('../../server/src/config/index.js'),
    import('../../server/src/services/badgeService.js'),
    import('../../server/src/services/activityService.js'),
    import('../../server/src/services/caService.js'),
    import('../../server/src/services/fabricGateway.js'),
  ]);
  StatusNames = fabricGateway.default?.StatusNames || fabricGateway.StatusNames;
  StatusCode = fabricGateway.default?.StatusCode || fabricGateway.StatusCode;
  config = configModule.default;
  createBadgeService = badgeModule.createBadgeService;
  activityEvaluate = activityModule.activityEvaluate;
  activitySubmit = activityModule.activitySubmit;
  registerAndEnrollUser = caModule.registerAndEnrollUser;
  closeGateway = gatewayModule.closeGateway;
  getContract = gatewayModule.getContract;
}

const DEFAULTS = Object.freeze({
  apiBase: 'http://127.0.0.1:3100/api/v2',
  rounds: 1,
  sameKey: 20,
  differentKeys: 20,
  bulkClients: 0,
  setupLeadSeconds: 0,
  skipReplay: false,
  skipRbac: false,
});

function parseArgs(argv) {
  const options = { ...DEFAULTS };
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === '--help') options.help = true;
    else if (token === '--skip-replay') options.skipReplay = true;
    else if (token === '--skip-rbac') options.skipRbac = true;
    else if (token.startsWith('--')) {
      const [rawName, inline] = token.slice(2).split('=', 2);
      const value = inline ?? argv[++index];
      const name = rawName.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
      if (!(name in options)) throw new Error(`unknown option --${rawName}`);
      if (name === 'apiBase') options[name] = String(value).replace(/\/$/, '');
      else options[name] = Number(value);
    } else throw new Error(`unexpected argument ${token}`);
  }
  for (const name of ['rounds', 'sameKey', 'differentKeys', 'bulkClients', 'setupLeadSeconds']) {
    if (!Number.isSafeInteger(options[name]) || options[name] < 0) throw new Error(`${name} must be a non-negative integer`);
  }
  if (options.rounds < 1) throw new Error('rounds must be at least 1');
  if (options.bulkClients !== 0 && options.bulkClients < 2) throw new Error('bulkClients must be 0 or at least 2');
  return options;
}

function usage() {
  return `Usage: node tests/badges/fabric-concurrency.mjs [options]

Options:
  --api-base URL             Existing EventChain V2 API (default :3100)
  --rounds N                 FAB-01 fresh series rounds; use 50 for release gate
  --same-key N               FAB-03 same-account/same-key fan-out (default 20)
  --different-keys N         FAB-03 same-account/different-key fan-out (default 20)
  --bulk-clients N           FAB-02 provision N fresh Student identities; use 100
  --setup-lead-seconds N     Override future application/eligibility opening lead
  --skip-replay              Skip FAB-03
  --skip-rbac                Skip direct-Gateway FAB-05

The harness never resets the ledger. It creates uniquely named test objects and
prints one JSON result. Progress is written to stderr.`;
}

function compactRunId() {
  return `${Date.now().toString(36)}${crypto.randomBytes(3).toString('hex')}`;
}

function sha(value) {
  return crypto.createHash('sha256').update(value).digest('hex');
}


function iso(milliseconds) {
  return new Date(milliseconds).toISOString();
}

function log(message) {
  process.stderr.write(`[badge-fabric] ${message}\n`);
}

function decode(bytes) {
  const text = new TextDecoder().decode(bytes || new Uint8Array());
  return text ? JSON.parse(text) : null;
}

function errorSummary(error) {
  return {
    name: error?.name || 'Error',
    code: error?.code ?? null,
    message: String(error?.message || error),
    transactionId: error?.transactionId || null,
  };
}

async function request(apiBase, method, path, token, body, idempotencyKey) {
  const headers = { authorization: `Bearer ${token}`, 'content-type': 'application/json' };
  if (idempotencyKey) headers['idempotency-key'] = idempotencyKey;
  const response = await fetch(`${apiBase}${path}`, {
    method,
    headers,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const error = new Error(`${method} ${path} returned ${response.status}: ${payload?.code || payload?.message || 'unknown error'}`);
    error.status = response.status;
    error.payload = payload;
    throw error;
  }
  return payload?.data;
}

async function login(apiBase, loginId, password) {
  const response = await fetch(`${apiBase}/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ studentID: loginId, password }),
  });
  const payload = await response.json();
  if (!response.ok) throw new Error(`login ${loginId} failed: ${payload?.message || response.status}`);
  return payload.data;
}

async function waitUntil(timestamp) {
  const waitMs = new Date(timestamp).getTime() - Date.now() + 500;
  if (waitMs > 0) {
    log(`waiting ${Math.ceil(waitMs / 1000)}s for application/eligibility window`);
    await new Promise((resolve) => setTimeout(resolve, waitMs));
  }
}

function seriesBody(seriesId, maxSupply, windows) {
  return {
    seriesId,
    seasonId: '2026-fabric-test',
    categoryId: 'basketball',
    title: `Fabric concurrency ${seriesId}`,
    description: 'Non-financial badge concurrency acceptance fixture.',
    assetUri: '/badges/basketball-2026-demo/v1/badge.png',
    assetSha256: sha(`asset:${seriesId}`),
    metadataSha256: sha(`metadata:${seriesId}`),
    eligibilityPolicyHash: sha(`policy:${seriesId}`),
    issuanceMode: 'CHECK_IN_GUARANTEED',
    maxSupply,
    eligibilityOpenAt: windows.eligibilityOpen,
    eligibilityCloseAt: windows.eligibilityClose,
    claimOpenAt: windows.claimOpen,
    claimCloseAt: windows.claimClose,
    supplyRationale: `Test supply equals frozen activity capacity ${maxSupply}.`,
  };
}

async function prepareExistingTwoStudentFixture(options, actors, runId) {
  const seriesCount = options.rounds + (options.skipReplay ? 0 : 2);
  const estimatedLead = Math.max(45, options.rounds * 8 + 30);
  const leadSeconds = options.setupLeadSeconds || estimatedLead;
  const now = Date.now();
  const windows = {
    applicationClose: iso(now + leadSeconds * 1000),
    startsAt: iso(now + (leadSeconds + 20) * 1000),
    endsAt: iso(now + (leadSeconds + 3600) * 1000),
    eligibilityOpen: iso(now + Math.max(15, leadSeconds - 10) * 1000),
    eligibilityClose: iso(now + 24 * 3600_000),
    claimOpen: iso(now + Math.max(15, leadSeconds - 10) * 1000),
    claimClose: iso(now + 30 * 24 * 3600_000),
  };
  const activityId = `badge-fab-${runId}`;
  const seed = `badge-fabric-seed-${runId}-000000000000`;
  const roundSeries = Array.from({ length: options.rounds }, (_, index) => `bf-${runId}-r${index}`);
  const sameSeries = options.skipReplay ? null : `bf-${runId}-same`;
  const differentSeries = options.skipReplay ? null : `bf-${runId}-diff`;
  const allSeries = [...roundSeries, ...[sameSeries, differentSeries].filter(Boolean)];

  log(`creating one capacity-2 activity and ${seriesCount} linked series`);
  await request(options.apiBase, 'POST', '/activities', actors.organizer.token, {
    id: activityId,
    categoryId: 'basketball',
    title: 'Badge Fabric concurrency fixture',
    capacity: 2,
    applicationCloseAt: windows.applicationClose,
    startsAt: windows.startsAt,
    endsAt: windows.endsAt,
  });
  for (const seriesId of allSeries) {
    await request(options.apiBase, 'POST', '/badges/series', actors.admin.token, seriesBody(seriesId, 2, windows));
    await request(options.apiBase, 'POST', `/badges/series/${seriesId}/activities/${activityId}`, actors.admin.token, { eligibilityQuota: 2 });
    await request(options.apiBase, 'POST', `/badges/series/${seriesId}/activate`, actors.admin.token, {});
  }
  await request(options.apiBase, 'POST', `/activities/${activityId}/draw-commitment`, actors.verifier.token, { seed });
  for (const student of actors.students) {
    await request(options.apiBase, 'POST', `/activities/${activityId}/applications`, student.token, {});
  }
  await waitUntil(windows.applicationClose);
  await request(options.apiBase, 'POST', `/activities/${activityId}/draw`, actors.organizer.token, { seed });

  for (let index = 0; index < actors.students.length; index += 1) {
    const student = actors.students[index];
    const secret = `badge-fabric-ticket-${runId}-${index}-000000000000`;
    const claimed = await request(
      options.apiBase,
      'POST',
      `/activities/${activityId}/ticket`,
      student.token,
      { secret },
      `ticket-${runId}-${index}`,
    );
    const timeSlice = Math.floor(Date.now() / 30_000);
    await request(options.apiBase, 'POST', '/activities/check-in/verify', actors.operator.token, {
      ticketId: claimed.ticket.ticketId,
      secret,
      timeSlice,
    });
  }
  return { activityId, roundSeries, sameSeries, differentSeries };
}

async function endorseClaim(accountId, seriesId, refId) {
  const contract = await getContract(accountId, config.fabric.chaincode.activity);
  const proposal = contract.newProposal('ClaimMyBadge', {
    arguments: [seriesId, refId],
    endorsingOrganizations: ['PlatformMSP', 'StudentMSP'],
  });
  const transaction = await proposal.endorse();
  return { accountId, seriesId, refId, transaction, transactionId: transaction.getTransactionId() };
}

async function barrierSubmit(endorsed) {
  const submitted = await Promise.all(endorsed.map(async (item) => ({ ...item, commit: await item.transaction.submit() })));
  const statuses = await Promise.all(submitted.map(async (item) => {
    try {
      const status = await item.commit.getStatus();
      return {
        accountId: item.accountId,
        refId: item.refId,
        transactionId: item.transactionId,
        successful: status.successful,
        code: status.code,
        status: StatusNames[status.code] || String(status.code),
      };
    } catch (error) {
      return { accountId: item.accountId, refId: item.refId, transactionId: item.transactionId, successful: false, error: errorSummary(error) };
    }
  }));
  if (statuses.some((status) => status.error?.name === 'TypeError')) {
    throw new Error('harness failed to decode a Fabric commit status');
  }
  return statuses;
}

async function reconcileClaims(items) {
  const badge = createBadgeService({ mvccRetries: 5, reconcileDelays: [100, 250, 500, 1000] });
  return Promise.all(items.map(async ({ accountId, seriesId, refId }) => {
    const result = await badge.claimMyBadge(accountId, seriesId, refId);
    if (result.pending) throw new Error(`claim remained pending for ${seriesId}/${accountId}`);
    return result.data;
  }));
}

async function runTwoClientRounds(options, actors, fixture) {
  const rounds = [];
  for (let index = 0; index < fixture.roundSeries.length; index += 1) {
    const seriesId = fixture.roundSeries[index];
    const claims = actors.students.map((student, studentIndex) => ({
      accountId: student.userId,
      seriesId,
      refId: `bf-${index}-${studentIndex}-${crypto.randomBytes(4).toString('hex')}`,
    }));
    const endorsed = await Promise.all(claims.map((claim) => endorseClaim(claim.accountId, claim.seriesId, claim.refId)));
    const rawStatuses = await barrierSubmit(endorsed);
    assertBarrierStatuses(rawStatuses, 2, `FAB-01 ${seriesId}`);
    const awards = await reconcileClaims(claims);
    const series = await request(options.apiBase, 'GET', `/badges/series/${seriesId}`, actors.students[0].token);
    const serials = awards.map((view) => view.instance.serialNumber).sort((a, b) => a - b);
    if (series.issuedCount !== 2 || JSON.stringify(serials) !== '[1,2]') {
      throw new Error(`FAB-01 invariant failed for ${seriesId}: count=${series.issuedCount}, serials=${serials}`);
    }
    rounds.push({ seriesId, rawStatuses, finalIssuedCount: series.issuedCount, finalSerials: serials });
    log(`FAB-01 round ${index + 1}/${fixture.roundSeries.length} passed`);
  }
  return rounds;
}

async function runReplayFanout(options, actors, fixture) {
  if (options.skipReplay) return { skipped: true };
  const accountId = actors.students[0].userId;
  const sameRef = `same-${crypto.randomBytes(8).toString('hex')}`;
  const sameClaims = Array.from({ length: options.sameKey }, () => ({ accountId, seriesId: fixture.sameSeries, refId: sameRef }));
  const sameEndorsed = await Promise.all(sameClaims.map((claim) => endorseClaim(claim.accountId, claim.seriesId, claim.refId)));
  const sameRaw = await barrierSubmit(sameEndorsed);
  assertBarrierStatuses(sameRaw, options.sameKey, 'FAB-03 same-key');
  const sameViews = await reconcileClaims(sameClaims);

  const differentClaims = Array.from({ length: options.differentKeys }, (_, index) => ({
    accountId,
    seriesId: fixture.differentSeries,
    refId: `diff-${index}-${crypto.randomBytes(6).toString('hex')}`,
  }));
  const differentEndorsed = await Promise.all(differentClaims.map((claim) => endorseClaim(claim.accountId, claim.seriesId, claim.refId)));
  const differentRaw = await barrierSubmit(differentEndorsed);
  assertBarrierStatuses(differentRaw, options.differentKeys, 'FAB-03 different-keys');
  const differentViews = await reconcileClaims(differentClaims);

  const sameInstances = [...new Set(sameViews.map((view) => view.instance.instanceId))];
  const differentInstances = [...new Set(differentViews.map((view) => view.instance.instanceId))];
  const sameSeries = await request(options.apiBase, 'GET', `/badges/series/${fixture.sameSeries}`, actors.students[0].token);
  const differentSeries = await request(options.apiBase, 'GET', `/badges/series/${fixture.differentSeries}`, actors.students[0].token);
  if (sameInstances.length !== 1 || differentInstances.length !== 1 || sameSeries.issuedCount !== 1 || differentSeries.issuedCount !== 1) {
    throw new Error('FAB-03 invariant failed: concurrent replay created more than one award or serial');
  }
  return {
    sameKey: { requests: options.sameKey, rawStatuses: sameRaw, uniqueInstances: sameInstances, issuedCount: sameSeries.issuedCount },
    differentKeys: { requests: options.differentKeys, rawStatuses: differentRaw, uniqueInstances: differentInstances, issuedCount: differentSeries.issuedCount },
  };
}

function assertBarrierStatuses(statuses, expectedCount, label) {
  if (statuses.length !== expectedCount || statuses.some((status) => status.error?.name === 'TypeError')) {
    throw new Error(`${label} did not produce ${expectedCount} readable Fabric commit statuses`);
  }
  const valid = statuses.filter((status) => status.successful && status.code === 0).length;
  const invalid = statuses.filter((status) => !status.successful && status.code === StatusCode.MVCC_READ_CONFLICT).length;
  if (valid !== 1 || valid + invalid !== expectedCount) {
    throw new Error(`${label} expected 1 VALID and ${expectedCount - 1} MVCC_READ_CONFLICT statuses; got ${JSON.stringify(statuses)}`);
  }
}

async function expectDirectRejection(accountId, transactionName, args) {
  const contract = await getContract(accountId, config.fabric.chaincode.activity);
  try {
    await contract.submit(transactionName, { arguments: args.map(String), endorsingOrganizations: ['PlatformMSP', 'StudentMSP'] });
    return { transactionName, rejected: false };
  } catch (error) {
    return { transactionName, rejected: true, error: errorSummary(error) };
  }
}

async function runDirectGatewayRbac(options, actors, fixture, runId) {
  if (options.skipRbac) return { skipped: true };
  const baseline = await request(options.apiBase, 'GET', '/badges/series', actors.admin.token);
  const targets = [actors.students[0], actors.organizer, actors.operator, actors.verifier];
  const attempts = [];
  for (const actor of targets) {
    const createBody = seriesBody(`deny-${runId}-${actor.role}`, 2, {
      eligibilityOpen: iso(Date.now() + 3600_000), eligibilityClose: iso(Date.now() + 7200_000),
      claimOpen: iso(Date.now() + 3600_000), claimClose: iso(Date.now() + 10_800_000),
    });
    const actorAttempts = await Promise.all([
      expectDirectRejection(actor.userId, 'CreateBadgeSeries', [JSON.stringify(createBody)]),
      expectDirectRejection(actor.userId, 'LinkBadgeActivity', [fixture.roundSeries[0], fixture.activityId, '2']),
      expectDirectRejection(actor.userId, 'PauseBadgeSeries', [fixture.roundSeries[0], sha('unauthorized pause')]),
      expectDirectRejection(actor.userId, 'CloseBadgeSeries', [fixture.roundSeries[0]]),
      expectDirectRejection(actor.userId, 'FinalizeBadgeSeries', [fixture.roundSeries[0]]),
    ]);
    attempts.push({ role: actor.role, accountId: actor.userId, attempts: actorAttempts });
  }
  const after = await request(options.apiBase, 'GET', '/badges/series', actors.admin.token);
  const allRejected = attempts.every((entry) => entry.attempts.every((attempt) => attempt.rejected));
  const ledgerSeriesCountUnchanged = after.length === baseline.length;
  if (!allRejected || !ledgerSeriesCountUnchanged) throw new Error('FAB-05 direct Gateway RBAC invariant failed');
  return { allRejected, ledgerSeriesCountUnchanged, attempts };
}

async function runBulkClients(options, actors, runId) {
  if (!options.bulkClients) return { skipped: true, requestedClients: 0 };
  const count = options.bulkClients;
  const leadSeconds = options.setupLeadSeconds || Math.max(120, Math.ceil(count / 5) * 8 + 60);
  const now = Date.now();
  const windows = {
    applicationClose: iso(now + leadSeconds * 1000), startsAt: iso(now + (leadSeconds + 20) * 1000),
    endsAt: iso(now + (leadSeconds + 3600) * 1000), eligibilityOpen: iso(now + Math.max(30, leadSeconds - 20) * 1000),
    eligibilityClose: iso(now + 24 * 3600_000), claimOpen: iso(now + Math.max(30, leadSeconds - 20) * 1000),
    claimClose: iso(now + 30 * 24 * 3600_000),
  };
  const activityId = `bulk-${runId}`;
  const seriesId = `bulk-series-${runId}`;
  const seed = `bulk-fabric-seed-${runId}-000000000000`;
  const students = Array.from({ length: count }, (_, index) => ({
    userId: `acct_badgetest_${runId}_${String(index).padStart(3, '0')}`,
    role: 'student',
  }));
  log(`FAB-02 provisioning ${count} fresh StudentMSP identities (bounded concurrency 5)`);
  for (let offset = 0; offset < students.length; offset += 5) {
    await Promise.all(students.slice(offset, offset + 5).map((student) => registerAndEnrollUser(student.userId, 'StudentMSP', 'student')));
  }
  await activitySubmit(actors.organizer.userId, 'CreateActivity', [activityId, 'basketball', 'Bulk badge Fabric fixture', count, windows.applicationClose, windows.startsAt, windows.endsAt]);
  const badge = createBadgeService();
  await badge.createSeries(actors.admin.userId, seriesBody(seriesId, count, windows));
  await badge.linkActivity(actors.admin.userId, seriesId, activityId, count);
  await badge.activateSeries(actors.admin.userId, seriesId);
  await activitySubmit(actors.verifier.userId, 'SetDrawCommitment', [activityId, sha(seed)]);
  for (let offset = 0; offset < students.length; offset += 10) {
    await Promise.all(students.slice(offset, offset + 10).map((student) => activitySubmit(student.userId, 'Apply', [activityId])));
  }
  await waitUntil(windows.applicationClose);
  await activitySubmit(actors.organizer.userId, 'RunDraw', [activityId, seed]);
  for (let offset = 0; offset < students.length; offset += 10) {
    await Promise.all(students.slice(offset, offset + 10).map(async (student, localIndex) => {
      const index = offset + localIndex;
      const secret = `bulk-ticket-${runId}-${index}-0000000000000000`;
      const ticketReceipt = await activitySubmit(student.userId, 'ClaimTicket', [activityId], { claim: { secret, refId: `bulk-ticket-${index}` } });
      const ticket = await activityEvaluate(student.userId, 'GetMyTicket', [activityId]);
      const timeSlice = Math.floor(Date.now() / 30_000);
      await activitySubmit(actors.operator.userId, 'CheckIn', [], {
        checkIn: { ticketId: ticketReceipt.ticketId, secret, timeSlice },
      });
    }));
  }
  const claims = students.map((student, index) => ({ accountId: student.userId, seriesId, refId: `bulk-badge-${index}` }));
  log(`FAB-02 endorsing ${count} proposals before the submit barrier`);
  const endorsed = await Promise.all(claims.map((claim) => endorseClaim(claim.accountId, claim.seriesId, claim.refId)));
  const rawStatuses = await barrierSubmit(endorsed);
  const awards = await reconcileClaims(claims);
  const series = await badge.getSeries(actors.admin.userId, seriesId);
  const serials = awards.map((view) => view.instance.serialNumber).sort((a, b) => a - b);
  const completeSerialRange = serials.length === count && serials.every((serial, index) => serial === index + 1);
  if (series.issuedCount !== count || !completeSerialRange) throw new Error('FAB-02 100-client invariant failed');
  for (const student of students) closeGateway(student.userId);
  return {
    requestedClients: count,
    provisionedClients: students.length,
    rawStatusCounts: rawStatuses.reduce((counts, status) => ({ ...counts, [status.status || status.error?.name]: (counts[status.status || status.error?.name] || 0) + 1 }), {}),
    finalIssuedCount: series.issuedCount,
    completeSerialRange,
  };
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    process.stdout.write(`${usage()}\n`);
    return;
  }
  await loadRuntime();
  const runId = compactRunId();
  const startedAt = new Date().toISOString();
  const actors = {
    admin: { ...(await login(options.apiBase, 'admin01', 'eventchain-admin-2026')), role: 'admin' },
    organizer: { ...(await login(options.apiBase, 'organizer01', 'eventchain-organizer-2026')), role: 'organizer' },
    verifier: { ...(await login(options.apiBase, 'verifier01', 'eventchain-verifier-2026')), role: 'verifier' },
    operator: { ...(await login(options.apiBase, 'operator01', 'eventchain-operator-2026')), role: 'operator' },
    students: [
      { ...(await login(options.apiBase, '3220100001', 'eventchain-student-2026')), role: 'student' },
      { ...(await login(options.apiBase, '3220100002', 'eventchain-student-2026')), role: 'student' },
    ],
  };
  const identitiesToClose = [actors.admin, actors.organizer, actors.verifier, actors.operator, ...actors.students];
  try {
    const fixture = await prepareExistingTwoStudentFixture(options, actors, runId);
    const fab01 = await runTwoClientRounds(options, actors, fixture);
    const fab03 = await runReplayFanout(options, actors, fixture);
    const fab05 = await runDirectGatewayRbac(options, actors, fixture, runId);
    const fab02 = await runBulkClients(options, actors, runId);
    const result = {
      ok: true,
      runId,
      startedAt,
      completedAt: new Date().toISOString(),
      parameters: options,
      network: { channel: config.fabric.channelName, chaincode: config.fabric.chaincode.activity, endorsers: ['PlatformMSP', 'StudentMSP'] },
      results: { fab01: { rounds: fab01.length, details: fab01 }, fab02, fab03, fab05 },
      untested: ['FAB-04 fault injection', 'FAB-06 upgrade history audit'],
    };
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  } finally {
    for (const actor of identitiesToClose) closeGateway(actor.userId);
  }
}

main().catch((error) => {
  process.stdout.write(`${JSON.stringify({ ok: false, error: errorSummary(error) }, null, 2)}\n`);
  process.exitCode = 1;
});
