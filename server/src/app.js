import express from 'express';
import cors from 'cors';
import { fileURLToPath } from 'node:url';
import config, { validateProductionConfig } from './config/index.js';
import { errorHandler } from './middleware/errorHandler.js';
import authRoutes from './routes/auth.js';
import userRoutes from './routes/users.js';
import financeRoutes from './routes/finance.js';
import activityRoutes from './routes/activities.js';
import { closeDB, initDB } from './services/wallet.js';
import { enrollAdmin } from './services/caService.js';
import { startClaimWorker } from './services/claimWorker.js';
import { closeAllGateways } from './services/fabricGateway.js';

export const app = express();

// --------------- Middleware ---------------
app.disable('x-powered-by');
app.use((_req, res, next) => {
  res.set({
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'no-referrer',
    'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
  });
  next();
});
app.use(cors({
  origin(origin, callback) {
    if (!origin || config.corsOrigins.includes(origin)) return callback(null, true);
    const error = new Error('该来源不允许访问接口');
    error.code = 'FORBIDDEN';
    return callback(error);
  },
}));
app.use(express.json({ limit: config.jsonLimit }));

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

app.use('/api', (req, _res, next) => {
  const error = new Error(`接口不存在：${req.method} ${req.originalUrl}`);
  error.code = 'NOT_FOUND';
  next(error);
});

// --------------- Error handler (must be last) ---------------
app.use(errorHandler);

// --------------- Start ---------------
async function main() {
  validateProductionConfig(config);
  initDB();

  // Enroll CA admins so public queries (e.g. GET /events) have a Fabric identity
  console.log('[EventChain] Enrolling CA admins...');
  await Promise.all(['PlatformMSP', 'OrganizerMSP', 'StudentMSP'].map(enrollAdmin));
  console.log('[EventChain] CA admins enrolled.');
  const claimWorker = startClaimWorker();

  const server = app.listen(config.port, () => {
    console.log(`[EventChain] Server listening on http://localhost:${config.port}`);
  });
  let stopping = false;
  const shutdown = (signal) => {
    if (stopping) return;
    stopping = true;
    console.log(`[EventChain] ${signal} received, shutting down...`);
    if (claimWorker) clearInterval(claimWorker);
    server.close(() => {
      closeAllGateways();
      closeDB();
      process.exit(0);
    });
    setTimeout(() => process.exit(1), 10_000).unref();
  };
  process.once('SIGINT', () => shutdown('SIGINT'));
  process.once('SIGTERM', () => shutdown('SIGTERM'));
}

if (process.argv[1] && fileURLToPath(import.meta.url) === fileURLToPath(new URL(`file://${process.argv[1]}`))) {
  main().catch((err) => {
    console.error('Failed to start server:', err);
    process.exit(1);
  });
}
