import {
  CommitError,
  CommitStatusError,
  StatusCode,
} from '@hyperledger/fabric-gateway';
import {
  activityEvaluate as defaultActivityEvaluate,
  activitySubmit as defaultActivitySubmit,
} from './activityService.js';

const BUSINESS_ERRORS = Object.freeze({
  BADGE_INELIGIBLE: { status: 403, message: '当前账户没有链上领取资格' },
  BADGE_ALREADY_CLAIMED: { status: 409, message: '当前账户已领取该系列徽章' },
  BADGE_SUPPLY_EXHAUSTED: { status: 409, message: '该系列已达固定发行上限' },
  IDEMPOTENCY_CONFLICT: { status: 409, message: '同一 Idempotency-Key 已用于不同请求' },
  BADGE_SERIES_NOT_ACTIVE: { status: 409, message: '该徽章系列当前不可领取' },
  BADGE_CLAIM_WINDOW_CLOSED: { status: 410, message: '该徽章系列的领取窗口已关闭' },
});

const MISSING_RECEIPT_TEXT = 'badge claim receipt not found';

export class BadgeHttpError extends Error {
  constructor(code, status, message, data) {
    super(message);
    this.name = 'BadgeHttpError';
    this.code = code;
    this.status = status;
    this.data = data;
  }
}

function isBadgeUnavailable(error) {
  const text = errorText(error).toLowerCase();
  // Keep this deployment check narrow. Fabric uses phrases such as
  // "badge claim receipt not found" for a normal empty lookup; those must
  // remain a null receipt and must not become a 503.
  return /(?:unknown transaction|unknown function|no such function|function\s+[a-z0-9_.-]+\s+not found|chaincode\s+(?:is\s+)?(?:not installed|unavailable)|chaincode .*cannot be found)/.test(text);
}

function errorText(error) {
  return [
    error?.message,
    error?.cause?.message,
    error?.cause?.details,
    ...(Array.isArray(error?.details) ? error.details.map((detail) => detail?.message) : []),
  ].filter(Boolean).join(' | ');
}

export function mapBadgeBusinessError(error) {
  const text = errorText(error);
  for (const [code, meta] of Object.entries(BUSINESS_ERRORS)) {
    if (new RegExp(`(?:^|[^A-Z0-9_])${code}(?:$|[^A-Z0-9_])`).test(text)) {
      return new BadgeHttpError(code, meta.status, meta.message, error?.data);
    }
  }
  return null;
}

export function isBadgeMvccConflict(error) {
  if ((error instanceof CommitError || error?.name === 'CommitError')
    && (error?.code === StatusCode.MVCC_READ_CONFLICT || error?.code === StatusCode.PHANTOM_READ_CONFLICT)) {
    return true;
  }
  return /\b(?:MVCC_READ_CONFLICT|PHANTOM_READ_CONFLICT)\b/.test(errorText(error));
}

export function isBadgeUnknownCommit(error) {
  return error instanceof CommitStatusError
    || error?.name === 'CommitStatusError'
    || error?.name === 'SubmitError';
}

