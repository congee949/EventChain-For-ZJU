import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.event;
const CC_PRED = config.fabric.chaincode.prediction;

// Optional auth for GET — attach user identity if token is present
router.use('/', (req, _res, next) => {
  if (req.method === 'GET' && req.headers.authorization) {
    return authenticate(req, _res, next);
  }
  next();
});

// GET /api/v1/events?status=PREDICTION_OPEN&type=basketball
router.get('/', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';

    const { status, type } = req.query;
    const result = await evaluateTransaction(
      queryUserId,
      CC,
      'ListEvents',
      status || '',
      type || ''
    );
    res.json({ error: false, data: result || [] });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/events/:id
router.get('/:id', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const event = await evaluateTransaction(queryUserId, CC, 'QueryEvent', req.params.id);
    if (!event) {
      const err = new Error('赛事不存在');
      err.code = 'EVENT_NOT_FOUND';
      throw err;
    }

    // Also fetch current odds
    let odds = null;
    try {
      odds = await evaluateTransaction(
        queryUserId,
        CC_PRED,
        'GetOdds',
        req.params.id
      );
    } catch {
      // odds may not be available if event hasn't opened predictions
    }

    res.json({ error: false, data: { ...event, odds } });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/events  [Organizer]
router.post('/', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { eventID, title, type, teams, ticketTotal, predictionOptions } = req.body;

    if (!eventID || !title || !type || !teams || !predictionOptions || ticketTotal == null) {
      const err = new Error('缺少必要字段 (eventID, title, type, teams, ticketTotal, predictionOptions)');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // eventID becomes part of the chaincode state key (`event:<eventID>`).
    // Restrict to alphanumerics + dash/underscore so it can't collide with the
    // namespace separator or contain weird characters.
    if (!/^[A-Za-z0-9_-]{1,64}$/.test(eventID)) {
      const err = new Error('eventID 格式无效（仅允许字母数字_-，长度 1-64）');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Chaincode requires ticketTotal > 0 (event.go validates and rejects <= 0).
    // Don't fall back to 0 silently — that just shifts the error to chaincode.
    const ticketTotalNum = Number(ticketTotal);
    if (!Number.isInteger(ticketTotalNum) || ticketTotalNum <= 0) {
      const err = new Error('ticketTotal 必须为正整数');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Chaincode CreateEvent signature:
    // (eventID, title, eventType, teamsJSON, ticketTotalStr, optionsJSON)
    await submitTransaction(
      req.user.userId,
      CC,
      'CreateEvent',
      eventID,
      title,
      type,
      JSON.stringify(teams),
      String(ticketTotalNum),
      JSON.stringify(predictionOptions)
    );

    // Read back the canonical event from chaincode (includes status, createdAt, etc.)
    // so the response isn't a fabricated stub.
    const created = await evaluateTransaction(req.user.userId, CC, 'QueryEvent', eventID);
    res.status(201).json({ error: false, data: created });
  } catch (err) {
    next(err);
  }
});

// PUT /api/v1/events/:id/status  [Organizer]
router.put('/:id/status', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { status } = req.body;
    if (!status) {
      const err = new Error('缺少 status 字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'UpdateStatus',
      req.params.id,
      status
    );

    // When opening prediction market, initialize the prediction pool
    // (the prediction chaincode requires a pool to exist before bets can be placed)
    //
    // CONSISTENCY WINDOW: UpdateStatus and InitializePool are two separate
    // chaincode submits. Fabric has no multi-chaincode atomic transaction,
    // so if InitializePool fails after UpdateStatus already committed, the
    // event is stuck in PREDICTION_OPEN with no pool, and the state machine
    // is forward-only — there's no way to revert.
    //
    // This is unhandled on purpose for the demo. In practice InitializePool
    // doesn't fail unless the chaincode itself crashes. For real deploys,
    // wrap this in try/catch and expose POST /events/:id/init-pool as a
    // recovery endpoint so admins can re-trigger it.
    if (status === 'PREDICTION_OPEN') {
      const evt = await evaluateTransaction(req.user.userId, CC, 'QueryEvent', req.params.id);
      if (evt && evt.predictionOptions && evt.predictionOptions.length >= 2) {
        await submitTransaction(
          req.user.userId,
          CC_PRED,
          'InitializePool',
          req.params.id,
          evt.predictionOptions[0],
          evt.predictionOptions[1]
        );
      }
    }

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// PUT /api/v1/events/:id/result  [Organizer]
// Records the result and triggers settlement in prediction-cc.
router.put('/:id/result', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { outcome } = req.body;
    if (!outcome) {
      const err = new Error('缺少 outcome 字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // 1. Record result on the event
    await submitTransaction(req.user.userId, CC, 'UpdateResult', req.params.id, outcome);

    // 2. Trigger settlement in prediction chaincode
    // Chaincode Settle signature: (eventID, winningOption)
    const settlement = await submitTransaction(
      req.user.userId,
      CC_PRED,
      'Settle',
      req.params.id,
      outcome
    );

    res.json({ error: false, data: settlement });
  } catch (err) {
    next(err);
  }
});

export default router;
