# EventChain Backend + Frontend Implementation Plan

---

## Task 1 — Server Scaffolding

- [ ] Create `server/package.json`
- [ ] Create `server/.env.example`
- [ ] Create `server/src/config/index.js`
- [ ] Create `server/src/config/fabric.js`
- [ ] Create `server/src/middleware/errorHandler.js`
- [ ] Create `server/src/app.js`

### server/package.json

```json
{
  "name": "eventchain-server",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "node --watch src/app.js",
    "start": "node src/app.js"
  },
  "dependencies": {
    "@hyperledger/fabric-gateway": "^1.7.0",
    "@hyperledger/grpc-signer": "^1.0.0",
    "fabric-ca-client": "^2.2.20",
    "express": "^4.21.0",
    "cors": "^2.8.5",
    "jsonwebtoken": "^9.0.2",
    "bcrypt": "^5.1.1",
    "better-sqlite3": "^11.6.0",
    "dotenv": "^16.4.7",
    "@grpc/grpc-js": "^1.12.0"
  }
}
```

### server/.env.example

```env
PORT=3000
JWT_SECRET=eventchain-dev-secret-change-in-production
JWT_EXPIRES_IN=24h

# Fabric connection
FABRIC_CHANNEL=eventchain
FABRIC_GATEWAY_PEER=peer0.platform.eventchain.com:7051
FABRIC_CA_URL=https://ca.student.eventchain.com:7054

# Chaincode names
CC_EVENT=event-cc
CC_PREDICTION=prediction-cc
CC_TICKET=ticket-cc
CC_TOKEN=token-cc

# Wallet path
WALLET_PATH=./wallet

# SQLite database path
DB_PATH=./data/users.db
```

### server/src/config/index.js

```js
import 'dotenv/config';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '../..');

export default {
  port: parseInt(process.env.PORT, 10) || 3000,

  jwt: {
    secret: process.env.JWT_SECRET || 'eventchain-dev-secret',
    expiresIn: process.env.JWT_EXPIRES_IN || '24h',
  },

  fabric: {
    channelName: process.env.FABRIC_CHANNEL || 'eventchain',
    peerEndpoint: process.env.FABRIC_GATEWAY_PEER || 'peer0.platform.eventchain.com:7051',
    caUrl: process.env.FABRIC_CA_URL || 'https://ca.student.eventchain.com:7054',
    chaincode: {
      event: process.env.CC_EVENT || 'event-cc',
      prediction: process.env.CC_PREDICTION || 'prediction-cc',
      ticket: process.env.CC_TICKET || 'ticket-cc',
      token: process.env.CC_TOKEN || 'token-cc',
    },
  },

  walletPath: path.resolve(root, process.env.WALLET_PATH || './wallet'),
  dbPath: path.resolve(root, process.env.DB_PATH || './data/users.db'),
};
```

### server/src/config/fabric.js

```js
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Connection profile for the Fabric Gateway SDK.
// In production the TLS cert paths come from the crypto material generated
// by the Fabric CA / cryptogen tool.  During development they live under the
// network's organizations/ directory.

const networkRoot = path.resolve(__dirname, '../../../fabric/network');

export function buildConnectionProfile(orgMSP) {
  const orgMap = {
    PlatformMSP: {
      mspId: 'PlatformMSP',
      peerHost: 'peer0.platform.eventchain.com',
      peerPort: 7051,
      caHost: 'ca.platform.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt'
      ),
    },
    OrganizerMSP: {
      mspId: 'OrganizerMSP',
      peerHost: 'peer0.organizer.eventchain.com',
      peerPort: 9051,
      caHost: 'ca.organizer.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt'
      ),
    },
    StudentMSP: {
      mspId: 'StudentMSP',
      peerHost: 'peer0.student.eventchain.com',
      peerPort: 11051,
      caHost: 'ca.student.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt'
      ),
    },
  };

  return orgMap[orgMSP] || orgMap.StudentMSP;
}
```

### server/src/middleware/errorHandler.js

```js
// Maps chaincode / application error codes to HTTP status + Chinese message.

const ERROR_MAP = {
  INSUFFICIENT_BALANCE: { status: 400, message: '余额不足' },
  EVENT_NOT_FOUND: { status: 404, message: '赛事不存在' },
  INVALID_STATUS: { status: 400, message: '赛事状态不允许此操作' },
  BET_CLOSED: { status: 400, message: '预测已关闭' },
  ALREADY_APPLIED: { status: 409, message: '已提交过申请' },
  TICKET_NOT_FOUND: { status: 404, message: '票据不存在' },
  UNAUTHORIZED: { status: 401, message: '未登录或登录已过期' },
  FORBIDDEN: { status: 403, message: '无权执行此操作' },
  USER_EXISTS: { status: 409, message: '该学号已注册' },
  INVALID_CREDENTIALS: { status: 401, message: '学号或密码错误' },
  VALIDATION_ERROR: { status: 400, message: '请求参数不合法' },
};

export function errorHandler(err, _req, res, _next) {
  console.error('[ErrorHandler]', err);

  // Application-level coded errors (thrown as { code, message })
  if (err.code && ERROR_MAP[err.code]) {
    const mapped = ERROR_MAP[err.code];
    return res.status(mapped.status).json({
      error: true,
      code: err.code,
      message: err.message || mapped.message,
    });
  }

  // Fabric Gateway errors often include a "details" array
  if (err.details && Array.isArray(err.details)) {
    const detail = err.details[0]?.message || err.message;
    // Try to extract a known code from the chaincode error string
    for (const [code, meta] of Object.entries(ERROR_MAP)) {
      if (detail.includes(code)) {
        return res.status(meta.status).json({
          error: true,
          code,
          message: meta.message,
        });
      }
    }
  }

  // Fallback
  const status = err.status || err.statusCode || 500;
  res.status(status).json({
    error: true,
    code: 'INTERNAL_ERROR',
    message: process.env.NODE_ENV === 'production' ? '服务器内部错误' : err.message,
  });
}

// Helper to throw coded errors from route handlers.
export function throwCoded(code, message) {
  const e = new Error(message || ERROR_MAP[code]?.message || code);
  e.code = code;
  throw e;
}
```

### server/src/app.js

```js
import express from 'express';
import cors from 'cors';
import config from './config/index.js';
import { errorHandler } from './middleware/errorHandler.js';
import authRoutes from './routes/auth.js';
import eventRoutes from './routes/events.js';
import predictionRoutes from './routes/predictions.js';
import ticketRoutes from './routes/tickets.js';
import userRoutes from './routes/users.js';
import { initDB } from './services/wallet.js';

const app = express();

// --------------- Middleware ---------------
app.use(cors());
app.use(express.json());

// --------------- Health check ---------------
app.get('/api/health', (_req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// --------------- Routes ---------------
app.use('/api/v1/auth', authRoutes);
app.use('/api/v1/events', eventRoutes);
app.use('/api/v1/predictions', predictionRoutes);
app.use('/api/v1/tickets', ticketRoutes);
app.use('/api/v1/users', userRoutes);

// --------------- Error handler (must be last) ---------------
app.use(errorHandler);

// --------------- Start ---------------
async function main() {
  initDB();
  app.listen(config.port, () => {
    console.log(`[EventChain] Server listening on http://localhost:${config.port}`);
  });
}

main().catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
```

---

## Task 2 — Fabric Services (Gateway + CA + Wallet)

- [ ] Create `server/src/services/wallet.js`
- [ ] Create `server/src/services/caService.js`
- [ ] Create `server/src/services/fabricGateway.js`

### server/src/services/wallet.js

```js
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
```

### server/src/services/caService.js

```js
import FabricCAServices from 'fabric-ca-client';
import { buildConnectionProfile } from '../config/fabric.js';
import { putIdentity, getIdentity } from './wallet.js';

// Cache CA client instances per org
const caClients = {};

function getCAClient(orgMSP) {
  if (caClients[orgMSP]) return caClients[orgMSP];

  const orgConfig = buildConnectionProfile(orgMSP);
  const caUrl = `https://${orgConfig.caHost}:7054`;
  const caClient = new FabricCAServices(caUrl, {
    trustedRoots: [],
    verify: false, // dev only — accept self-signed CA certs
  }, orgConfig.caHost);

  caClients[orgMSP] = caClient;
  return caClient;
}

