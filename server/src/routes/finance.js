import crypto from 'node:crypto';
import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import {
  asJSON, asText, financeEvaluate, financeSubmit, hashEvidence, idempotencyKey, ok, validation,
} from '../services/financeService.js';
import { processSettlement, runClaimCycle } from '../services/claimWorker.js';

const router = Router();
router.use(authenticate);

const route = (handler) => async (req, res, next) => {
  try { await handler(req, res); } catch (error) { next(error); }
};

router.get('/config', route(async (req, res) => ok(res, await financeEvaluate(req.user.userId, 'GetConfig'))));
router.get('/categories', route(async (req, res) => ok(res, await financeEvaluate(req.user.userId, 'ListCategories'))));
router.get('/wallet', route(async (req, res) => ok(res, await financeEvaluate(req.user.userId, 'GetMyWallet'))));
router.get('/wallet/:accountId', requireRole('admin', 'operator'), route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetWalletForAccount', [req.params.accountId]))));

router.post('/admin/initialize', requireRole('admin'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'InitializeV2'), 201)));
router.post('/admin/categories', requireRole('admin'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'CreateCategory', [asText(req.body.id, 'id'), asText(req.body.name, 'name')]);
  ok(res, data, 201);
}));
router.post('/admin/demo/a-credit', requireRole('admin'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'CreditDemoA', [
    asText(req.body.accountId, 'accountId'), asText(req.body.amount, 'amount'),
    asText(req.body.reason, 'reason'), idempotencyKey(req),
  ]);
  ok(res, data, 201);
}));
router.post('/admin/demo/bonus-credit', requireRole('admin'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'CreditDemoBonus', [
    asText(req.body.accountId, 'accountId'), asText(req.body.categoryId, 'categoryId'),
    asText(req.body.amount, 'amount'), idempotencyKey(req),
  ]);
  ok(res, data, 201);
}));

router.post('/wallet/convert', requireRole('student'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'ConvertAToPaidB', [
    asText(req.body.categoryId, 'categoryId'), asText(req.body.amount, 'amount'), idempotencyKey(req),
  ]);
  ok(res, data, 201);
}));
router.post('/wallet/redeem', requireRole('student'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'RedeemPaidBToA', [
    asText(req.body.categoryId, 'categoryId'), asText(req.body.amount, 'amount'), idempotencyKey(req),
  ]);
  ok(res, data, 201);
}));
router.post('/wallet/transfer', requireRole('student'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'TransferPaidB', [
    asText(req.body.categoryId, 'categoryId'), asText(req.body.toAccountId, 'toAccountId'),
    asText(req.body.amount, 'amount'), idempotencyKey(req),
  ]);
  ok(res, data, 201);
}));
router.post('/wallet/sweep-expired', route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'SweepExpiredLots', [
    req.user.userId, asText(req.body.categoryId, 'categoryId'), asText(req.body.limit ?? 100, 'limit'),
  ]);
  ok(res, data);
}));

router.get('/markets', route(async (req, res) => ok(res, await financeEvaluate(req.user.userId, 'ListMarkets'))));
router.get('/markets/:marketId', route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetMarket', [req.params.marketId]))));
router.post('/markets', requireRole('organizer'), route(async (req, res) => {
  const stakeBucket = String(req.body.stakeBucket || 'BONUS').toUpperCase();
  if (!['BONUS', 'PAID'].includes(stakeBucket)) validation('stakeBucket must be BONUS or PAID');
  const data = await financeSubmit(req.user.userId, 'CreateMarket', [
    asText(req.body.marketId, 'marketId'), asText(req.body.eventId, 'eventId'),
    asText(req.body.categoryId, 'categoryId'), asJSON(req.body.outcomes, 'outcomes'),
    asText(req.body.closeAt, 'closeAt'), stakeBucket, asText(req.body.marketCap, 'marketCap', { optional: true }),
  ]);
  ok(res, data, 201);
}));
for (const [path, fn, roles] of [
  ['open', 'OpenMarket', ['organizer', 'admin']], ['lock', 'LockMarket', ['organizer', 'admin']],
  ['confirm', 'ConfirmResult', ['verifier', 'admin']], ['finalize', 'FinalizeMarket', ['operator', 'admin']],
]) {
  router.post(`/markets/:marketId/${path}`, requireRole(...roles), route(async (req, res) =>
    ok(res, await financeSubmit(req.user.userId, fn, [req.params.marketId]))));
}
router.post('/markets/:marketId/positions', requireRole('student'), route(async (req, res) => {
  const request = {
    marketId: req.params.marketId,
    outcomeId: asText(req.body.outcomeId, 'outcomeId'),
    amount: asText(req.body.amount, 'amount'),
    refId: idempotencyKey(req),
  };
  ok(res, await financeSubmit(req.user.userId, 'PlacePosition', [], { order: request }), 201);
}));
router.post('/markets/:marketId/result', requireRole('organizer'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'ProposeResult', [
    req.params.marketId, asText(req.body.outcomeId, 'outcomeId'), hashEvidence(req.body.evidenceHash || req.body.evidence),
  ]);
  ok(res, data);
}));
router.post('/markets/:marketId/challenge', requireRole('student'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'ChallengeResult', [
    req.params.marketId, hashEvidence(req.body.reasonHash || req.body.reason),
  ]);
  ok(res, data);
}));
router.post('/markets/:marketId/vote', requireRole('arbitrator'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'VoteDispute', [req.params.marketId, asText(req.body.outcomeId, 'outcomeId')]))));
router.post('/admin/arbitrators/:accountId', requireRole('admin'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'AddArbitrator', [req.params.accountId]), 201)));

