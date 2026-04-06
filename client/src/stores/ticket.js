import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useTicketStore = defineStore('ticket', () => {
  const myTickets = ref([]);
  const loading = ref(false);

  async function applyTicket(eventID) {
    loading.value = true;
    try {
      const result = await api.post('/tickets/apply', { eventID });
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function runLottery(eventID) {
    const result = await api.post(`/tickets/lottery/${eventID}`);
    return result;
  }

  async function fetchMyTickets() {
    myTickets.value = await api.get('/tickets/mine');
    return myTickets.value;
  }

  async function claimTicket(ticketID) {
    const result = await api.post(`/tickets/claim/${ticketID}`);
    // Refresh ticket list to show new claim hash
    await fetchMyTickets();
    return result;
  }

  async function verifyTicket(ticketID, hash) {
    const result = await api.get(`/tickets/verify/${ticketID}`, { params: { hash } });
    return result;
  }

  async function refundTicket(ticketID) {
    const result = await api.post(`/tickets/refund/${ticketID}`);
    await fetchMyTickets();
    return result;
  }

  return {
    myTickets,
    loading,
    applyTicket,
    runLottery,
    fetchMyTickets,
    claimTicket,
    verifyTicket,
    refundTicket,
  };
});
