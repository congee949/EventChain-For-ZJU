import { computed, ref } from 'vue';
import { defineStore } from 'pinia';
import api, { idempotencyHeaders } from '../api/index.js';

export const SCALE = 1_000_000;
export const formatAmount = (value) => (Number(value || 0) / SCALE).toLocaleString('zh-CN', { maximumFractionDigits: 6 });

export const useFinanceStore = defineStore('financeV2', () => {
  const wallet = ref(null);
  const categories = ref([]);
  const markets = ref([]);
  const offers = ref([]);
  const currentMarket = ref(null);
  const loading = ref(false);
  const error = ref('');
  const aBalance = computed(() => formatAmount(wallet.value?.aAvailable));

  async function refresh() {
    loading.value = true;
    error.value = '';
    try {
      [wallet.value, categories.value, markets.value, offers.value] = await Promise.all([
        api.get('/finance/wallet'), api.get('/finance/categories'), api.get('/finance/markets'), api.get('/finance/offers'),
      ]);
    } catch (cause) {
      error.value = cause?.message || '财务数据加载失败';
      throw cause;
    } finally { loading.value = false; }
  }
  async function refreshWallet() { wallet.value = await api.get('/finance/wallet'); return wallet.value; }
  async function fetchMarket(marketId) { currentMarket.value = await api.get(`/finance/markets/${marketId}`); return currentMarket.value; }
  async function convert(categoryId, amount) {
    await api.post('/finance/wallet/convert', { categoryId, amount }, { headers: idempotencyHeaders('convert') });
    await refresh();
  }
  async function redeem(categoryId, amount) {
    await api.post('/finance/wallet/redeem', { categoryId, amount }, { headers: idempotencyHeaders('redeem') });
    await refresh();
  }
  async function transfer(categoryId, toAccountId, amount) {
    await api.post('/finance/wallet/transfer', { categoryId, toAccountId, amount }, { headers: idempotencyHeaders('transfer') });
    await refresh();
  }
  async function placePosition(marketId, outcomeId, amount) {
    const result = await api.post(`/finance/markets/${marketId}/positions`, { outcomeId, amount }, { headers: idempotencyHeaders('position') });
    await refresh();
    return result;
  }
  async function claim(marketId, epoch) {
    return api.post(`/finance/markets/${marketId}/settlements/${epoch}/claim`);
  }
  async function mature(marketId, epoch) {
    const result = await api.post(`/finance/markets/${marketId}/settlements/${epoch}/mature`);
    await refresh();
    return result;
  }
  async function order(offerId, scheduledAt) {
    const result = await api.post('/finance/orders', { offerId, scheduledAt }, { headers: idempotencyHeaders('service') });
    await refresh();
    return result;
  }
  return { wallet, categories, markets, offers, currentMarket, loading, error, aBalance, refresh, refreshWallet, fetchMarket, convert, redeem, transfer, placePosition, claim, mature, order };
});
