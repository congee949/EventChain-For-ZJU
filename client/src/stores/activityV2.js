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
    const bytes = new Uint8Array(32);
    crypto.getRandomValues(bytes);
    const secret = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
    localStorage.setItem(`ec_ticket_secret:${activityId}`, secret);
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
