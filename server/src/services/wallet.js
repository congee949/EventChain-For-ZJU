import fs from 'node:fs';
import path from 'node:path';
import Database from 'better-sqlite3';
import bcrypt from 'bcrypt';
import config from '../config/index.js';

// ---- SQLite for password hashes ----

let db;

export function initDB() {
  const dir = path.dirname(config.dbPath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });

  db = new Database(config.dbPath);
  db.pragma('journal_mode = WAL');

  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      user_id   TEXT PRIMARY KEY,
      name      TEXT NOT NULL,
      password  TEXT NOT NULL,
      org_msp   TEXT NOT NULL DEFAULT 'StudentMSP',
      role      TEXT NOT NULL DEFAULT 'student',
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    )
  `);
}

export function getDB() {
  return db;
}

export function createUser(userId, name, passwordHash, orgMSP = 'StudentMSP', role = 'student') {
  const stmt = db.prepare(
    'INSERT INTO users (user_id, name, password, org_msp, role) VALUES (?, ?, ?, ?, ?)'
  );
  stmt.run(userId, name, passwordHash, orgMSP, role);
}

export function findUser(userId) {
  return db.prepare('SELECT * FROM users WHERE user_id = ?').get(userId);
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
    fs.mkdirSync(config.walletPath, { recursive: true });
  }
  return config.walletPath;
}

export function putIdentity(userId, mspId, certificate, privateKey) {
  const identityPath = path.join(walletDir(), `${userId}.json`);
  const identity = {
    credentials: { certificate, privateKey },
    mspId,
    type: 'X.509',
  };
  fs.writeFileSync(identityPath, JSON.stringify(identity, null, 2));
}

export function getIdentity(userId) {
  const identityPath = path.join(walletDir(), `${userId}.json`);
  if (!fs.existsSync(identityPath)) return null;
  return JSON.parse(fs.readFileSync(identityPath, 'utf-8'));
}

export function identityExists(userId) {
  return fs.existsSync(path.join(walletDir(), `${userId}.json`));
}