function isMissingReceipt(error) {
  return errorText(error).toLowerCase().includes(MISSING_RECEIPT_TEXT);
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function createBadgeService({
  activityEvaluate = defaultActivityEvaluate,
  activitySubmit = defaultActivitySubmit,
  wait = delay,
  mvccRetries = 3,
  reconcileDelays = [100, 250, 500],
  pendingTracker = null,
} = {}) {
  async function evaluate(accountId, transactionName, args = []) {
    try {
      return await activityEvaluate(accountId, transactionName, args);
    } catch (error) {
      if (isBadgeUnavailable(error)) {
        throw new BadgeHttpError('BADGE_CHAINCODE_UNAVAILABLE', 503, '徽章链码未升级或当前不可用');
      }
      throw error;
    }
  }

  async function submit(accountId, transactionName, args = []) {
    try {
      return await activitySubmit(accountId, transactionName, args);
    } catch (error) {
      if (isBadgeUnavailable(error)) {
        throw new BadgeHttpError('BADGE_CHAINCODE_UNAVAILABLE', 503, '徽章链码未升级或当前不可用');
      }
      throw error;
    }
  }

  async function findClaimReceipt(accountId, seriesId, refId) {
    try {
      return await evaluate(accountId, 'GetMyBadgeClaimReceipt', [seriesId, refId]);
    } catch (error) {
      if (isMissingReceipt(error)) return null;
      throw error;
    }
  }

  async function reconcileClaim(accountId, seriesId, refId, delays = reconcileDelays) {
    let immediate;
    try {
      immediate = await findClaimReceipt(accountId, seriesId, refId);
    } catch (error) {
      if (error instanceof BadgeHttpError) throw error;
      immediate = null;
    }
    if (immediate) return immediate;
    for (const milliseconds of delays) {
      await wait(milliseconds);
      let receipt;
      try {
        receipt = await findClaimReceipt(accountId, seriesId, refId);
      } catch (error) {
        if (error instanceof BadgeHttpError) throw error;
        receipt = null;
      }
      if (receipt) return receipt;
    }
    return null;
  }

  async function getClaimStatus(accountId, seriesId, refId) {
    const tracked = pendingTracker?.getStatus(accountId, seriesId, refId);
    if (tracked?.status === 'CONFIRMED') {
      return { pending: false, data: tracked.result };
    }
    if (tracked?.status === 'INVALID') {
      const meta = BUSINESS_ERRORS[tracked.result?.code];
      throw new BadgeHttpError(
        tracked.result?.code || 'BADGE_CLAIM_INVALID',
        meta?.status || 409,
        meta?.message || '徽章领取交易已被账本拒绝',
      );
    }
    // UNKNOWN is a journal hint; re-probe the ledger so a recovered peer can
    // confirm the receipt and an outage can surface as 503.
    try {
      const receipt = await findClaimReceipt(accountId, seriesId, refId);
      if (receipt) return { pending: false, data: await evaluate(accountId, 'GetMyBadge', [seriesId]) };
      return { pending: true, data: { seriesId, refId, status: 'UNKNOWN' } };
    } catch (error) {
      // A mapped chaincode/business error is authoritative. Only an absent
      // receipt or an otherwise indeterminate read may be represented as 202.
      if (error instanceof BadgeHttpError) throw error;
      const businessError = mapBadgeBusinessError(error);
      if (businessError) throw businessError;
      return { pending: true, data: { seriesId, refId, status: 'UNKNOWN' } };
    }
  }

  async function claimMyBadge(accountId, seriesId, refId) {
    for (let attempt = 0; attempt < mvccRetries; attempt += 1) {
      const existing = await findClaimReceipt(accountId, seriesId, refId);
      if (existing) {
        return { pending: false, data: await evaluate(accountId, 'GetMyBadge', [seriesId]) };
      }

      try {
        const award = await submit(accountId, 'ClaimMyBadge', [seriesId, refId]);
        return { pending: false, data: award };
      } catch (error) {
        const businessError = mapBadgeBusinessError(error);
        if (businessError) {
          if (businessError.code === 'BADGE_ALREADY_CLAIMED') {
            try {
              businessError.data = await evaluate(accountId, 'GetMyBadge', [seriesId]);
            } catch {
              // Preserve the explicit ledger error even if the convenience view is unavailable.
            }
          }
          throw businessError;
        }

        if (isBadgeMvccConflict(error)) {
          const receipt = await findClaimReceipt(accountId, seriesId, refId);
          if (receipt) {
            return { pending: false, data: await evaluate(accountId, 'GetMyBadge', [seriesId]) };
          }
          if (attempt + 1 < mvccRetries) {
            await wait(100 * (2 ** attempt));
            continue;
          }
          throw new BadgeHttpError(
            'BADGE_CONCURRENT_UPDATE',
            409,
            '徽章供应正在并发更新，请使用原 Idempotency-Key 重试',
          );
        }

        if (isBadgeUnknownCommit(error)) {
          const receipt = await reconcileClaim(accountId, seriesId, refId);
          if (receipt) {
            return { pending: false, data: await evaluate(accountId, 'GetMyBadge', [seriesId]) };
          }
          pendingTracker?.registerPending({ accountId, seriesId, refId });
          return {
            pending: true,
            code: 'BADGE_CLAIM_PENDING',
            data: { seriesId, refId, status: 'UNKNOWN' },
          };
        }
        throw error;
      }
    }
    throw new BadgeHttpError('BADGE_CONCURRENT_UPDATE', 409, '徽章供应正在并发更新');
  }

  return Object.freeze({
    setPendingTracker: (tracker) => { pendingTracker = tracker; },
    listSeries: (accountId) => evaluate(accountId, 'ListBadgeSeries'),
    getSeries: (accountId, seriesId) => evaluate(accountId, 'GetBadgeSeries', [seriesId]),
    listActivityLinks: (accountId, seriesId) => evaluate(accountId, 'ListBadgeActivityLinks', [seriesId]),
    getPublicInstanceById: (accountId, instanceId) => evaluate(accountId, 'GetBadgePublicInstanceByID', [instanceId]),
    listMyAwards: (accountId) => evaluate(accountId, 'ListMyBadges'),
    getMyAward: (accountId, seriesId) => evaluate(accountId, 'GetMyBadge', [seriesId]),
    findClaimReceipt,
    getClaimStatus,
    reconcileClaim,
    claimMyBadge,
    createSeries: (accountId, series) => submit(accountId, 'CreateBadgeSeries', [JSON.stringify(series)]),
    linkActivity: (accountId, seriesId, activityId, eligibilityQuota) => submit(
      accountId,
      'LinkBadgeActivity',
      [seriesId, activityId, String(eligibilityQuota)],
    ),
    activateSeries: (accountId, seriesId) => submit(accountId, 'ActivateBadgeSeries', [seriesId]),
    pauseSeries: (accountId, seriesId, reasonHash) => submit(accountId, 'PauseBadgeSeries', [seriesId, reasonHash]),
    resumeSeries: (accountId, seriesId) => submit(accountId, 'ResumeBadgeSeries', [seriesId]),
    closeSeries: (accountId, seriesId) => submit(accountId, 'CloseBadgeSeries', [seriesId]),
    finalizeSeries: (accountId, seriesId) => submit(accountId, 'FinalizeBadgeSeries', [seriesId]),
  });
}

export const badgeService = createBadgeService();
