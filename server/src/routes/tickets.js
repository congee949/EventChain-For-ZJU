import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.ticket;

// POST /api/v1/tickets/apply
// Body: { eventID }
router.post('/apply', authenticate, async (req, res, next) => {
  try {
    const { eventID } = req.body;
    if (!eventID) {
      const err = new Error('eventID 为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ApplyTicket',
      eventID
    );

    res.status(201).json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/lottery/:eventID  [Organizer]
router.post('/lottery/:eventID', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RunLottery',
      req.params.eventID
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/tickets/mine
router.get('/mine', authenticate, async (req, res, next) => {
  try {
    const tickets = await evaluateTransaction(
      req.user.userId,
      CC,
      'GetUserTickets',
      req.user.userId
    );
    res.json({ error: false, data: tickets || [] });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/claim/:ticketID
router.post('/claim/:ticketID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ClaimTicket',
      req.params.ticketID
    );

    // result includes { ticketID, claimHash } — frontend uses claimHash for QR
    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/tickets/verify/:ticketID
router.get('/verify/:ticketID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const result = await evaluateTransaction(
      queryUserId,
      CC,
      'VerifyTicket',
      req.params.ticketID,
      req.query.hash || ''
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/refund/:ticketID
router.post('/refund/:ticketID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RefundTicket',
      req.params.ticketID
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

export default router;