// Enroll the bootstrap admin identity for a given org CA.
// This admin identity is used to register new users.
export async function enrollAdmin(orgMSP) {
  if (getIdentity(`admin-${orgMSP}`)) return getIdentity(`admin-${orgMSP}`);

  const caClient = getCAClient(orgMSP);
  const enrollment = await caClient.enroll({
    enrollmentID: 'admin',
    enrollmentSecret: 'adminpw',
  });

  putIdentity(`admin-${orgMSP}`, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(`admin-${orgMSP}`);
}

// Register and enroll a new user with the org's CA.
export async function registerAndEnrollUser(userId, orgMSP) {
  const caClient = getCAClient(orgMSP);

  // Ensure the CA admin is enrolled first
  const adminIdentity = await enrollAdmin(orgMSP);

  // Build an admin User object that fabric-ca-client can use for registration
  const adminUser = {
    getName: () => `admin-${orgMSP}`,
    getMSPId: () => orgMSP,
    getIdentity: () => ({
      serialize: () => adminIdentity.credentials.certificate,
    }),
    getSigningIdentity: () => ({
      sign: (msg) => {
        const { createSign } = require('node:crypto');
        const sign = createSign('SHA256');
        sign.update(msg);
        return sign.sign(adminIdentity.credentials.privateKey);
      },
    }),
  };

  // Register
  const secret = await caClient.register(
    {
      affiliation: '',
      enrollmentID: userId,
      role: 'client',
    },
    adminUser
  );

  // Enroll
  const enrollment = await caClient.enroll({
    enrollmentID: userId,
    enrollmentSecret: secret,
  });

  putIdentity(userId, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(userId);
}
```

### server/src/services/fabricGateway.js

```js
import * as grpc from '@grpc/grpc-js';
import { connect, signers } from '@hyperledger/fabric-gateway';
import fs from 'node:fs';
import crypto from 'node:crypto';
import { buildConnectionProfile } from '../config/fabric.js';
import { getIdentity } from './wallet.js';
import config from '../config/index.js';

// Cache gateway connections per user to avoid reconnecting on every request.
const gatewayCache = new Map();

function newGrpcConnection(orgMSP) {
  const orgConfig = buildConnectionProfile(orgMSP);
  const tlsCert = fs.readFileSync(orgConfig.tlsCertPath);
  const tlsCredentials = grpc.credentials.createSsl(tlsCert);
  return new grpc.Client(
    `${orgConfig.peerHost}:${orgConfig.peerPort}`,
    tlsCredentials,
    { 'grpc.ssl_target_name_override': orgConfig.peerHost }
  );
}

function newIdentity(identity) {
  const certificate = identity.credentials.certificate;
  return { mspId: identity.mspId, credentials: Buffer.from(certificate) };
}

function newSigner(identity) {
  const privateKeyPem = identity.credentials.privateKey;
  const privateKey = crypto.createPrivateKey(privateKeyPem);
  return signers.newPrivateKeySigner(privateKey);
}

// Get or create a Gateway connection for a given user.
export async function getGateway(userId) {
  if (gatewayCache.has(userId)) return gatewayCache.get(userId);

  const identity = getIdentity(userId);
  if (!identity) throw Object.assign(new Error('身份未找到'), { code: 'UNAUTHORIZED' });

  const grpcClient = newGrpcConnection(identity.mspId);
  const gateway = connect({
    client: grpcClient,
    identity: newIdentity(identity),
    signer: newSigner(identity),
    evaluateOptions: () => ({ deadline: Date.now() + 5000 }),
    endorseOptions: () => ({ deadline: Date.now() + 15000 }),
    submitOptions: () => ({ deadline: Date.now() + 5000 }),
    commitStatusOptions: () => ({ deadline: Date.now() + 60000 }),
  });

  gatewayCache.set(userId, { gateway, grpcClient });
  return { gateway, grpcClient };
}

// Get a contract handle for a given chaincode.
export async function getContract(userId, chaincodeName) {
  const { gateway } = await getGateway(userId);
  const network = gateway.getNetwork(config.fabric.channelName);
  return network.getContract(chaincodeName);
}

// Submit a transaction (read-write).
export async function submitTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.submitTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

// Evaluate a transaction (read-only query).
export async function evaluateTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.evaluateTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

// Close a user's cached connection when they log out (optional).
export function closeGateway(userId) {
  const cached = gatewayCache.get(userId);
  if (cached) {
    cached.gateway.close();
    cached.grpcClient.close();
    gatewayCache.delete(userId);
  }
}
```

---

## Task 3 — Auth System (JWT Middleware + Auth Routes)

- [ ] Create `server/src/middleware/auth.js`
- [ ] Create `server/src/routes/auth.js`

### server/src/middleware/auth.js

```js
import jwt from 'jsonwebtoken';
import config from '../config/index.js';

// JWT verification middleware.
// Sets req.user = { userId, orgMSP } on success.
export function authenticate(req, _res, next) {
  const header = req.headers.authorization;
  if (!header || !header.startsWith('Bearer ')) {
    const err = new Error('未提供认证令牌');
    err.code = 'UNAUTHORIZED';
    return next(err);
  }

  const token = header.slice(7);
  try {
    const payload = jwt.verify(token, config.jwt.secret);
    req.user = { userId: payload.userId, orgMSP: payload.orgMSP, role: payload.role };
    next();
  } catch {
    const err = new Error('认证令牌无效或已过期');
    err.code = 'UNAUTHORIZED';
    next(err);
  }
}

// Require a specific role (e.g. 'organizer', 'admin').
export function requireRole(...roles) {
  return (req, _res, next) => {
    if (!roles.includes(req.user.role)) {
      const err = new Error('无权执行此操作');
      err.code = 'FORBIDDEN';
      return next(err);
    }
    next();
  };
}
```

### server/src/routes/auth.js

```js
import { Router } from 'express';
import jwt from 'jsonwebtoken';
import config from '../config/index.js';
import { createUser, findUser, hashPassword, verifyPassword } from '../services/wallet.js';
import { registerAndEnrollUser } from '../services/caService.js';
import { submitTransaction } from '../services/fabricGateway.js';

const router = Router();

// POST /api/v1/auth/register
// Body: { studentID, password, name }
router.post('/register', async (req, res, next) => {
  try {
    const { studentID, password, name } = req.body;

    if (!studentID || !password || !name) {
      const err = new Error('studentID, password, name 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Check if user already exists in SQLite
    if (findUser(studentID)) {
      const err = new Error('该学号已注册');
      err.code = 'USER_EXISTS';
      throw err;
    }

    // 1. Hash password and store in SQLite
    const passwordHash = await hashPassword(password);
    createUser(studentID, name, passwordHash, 'StudentMSP', 'student');

    // 2. Register + enroll with Fabric CA → cert stored in wallet
    await registerAndEnrollUser(studentID, 'StudentMSP');

    // 3. Mint 1000 tokens as registration bonus
    await submitTransaction(studentID, config.fabric.chaincode.token, 'Mint', studentID, '1000');

    // 4. Issue JWT
    const token = jwt.sign(
      { userId: studentID, orgMSP: 'StudentMSP', role: 'student' },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.status(201).json({
      error: false,
      data: { token, userId: studentID, name, balance: 1000 },
    });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/auth/login
// Body: { studentID, password }
router.post('/login', async (req, res, next) => {
  try {
    const { studentID, password } = req.body;

    if (!studentID || !password) {
      const err = new Error('studentID 和 password 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const user = findUser(studentID);
    if (!user) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const valid = await verifyPassword(password, user.password);
    if (!valid) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const token = jwt.sign(
      { userId: user.user_id, orgMSP: user.org_msp, role: user.role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.json({
      error: false,
      data: { token, userId: user.user_id, name: user.name, role: user.role },
    });
  } catch (err) {
    next(err);
  }
});

export default router;
```

---

## Task 4 — Event Routes

- [ ] Create `server/src/routes/events.js`

### server/src/routes/events.js

```js
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
```

---

## Task 5 — Prediction Routes

- [ ] Create `server/src/routes/predictions.js`

### server/src/routes/predictions.js

```js
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

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'PlaceBet',
      eventID,
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
    const bets = await evaluateTransaction(req.user.userId, CC, 'GetUserBets', req.user.userId);
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
```

---

## Task 6 — Ticket + User Routes

- [ ] Create `server/src/routes/tickets.js`
- [ ] Create `server/src/routes/users.js`

### server/src/routes/tickets.js

```js
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
```

### server/src/routes/users.js

```js
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
```

---

## Task 7 — Client Scaffolding + Liquid Glass CSS

- [ ] Create `client/package.json`
- [ ] Create `client/vite.config.js`
- [ ] Create `client/index.html`
- [ ] Create `client/src/main.js`
- [ ] Create `client/src/App.vue`
- [ ] Create `client/src/assets/styles/variables.css`
- [ ] Create `client/src/assets/styles/liquid-glass.css`
- [ ] Create `client/src/assets/styles/global.css`
- [ ] Create `client/src/composables/useGlassHighlight.js`
- [ ] Create `client/src/api/index.js`
- [ ] Create `client/src/router/index.js`

### client/package.json

```json
{
  "name": "eventchain-client",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.5.13",
    "vue-router": "^4.5.0",
    "pinia": "^2.3.0",
    "axios": "^1.7.9",
    "element-plus": "^2.9.1",
    "echarts": "^5.6.0",
    "vue-echarts": "^7.0.3",
    "qrcode": "^1.5.4"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.1",
    "vite": "^6.0.0"
  }
}
```

### client/vite.config.js

```js
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
```

### client/index.html

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>EventChain — 校园赛事预测市场</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.js"></script>
  </body>
</html>
```

### client/src/main.js

```js
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import App from './App.vue';
import router from './router/index.js';
import './assets/styles/variables.css';
import './assets/styles/liquid-glass.css';
import './assets/styles/global.css';

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

app.mount('#app');
```

### client/src/App.vue

```vue
<script setup>
import GlassNavbar from './components/GlassNavbar.vue';
import { useAuthStore } from './stores/auth.js';

const auth = useAuthStore();
auth.restoreSession();
</script>

<template>
  <div class="app-shell">
    <!-- Floating background orbs -->
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="orb orb-3"></div>

    <GlassNavbar />
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
}

.main-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 88px 24px 48px;
}
</style>
```

### client/src/assets/styles/variables.css

```css
:root {
  /* ---------- Brand palette ---------- */
  --color-primary: #6366f1;
  --color-primary-light: #818cf8;
  --color-primary-dark: #4f46e5;
  --color-success: #22c55e;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-info: #3b82f6;

  /* ---------- Neutral ---------- */
  --color-text: #1e293b;
  --color-text-secondary: #64748b;
  --color-text-tertiary: #94a3b8;
  --color-border: rgba(255, 255, 255, 0.35);

  /* ---------- Glass material tokens ---------- */
  --glass-dense-bg: rgba(255, 255, 255, 0.35);
  --glass-dense-blur: 32px;
  --glass-dense-saturate: 2;

  --glass-bg: rgba(255, 255, 255, 0.22);
  --glass-blur: 24px;
  --glass-saturate: 1.8;

  --glass-subtle-bg: rgba(255, 255, 255, 0.12);
  --glass-subtle-blur: 16px;
  --glass-subtle-saturate: 1.4;

  /* ---------- Specular highlight ---------- */
  --highlight-x: 30%;
  --highlight-y: 20%;

  /* ---------- Radius ---------- */
  --radius-sm: 12px;
  --radius-md: 18px;
  --radius-lg: 24px;

  /* ---------- Shadows ---------- */
  --shadow-card: 0 8px 40px rgba(0, 0, 0, 0.06);
  --shadow-card-hover: 0 12px 48px rgba(0, 0, 0, 0.1);

  /* ---------- Transitions ---------- */
  --transition-base: 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
```

### client/src/assets/styles/liquid-glass.css

```css
/* ============================================================
   Liquid Glass Material System — three tiers
   ============================================================ */

/* ---------- Dense — navigation bars, headers ---------- */
.glass-dense {
  position: relative;
  background: var(--glass-dense-bg);
  backdrop-filter: blur(var(--glass-dense-blur)) saturate(var(--glass-dense-saturate));
  -webkit-backdrop-filter: blur(var(--glass-dense-blur)) saturate(var(--glass-dense-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.45);
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.6),
    var(--shadow-card);
  border-radius: var(--radius-md);
}

/* ---------- Regular — cards, panels ---------- */
.glass {
  position: relative;
  background: var(--glass-bg);
  backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  -webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.4);
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card);
  border-radius: var(--radius-md);
  transition: box-shadow var(--transition-base), transform var(--transition-base);
}

.glass:hover {
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card-hover);
  transform: translateY(-2px);
}

/* ---------- Subtle — nested elements, badges ---------- */
.glass-subtle {
  position: relative;
  background: var(--glass-subtle-bg);
  backdrop-filter: blur(var(--glass-subtle-blur)) saturate(var(--glass-subtle-saturate));
  -webkit-backdrop-filter: blur(var(--glass-subtle-blur)) saturate(var(--glass-subtle-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.25);
  border-radius: var(--radius-sm);
}

/* ---------- Specular highlight layer (shared) ---------- */
.glass::before,
.glass-dense::before,
.glass-subtle::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: radial-gradient(
    ellipse at var(--highlight-x, 30%) var(--highlight-y, 20%),
    rgba(255, 255, 255, 0.45) 0%,
    transparent 65%
  );
  mix-blend-mode: overlay;
  pointer-events: none;
  z-index: 1;
}

/* Ensure card content sits above the pseudo-element */
.glass > *,
.glass-dense > *,
.glass-subtle > * {
  position: relative;
  z-index: 2;
}
```

### client/src/assets/styles/global.css

```css
/* ============================================================
   Global — body background, orbs, reset
   ============================================================ */

*,
*::before,
*::after {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family:
    -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI',
    Roboto, 'Helvetica Neue', sans-serif;
  color: var(--color-text);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;

  /* Mesh gradient background */
  background:
    radial-gradient(ellipse at 20% 20%, rgba(99, 102, 241, 0.25) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 30%, rgba(236, 72, 153, 0.20) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 80%, rgba(34, 197, 94, 0.18) 0%, transparent 50%),
    radial-gradient(ellipse at 10% 70%, rgba(245, 158, 11, 0.15) 0%, transparent 50%),
    linear-gradient(135deg, #f0f4ff 0%, #fdf2f8 50%, #f0fdf4 100%);
  background-attachment: fixed;
  min-height: 100vh;
}

/* ---------- Floating orbs ---------- */
.orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.5;
  pointer-events: none;
  z-index: 0;
  animation: float 20s ease-in-out infinite;
}

.orb-1 {
  width: 400px;
  height: 400px;
  background: rgba(99, 102, 241, 0.3);
  top: -100px;
  left: -100px;
  animation-delay: 0s;
}

.orb-2 {
  width: 350px;
  height: 350px;
  background: rgba(236, 72, 153, 0.25);
  top: 50%;
  right: -80px;
  animation-delay: -7s;
}

.orb-3 {
  width: 300px;
  height: 300px;
  background: rgba(34, 197, 94, 0.2);
  bottom: -60px;
  left: 30%;
  animation-delay: -14s;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) scale(1);
  }
  25% {
    transform: translate(30px, -40px) scale(1.05);
  }
  50% {
    transform: translate(-20px, 20px) scale(0.95);
  }
  75% {
    transform: translate(15px, 35px) scale(1.02);
  }
}

/* ---------- Scrollbar ---------- */
::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.15);
  border-radius: 3px;
}

/* ---------- Selection ---------- */
::selection {
  background: rgba(99, 102, 241, 0.25);
}
```

### client/src/composables/useGlassHighlight.js

```js
import { onMounted, onUnmounted, ref } from 'vue';

/**
 * Tracks the mouse position relative to a target element and updates
 * CSS custom properties --highlight-x and --highlight-y so the
 * radial-gradient specular highlight follows the cursor.
 *
 * Usage:
 *   const { elementRef } = useGlassHighlight();
 *   <div ref="elementRef" class="glass"> ... </div>
 */
export function useGlassHighlight() {
  const elementRef = ref(null);
  let rafId = null;

  function onMouseMove(e) {
    if (!elementRef.value) return;

    // Cancel any pending frame to avoid piling up
    if (rafId) cancelAnimationFrame(rafId);

    rafId = requestAnimationFrame(() => {
      const rect = elementRef.value.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      elementRef.value.style.setProperty('--highlight-x', `${x}%`);
      elementRef.value.style.setProperty('--highlight-y', `${y}%`);
    });
  }

  function onMouseLeave() {
    if (!elementRef.value) return;
    // Reset to default resting position
    elementRef.value.style.setProperty('--highlight-x', '30%');
    elementRef.value.style.setProperty('--highlight-y', '20%');
  }

  onMounted(() => {
    if (elementRef.value) {
      elementRef.value.addEventListener('mousemove', onMouseMove);
      elementRef.value.addEventListener('mouseleave', onMouseLeave);
    }
  });

  onUnmounted(() => {
    if (rafId) cancelAnimationFrame(rafId);
    if (elementRef.value) {
      elementRef.value.removeEventListener('mousemove', onMouseMove);
      elementRef.value.removeEventListener('mouseleave', onMouseLeave);
    }
  });

  return { elementRef };
}
```

### client/src/api/index.js

```js
import axios from 'axios';
import { ElMessage } from 'element-plus';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

// ---- Request interceptor: attach JWT ----
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ec_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// ---- Response interceptor: unwrap data, handle errors ----
api.interceptors.response.use(
  (response) => {
    // Successful responses have { error: false, data: ... }
    return response.data?.data !== undefined ? response.data.data : response.data;
  },
  (error) => {
    const resp = error.response;
    if (resp) {
      const body = resp.data;
      const message = body?.message || '请求失败';

      // Auto-logout on 401
      if (resp.status === 401) {
        localStorage.removeItem('ec_token');
        localStorage.removeItem('ec_user');
        window.location.href = '/login';
      }

      ElMessage.error(message);
      return Promise.reject(body);
    }

    ElMessage.error('网络连接失败，请稍后重试');
    return Promise.reject(error);
  }
);

export default api;
```

### client/src/router/index.js

```js
import { createRouter, createWebHistory } from 'vue-router';

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/event/:id',
    name: 'EventDetail',
    component: () => import('../views/EventDetailView.vue'),
    props: true,
  },
  {
    path: '/tickets',
    name: 'TicketHall',
    component: () => import('../views/TicketHallView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/me',
    name: 'Profile',
    component: () => import('../views/ProfileView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/AdminView.vue'),
    meta: { requiresAuth: true, requiresRole: ['organizer', 'admin'] },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// Navigation guard
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('ec_token');
  const user = JSON.parse(localStorage.getItem('ec_user') || 'null');

  if (to.meta.requiresAuth && !token) {
    return next({ name: 'Login', query: { redirect: to.fullPath } });
  }

  if (to.meta.requiresRole && user) {
    const allowed = to.meta.requiresRole;
    if (!allowed.includes(user.role)) {
      return next({ name: 'Home' });
    }
  }

  next();
});

export default router;
```

---

## Task 8 — Reusable Components

- [ ] Create `client/src/components/GlassCard.vue`
- [ ] Create `client/src/components/GlassNavbar.vue`
- [ ] Create `client/src/components/ProbabilityBar.vue`
- [ ] Create `client/src/components/SparkLine.vue`
- [ ] Create `client/src/components/CountdownTimer.vue`
- [ ] Create `client/src/components/QRCode.vue`

### client/src/components/GlassCard.vue

```vue
<script setup>
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const props = defineProps({
  variant: {
    type: String,
    default: 'regular',
    validator: (v) => ['dense', 'regular', 'subtle'].includes(v),
  },
  hoverable: {
    type: Boolean,
    default: true,
  },
  padding: {
    type: String,
    default: '24px',
  },
});

const { elementRef } = useGlassHighlight();

const classMap = {
  dense: 'glass-dense',
  regular: 'glass',
  subtle: 'glass-subtle',
};
</script>

<template>
  <div
    ref="elementRef"
    :class="[classMap[props.variant], { 'no-hover': !props.hoverable }]"
    :style="{ padding: props.padding }"
    class="glass-card"
  >
    <slot />
  </div>
</template>

<style scoped>
.glass-card {
  overflow: hidden;
}

.glass-card.no-hover:hover {
  transform: none;
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card);
}
</style>
```

### client/src/components/GlassNavbar.vue

```vue
<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { useUserStore } from '../stores/user.js';
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const router = useRouter();
const auth = useAuthStore();
const userStore = useUserStore();
const { elementRef } = useGlassHighlight();

const isLoggedIn = computed(() => auth.isLoggedIn);
const displayName = computed(() => auth.user?.name || auth.user?.userId || '');
const balance = computed(() => userStore.balance);

function handleLogout() {
  auth.logout();
  router.push('/login');
}

const navLinks = [
  { label: '首页', path: '/' },
  { label: '票务大厅', path: '/tickets' },
  { label: '我的', path: '/me' },
];
</script>

<template>
  <nav ref="elementRef" class="glass-dense navbar">
    <div class="navbar-inner">
      <router-link to="/" class="brand">
        <span class="brand-icon">&#9830;</span>
        <span class="brand-text">EventChain</span>
      </router-link>

      <div class="nav-links">
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          class="nav-link"
          active-class="nav-link--active"
        >
          {{ link.label }}
        </router-link>

        <router-link
          v-if="auth.user?.role === 'organizer' || auth.user?.role === 'admin'"
          to="/admin"
          class="nav-link"
          active-class="nav-link--active"
        >
          管理
        </router-link>
      </div>

      <div class="nav-right">
        <template v-if="isLoggedIn">
          <div class="balance-badge glass-subtle">
            <span class="balance-icon">&#9733;</span>
            <span class="balance-amount">{{ balance }}</span>
          </div>
          <div class="user-info">
            <span class="user-name">{{ displayName }}</span>
            <button class="logout-btn" @click="handleLogout">退出</button>
          </div>
        </template>
        <template v-else>
          <router-link to="/login" class="login-btn">登录</router-link>
        </template>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  position: fixed;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  width: calc(100% - 48px);
  max-width: 1200px;
  z-index: 1000;
  padding: 0 24px;
  border-radius: var(--radius-lg);
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--color-text);
  font-weight: 700;
  font-size: 18px;
}

.brand-icon {
  font-size: 22px;
  color: var(--color-primary);
}

.nav-links {
  display: flex;
  gap: 8px;
}

.nav-link {
  text-decoration: none;
  color: var(--color-text-secondary);
  padding: 6px 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 500;
  transition: all var(--transition-base);
}

.nav-link:hover {
  color: var(--color-text);
  background: rgba(255, 255, 255, 0.2);
}

.nav-link--active {
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.balance-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-warning);
}

.balance-icon {
  font-size: 16px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text);
}

.logout-btn {
  background: none;
  border: none;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  margin-left: 4px;
  border-radius: 6px;
  transition: all var(--transition-base);
}

.logout-btn:hover {
  color: var(--color-danger);
  background: rgba(239, 68, 68, 0.08);
}

.login-btn {
  text-decoration: none;
  padding: 6px 20px;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  transition: background var(--transition-base);
}

.login-btn:hover {
  background: var(--color-primary-dark);
}
</style>
```

### client/src/components/ProbabilityBar.vue

```vue
<script setup>
import { computed } from 'vue';

const props = defineProps({
  probA: { type: Number, required: true },  // 0-1
  labelA: { type: String, default: 'A' },
  labelB: { type: String, default: 'B' },
  colorA: { type: String, default: '#6366f1' },
  colorB: { type: String, default: '#ec4899' },
  height: { type: String, default: '32px' },
});

const pctA = computed(() => Math.round(props.probA * 100));
const pctB = computed(() => 100 - pctA.value);
</script>

<template>
  <div class="probability-bar" :style="{ height: props.height }">
    <div
      class="bar-segment bar-a"
      :style="{
        width: pctA + '%',
        background: `linear-gradient(90deg, ${props.colorA}, ${props.colorA}dd)`,
      }"
    >
      <span v-if="pctA >= 20" class="bar-label">{{ props.labelA }} {{ pctA }}%</span>
    </div>
    <div
      class="bar-segment bar-b"
      :style="{
        width: pctB + '%',
        background: `linear-gradient(90deg, ${props.colorB}dd, ${props.colorB})`,
      }"
    >
      <span v-if="pctB >= 20" class="bar-label">{{ props.labelB }} {{ pctB }}%</span>
    </div>
  </div>
</template>

<style scoped>
.probability-bar {
  display: flex;
  border-radius: 999px;
  overflow: hidden;
  width: 100%;
}

.bar-segment {
  display: flex;
  align-items: center;
  justify-content: center;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  min-width: 4px;
}

.bar-label {
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}
</style>
```

### client/src/components/SparkLine.vue

```vue
<script setup>
import { computed } from 'vue';

const props = defineProps({
  data: { type: Array, required: true },   // array of numbers
  width: { type: Number, default: 80 },
  height: { type: Number, default: 24 },
  color: { type: String, default: '#6366f1' },
  filled: { type: Boolean, default: true },
});

const pathD = computed(() => {
  const pts = props.data;
  if (!pts.length) return '';

  const max = Math.max(...pts);
  const min = Math.min(...pts);
  const range = max - min || 1;
  const stepX = props.width / (pts.length - 1 || 1);
  const pad = 2;
  const usableH = props.height - pad * 2;

  const points = pts.map((v, i) => {
    const x = i * stepX;
    const y = pad + usableH - ((v - min) / range) * usableH;
    return `${x},${y}`;
  });

  return 'M' + points.join(' L');
});

const fillD = computed(() => {
  if (!props.filled || !props.data.length) return '';
  return `${pathD.value} L${props.width},${props.height} L0,${props.height} Z`;
});
</script>

<template>
  <svg
    :viewBox="`0 0 ${props.width} ${props.height}`"
    :width="props.width"
    :height="props.height"
    class="sparkline"
  >
    <path
      v-if="props.filled"
      :d="fillD"
      :fill="`${props.color}20`"
    />
    <path
      :d="pathD"
      fill="none"
      :stroke="props.color"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<style scoped>
.sparkline {
  display: inline-block;
  vertical-align: middle;
}
</style>
```

### client/src/components/CountdownTimer.vue

```vue
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';

const props = defineProps({
  targetTime: { type: String, required: true },  // ISO string
});

const now = ref(Date.now());
let timer = null;

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now(); }, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});

const remaining = computed(() => {
  const diff = new Date(props.targetTime).getTime() - now.value;
  if (diff <= 0) return { days: 0, hours: 0, mins: 0, secs: 0, expired: true };

  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  const secs = Math.floor((diff % 60000) / 1000);

  return { days, hours, mins, secs, expired: false };
});

function pad(n) {
  return String(n).padStart(2, '0');
}
</script>

<template>
  <div class="countdown" :class="{ 'countdown--expired': remaining.expired }">
    <template v-if="remaining.expired">
      <span class="countdown-label">已截止</span>
    </template>
    <template v-else>
      <div v-if="remaining.days > 0" class="countdown-unit">
        <span class="countdown-value">{{ remaining.days }}</span>
        <span class="countdown-label">天</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.hours) }}</span>
        <span class="countdown-label">时</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.mins) }}</span>
        <span class="countdown-label">分</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.secs) }}</span>
        <span class="countdown-label">秒</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.countdown {
  display: flex;
  align-items: center;
  gap: 4px;
}

.countdown-unit {
  display: flex;
  align-items: baseline;
  gap: 1px;
}

.countdown-value {
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}

.countdown-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.countdown--expired .countdown-label {
  color: var(--color-danger);
  font-weight: 600;
  font-size: 14px;
}
</style>
```

### client/src/components/QRCode.vue

```vue
<script setup>
import { ref, watch, onMounted } from 'vue';
import QRCodeLib from 'qrcode';

const props = defineProps({
  value: { type: String, required: true },
  size: { type: Number, default: 200 },
});

const canvasRef = ref(null);

async function render() {
  if (!canvasRef.value || !props.value) return;
  await QRCodeLib.toCanvas(canvasRef.value, props.value, {
    width: props.size,
    margin: 2,
    color: { dark: '#1e293b', light: '#ffffff' },
  });
}

onMounted(render);
watch(() => props.value, render);
watch(() => props.size, render);
</script>

<template>
  <canvas ref="canvasRef" class="qrcode-canvas" />
</template>

<style scoped>
.qrcode-canvas {
  border-radius: var(--radius-sm);
}
</style>
```

---

## Task 9 — Pinia Stores

- [ ] Create `client/src/stores/auth.js`
- [ ] Create `client/src/stores/events.js`
- [ ] Create `client/src/stores/prediction.js`
- [ ] Create `client/src/stores/ticket.js`
- [ ] Create `client/src/stores/user.js`

### client/src/stores/auth.js

```js
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../api/index.js';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(null);
  const user = ref(null); // { userId, name, role }

  const isLoggedIn = computed(() => !!token.value);

  // Restore session from localStorage on app init
  function restoreSession() {
    const savedToken = localStorage.getItem('ec_token');
    const savedUser = localStorage.getItem('ec_user');
    if (savedToken && savedUser) {
      token.value = savedToken;
      user.value = JSON.parse(savedUser);
    }
  }

  function setSession(jwt, userData) {
    token.value = jwt;
    user.value = userData;
    localStorage.setItem('ec_token', jwt);
    localStorage.setItem('ec_user', JSON.stringify(userData));
  }

  async function register(studentID, password, name) {
    const data = await api.post('/auth/register', { studentID, password, name });
    setSession(data.token, {
      userId: data.userId,
      name: data.name || name,
      role: 'student',
    });
    return data;
  }

  async function login(studentID, password) {
    const data = await api.post('/auth/login', { studentID, password });
    setSession(data.token, {
      userId: data.userId,
      name: data.name,
      role: data.role,
    });
    return data;
  }

  function logout() {
    token.value = null;
    user.value = null;
    localStorage.removeItem('ec_token');
    localStorage.removeItem('ec_user');
  }

  return { token, user, isLoggedIn, restoreSession, register, login, logout };
});
```

### client/src/stores/events.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useEventStore = defineStore('events', () => {
  const events = ref([]);
  const currentEvent = ref(null);
  const loading = ref(false);

  async function fetchEvents(filters = {}) {
    loading.value = true;
    try {
      const params = new URLSearchParams();
      if (filters.status) params.set('status', filters.status);
      if (filters.type) params.set('type', filters.type);
      const qs = params.toString();
      events.value = await api.get(`/events${qs ? '?' + qs : ''}`);
    } finally {
      loading.value = false;
    }
  }

  async function fetchEvent(eventId) {
    loading.value = true;
    try {
      currentEvent.value = await api.get(`/events/${eventId}`);
      return currentEvent.value;
    } finally {
      loading.value = false;
    }
  }

  async function createEvent(payload) {
    const result = await api.post('/events', payload);
    await fetchEvents();
    return result;
  }

  async function updateStatus(eventId, status) {
    const result = await api.put(`/events/${eventId}/status`, { status });
    await fetchEvents();
    return result;
  }

  async function recordResult(eventId, outcome) {
    const result = await api.put(`/events/${eventId}/result`, { outcome });
    await fetchEvents();
    return result;
  }

  return {
    events,
    currentEvent,
    loading,
    fetchEvents,
    fetchEvent,
    createEvent,
    updateStatus,
    recordResult,
  };
});
```

### client/src/stores/prediction.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const usePredictionStore = defineStore('prediction', () => {
  const odds = ref(null);      // { probA, probB } for current event
  const pool = ref(null);      // { poolA, poolB, k, totalVolume }
  const myBets = ref([]);
  const myScore = ref(null);   // { totalBets, correctBets, accuracyRate }
  const loading = ref(false);

  async function fetchOdds(eventID) {
    odds.value = await api.get(`/predictions/odds/${eventID}`);
    return odds.value;
  }

  async function fetchPool(eventID) {
    pool.value = await api.get(`/predictions/pool/${eventID}`);
    return pool.value;
  }

  async function placeBet(eventID, option, amount) {
    loading.value = true;
    try {
      const result = await api.post('/predictions/bet', { eventID, option, amount });
      // result: { shares, newOddsA, newOddsB }
      odds.value = { probA: result.newOddsA, probB: result.newOddsB };
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function fetchMyBets() {
    myBets.value = await api.get('/predictions/mine');
    return myBets.value;
  }

  async function fetchMyScore() {
    myScore.value = await api.get('/predictions/score');
    return myScore.value;
  }

  return {
    odds,
    pool,
    myBets,
    myScore,
    loading,
    fetchOdds,
    fetchPool,
    placeBet,
    fetchMyBets,
    fetchMyScore,
  };
});
```

### client/src/stores/ticket.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useTicketStore = defineStore('ticket', () => {
  const myTickets = ref([]);
  const loading = ref(false);

  async function applyTicket(eventID) {
    loading.value = true;
    try {
      const result = await api.post('/tickets/apply', { eventID });
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function runLottery(eventID) {
    const result = await api.post(`/tickets/lottery/${eventID}`);
    return result;
  }

  async function fetchMyTickets() {
    myTickets.value = await api.get('/tickets/mine');
    return myTickets.value;
  }

  async function claimTicket(ticketID) {
    const result = await api.post(`/tickets/claim/${ticketID}`);
    // Refresh ticket list to show new claim hash
    await fetchMyTickets();
    return result;
  }

  async function verifyTicket(ticketID, hash) {
    const result = await api.get(`/tickets/verify/${ticketID}`, { params: { hash } });
    return result;
  }

  async function refundTicket(ticketID) {
    const result = await api.post(`/tickets/refund/${ticketID}`);
    await fetchMyTickets();
    return result;
  }

  return {
    myTickets,
    loading,
    applyTicket,
    runLottery,
    fetchMyTickets,
    claimTicket,
    verifyTicket,
    refundTicket,
  };
});
```

### client/src/stores/user.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useUserStore = defineStore('user', () => {
  const balance = ref(0);
  const profile = ref(null);
  const leaderboard = ref([]);
  const loading = ref(false);

  async function fetchProfile() {
    loading.value = true;
    try {
      profile.value = await api.get('/users/profile');
      balance.value = profile.value.balance ?? 0;
      return profile.value;
    } finally {
      loading.value = false;
    }
  }

  async function fetchLeaderboard() {
    leaderboard.value = await api.get('/users/leaderboard');
    return leaderboard.value;
  }

  // Called after any action that changes balance (bet, settlement, etc.)
  async function refreshBalance() {
    const p = await api.get('/users/profile');
    balance.value = p.balance ?? 0;
  }

  return {
    balance,
    profile,
    leaderboard,
    loading,
    fetchProfile,
    fetchLeaderboard,
    refreshBalance,
  };
});
```

---

## Task 10 — Home Page

- [ ] Create `client/src/views/HomeView.vue`
- [ ] Create `client/src/views/LoginView.vue`

### client/src/views/HomeView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useEventStore } from '../stores/events.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';
import SparkLine from '../components/SparkLine.vue';

const router = useRouter();
const eventStore = useEventStore();
const userStore = useUserStore();
const auth = useAuthStore();

onMounted(async () => {
  await eventStore.fetchEvents();
  if (auth.isLoggedIn) {
    await Promise.all([
      userStore.fetchProfile(),
      userStore.fetchLeaderboard(),
    ]);
  }
});

const hotEvents = computed(() =>
  eventStore.events.filter((e) =>
    ['PREDICTION_OPEN', 'ONGOING'].includes(e.status)
  )
);

const typeEmoji = {
  basketball: '\u{1F3C0}',
  football: '\u26BD',
  esports: '\u{1F3AE}',
  badminton: '\u{1F3F8}',
  track: '\u{1F3C3}',
};

function goToEvent(id) {
  router.push(`/event/${id}`);
}
</script>

<template>
  <div class="home">
    <!-- Hero -->
    <section class="hero">
      <h1 class="hero-title">EventChain</h1>
      <p class="hero-subtitle">校园赛事预测市场 &mdash; 预测即力量</p>
    </section>

    <div class="home-grid">
      <!-- Event cards -->
      <section class="events-section">
        <h2 class="section-title">热门赛事</h2>
        <div class="events-grid">
          <GlassCard
            v-for="event in hotEvents"
            :key="event.id"
            class="event-card"
            @click="goToEvent(event.id)"
          >
            <div class="event-card-header">
              <span class="event-type-emoji">{{ typeEmoji[event.type] || '\u{1F3C6}' }}</span>
              <span class="event-status glass-subtle">{{ event.status }}</span>
            </div>
            <h3 class="event-title">{{ event.title }}</h3>
            <div class="event-teams">
              {{ event.teams?.[0] }} <span class="vs">VS</span> {{ event.teams?.[1] }}
            </div>
            <ProbabilityBar
              v-if="event.odds"
              :prob-a="event.odds.probA"
              :label-a="event.predictionOptions?.[0] || 'A'"
              :label-b="event.predictionOptions?.[1] || 'B'"
              height="28px"
              style="margin-top: 12px"
            />
            <div class="event-card-footer">
              <SparkLine
                v-if="event.oddsHistory"
                :data="event.oddsHistory"
                :width="80"
                :height="24"
              />
              <span class="event-volume">
                {{ event.pool?.totalVolume || 0 }} 浙币参与
              </span>
            </div>
          </GlassCard>
        </div>
      </section>

      <!-- Sidebar -->
      <aside class="sidebar">
        <!-- Stats -->
        <GlassCard v-if="auth.isLoggedIn && userStore.profile" class="sidebar-card">
          <h3 class="sidebar-title">我的数据</h3>
          <div class="stats-row">
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.balance }}</span>
              <span class="stat-label">浙币</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ (userStore.profile.accuracyRate * 100).toFixed(1) }}%</span>
              <span class="stat-label">准确率</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.totalBets }}</span>
              <span class="stat-label">总预测</span>
            </div>
          </div>
        </GlassCard>

        <!-- Leaderboard -->
        <GlassCard class="sidebar-card">
          <h3 class="sidebar-title">预测之星</h3>
          <ol class="leaderboard-list">
            <li
              v-for="(entry, idx) in userStore.leaderboard.slice(0, 10)"
              :key="entry.userId"
              class="leaderboard-item"
            >
              <span class="lb-rank">{{ idx + 1 }}</span>
              <span class="lb-name">{{ entry.userId }}</span>
              <span class="lb-score">{{ (entry.accuracyRate * 100).toFixed(1) }}%</span>
            </li>
          </ol>
        </GlassCard>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.hero {
  text-align: center;
  padding: 48px 0 32px;
}

.hero-title {
  font-size: 48px;
  font-weight: 800;
  background: linear-gradient(135deg, var(--color-primary), #ec4899);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  font-size: 18px;
  color: var(--color-text-secondary);
  margin-top: 8px;
}

.home-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.events-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.event-card {
  cursor: pointer;
}

.event-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.event-type-emoji {
  font-size: 24px;
}

.event-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.event-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 4px;
}

.event-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.vs {
  color: var(--color-danger);
  font-weight: 700;
  margin: 0 4px;
}

.event-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

.event-volume {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

/* Sidebar */
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sidebar-card {
  padding: 20px;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 12px;
}

.stats-row {
  display: flex;
  justify-content: space-between;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.leaderboard-list {
  list-style: none;
  padding: 0;
}

.leaderboard-item {
  display: flex;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.leaderboard-item:last-child {
  border-bottom: none;
}

.lb-rank {
  width: 24px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-tertiary);
}

.lb-name {
  flex: 1;
  font-size: 14px;
}

.lb-score {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
}
</style>
```

### client/src/views/LoginView.vue

```vue
<script setup>
import { ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const mode = ref('login'); // 'login' | 'register'
const form = ref({ studentID: '', password: '', name: '' });
const submitting = ref(false);
const errorMsg = ref('');

const buttonLabel = computed(() => (mode.value === 'login' ? '登录' : '注册'));

async function handleSubmit() {
  errorMsg.value = '';
  submitting.value = true;

  try {
    if (mode.value === 'register') {
      if (!form.value.name) {
        errorMsg.value = '请输入姓名';
        return;
      }
      await auth.register(form.value.studentID, form.value.password, form.value.name);
    } else {
      await auth.login(form.value.studentID, form.value.password);
    }
    // Redirect to the page they tried to access, or home
    const redirect = route.query.redirect || '/';
    router.push(redirect);
  } catch (err) {
    errorMsg.value = err?.message || '操作失败';
  } finally {
    submitting.value = false;
  }
}

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login';
  errorMsg.value = '';
}
</script>

<template>
  <div class="login-page">
    <GlassCard class="login-card" padding="40px">
      <h2 class="login-title">
        {{ mode === 'login' ? '欢迎回来' : '加入 EventChain' }}
      </h2>
      <p class="login-subtitle">
        {{ mode === 'login' ? '登录你的账号' : '注册即送 1000 浙币' }}
      </p>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">学号</label>
          <input
            v-model="form.studentID"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入学号"
            required
          />
        </div>

        <div v-if="mode === 'register'" class="form-group">
          <label class="form-label">姓名</label>
          <input
            v-model="form.name"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入姓名"
          />
        </div>

        <div class="form-group">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            class="form-input glass-subtle"
            placeholder="请输入密码"
            required
          />
        </div>

        <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>

        <button
          type="submit"
          class="submit-btn"
          :disabled="submitting"
        >
          {{ submitting ? '处理中...' : buttonLabel }}
        </button>
      </form>

      <p class="toggle-text">
        {{ mode === 'login' ? '还没有账号？' : '已有账号？' }}
        <a class="toggle-link" @click.prevent="toggleMode">
          {{ mode === 'login' ? '立即注册' : '去登录' }}
        </a>
      </p>
    </GlassCard>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: calc(100vh - 120px);
}

.login-card {
  width: 100%;
  max-width: 420px;
}

.login-title {
  font-size: 24px;
  font-weight: 800;
  text-align: center;
}

.login-subtitle {
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 14px;
  margin-top: 4px;
  margin-bottom: 28px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 10px 14px;
  border: none;
  outline: none;
  font-size: 15px;
  color: var(--color-text);
  border-radius: var(--radius-sm);
}

.form-input::placeholder {
  color: var(--color-text-tertiary);
}

.form-input:focus {
  box-shadow: 0 0 0 2px var(--color-primary-light);
}

.error-text {
  color: var(--color-danger);
  font-size: 13px;
  text-align: center;
}

.submit-btn {
  margin-top: 8px;
  padding: 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.submit-btn:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-text {
  text-align: center;
  margin-top: 20px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.toggle-link {
  color: var(--color-primary);
  font-weight: 600;
  cursor: pointer;
}

.toggle-link:hover {
  text-decoration: underline;
}
</style>
```

---

## Task 11 — Event Detail Page

- [ ] Create `client/src/views/EventDetailView.vue`

### client/src/views/EventDetailView.vue

```vue
<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
} from 'echarts/components';
import VChart from 'vue-echarts';
import { ElMessage, ElInputNumber, ElSlider, ElRadioGroup, ElRadioButton } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent]);

const route = useRoute();
const eventStore = useEventStore();
const predStore = usePredictionStore();
const userStore = useUserStore();
const auth = useAuthStore();

const eventId = computed(() => route.params.id);
const event = computed(() => eventStore.currentEvent);
const odds = computed(() => predStore.odds);

// Bet form
const selectedOption = ref('A');
const betAmount = ref(50);
const submitting = ref(false);

onMounted(async () => {
  await eventStore.fetchEvent(eventId.value);
  await predStore.fetchOdds(eventId.value);
  await predStore.fetchPool(eventId.value);
});

watch(eventId, async (newId) => {
  if (newId) {
    await eventStore.fetchEvent(newId);
    await predStore.fetchOdds(newId);
    await predStore.fetchPool(newId);
  }
});

// Payout preview based on current AMM state
const payoutPreview = computed(() => {
  if (!predStore.pool || !betAmount.value) return 0;
  const pool = predStore.pool;
  const d = betAmount.value;

  if (selectedOption.value === 'A') {
    const newPoolB = pool.poolB + d;
    const newPoolA = pool.k / newPoolB;
    const shares = pool.poolA - newPoolA;
    return shares.toFixed(2);
  } else {
    const newPoolA = pool.poolA + d;
    const newPoolB = pool.k / newPoolA;
    const shares = pool.poolB - newPoolB;
    return shares.toFixed(2);
  }
});

async function handleBet() {
  if (!auth.isLoggedIn) {
    ElMessage.warning('请先登录');
    return;
  }
  submitting.value = true;
  try {
    const result = await predStore.placeBet(eventId.value, selectedOption.value, betAmount.value);
    ElMessage.success(`下注成功！获得 ${result.shares.toFixed(2)} 份额`);
    await userStore.refreshBalance();
  } catch {
    // Error handled by interceptor
  } finally {
    submitting.value = false;
  }
}

// ECharts config for odds history
const chartOption = computed(() => {
  // Use event.oddsHistory if available, otherwise generate sample points
  const history = event.value?.oddsHistory || [];
  const times = history.map((h) => h.time || '');
  const probAData = history.map((h) => ((h.probA ?? 0.5) * 100).toFixed(1));
  const probBData = history.map((h) => ((h.probB ?? 0.5) * 100).toFixed(1));

  const optionA = event.value?.predictionOptions?.[0] || '选项 A';
  const optionB = event.value?.predictionOptions?.[1] || '选项 B';

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b', fontSize: 13 },
      formatter(params) {
        let html = `<div style="font-weight:600;margin-bottom:4px">${params[0].axisValue}</div>`;
        for (const p of params) {
          html += `<div>${p.marker} ${p.seriesName}: <b>${p.value}%</b></div>`;
        }
        return html;
      },
    },
    legend: {
      data: [optionA, optionB],
      bottom: 0,
      textStyle: { fontSize: 13 },
    },
    grid: {
      top: 20,
      right: 20,
      bottom: 40,
      left: 50,
      containLabel: false,
    },
    xAxis: {
      type: 'category',
      data: times,
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%', color: '#94a3b8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
    },
    series: [
      {
        name: optionA,
        type: 'line',
        data: probAData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#6366f1' },
        itemStyle: { color: '#6366f1' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(99,102,241,0.25)' },
              { offset: 1, color: 'rgba(99,102,241,0.02)' },
            ],
          },
        },
      },
      {
        name: optionB,
        type: 'line',
        data: probBData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#ec4899' },
        itemStyle: { color: '#ec4899' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(236,72,153,0.25)' },
              { offset: 1, color: 'rgba(236,72,153,0.02)' },
            ],
          },
        },
      },
    ],
  };
});

