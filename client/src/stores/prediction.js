import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const usePredictionStore = defineStore('prediction', () => {
  const odds = ref(null);      // { probA, probB } for current event
  const pool = ref(null);      // { poolA, poolB, k, totalVolume }
  const myBets = ref([]);
  const myScore = ref(null);   // { totalBets, correctBets, accuracyRate }
  const loading = ref(false);

  async function fetchOdds(eventID) {
    odds.value = await api.get(`/predictions/odds/${eventID}`);
    return odds.value;
  }

  async function fetchPool(eventID) {
    pool.value = await api.get(`/predictions/pool/${eventID}`);
    return pool.value;
  }

  async function placeBet(eventID, option, amount) {
    loading.value = true;
    try {
      const result = await api.post('/predictions/bet', { eventID, option, amount });
      // result: { shares, newOddsA, newOddsB }
      odds.value = { probA: result.newOddsA, probB: result.newOddsB };
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function fetchMyBets() {
    myBets.value = await api.get('/predictions/mine');
    return myBets.value;
  }

  async function fetchMyScore() {
    myScore.value = await api.get('/predictions/score');
    return myScore.value;
  }

  return {
    odds,
    pool,
    myBets,
    myScore,
    loading,
    fetchOdds,
    fetchPool,
    placeBet,
    fetchMyBets,
    fetchMyScore,
  };
});
