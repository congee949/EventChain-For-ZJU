import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.ticket;
const CC_EVENT = config.fabric.chaincode.event;

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

    const event = await evaluateTransaction(req.user.userId, CC_EVENT, 'QueryEvent', eventID);
    if (event?.status !== 'TICKET_OPEN') {
      const err = new Error('赛事当前未开放购票申请');
      err.code = 'INVALID_STATUS';
      throw err;
    }

    // Chaincode ApplyTicket signature: (eventID, userID)
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ApplyTicket',
      eventID,
      req.user.userId
    );

    res.status(201).json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/lottery/:eventID  [Organizer]
// Body: { ticketCount }  — number of winners to draw
router.post('/lottery/:eventID', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { ticketCount } = req.body;
    const ticketCountNum = Number(ticketCount);
    if (!Number.isInteger(ticketCountNum) || ticketCountNum <= 0) {
      const err = new Error('ticketCount 必须为正整数');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const event = await evaluateTransaction(
      req.user.userId,
      CC_EVENT,
      'QueryEvent',
      req.params.eventID
    );
    if (event?.status !== 'TICKET_OPEN') {
      const err = new Error('赛事当前不允许执行抽签');
      err.code = 'INVALID_STATUS';
      throw err;
    }
    if (ticketCountNum > event.ticketTotal) {
      const err = new Error(`抽签人数不能超过票务总量 ${event.ticketTotal}`);
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Chaincode RunLottery signature: (eventID, ticketCountStr)
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RunLottery',
      req.params.eventID,
      String(ticketCountNum)
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

// GET /api/v1/tickets/applications/mine
router.get('/applications/mine', authenticate, async (req, res, next) => {
  try {
    const applications = await evaluateTransaction(
      req.user.userId,
      CC,
      'GetUserApplications',
      req.user.userId
    );
    res.json({ error: false, data: applications || [] });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/claim/:eventID
// Chaincode ClaimTicket(eventID, userID) — claims the ticket the caller won
// in the lottery for that event. Route param is the EVENT id, not a ticket id.
router.post('/claim/:eventID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ClaimTicket',
      req.params.eventID,
      req.user.userId
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
// Chaincode RefundTicket(ticketID, userID)
router.post('/refund/:ticketID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RefundTicket',
      req.params.ticketID,
      req.user.userId
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

export default router;