const statusLabel = {
  CREATED: '已创建',
  PREDICTION_OPEN: '预测中',
  TICKET_OPEN: '购票中',
  ONGOING: '进行中',
  SETTLED: '已结算',
};
</script>

<template>
  <div v-if="event" class="event-detail">
    <!-- Header -->
    <GlassCard class="event-header" padding="32px">
      <div class="header-top">
        <span class="event-type">{{ event.type }}</span>
        <span class="status-badge glass-subtle">{{ statusLabel[event.status] || event.status }}</span>
      </div>
      <h1 class="event-title">{{ event.title }}</h1>
      <div class="teams-display">
        <span class="team team-a">{{ event.teams?.[0] }}</span>
        <span class="vs-badge">VS</span>
        <span class="team team-b">{{ event.teams?.[1] }}</span>
      </div>
      <ProbabilityBar
        v-if="odds"
        :prob-a="odds.probA"
        :label-a="event.predictionOptions?.[0] || 'A'"
        :label-b="event.predictionOptions?.[1] || 'B'"
        height="36px"
        style="margin-top: 20px"
      />
    </GlassCard>

    <div class="detail-grid">
      <!-- Odds chart -->
      <GlassCard class="chart-card" padding="24px">
        <h2 class="card-title">概率走势</h2>
        <VChart
          :option="chartOption"
          style="height: 320px; width: 100%"
          autoresize
        />
      </GlassCard>

      <!-- Bet panel -->
      <GlassCard class="bet-panel" padding="24px">
        <h2 class="card-title">下注预测</h2>

        <div class="bet-options">
          <ElRadioGroup v-model="selectedOption" size="large">
            <ElRadioButton value="A">
              {{ event.predictionOptions?.[0] || 'A' }}
            </ElRadioButton>
            <ElRadioButton value="B">
              {{ event.predictionOptions?.[1] || 'B' }}
            </ElRadioButton>
          </ElRadioGroup>
        </div>

        <div class="bet-amount">
          <label class="form-label">投注金额（浙币）</label>
          <ElSlider v-model="betAmount" :min="1" :max="500" :step="10" show-input />
        </div>

        <div class="payout-preview glass-subtle">
          <div class="preview-row">
            <span class="preview-label">投注</span>
            <span class="preview-value">{{ betAmount }} 浙币</span>
          </div>
          <div class="preview-row">
            <span class="preview-label">预计份额</span>
            <span class="preview-value highlight">{{ payoutPreview }}</span>
          </div>
        </div>

        <button
          class="bet-btn"
          :disabled="submitting || event.status !== 'PREDICTION_OPEN'"
          @click="handleBet"
        >
          {{ submitting ? '提交中...' : event.status === 'PREDICTION_OPEN' ? '确认下注' : '预测未开放' }}
        </button>

        <!-- Pool info -->
        <div v-if="predStore.pool" class="pool-info">
          <span>总投注量: {{ predStore.pool.totalVolume }} 浙币</span>
        </div>
      </GlassCard>
    </div>
  </div>

  <div v-else class="loading-state">
    <p>加载中...</p>
  </div>
