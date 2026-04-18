import { Router } from 'express';
import { authenticate } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.prediction;

// POST /api/v1/predictions/bet
// Body: { eventID, option, amount }
router.post('/bet', authenticate, async (req, res, next) => {
  try {
    const { eventID, option, amount } = req.body;

    if (!eventID || !option || !amount || amount <= 0) {
      const err = new Error('eventID, option, amount(>0) 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Chaincode PlaceBet signature: (eventID, userID, option, amountStr)
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'PlaceBet',
      eventID,
      req.user.userId,
      option,
      String(amount)
    );

    // result: { shares, newOddsA, newOddsB }
    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/odds/:eventID
router.get('/odds/:eventID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const odds = await evaluateTransaction(queryUserId, CC, 'GetOdds', req.params.eventID);
    res.json({ error: false, data: odds });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/pool/:eventID
router.get('/pool/:eventID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const pool = await evaluateTransaction(queryUserId, CC, 'GetPool', req.params.eventID);
    res.json({ error: false, data: pool });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/mine
router.get('/mine', authenticate, async (req, res, next) => {
  try {
    const bets = await evaluateTransaction(req.user.userId, CC, 'GetUserBets', req.user.userId, '');
    res.json({ error: false, data: bets || [] });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/score
router.get('/score', authenticate, async (req, res, next) => {
  try {
    const score = await evaluateTransaction(req.user.userId, CC, 'GetUserScore', req.user.userId);
    res.json({ error: false, data: score });
  } catch (err) {
    next(err);
  }
});

export default router;
