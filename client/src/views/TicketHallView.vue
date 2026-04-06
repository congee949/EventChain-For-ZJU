<script setup>
import { onMounted, computed } from 'vue';
import { ElMessage } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';
import CountdownTimer from '../components/CountdownTimer.vue';
import QRCode from '../components/QRCode.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    eventStore.fetchEvents({ status: 'TICKET_OPEN' }),
    ticketStore.fetchMyTickets(),
  ]);
});

const ticketEvents = computed(() =>
  eventStore.events.filter((e) => e.status === 'TICKET_OPEN')
);

const wonTickets = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'WON' || t.status === 'CLAIMED')
);

const pendingApplications = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'PENDING')
);

async function handleApply(eventID) {
  try {
    await ticketStore.applyTicket(eventID);
    ElMessage.success('申请已提交');
    await ticketStore.fetchMyTickets();
  } catch {
    // Error handled by interceptor
  }
}

async function handleClaim(ticketID) {
  try {
    await ticketStore.claimTicket(ticketID);
    ElMessage.success('票据已领取');
  } catch {
    // Error handled by interceptor
  }
}

async function handleRefund(ticketID) {
  try {
    await ticketStore.refundTicket(ticketID);
    ElMessage.success('退票成功');
  } catch {
    // Error handled by interceptor
  }
}
</script>

<template>
  <div class="ticket-hall">
    <h1 class="page-title">票务大厅</h1>

    <!-- Available ticket events -->
    <section class="section">
      <h2 class="section-title">正在售票的赛事</h2>
      <div class="ticket-events-grid">
        <GlassCard
          v-for="event in ticketEvents"
          :key="event.id"
          class="ticket-event-card"
        >
          <div class="te-header">
            <h3 class="te-title">{{ event.title }}</h3>
            <span class="te-quota glass-subtle">余票 {{ event.ticketTotal }}</span>
          </div>
          <div class="te-teams">{{ event.teams?.[0] }} VS {{ event.teams?.[1] }}</div>
          <div class="te-countdown">
            <span class="te-countdown-label">抽签倒计时</span>
            <CountdownTimer :target-time="event.lotteryTime || '2026-04-10T20:00:00'" />
          </div>
          <button class="apply-btn" @click="handleApply(event.id)">
            申请购票
          </button>
        </GlassCard>
      </div>
      <p v-if="!ticketEvents.length" class="empty-text">暂无售票中的赛事</p>
    </section>

    <!-- My applications -->
    <section class="section">
      <h2 class="section-title">我的申请</h2>
      <div class="applications-list">
        <GlassCard
          v-for="app in pendingApplications"
          :key="app.eventID + app.userID"
          variant="subtle"
          padding="16px"
          class="app-item"
        >
          <span class="app-event">{{ app.eventID }}</span>
          <span class="app-status pending">等待抽签</span>
        </GlassCard>
      </div>
      <p v-if="!pendingApplications.length" class="empty-text">暂无待处理的申请</p>
    </section>

    <!-- Won tickets -->
    <section class="section">
      <h2 class="section-title">我的票据</h2>
      <div class="tickets-grid">
        <GlassCard
          v-for="ticket in wonTickets"
          :key="ticket.ticketID"
          class="ticket-card"
        >
          <h3 class="ticket-event-name">{{ ticket.eventID }}</h3>
          <div class="ticket-status">
            <span v-if="ticket.status === 'WON'" class="status-won">中签 - 待领取</span>
            <span v-else-if="ticket.status === 'CLAIMED'" class="status-claimed">已领取</span>
          </div>

          <!-- QR code for claimed tickets -->
          <div v-if="ticket.status === 'CLAIMED' && ticket.claimHash" class="qr-section">
            <QRCode :value="ticket.claimHash" :size="180" />
            <p class="qr-hint">入场时出示此二维码</p>
          </div>

          <div class="ticket-actions">
            <button
              v-if="ticket.status === 'WON'"
              class="claim-btn"
              @click="handleClaim(ticket.ticketID)"
            >
              领取票据
            </button>
            <button
              class="refund-btn"
              @click="handleRefund(ticket.ticketID)"
            >
              退票
            </button>
          </div>
        </GlassCard>
      </div>
      <p v-if="!wonTickets.length" class="empty-text">暂无票据</p>
    </section>
  </div>
</template>

<style scoped>
.ticket-hall {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.section {
  margin-bottom: 40px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.ticket-events-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.te-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.te-title {
  font-size: 16px;
  font-weight: 700;
}

.te-quota {
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-success);
}

.te-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.te-countdown {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.te-countdown-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.apply-btn {
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.apply-btn:hover {
  background: var(--color-primary-dark);
}

.applications-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.app-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.app-event {
  font-weight: 600;
}

.app-status.pending {
  color: var(--color-warning);
  font-weight: 600;
  font-size: 13px;
}

.tickets-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.ticket-card {
  text-align: center;
}

.ticket-event-name {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
}

.status-won {
  color: var(--color-warning);
  font-weight: 600;
}

.status-claimed {
  color: var(--color-success);
  font-weight: 600;
}

.qr-section {
  margin: 16px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qr-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.ticket-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.claim-btn {
  flex: 1;
  padding: 8px;
  border: none;
  border-radius: 8px;
  background: var(--color-success);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.refund-btn {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  background: transparent;
  color: var(--color-danger);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-base);
}

.refund-btn:hover {
  background: rgba(239, 68, 68, 0.08);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  padding: 24px 0;
  text-align: center;
}
</style>