router.get('/markets/:marketId/settlements/:epoch', route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetSettlement', [req.params.marketId, req.params.epoch]))));
router.get('/markets/:marketId/settlements/:epoch/claim', route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetClaim', [req.params.marketId, req.params.epoch, req.user.userId]))));
router.post('/markets/:marketId/settlements/:epoch/claim', route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'ClaimOne', [req.params.marketId, req.params.epoch, req.user.userId]), 201)));
router.post('/markets/:marketId/settlements/:epoch/mature', route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'MatureClaimOne', [req.params.marketId, req.params.epoch, req.user.userId]))));
router.post('/markets/:marketId/settlements/:epoch/activate', requireRole('operator', 'admin'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'ActivateSettlement', [req.params.marketId, req.params.epoch]))));
router.post('/markets/:marketId/settlements/:epoch/close', requireRole('operator', 'admin'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'CloseSettlement', [req.params.marketId, req.params.epoch]))));
router.post('/admin/claims/process', requireRole('operator', 'admin'), route(async (_req, res) =>
  ok(res, await runClaimCycle())));
router.post('/admin/claims/:marketId/:epoch', requireRole('operator', 'admin'), route(async (req, res) =>
  ok(res, await processSettlement(req.user.userId, req.params.marketId, req.params.epoch))));

router.get('/offers', route(async (req, res) => ok(res, await financeEvaluate(req.user.userId, 'ListServiceOffers'))));
router.get('/offers/:offerId', route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetServiceOffer', [req.params.offerId]))));
router.post('/offers', requireRole('organizer'), route(async (req, res) => {
  const data = await financeSubmit(req.user.userId, 'CreateServiceOffer', [
    asText(req.body.offerId, 'offerId'), asText(req.body.categoryId, 'categoryId'),
    asText(req.body.offerType, 'offerType'), asText(req.body.title, 'title'), asText(req.body.price, 'price'),
    asText(req.body.cancellationBps, 'cancellationBps', { optional: true }),
    asText(req.body.guaranteeBudget, 'guaranteeBudget', { optional: true }),
    asText(req.body.bonusBudget, 'bonusBudget', { optional: true }), asText(req.body.inventory, 'inventory'),
  ]);
  ok(res, data, 201);
}));
router.get('/orders/:orderId', route(async (req, res) =>
  ok(res, await financeEvaluate(req.user.userId, 'GetServiceOrder', [req.params.orderId]))));
router.post('/orders', requireRole('student'), route(async (req, res) => {
  const order = {
    offerId: asText(req.body.offerId, 'offerId'), scheduledAt: asText(req.body.scheduledAt, 'scheduledAt'), refId: idempotencyKey(req),
  };
  ok(res, await financeSubmit(req.user.userId, 'CreateServiceOrder', [], { order }), 201);
}));
for (const [path, fn, roles] of [
  ['fulfill', 'FulfillServiceOrder', ['organizer']], ['delay-compensation', 'CompensateServiceDelay', ['organizer', 'operator', 'admin']],
]) {
  router.post(`/orders/:orderId/${path}`, requireRole(...roles), route(async (req, res) =>
    ok(res, await financeSubmit(req.user.userId, fn, [req.params.orderId]))));
}
router.post('/orders/:orderId/cancel', route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'CancelServiceOrder', [req.params.orderId, asText(req.body.reason, 'reason')]))));

router.post('/admin/emergency-approvers/:accountId', requireRole('admin'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'AddEmergencyApprover', [req.params.accountId]), 201)));
router.post('/emergencies', requireRole('admin', 'operator', 'verifier', 'arbitrator'), route(async (req, res) => {
  const actionId = asText(req.body.actionId, 'actionId');
  const nonce = asText(req.body.nonce || crypto.randomBytes(16).toString('hex'), 'nonce');
  const data = await financeSubmit(req.user.userId, 'ProposeEmergency', [
    actionId, asText(req.body.actionType, 'actionType'), asJSON(req.body.payload, 'payload'), nonce, asText(req.body.expiresAt, 'expiresAt'),
  ]);
  ok(res, data, 201);
}));
router.post('/emergencies/:actionId/approve', requireRole('admin', 'operator', 'verifier', 'arbitrator'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'ApproveEmergency', [req.params.actionId, asText(req.body.payloadHash, 'payloadHash')]))));
router.post('/emergencies/:actionId/execute', requireRole('admin', 'operator'), route(async (req, res) =>
  ok(res, await financeSubmit(req.user.userId, 'ExecuteEmergency', [req.params.actionId]))));

export default router;
