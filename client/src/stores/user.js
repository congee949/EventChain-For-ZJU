import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useUserStore = defineStore('user', () => {
  const balance = ref(0);
  const profile = ref(null);
  const leaderboard = ref([]);
  const loading = ref(false);

  async function fetchProfile() {
    loading.value = true;
    try {
      profile.value = await api.get('/users/profile');
      balance.value = profile.value.balance ?? 0;
      return profile.value;
    } finally {
      loading.value = false;
    }
  }

  async function fetchLeaderboard() {
    leaderboard.value = await api.get('/users/leaderboard');
    return leaderboard.value;
  }

  // Called after any action that changes balance (bet, settlement, etc.)
  async function refreshBalance() {
    const p = await api.get('/users/profile');
    balance.value = p.balance ?? 0;
    profile.value = p;
  }

  return {
    balance,
    profile,
    leaderboard,
    loading,
    fetchProfile,
    fetchLeaderboard,
    refreshBalance,
  };
});
