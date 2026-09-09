import { ref } from 'vue';
import { defineStore } from 'pinia';
import api from '../api/index.js';

const PENDING_KEY_PREFIX = 'ec_badge_claim_ref:';
const TERMINAL_CODES = new Set([
  'BADGE_INELIGIBLE',
  'BADGE_SUPPLY_EXHAUSTED',
  'BADGE_SERIES_NOT_ACTIVE',
  'BADGE_CLAIM_WINDOW_CLOSED',
  'IDEMPOTENCY_CONFLICT',
]);

const wait = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

function newClaimRef() {
  const random = globalThis.crypto?.randomUUID?.()
    || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `badge-claim-${random}`.slice(0, 64).toLowerCase();
}

function listFrom(payload, key) {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.[key])) return payload[key];
  if (Array.isArray(payload?.items)) return payload.items;
  return [];
}

function isPending(payload) {
  return payload?.code === 'BADGE_CLAIM_PENDING'
    || payload?.status === 'PENDING'
    || payload?.status === 'UNKNOWN'
    || payload?.claimStatus === 'PENDING';
}

function pendingStorageKey(seriesId) {
  let account = 'anonymous';
  try { account = JSON.parse(localStorage.getItem('ec_user') || '{}')?.userId || account; } catch { /* ignore malformed stale session */ }
  return `${PENDING_KEY_PREFIX}${account}:${seriesId}`;
}

export function badgeErrorMessage(error) {
  const code = error?.code;
  const messages = {
    BADGE_INELIGIBLE: '你尚未取得这个系列的领取资格。完成指定活动签到后再来查看。',
    BADGE_SUPPLY_EXHAUSTED: '固定发行上限异常耗尽，领取已停止并等待运营方核查。',
    BADGE_SERIES_NOT_ACTIVE: '该系列当前未开放领取。',
    BADGE_CLAIM_WINDOW_CLOSED: '该系列的领取窗口已经结束。',
    IDEMPOTENCY_CONFLICT: '本次领取标识发生冲突，请联系运营方核查。',
  };
  return messages[code] || error?.message || '操作未完成，请稍后重试。';
}

