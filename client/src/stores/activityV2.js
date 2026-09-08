import { ref } from 'vue';
import { defineStore } from 'pinia';
import api, { idempotencyHeaders } from '../api/index.js';

export const useActivityV2Store = defineStore('activityV2', () => {
  const activities = ref([]);
  const applications = ref({});
  const tickets = ref({});
  const loading = ref(false);
  const error = ref('');
  async function refresh() {
    loading.value = true; error.value = '';
    try { activities.value = await api.get('/activities'); return activities.value; }
    catch (cause) { error.value = cause?.message || '活动数据加载失败'; throw cause; }
    finally { loading.value = false; }
  }
  async function apply(activityId) {
    const result = await api.post(`/activities/${activityId}/applications`);
    applications.value[activityId] = result;
    await refresh();
    return result;
  }
  async function loadMine(activityId) {
    try { applications.value[activityId] = await api.get(`/activities/${activityId}/application`, { silent: true }); } catch { applications.value[activityId] = null; }
    try { tickets.value[activityId] = await api.get(`/activities/${activityId}/ticket`, { silent: true }); } catch { tickets.value[activityId] = null; }
  }
  async function hydrateMine() { await Promise.all(activities.value.map((item) => loadMine(item.id))); }
  async function claim(activityId) {
    const storageKey = `ec_ticket_secret:${activityId}`;
    let secret = localStorage.getItem(storageKey);
    if (!secret) {
      const bytes = new Uint8Array(32);
      crypto.getRandomValues(bytes);
      secret = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
      // Keep the same secret across retries. The transaction may have committed
      // even when the browser did not receive its response.
      localStorage.setItem(storageKey, secret);
    }
    const result = await api.post(`/activities/${activityId}/ticket`, { secret }, { headers: idempotencyHeaders('ticket') });
    tickets.value[activityId] = result.ticket;
    return result;
  }
  function qrProof(activityId) {
    const ticket = tickets.value[activityId];
    const secret = localStorage.getItem(`ec_ticket_secret:${activityId}`);
    if (!ticket || !secret) return null;
    return JSON.stringify({ ticketId: ticket.ticketId, secret, timeSlice: Math.floor(Date.now() / 30000) });
  }
  return { activities, applications, tickets, loading, error, refresh, apply, loadMine, hydrateMine, claim, qrProof };
});