</template>

<style scoped>
.event-detail {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.event-header {
  text-align: center;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.event-type {
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-secondary);
  letter-spacing: 0.05em;
}

.status-badge {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
}

.event-title {
  font-size: 28px;
  font-weight: 800;
}

.teams-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 8px;
  font-size: 20px;
}

.team {
  font-weight: 700;
}

.team-a { color: var(--color-primary); }
.team-b { color: #ec4899; }

.vs-badge {
  font-size: 14px;
  font-weight: 800;
  color: var(--color-danger);
  padding: 4px 10px;
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.08);
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* Bet panel */
.bet-options {
  margin-bottom: 20px;
}

.bet-amount {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.payout-preview {
  padding: 16px;
  margin-bottom: 16px;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
}

.preview-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.preview-value {
  font-size: 14px;
  font-weight: 600;
}

.preview-value.highlight {
  color: var(--color-primary);
  font-size: 16px;
}

.bet-btn {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--color-primary), #8b5cf6);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-base);
}

.bet-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.3);
}

.bet-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pool-info {
  text-align: center;
  margin-top: 12px;
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.loading-state {
  text-align: center;
  padding: 80px 0;
  color: var(--color-text-tertiary);
}
</style>
```

---

## Task 12 — Remaining Pages (Ticket Hall, Profile, Admin)

- [ ] Create `client/src/views/TicketHallView.vue`
- [ ] Create `client/src/views/ProfileView.vue`
- [ ] Create `client/src/views/AdminView.vue`

### client/src/views/TicketHallView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { ElMessage } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';
import CountdownTimer from '../components/CountdownTimer.vue';
import QRCode from '../components/QRCode.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    eventStore.fetchEvents({ status: 'TICKET_OPEN' }),
    ticketStore.fetchMyTickets(),
  ]);
});

const ticketEvents = computed(() =>
  eventStore.events.filter((e) => e.status === 'TICKET_OPEN')
);

const wonTickets = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'WON' || t.status === 'CLAIMED')
);

const pendingApplications = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'PENDING')
);

async function handleApply(eventID) {
  try {
    await ticketStore.applyTicket(eventID);
    ElMessage.success('申请已提交');
    await ticketStore.fetchMyTickets();
  } catch {
    // Error handled by interceptor
  }
}

async function handleClaim(ticketID) {
  try {
    await ticketStore.claimTicket(ticketID);
    ElMessage.success('票据已领取');
  } catch {
    // Error handled by interceptor
  }
}

async function handleRefund(ticketID) {
  try {
    await ticketStore.refundTicket(ticketID);
    ElMessage.success('退票成功');
  } catch {
    // Error handled by interceptor
  }
}
</script>

<template>
  <div class="ticket-hall">
    <h1 class="page-title">票务大厅</h1>

    <!-- Available ticket events -->
    <section class="section">
      <h2 class="section-title">正在售票的赛事</h2>
      <div class="ticket-events-grid">
        <GlassCard
          v-for="event in ticketEvents"
          :key="event.id"
          class="ticket-event-card"
        >
          <div class="te-header">
            <h3 class="te-title">{{ event.title }}</h3>
            <span class="te-quota glass-subtle">余票 {{ event.ticketTotal }}</span>
          </div>
          <div class="te-teams">{{ event.teams?.[0] }} VS {{ event.teams?.[1] }}</div>
          <div class="te-countdown">
            <span class="te-countdown-label">抽签倒计时</span>
            <CountdownTimer :target-time="event.lotteryTime || '2026-04-10T20:00:00'" />
          </div>
          <button class="apply-btn" @click="handleApply(event.id)">
            申请购票
          </button>
        </GlassCard>
      </div>
      <p v-if="!ticketEvents.length" class="empty-text">暂无售票中的赛事</p>
    </section>

    <!-- My applications -->
    <section class="section">
      <h2 class="section-title">我的申请</h2>
      <div class="applications-list">
        <GlassCard
          v-for="app in pendingApplications"
          :key="app.eventID + app.userID"
          variant="subtle"
          padding="16px"
          class="app-item"
        >
          <span class="app-event">{{ app.eventID }}</span>
          <span class="app-status pending">等待抽签</span>
        </GlassCard>
      </div>
      <p v-if="!pendingApplications.length" class="empty-text">暂无待处理的申请</p>
    </section>

    <!-- Won tickets -->
    <section class="section">
      <h2 class="section-title">我的票据</h2>
      <div class="tickets-grid">
        <GlassCard
          v-for="ticket in wonTickets"
          :key="ticket.ticketID"
          class="ticket-card"
        >
          <h3 class="ticket-event-name">{{ ticket.eventID }}</h3>
          <div class="ticket-status">
            <span v-if="ticket.status === 'WON'" class="status-won">中签 - 待领取</span>
            <span v-else-if="ticket.status === 'CLAIMED'" class="status-claimed">已领取</span>
          </div>

          <!-- QR code for claimed tickets -->
          <div v-if="ticket.status === 'CLAIMED' && ticket.claimHash" class="qr-section">
            <QRCode :value="ticket.claimHash" :size="180" />
            <p class="qr-hint">入场时出示此二维码</p>
          </div>

          <div class="ticket-actions">
            <button
              v-if="ticket.status === 'WON'"
              class="claim-btn"
              @click="handleClaim(ticket.ticketID)"
            >
              领取票据
            </button>
            <button
              class="refund-btn"
              @click="handleRefund(ticket.ticketID)"
            >
              退票
            </button>
          </div>
        </GlassCard>
      </div>
      <p v-if="!wonTickets.length" class="empty-text">暂无票据</p>
    </section>
  </div>
</template>

<style scoped>
.ticket-hall {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.section {
  margin-bottom: 40px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.ticket-events-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.te-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.te-title {
  font-size: 16px;
  font-weight: 700;
}

.te-quota {
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-success);
}

.te-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.te-countdown {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.te-countdown-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.apply-btn {
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.apply-btn:hover {
  background: var(--color-primary-dark);
}

.applications-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.app-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.app-event {
  font-weight: 600;
}

.app-status.pending {
  color: var(--color-warning);
  font-weight: 600;
  font-size: 13px;
}

.tickets-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.ticket-card {
  text-align: center;
}

.ticket-event-name {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
}

.status-won {
  color: var(--color-warning);
  font-weight: 600;
}

.status-claimed {
  color: var(--color-success);
  font-weight: 600;
}

.qr-section {
  margin: 16px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qr-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.ticket-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.claim-btn {
  flex: 1;
  padding: 8px;
  border: none;
  border-radius: 8px;
  background: var(--color-success);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.refund-btn {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  background: transparent;
  color: var(--color-danger);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-base);
}

.refund-btn:hover {
  background: rgba(239, 68, 68, 0.08);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  padding: 24px 0;
  text-align: center;
}
</style>
```

### client/src/views/ProfileView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { RadarChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import VChart from 'vue-echarts';
import { useUserStore } from '../stores/user.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

use([CanvasRenderer, RadarChart, TitleComponent, TooltipComponent, LegendComponent]);

const userStore = useUserStore();
const predStore = usePredictionStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    userStore.fetchProfile(),
    predStore.fetchMyBets(),
    predStore.fetchMyScore(),
    ticketStore.fetchMyTickets(),
  ]);
});

