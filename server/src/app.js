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
