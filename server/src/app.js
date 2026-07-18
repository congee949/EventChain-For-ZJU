import express from 'express';
import cors from 'cors';
import { fileURLToPath } from 'node:url';
import config from './config/index.js';
import { errorHandler } from './middleware/errorHandler.js';
import authRoutes from './routes/auth.js';
import userRoutes from './routes/users.js';
import financeRoutes from './routes/finance.js';
import activityRoutes from './routes/activities.js';
import { initDB } from './services/wallet.js';
import { enrollAdmin } from './services/caService.js';
import { startClaimWorker } from './services/claimWorker.js';

export const app = express();

// --------------- Middleware ---------------
app.use(cors());
app.use(express.json());

// --------------- Health check ---------------
app.get('/api/health', (_req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// --------------- Routes ---------------
app.use('/api/v1', (_req, res) => res.status(410).json({
  error: true,
  code: 'V1_RETIRED',
  message: 'V1 已停用，请使用 /api/v2；旧积分与旧预测入口不再接受交易。',
}));
app.use('/api/v2/auth', authRoutes);
app.use('/api/v2/users', userRoutes);
app.use('/api/v2/finance', financeRoutes);
app.use('/api/v2/activities', activityRoutes);

// --------------- Error handler (must be last) ---------------
app.use(errorHandler);

// --------------- Start ---------------
async function main() {
  initDB();

  // Enroll CA admins so public queries (e.g. GET /events) have a Fabric identity
  console.log('[EventChain] Enrolling CA admins...');
  await enrollAdmin('PlatformMSP');
  await enrollAdmin('OrganizerMSP');
  await enrollAdmin('StudentMSP');
  console.log('[EventChain] CA admins enrolled.');
  startClaimWorker();

  app.listen(config.port, () => {
    console.log(`[EventChain] Server listening on http://localhost:${config.port}`);
  });
}

if (process.argv[1] && fileURLToPath(import.meta.url) === fileURLToPath(new URL(`file://${process.argv[1]}`))) {
  main().catch((err) => {
    console.error('Failed to start server:', err);
    process.exit(1);
  });
}
