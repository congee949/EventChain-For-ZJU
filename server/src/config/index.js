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
