import fs from 'node:fs';
import path from 'node:path';
import Database from 'better-sqlite3';
import bcrypt from 'bcrypt';
import crypto from 'node:crypto';
import config from '../config/index.js';

// ---- SQLite for password hashes ----

let db;

export function initDB() {
  const dir = path.dirname(config.dbPath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  fs.chmodSync(dir, 0o700);

  db = new Database(config.dbPath);
  fs.chmodSync(config.dbPath, 0o600);
  db.pragma('journal_mode = WAL');

  db.exec(`
    CREATE TABLE IF NOT EXISTS users_v2 (
      account_id        TEXT PRIMARY KEY,
      login_lookup      TEXT NOT NULL UNIQUE,
      login_ciphertext  TEXT NOT NULL,
      login_iv          TEXT NOT NULL,
      login_tag         TEXT NOT NULL,
      name              TEXT NOT NULL,
      password          TEXT NOT NULL,
      org_msp           TEXT NOT NULL DEFAULT 'StudentMSP',
      role              TEXT NOT NULL DEFAULT 'student',
      created_at        TEXT NOT NULL DEFAULT (datetime('now'))
    )
  `);
}

export function getDB() {
  return db;
}

function normalizeLoginId(loginId) {
  return String(loginId || '').trim().toLowerCase();
}

function keyBuffer(hex, label) {
  if (!/^[0-9a-f]{64}$/i.test(hex || '')) {
    throw new Error(`${label} must be a 32-byte hex key`);
  }
  return Buffer.from(hex, 'hex');
}

function lookupLogin(loginId) {
  return crypto.createHmac('sha256', keyBuffer(config.identity.lookupKey, 'IDENTITY_LOOKUP_KEY'))
    .update(normalizeLoginId(loginId))
    .digest('hex');
}

function encryptLogin(loginId) {
  const iv = crypto.randomBytes(12);
  const cipher = crypto.createCipheriv('aes-256-gcm', keyBuffer(config.identity.encryptionKey, 'IDENTITY_ENCRYPTION_KEY'), iv);
  const ciphertext = Buffer.concat([cipher.update(normalizeLoginId(loginId), 'utf8'), cipher.final()]);
  return { ciphertext: ciphertext.toString('base64'), iv: iv.toString('base64'), tag: cipher.getAuthTag().toString('base64') };
}

export function createUser(accountId, loginId, name, passwordHash, orgMSP = 'StudentMSP', role = 'student') {
  const encrypted = encryptLogin(loginId);
  const stmt = db.prepare(
    `INSERT INTO users_v2
      (account_id, login_lookup, login_ciphertext, login_iv, login_tag, name, password, org_msp, role)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
  );
  stmt.run(accountId, lookupLogin(loginId), encrypted.ciphertext, encrypted.iv, encrypted.tag, name, passwordHash, orgMSP, role);
}

export function findUser(loginId) {
  return db.prepare('SELECT * FROM users_v2 WHERE login_lookup = ?').get(lookupLogin(loginId));
}

export function findUserByAccountId(accountId) {
  return db.prepare('SELECT * FROM users_v2 WHERE account_id = ?').get(accountId);
}

export function findFirstUserByRole(role) {
  return db.prepare('SELECT * FROM users_v2 WHERE role = ? ORDER BY created_at, account_id LIMIT 1').get(role);
}

export async function hashPassword(plain) {
  return bcrypt.hash(plain, 10);
}

export async function verifyPassword(plain, hash) {
  return bcrypt.compare(plain, hash);
}

// ---- Fabric file-system wallet for X.509 identities ----

export function walletDir() {
  if (!fs.existsSync(config.walletPath)) {
    fs.mkdirSync(config.walletPath, { recursive: true, mode: 0o700 });
  }
  fs.chmodSync(config.walletPath, 0o700);
  return config.walletPath;
}

function identityPath(userId) {
  const safeId = String(userId || '');
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(safeId)) {
    throw new Error('invalid identity id');
  }
  return path.join(walletDir(), `${safeId}.json`);
}

export function putIdentity(userId, mspId, certificate, privateKey) {
  const targetPath = identityPath(userId);
  const temporaryPath = `${targetPath}.${process.pid}.tmp`;
  const identity = {
    credentials: { certificate, privateKey },
    mspId,
    type: 'X.509',
  };
  fs.writeFileSync(temporaryPath, JSON.stringify(identity, null, 2), { mode: 0o600 });
  fs.renameSync(temporaryPath, targetPath);
  fs.chmodSync(targetPath, 0o600);
}

export function getIdentity(userId) {
  const targetPath = identityPath(userId);
  if (!fs.existsSync(targetPath)) return null;
  return JSON.parse(fs.readFileSync(targetPath, 'utf-8'));
}

export function identityExists(userId) {
  return fs.existsSync(identityPath(userId));
}

export function closeDB() {
  if (!db) return;
  db.close();
  db = undefined;
}