const profile = computed(() => userStore.profile);

// Radar chart — accuracy by event type
const radarOption = computed(() => {
  // Group bets by event type and compute per-type accuracy
  const typeMap = {};
  const types = ['basketball', 'football', 'esports', 'badminton', 'track'];
  const typeLabels = {
    basketball: '篮球',
    football: '足球',
    esports: '电竞',
    badminton: '羽毛球',
    track: '田径',
  };

  for (const t of types) {
    typeMap[t] = { total: 0, correct: 0 };
  }

  for (const bet of predStore.myBets) {
    const t = bet.eventType || 'basketball';
    if (typeMap[t]) {
      typeMap[t].total++;
      if (bet.won) typeMap[t].correct++;
    }
  }

  const indicators = types.map((t) => ({
    name: typeLabels[t] || t,
    max: 100,
  }));

  const values = types.map((t) => {
    const { total, correct } = typeMap[t];
    return total > 0 ? Math.round((correct / total) * 100) : 0;
  });

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b' },
    },
    radar: {
      indicator: indicators,
      shape: 'polygon',
      axisName: {
        color: '#64748b',
        fontSize: 13,
      },
      splitArea: {
        areaStyle: {
          color: [
            'rgba(99,102,241,0.03)',
            'rgba(99,102,241,0.06)',
            'rgba(99,102,241,0.09)',
            'rgba(99,102,241,0.12)',
            'rgba(99,102,241,0.15)',
          ],
        },
      },
      splitLine: {
        lineStyle: { color: 'rgba(0,0,0,0.06)' },
      },
      axisLine: {
        lineStyle: { color: 'rgba(0,0,0,0.08)' },
      },
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            value: values,
            name: '准确率',
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: '#6366f1', width: 2 },
            itemStyle: { color: '#6366f1' },
            areaStyle: { color: 'rgba(99,102,241,0.2)' },
          },
        ],
      },
    ],
  };
});

