import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.event;
const CC_PRED = config.fabric.chaincode.prediction;

// GET /api/v1/events?status=PREDICTION_OPEN&type=basketball
router.get('/', async (req, res, next) => {
  try {
    // Public — use a platform admin identity for read-only queries
    const userId = req.headers.authorization ? undefined : 'admin-PlatformMSP';
    const queryUserId = req.user?.userId || userId || 'admin-PlatformMSP';

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

// Use auth middleware optionally for GET — pass through if no token
router.get('/', (req, _res, next) => {
  if (req.headers.authorization) {
    return authenticate(req, _res, next);
  }
  next();
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
    const { title, type, teams, ticketTotal, predictionOptions } = req.body;

    if (!title || !type || !teams || !predictionOptions) {
      const err = new Error('缺少必要字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'CreateEvent',
      JSON.stringify({ title, type, teams, ticketTotal: ticketTotal || 0, predictionOptions })
    );

    res.status(201).json({ error: false, data: result });
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
    const settlement = await submitTransaction(
      req.user.userId,
      CC_PRED,
      'Settle',
      req.params.id
    );

    res.json({ error: false, data: settlement });
  } catch (err) {
    next(err);
  }
});

export default router;
