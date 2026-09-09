import { Router } from 'express';
import crypto from 'node:crypto';
import { authenticate, requireRole } from '../middleware/auth.js';
import { BadgeHttpError, badgeService as defaultBadgeService } from '../services/badgeService.js';
import { idempotencyKey } from '../services/financeService.js';

const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/;
// This is deliberately narrower than the HTTP idempotency-key syntax. The
// activity chaincode persists badge refs and accepts only lowercase
// [a-z0-9_-], 8-64 characters.
const REF_PATTERN = /^[a-z0-9][a-z0-9_-]{7,63}$/;
const HASH_PATTERN = /^[0-9a-f]{64}$/;

function validation(message) {
  const error = new Error(message);
  error.code = 'VALIDATION_ERROR';
  throw error;
}

function id(value, field) {
  if (typeof value !== 'string' || !ID_PATTERN.test(value)) validation(`${field} is invalid`);
  return value;
}

function emptyBody(req) {
  if (req.body == null) return;
  if (typeof req.body !== 'object' || Array.isArray(req.body) || Object.keys(req.body).length !== 0) {
    validation('claim body must be empty');
  }
}

function objectBody(req) {
  if (!req.body || typeof req.body !== 'object' || Array.isArray(req.body)) {
    validation('JSON object body is required');
  }
  return req.body;
}

function positiveInteger(value, field) {
  const number = Number(value);
  if (!Number.isSafeInteger(number) || number < 1 || number > 100000) {
    validation(`${field} must be an integer from 1 to 100000`);
  }
  return number;
}

function reasonHash(value) {
  if (typeof value !== 'string' || !HASH_PATTERN.test(value)) {
    validation('reasonHash must be a lowercase SHA-256 digest');
  }
  return value;
}

/**
 * Map a client idempotency key to an opaque, chaincode-safe badge ref. The
 * account id is part of the digest so a key reused by two students cannot
 * collide, while retries by the same student remain stable.
 */
export function badgeRefId(accountId, incomingKey) {
  const digest = crypto.createHash('sha256')
    .update(`badge:${accountId}\u0000${incomingKey}`, 'utf8')
    .digest('hex');
  return `b_${digest.slice(0, 61)}`;
}

function responseDataWithRef(data, refId) {
  if (data && typeof data === 'object' && !Array.isArray(data)) return { ...data, refId };
  return { result: data, refId };
}

function sendBadgeError(res, error) {
  if (!(error instanceof BadgeHttpError)) return false;
  res.status(error.status).json({
    error: true,
    code: error.code,
    message: error.message,
    ...(error.data === undefined ? {} : { data: error.data }),
  });
  return true;
}

export function createBadgesRouter(service = defaultBadgeService) {
  const router = Router();
  router.use(authenticate);

  const route = (handler) => async (req, res, next) => {
    try {
      await handler(req, res);
    } catch (error) {
      if (!sendBadgeError(res, error)) next(error);
    }
  };

  router.get('/series', route(async (req, res) => {
    res.json({ error: false, data: await service.listSeries(req.user.userId) });
  }));
  router.get('/my-awards', route(async (req, res) => {
    res.json({ error: false, data: await service.listMyAwards(req.user.userId) });
  }));
  router.get('/public-instances/:instanceId', route(async (req, res) => {
    const data = await service.getPublicInstanceById(req.user.userId, id(req.params.instanceId, 'instanceId'));
    res.json({ error: false, data });
  }));
  router.get('/series/:seriesId', route(async (req, res) => {
    const data = await service.getSeries(req.user.userId, id(req.params.seriesId, 'seriesId'));
    res.json({ error: false, data });
  }));
  router.get('/series/:seriesId/activities', route(async (req, res) => {
    const data = await service.listActivityLinks(req.user.userId, id(req.params.seriesId, 'seriesId'));
    res.json({ error: false, data });
  }));
  router.get('/series/:seriesId/my-award', route(async (req, res) => {
    const data = await service.getMyAward(req.user.userId, id(req.params.seriesId, 'seriesId'));
    res.json({ error: false, data });
  }));
  router.get('/series/:seriesId/claims/:refId', route(async (req, res) => {
    const seriesId = id(req.params.seriesId, 'seriesId');
    const refId = req.params.refId;
    if (!REF_PATTERN.test(refId)) validation('refId is invalid');
    const result = await service.getClaimStatus(req.user.userId, seriesId, refId);
    if (!result.pending) return res.json({ error: false, data: result.data });
    return res.status(202).json({
      error: false,
      code: 'BADGE_CLAIM_PENDING',
      data: result.data,
    });
  }));
  router.post('/series/:seriesId/claim', requireRole('student'), route(async (req, res) => {
    emptyBody(req);
    const seriesId = id(req.params.seriesId, 'seriesId');
    const refId = badgeRefId(req.user.userId, idempotencyKey(req));
    const result = await service.claimMyBadge(req.user.userId, seriesId, refId);
    const data = responseDataWithRef(result.data, refId);
    if (result.pending) {
      return res.status(202).json({ error: false, code: result.code, data });
    }
    return res.status(201).json({ error: false, data });
  }));

  router.post('/series', requireRole('admin'), route(async (req, res) => {
    const data = await service.createSeries(req.user.userId, objectBody(req));
    res.status(201).json({ error: false, data });
  }));
  router.post('/series/:seriesId/activities/:activityId', requireRole('admin'), route(async (req, res) => {
    const data = await service.linkActivity(
      req.user.userId,
      id(req.params.seriesId, 'seriesId'),
      id(req.params.activityId, 'activityId'),
      positiveInteger(objectBody(req).eligibilityQuota, 'eligibilityQuota'),
    );
    res.status(201).json({ error: false, data });
  }));
  router.post('/series/:seriesId/activate', requireRole('admin'), route(async (req, res) => {
    emptyBody(req);
    res.json({ error: false, data: await service.activateSeries(req.user.userId, id(req.params.seriesId, 'seriesId')) });
  }));
  router.post('/series/:seriesId/pause', requireRole('admin'), route(async (req, res) => {
    const body = objectBody(req);
    if (Object.keys(body).some((key) => key !== 'reasonHash')) validation('pause body only accepts reasonHash');
    const data = await service.pauseSeries(
      req.user.userId,
      id(req.params.seriesId, 'seriesId'),
      reasonHash(body.reasonHash),
    );
    res.json({ error: false, data });
  }));
  for (const [path, method] of [
    ['resume', 'resumeSeries'],
    ['close', 'closeSeries'],
    ['finalize', 'finalizeSeries'],
  ]) {
    router.post(`/series/:seriesId/${path}`, requireRole('admin'), route(async (req, res) => {
      emptyBody(req);
      const data = await service[method](req.user.userId, id(req.params.seriesId, 'seriesId'));
      res.json({ error: false, data });
    }));
  }

  return router;
}

export default createBadgesRouter();
