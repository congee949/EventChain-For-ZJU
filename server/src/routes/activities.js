import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { activityEvaluate, activitySubmit } from '../services/activityService.js';
import { asText, financeSubmit, hashEvidence, idempotencyKey, ok, validation } from '../services/financeService.js';

const router = Router();
router.use(authenticate);
const route = (handler) => async (req, res, next) => { try { await handler(req, res); } catch (error) { next(error); } };

router.get('/', route(async (req, res) => ok(res, await activityEvaluate(req.user.userId, 'ListActivities'))));
router.get('/:activityId', route(async (req, res) => ok(res, await activityEvaluate(req.user.userId, 'GetActivity', [req.params.activityId]))));
router.post('/', requireRole('organizer'), route(async (req, res) => {
  const result = await activitySubmit(req.user.userId, 'CreateActivity', [
    asText(req.body.id, 'id'), asText(req.body.categoryId, 'categoryId'), asText(req.body.title, 'title'),
    asText(req.body.capacity, 'capacity'), asText(req.body.applicationCloseAt, 'applicationCloseAt'),
    asText(req.body.startsAt, 'startsAt'), asText(req.body.endsAt, 'endsAt'),
  ]);
  ok(res, result, 201);
}));
router.post('/:activityId/draw-commitment', requireRole('verifier', 'admin'), route(async (req, res) =>
  ok(res, await activitySubmit(req.user.userId, 'SetDrawCommitment', [req.params.activityId, hashEvidence(req.body.seedHash || req.body.seed)]))));
router.post('/:activityId/applications', requireRole('student'), route(async (req, res) =>
  ok(res, await activitySubmit(req.user.userId, 'Apply', [req.params.activityId]), 201)));
router.get('/:activityId/application', requireRole('student'), route(async (req, res) =>
  ok(res, await activityEvaluate(req.user.userId, 'GetMyApplication', [req.params.activityId]))));
router.post('/:activityId/draw', requireRole('organizer'), route(async (req, res) =>
  ok(res, await activitySubmit(req.user.userId, 'RunDraw', [req.params.activityId, asText(req.body.seed, 'seed')]))));
router.put('/:activityId/status', requireRole('organizer', 'admin'), route(async (req, res) =>
  ok(res, await activitySubmit(req.user.userId, 'UpdateActivityStatus', [req.params.activityId, asText(req.body.status, 'status')]))));

router.get('/:activityId/ticket', requireRole('student'), route(async (req, res) =>
  ok(res, await activityEvaluate(req.user.userId, 'GetMyTicket', [req.params.activityId]))));
router.post('/:activityId/ticket', requireRole('student'), route(async (req, res) => {
  const secret = asText(req.body.secret, 'secret');
  if (secret.length < 32 || secret.length > 256) validation('secret must contain 32-256 characters');
  const result = await activitySubmit(req.user.userId, 'ClaimTicket', [], {
    claim: { secret, refId: idempotencyKey(req) },
  });
  ok(res, { ticket: result }, 201);
}));

router.post('/check-in/verify', requireRole('operator', 'admin'), route(async (req, res) => {
  const proof = {
    ticketId: asText(req.body.ticketId, 'ticketId'),
    secret: asText(req.body.secret, 'secret'),
    timeSlice: Number(req.body.timeSlice),
  };
  if (!Number.isSafeInteger(proof.timeSlice)) validation('timeSlice must be an integer');
  const receipt = await activitySubmit(req.user.userId, 'CheckIn', [], { checkIn: proof });
  try {
    const reward = await financeSubmit(req.user.userId, 'GrantCheckInBonus', [
      receipt.accountId, receipt.categoryId, receipt.activityId, receipt.refId,
    ]);
    ok(res, { receipt, reward, rewardPending: false });
  } catch (error) {
    res.status(202).json({
      error: false,
      data: { receipt, reward: null, rewardPending: true, retryable: true },
      warning: `签到已确认，奖励待重试：${error.message}`,
    });
  }
}));

export default router;
