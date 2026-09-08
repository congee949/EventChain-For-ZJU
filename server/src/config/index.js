import 'dotenv/config';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '../..');

export default {
  demoMode: process.env.DEMO_MODE !== 'false',
  openStudentRegistration: process.env.OPEN_STUDENT_REGISTRATION === 'true',
  port: parseInt(process.env.PORT, 10) || 3000,
  corsOrigins: (process.env.CORS_ORIGINS || 'http://localhost:5173,http://127.0.0.1:5173')
    .split(',').map((value) => value.trim()).filter(Boolean),
  jsonLimit: process.env.JSON_LIMIT || '256kb',

  jwt: {
    secret: process.env.JWT_SECRET || 'eventchain-dev-secret',
    expiresIn: process.env.JWT_EXPIRES_IN || '24h',
  },

  fabric: {
    channelName: process.env.FABRIC_CHANNEL || 'eventchain',
    chaincode: {
      event: process.env.CC_EVENT || 'event-cc',
      prediction: process.env.CC_PREDICTION || 'prediction-cc',
      ticket: process.env.CC_TICKET || 'ticket-cc',
      token: process.env.CC_TOKEN || 'token-cc',
      finance: process.env.CC_FINANCE || 'finance',
      activity: process.env.CC_ACTIVITY || 'activity',
    },
  },

  identity: {
    lookupKey: process.env.IDENTITY_LOOKUP_KEY || crypto.createHash('sha256').update(`${process.env.JWT_SECRET || 'eventchain-dev-secret'}:lookup`).digest('hex'),
    encryptionKey: process.env.IDENTITY_ENCRYPTION_KEY || crypto.createHash('sha256').update(`${process.env.JWT_SECRET || 'eventchain-dev-secret'}:encryption`).digest('hex'),
    demoBootstrapKey: process.env.DEMO_BOOTSTRAP_KEY || 'eventchain-demo-bootstrap',
  },

  claimWorker: {
    enabled: process.env.CLAIM_WORKER_ENABLED !== 'false',
    intervalMs: Math.max(60_000, parseInt(process.env.CLAIM_WORKER_INTERVAL_MS, 10) || 300_000),
    concurrency: Math.min(10, Math.max(1, parseInt(process.env.CLAIM_WORKER_CONCURRENCY, 10) || 5)),
    pageSize: Math.min(100, Math.max(1, parseInt(process.env.CLAIM_WORKER_PAGE_SIZE, 10) || 100)),
  },

  gatewayCache: {
    ttlMs: Math.max(60_000, parseInt(process.env.GATEWAY_CACHE_TTL_MS, 10) || 600_000),
  },

  walletPath: path.resolve(root, process.env.WALLET_PATH || './wallet'),
  dbPath: path.resolve(root, process.env.DB_PATH || './data/users.db'),
};

export function validateProductionConfig(currentConfig) {
  if (process.env.NODE_ENV !== 'production') return;
  const insecure = [];
  if (currentConfig.jwt.secret.length < 32 || currentConfig.jwt.secret.includes('eventchain-dev-secret')) insecure.push('JWT_SECRET');
  if (!/^[0-9a-f]{64}$/i.test(process.env.IDENTITY_LOOKUP_KEY || '')) insecure.push('IDENTITY_LOOKUP_KEY');
  if (!/^[0-9a-f]{64}$/i.test(process.env.IDENTITY_ENCRYPTION_KEY || '')) insecure.push('IDENTITY_ENCRYPTION_KEY');
  if (currentConfig.demoMode && (currentConfig.identity.demoBootstrapKey === 'eventchain-demo-bootstrap' || currentConfig.identity.demoBootstrapKey.startsWith('replace-with-'))) insecure.push('DEMO_BOOTSTRAP_KEY');
  if (insecure.length) {
    throw new Error(`生产环境密钥配置不安全：${insecure.join(', ')}`);
  }
}
