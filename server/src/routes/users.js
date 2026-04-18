import { Router } from 'express';
import { authenticate } from '../middleware/auth.js';
import { evaluateTransaction } from '../services/fabricGateway.js';
import { findUser } from '../services/wallet.js';
import config from '../config/index.js';

const router = Router();

// GET /api/v1/users/profile
router.get('/profile', authenticate, async (req, res, next) => {
  try {
    const userId = req.user.userId;

    // Fetch balance from token chaincode
    const balance = await evaluateTransaction(
      userId,
      config.fabric.chaincode.token,
      'BalanceOf',
      userId
    );

    // Fetch prediction score
    const score = await evaluateTransaction(
      userId,
      config.fabric.chaincode.prediction,
      'GetUserScore',
      userId
    );

    // Fetch user meta from SQLite
    const userRow = findUser(userId);

    res.json({
      error: false,
      data: {
        userId,
        name: userRow?.name || userId,
        role: userRow?.role || 'student',
        balance: balance?.balance ?? 0,
        // placedBets: # of PlaceBet calls (incremented on every bet)
        // totalBets:  # of settled events (denominator for accuracy)
        placedBets: score?.placedBets ?? 0,
        totalBets: score?.totalBets ?? 0,
        correctBets: score?.correctBets ?? 0,
        accuracyRate: score?.accuracyRate ?? 0,
      },
    });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/users/leaderboard
router.get('/leaderboard', async (req, res, next) => {
  try {
    // This calls a CouchDB rich query inside prediction-cc
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const leaderboard = await evaluateTransaction(
      queryUserId,
      config.fabric.chaincode.prediction,
      'GetLeaderboard'
    );

    res.json({ error: false, data: leaderboard || [] });
  } catch (err) {
    next(err);
  }
});

export default router;
