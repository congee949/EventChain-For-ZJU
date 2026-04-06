import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useEventStore = defineStore('events', () => {
  const events = ref([]);
  const currentEvent = ref(null);
  const loading = ref(false);

  async function fetchEvents(filters = {}) {
    loading.value = true;
    try {
      const params = new URLSearchParams();
      if (filters.status) params.set('status', filters.status);
      if (filters.type) params.set('type', filters.type);
      const qs = params.toString();
      events.value = await api.get(`/events${qs ? '?' + qs : ''}`);
    } finally {
      loading.value = false;
    }
  }

  async function fetchEvent(eventId) {
    loading.value = true;
    try {
      currentEvent.value = await api.get(`/events/${eventId}`);
      return currentEvent.value;
    } finally {
      loading.value = false;
    }
  }

  async function createEvent(payload) {
    const result = await api.post('/events', payload);
    await fetchEvents();
    return result;
  }

  async function updateStatus(eventId, status) {
    const result = await api.put(`/events/${eventId}/status`, { status });
    await fetchEvents();
    return result;
  }

  async function recordResult(eventId, outcome) {
    const result = await api.put(`/events/${eventId}/result`, { outcome });
    await fetchEvents();
    return result;
  }

  return {
    events,
    currentEvent,
    loading,
    fetchEvents,
    fetchEvent,
    createEvent,
    updateStatus,
    recordResult,
  };
});