// Achievement badges
const achievements = computed(() => {
  const list = [];
  const p = profile.value;
  if (!p) return list;

  if (p.totalBets >= 1) list.push({ label: '初出茅庐', desc: '完成第一次预测', icon: '\u{1F3AF}' });
  if (p.totalBets >= 50) list.push({ label: '预测达人', desc: '完成 50 次预测', icon: '\u{1F525}' });
  if (p.accuracyRate >= 0.8 && p.totalBets >= 10) list.push({ label: '神算子', desc: '准确率超过 80%', icon: '\u{1F52E}' });
  if (p.balance >= 5000) list.push({ label: '富甲一方', desc: '余额超过 5000', icon: '\u{1F4B0}' });

  return list;
});
</script>

<template>
  <div class="profile-page">
    <h1 class="page-title">个人中心</h1>

    <div class="profile-grid" v-if="profile">
      <!-- Balance + stats -->
      <GlassCard class="balance-card" padding="32px">
        <div class="balance-header">
          <h2 class="balance-title">{{ profile.name }}</h2>
          <span class="user-role glass-subtle">{{ profile.role }}</span>
        </div>
        <div class="stats-row">
          <div class="stat-item">
            <span class="stat-value primary">{{ profile.balance }}</span>
            <span class="stat-label">浙币余额</span>
          </div>
          <div class="stat-item">
            <span class="stat-value success">{{ (profile.accuracyRate * 100).toFixed(1) }}%</span>
            <span class="stat-label">预测准确率</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.totalBets }}</span>
            <span class="stat-label">总预测次数</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.correctBets }}</span>
            <span class="stat-label">正确次数</span>
          </div>
        </div>
      </GlassCard>

      <!-- Radar chart -->
      <GlassCard class="radar-card" padding="24px">
        <h2 class="card-title">分项准确率</h2>
        <VChart
          :option="radarOption"
          style="height: 300px; width: 100%"
          autoresize
        />
      </GlassCard>

      <!-- Bet history -->
      <GlassCard class="history-card" padding="24px">
        <h2 class="card-title">预测记录</h2>
        <div class="bet-list">
          <div
            v-for="bet in predStore.myBets"
            :key="bet.betID || bet.eventID + bet.timestamp"
            class="bet-item glass-subtle"
          >
            <div class="bet-info">
              <span class="bet-event">{{ bet.eventID }}</span>
              <span class="bet-option">{{ bet.option }}</span>
            </div>
            <div class="bet-meta">
              <span class="bet-amount">{{ bet.amount }} 浙币</span>
              <span class="bet-shares">{{ bet.shares?.toFixed(2) }} 份额</span>
              <span
                class="bet-result"
                :class="bet.won ? 'won' : bet.won === false ? 'lost' : 'pending'"
              >
                {{ bet.won ? '胜' : bet.won === false ? '负' : '待定' }}
              </span>
            </div>
          </div>
        </div>
        <p v-if="!predStore.myBets.length" class="empty-text">暂无预测记录</p>
      </GlassCard>

      <!-- Achievements -->
      <GlassCard class="achievements-card" padding="24px">
        <h2 class="card-title">成就徽章</h2>
        <div class="badge-grid">
          <div
            v-for="badge in achievements"
            :key="badge.label"
            class="badge-item glass-subtle"
          >
            <span class="badge-icon">{{ badge.icon }}</span>
            <span class="badge-label">{{ badge.label }}</span>
            <span class="badge-desc">{{ badge.desc }}</span>
          </div>
        </div>
        <p v-if="!achievements.length" class="empty-text">继续努力，解锁成就吧</p>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.profile-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

