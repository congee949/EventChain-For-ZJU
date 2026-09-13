import config from '../config/index.js';
import { badgeService, mapBadgeBusinessError } from './badgeService.js';
import { createBadgeReconcileWorker } from './badgeReconcileWorker.js';

export const badgeReconcileWorker = createBadgeReconcileWorker({
  journalPath: config.badgePendingJournalPath,
  getClaimReceipt: (accountId, seriesId, refId) => badgeService.findClaimReceipt(accountId, seriesId, refId),
  getMyBadge: (accountId, seriesId) => badgeService.getMyAward(accountId, seriesId),
  classifyTerminalError: mapBadgeBusinessError,
});

badgeService.setPendingTracker(badgeReconcileWorker);

export function startBadgeReconcileWorker() {
  return badgeReconcileWorker.start();
}

export function stopBadgeReconcileWorker() {
  return badgeReconcileWorker.stop();
}
