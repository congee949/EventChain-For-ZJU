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

  walletPath: path.resolve(root, process.env.WALLET_PATH || './wallet'),
  dbPath: path.resolve(root, process.env.DB_PATH || './data/users.db'),
};