/* Balance card spans full width */
.balance-card {
  grid-column: 1 / -1;
}

.balance-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.balance-title {
  font-size: 24px;
  font-weight: 800;
}

.user-role {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.stats-row {
  display: flex;
  justify-content: space-around;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
}

.stat-value.primary { color: var(--color-primary); }
.stat-value.success { color: var(--color-success); }

.stat-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* History card spans full width */
.history-card {
  grid-column: 1 / -1;
}

.bet-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.bet-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.bet-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bet-event {
  font-weight: 600;
}

.bet-option {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.bet-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
}

.bet-amount { color: var(--color-text-secondary); }
.bet-shares { color: var(--color-text-tertiary); }

.bet-result {
  font-weight: 700;
  padding: 2px 10px;
  border-radius: 6px;
}

.bet-result.won { color: var(--color-success); background: rgba(34, 197, 94, 0.1); }
.bet-result.lost { color: var(--color-danger); background: rgba(239, 68, 68, 0.1); }
.bet-result.pending { color: var(--color-warning); background: rgba(245, 158, 11, 0.1); }

/* Achievements */
.achievements-card {
  grid-column: 1 / -1;
}

.badge-grid {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.badge-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 20px;
  min-width: 120px;
  text-align: center;
  gap: 4px;
}

.badge-icon {
  font-size: 32px;
}

.badge-label {
  font-size: 14px;
  font-weight: 700;
}

.badge-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  text-align: center;
  padding: 20px 0;
}
</style>
```

### client/src/views/AdminView.vue

```vue
<script setup>
import { ref, onMounted, computed } from 'vue';
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElInputNumber,
  ElButton,
  ElTable,
  ElTableColumn,
  ElTag,
  ElMessage,
  ElMessageBox,
} from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await eventStore.fetchEvents();
});