export const useBadgesStore = defineStore('badges', () => {
  const series = ref([]);
  const awards = ref([]);
  const currentSeries = ref(null);
  const currentView = ref(null);
  const loading = ref(false);
  const detailLoading = ref(false);
  const error = ref('');
  const claimStates = ref({});

  function setClaimState(seriesId, state) {
    claimStates.value = {
      ...claimStates.value,
      [seriesId]: { ...(claimStates.value[seriesId] || {}), ...state },
    };
  }

  function pendingRef(seriesId) {
    return localStorage.getItem(pendingStorageKey(seriesId));
  }

  function rememberPendingRef(seriesId, refId) {
    localStorage.setItem(pendingStorageKey(seriesId), refId);
  }

  function clearPendingRef(seriesId) {
    localStorage.removeItem(pendingStorageKey(seriesId));
    localStorage.removeItem(`${pendingStorageKey(seriesId)}:request`);
  }

  async function loadSeriesList() {
    loading.value = true;
    error.value = '';
    try {
      const payload = await api.get('/badges/series', { silent: true });
      series.value = listFrom(payload, 'series');
      return series.value;
    } catch (cause) {
      error.value = badgeErrorMessage(cause);
      throw cause;
    } finally {
      loading.value = false;
    }
  }

  async function loadAwards() {
    const payload = await api.get('/badges/my-awards', { silent: true });
    awards.value = listFrom(payload, 'awards');
    return awards.value;
  }

  async function loadSeries(seriesId) {
    const payload = await api.get(`/badges/series/${encodeURIComponent(seriesId)}`, { silent: true });
    currentSeries.value = payload?.series || payload;
    return currentSeries.value;
  }

  async function loadSeriesLinks(seriesId) {
    const payload = await api.get(`/badges/series/${encodeURIComponent(seriesId)}/activities`, { silent: true });
    return listFrom(payload, 'activities');
  }

  async function loadMyAward(seriesId) {
    const payload = await api.get(`/badges/series/${encodeURIComponent(seriesId)}/my-award`, { silent: true });
    currentView.value = payload || null;
    if (payload?.series) currentSeries.value = payload.series;
    if (payload?.award) {
      clearPendingRef(seriesId);
      setClaimState(seriesId, { status: 'success', message: '徽章已写入链上。', refId: '' });
    } else if (pendingRef(seriesId)) {
      setClaimState(seriesId, {
        status: 'pending',
        message: '正在确认链上结果，请勿使用新的领取标识。',
        refId: pendingRef(seriesId),
      });
    }
    return currentView.value;
  }

  async function loadDetail(seriesId) {
    detailLoading.value = true;
    error.value = '';
    try {
      const [loadedSeries, view, activities] = await Promise.all([
        loadSeries(seriesId),
        loadMyAward(seriesId),
        loadSeriesLinks(seriesId),
      ]);
      currentSeries.value = { ...loadedSeries, activities };
      return { series: currentSeries.value, view };
    } catch (cause) {
      error.value = badgeErrorMessage(cause);
      throw cause;
    } finally {
      detailLoading.value = false;
    }
  }

  async function loadPublicInstance(instanceId) {
    return api.get(`/badges/public-instances/${encodeURIComponent(instanceId)}`, { silent: true });
  }

  async function finishClaim(seriesId) {
    const [loadedSeries, view, activities] = await Promise.all([
      loadSeries(seriesId),
      loadMyAward(seriesId),
      loadSeriesLinks(seriesId),
      loadAwards().catch(() => []),
    ]);
    currentSeries.value = { ...loadedSeries, activities };
    clearPendingRef(seriesId);
    setClaimState(seriesId, { status: 'success', message: '领取成功，徽章已写入链上。', refId: '' });
    return view;
  }

  async function reconcileClaim(seriesId, refId, attempts = 20) {
    // The server may return a canonical, hashed receipt reference that differs
    // from the browser's Idempotency-Key. Persist that reference so reloads and
    // retries query the exact ledger receipt instead of issuing a new claim.
    rememberPendingRef(seriesId, refId);
    setClaimState(seriesId, {
      status: 'pending',
      message: '正在确认链上结果，请勿关闭或重复领取。',
      refId,
    });
    for (let attempt = 0; attempt < attempts; attempt += 1) {
      if (attempt > 0) await wait(3000);
      try {
        const receipt = await api.get(
          `/badges/series/${encodeURIComponent(seriesId)}/claims/${encodeURIComponent(refId)}`,
          { silent: true },
        );
        if (receipt && !isPending(receipt)) return finishClaim(seriesId);
      } catch {
        // A missing receipt is expected while an ambiguous submission is settling.
      }
    }
    setClaimState(seriesId, {
      status: 'pending',
      message: '链上结果仍在确认。请稍后回到本页继续查询；系统会沿用原领取标识。',
      refId,
    });
    return null;
  }

  async function claim(seriesId) {
    const existingRef = pendingRef(seriesId);
    if (existingRef) return reconcileClaim(seriesId, existingRef);
    const requestStorageKey = `${pendingStorageKey(seriesId)}:request`;
    const refId = localStorage.getItem(requestStorageKey) || newClaimRef();
    localStorage.setItem(requestStorageKey, refId);
    setClaimState(seriesId, { status: 'submitting', message: '正在提交免费领取…', refId });
    try {
      const result = await api.post(
        `/badges/series/${encodeURIComponent(seriesId)}/claim`,
        {},
        { headers: { 'Idempotency-Key': refId }, silent: true },
      );
      if (isPending(result)) {
        if (!result?.refId) throw new Error('领取响应缺少回执编号，请使用原请求重试');
        return reconcileClaim(seriesId, result.refId);
      }
      return finishClaim(seriesId);
    } catch (cause) {
      if (cause?.code === 'BADGE_ALREADY_CLAIMED') return finishClaim(seriesId);
      if (TERMINAL_CODES.has(cause?.code)) {
        clearPendingRef(seriesId);
        setClaimState(seriesId, { status: 'error', message: badgeErrorMessage(cause), refId: '' });
        throw cause;
      }
      // Without a response we only know the request key, not its server-derived
      // ledger reference. A subsequent click reuses this exact request key.
      setClaimState(seriesId, { status: 'error', message: '领取响应未确认，请重试；将沿用原请求编号。', refId: '' });
      throw cause;
    }
  }

  async function resumePending(seriesId) {
    const refId = pendingRef(seriesId);
    if (!refId || currentView.value?.award) return null;
    return reconcileClaim(seriesId, refId);
  }

  return {
    series,
    awards,
    currentSeries,
    currentView,
    loading,
    detailLoading,
    error,
    claimStates,
    loadSeriesList,
    loadAwards,
    loadSeries,
    loadSeriesLinks,
    loadMyAward,
    loadDetail,
    loadPublicInstance,
    claim,
    resumePending,
  };
});
