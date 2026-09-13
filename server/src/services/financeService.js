import crypto from 'node:crypto';
import config from '../config/index.js';
import { evaluateTransactionOptions, submitTransactionOptions } from './fabricGateway.js';

const FINANCE_ENDORSERS = ['PlatformMSP', 'StudentMSP'];
const FINANCE_READERS = ['PlatformMSP'];

// Shared validation for request idempotency keys.  Individual domains may
// derive their own ledger reference from this opaque client value; finance
// keeps passing its existing key through unchanged.
export const IDEMPOTENCY_PATTERN = /^[A-Za-z0-9._:-]{8,128}$/;

export function asText(value, field, { optional = false } = {}) {
  if ((value === undefined || value === null || value === '') && optional) return '';
  if (typeof value !== 'string' && typeof value !== 'number') validation(`${field} must be a string or number`);
  const text = String(value).trim();
  if (!text || text.length > 256) validation(`${field} is required`);
  return text;
}

export function asJSON(value, field) {
  if (value === undefined || value === null) validation(`${field} is required`);
  const text = JSON.stringify(value);
  if (text.length > 8192) validation(`${field} is too large`);
  return text;
}

export function idempotencyKey(req) {
  const value = req.get('Idempotency-Key');
  if (!value || !IDEMPOTENCY_PATTERN.test(value)) {
    validation('Idempotency-Key header must contain 8-128 safe characters');
  }
  return value;
}

export function hashEvidence(value) {
  const text = asText(value, 'evidence');
  if (/^[0-9a-f]{64}$/i.test(text)) return text.toLowerCase();
  return crypto.createHash('sha256').update(text, 'utf8').digest('hex');
}

export function validation(message) {
  const error = new Error(message);
  error.code = 'VALIDATION_ERROR';
  throw error;
}

export async function financeSubmit(userId, fn, args = [], transientData) {
  const transient = transientData
    ? Object.fromEntries(Object.entries(transientData).map(([key, value]) => [key, Buffer.from(JSON.stringify(value))]))
    : undefined;
  return submitTransactionOptions(userId, config.fabric.chaincode.finance, fn, {
    arguments: args.map(String),
    transientData: transient,
    endorsingOrganizations: FINANCE_ENDORSERS,
  });
}

export async function financeEvaluate(userId, fn, args = []) {
  return evaluateTransactionOptions(userId, config.fabric.chaincode.finance, fn, {
    arguments: args.map(String),
    endorsingOrganizations: FINANCE_READERS,
  });
}

export function ok(res, data, status = 200) {
  return res.status(status).json({ error: false, data });
}