// --- Create Event Form ---
const createForm = ref({
  title: '',
  type: 'basketball',
  teamA: '',
  teamB: '',
  ticketTotal: 100,
  optionA: '',
  optionB: '',
});
const creating = ref(false);

const eventTypes = [
  { value: 'basketball', label: '篮球' },
  { value: 'football', label: '足球' },
  { value: 'esports', label: '电竞' },
  { value: 'badminton', label: '羽毛球' },
  { value: 'track', label: '田径' },
];

async function handleCreate() {
  const f = createForm.value;
  if (!f.title || !f.teamA || !f.teamB || !f.optionA || !f.optionB) {
    ElMessage.warning('请填写所有必填字段');
    return;
  }

  creating.value = true;
  try {
    await eventStore.createEvent({
      title: f.title,
      type: f.type,
      teams: [f.teamA, f.teamB],
      ticketTotal: f.ticketTotal,
      predictionOptions: [f.optionA, f.optionB],
    });
    ElMessage.success('赛事创建成功');
    // Reset form
    createForm.value = {
      title: '', type: 'basketball', teamA: '', teamB: '',
      ticketTotal: 100, optionA: '', optionB: '',
    };
  } catch {
    // Handled by interceptor
  } finally {
    creating.value = false;
  }
}

// --- Event Manager ---
const statusFlow = {
  CREATED: 'PREDICTION_OPEN',
  PREDICTION_OPEN: 'TICKET_OPEN',
  TICKET_OPEN: 'ONGOING',
};

const statusTagType = {
  CREATED: 'info',
  PREDICTION_OPEN: 'warning',
  TICKET_OPEN: '',
  ONGOING: 'success',
  SETTLED: 'danger',
};

async function advanceStatus(event) {
  const nextStatus = statusFlow[event.status];
  if (!nextStatus) return;

  try {
    await ElMessageBox.confirm(
      `确认将「${event.title}」状态推进为 ${nextStatus}？`,
      '确认操作'
    );
    await eventStore.updateStatus(event.id, nextStatus);
    ElMessage.success('状态已更新');
  } catch {
    // Cancelled or error
  }
}

async function settleEvent(event) {
  try {
    const { value: outcome } = await ElMessageBox.prompt(
      '请输入赛事结果（选项名称）',
      '录入结果',
      { inputPlaceholder: event.predictionOptions?.join(' 或 ') }
    );
    if (!outcome) return;
    await eventStore.recordResult(event.id, outcome);
    ElMessage.success('结算完成');
  } catch {
    // Cancelled or error
  }
}

async function runLottery(event) {
  try {
    await ElMessageBox.confirm(`确认对「${event.title}」执行抽签？`, '确认');
    await ticketStore.runLottery(event.id);
    ElMessage.success('抽签完成');
  } catch {
    // Cancelled or error
  }
}

// --- System stats ---
const totalEvents = computed(() => eventStore.events.length);
const activeEvents = computed(() =>
  eventStore.events.filter((e) => !['SETTLED', 'CREATED'].includes(e.status)).length
);
</script>

<template>
  <div class="admin-page">
    <h1 class="page-title">管理控制台</h1>

    <div class="admin-grid">
      <!-- System stats -->
      <GlassCard class="stats-card" padding="24px">
        <h2 class="card-title">系统概览</h2>
        <div class="admin-stats">
          <div class="admin-stat">
            <span class="admin-stat-value">{{ totalEvents }}</span>
            <span class="admin-stat-label">总赛事</span>
          </div>
          <div class="admin-stat">
            <span class="admin-stat-value">{{ activeEvents }}</span>
            <span class="admin-stat-label">进行中</span>
          </div>
        </div>
      </GlassCard>

      <!-- Create event form -->
      <GlassCard class="create-card" padding="28px">
        <h2 class="card-title">创建赛事</h2>
        <ElForm label-position="top" :model="createForm">
          <ElFormItem label="赛事名称">
            <ElInput v-model="createForm.title" placeholder="例：院际篮球决赛" />
          </ElFormItem>

          <ElFormItem label="赛事类型">
            <ElSelect v-model="createForm.type" style="width: 100%">
              <ElOption
                v-for="t in eventTypes"
                :key="t.value"
                :label="t.label"
                :value="t.value"
              />
            </ElSelect>
          </ElFormItem>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="队伍 A">
              <ElInput v-model="createForm.teamA" placeholder="队伍名称" />
            </ElFormItem>
            <ElFormItem label="队伍 B">
              <ElInput v-model="createForm.teamB" placeholder="队伍名称" />
            </ElFormItem>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="预测选项 A">
              <ElInput v-model="createForm.optionA" placeholder="例：教院胜" />
            </ElFormItem>
            <ElFormItem label="预测选项 B">
              <ElInput v-model="createForm.optionB" placeholder="例：丹青胜" />
            </ElFormItem>
          </div>

          <ElFormItem label="票务总量">
            <ElInputNumber v-model="createForm.ticketTotal" :min="0" :max="10000" style="width: 100%" />
          </ElFormItem>

          <ElButton
            type="primary"
            :loading="creating"
            style="width: 100%; margin-top: 8px"
            @click="handleCreate"
          >
            创建赛事
          </ElButton>
        </ElForm>
      </GlassCard>

      <!-- Event manager table -->
      <GlassCard class="manager-card" padding="24px">
        <h2 class="card-title">赛事管理</h2>
        <ElTable :data="eventStore.events" stripe style="width: 100%">
          <ElTableColumn prop="title" label="赛事名称" min-width="180" />
          <ElTableColumn prop="type" label="类型" width="80" />
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }">
              <ElTag :type="statusTagType[row.status] || 'info'" size="small">
                {{ row.status }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="280">
            <template #default="{ row }">
              <ElButton
                v-if="statusFlow[row.status]"
                type="primary"
                size="small"
                @click="advanceStatus(row)"
              >
                推进状态
              </ElButton>
              <ElButton
                v-if="row.status === 'ONGOING'"
                type="warning"
                size="small"
                @click="settleEvent(row)"
              >
                录入结果
              </ElButton>
              <ElButton
                v-if="row.status === 'TICKET_OPEN'"
                type="success"
                size="small"
                @click="runLottery(row)"
              >
                执行抽签
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.admin-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.stats-card {
  grid-column: 1 / -1;
}

.manager-card {
  grid-column: 1 / -1;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

.admin-stats {
  display: flex;
  gap: 48px;
}

.admin-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.admin-stat-value {
  font-size: 36px;
  font-weight: 800;
  color: var(--color-primary);
}

.admin-stat-label {
  font-size: 14px;
  color: var(--color-text-tertiary);
}
</style>
```

---

## Summary — Task Dependency Order

| # | Task | Depends on |
|---|------|-----------|
| 1 | Server Scaffolding (app.js, config, errorHandler) | -- |
| 2 | Fabric Services (wallet, CA, gateway) | Task 1 |
| 3 | Auth System (JWT middleware + routes) | Task 2 |
| 4 | Event Routes | Task 3 |
| 5 | Prediction Routes | Task 3 |
| 6 | Ticket + User Routes | Task 3 |
| 7 | Client Scaffolding + Liquid Glass CSS + Composables + API + Router | -- |
| 8 | Reusable Components (GlassCard, Navbar, ProbabilityBar, SparkLine, Countdown, QR) | Task 7 |
| 9 | Pinia Stores (auth, events, prediction, ticket, user) | Task 7 |
| 10 | Home Page + Login Page | Tasks 8, 9 |
| 11 | Event Detail Page (ECharts odds chart + bet panel) | Tasks 8, 9 |
| 12 | Remaining Pages (Ticket Hall, Profile with radar chart, Admin) | Tasks 8, 9 |

Tasks 1-6 (backend) and Tasks 7-9 (frontend foundation) can proceed in parallel. Tasks 10-12 depend on Tasks 8 and 9 being complete.
