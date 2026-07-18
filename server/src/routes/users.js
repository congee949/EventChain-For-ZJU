import { Router } from 'express';
import { authenticate } from '../middleware/auth.js';
import { findUserByAccountId } from '../services/wallet.js';
import { financeEvaluate } from '../services/financeService.js';

const router = Router();

// GET /api/v1/users/profile
router.get('/profile', authenticate, async (req, res, next) => {
  try {
    const userId = req.user.userId;

    const wallet = await financeEvaluate(userId, 'GetMyWallet');

    // Fetch user meta from SQLite
    const userRow = findUserByAccountId(userId);

    res.json({
      error: false,
      data: {
        userId,
        name: userRow?.name || userId,
        role: userRow?.role || 'student',
        wallet,
      },
    });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/users/leaderboard
export default router;
