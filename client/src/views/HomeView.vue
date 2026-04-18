<script setup>
import { onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useEventStore } from '../stores/events.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';
import SparkLine from '../components/SparkLine.vue';

const router = useRouter();
const eventStore = useEventStore();
const userStore = useUserStore();
const auth = useAuthStore();

onMounted(async () => {
  await eventStore.fetchEvents();
  if (auth.isLoggedIn) {
    await Promise.all([
      userStore.fetchProfile(),
      userStore.fetchLeaderboard(),
    ]);
  }
});

const hotEvents = computed(() =>
  eventStore.events.filter((e) =>
    ['PREDICTION_OPEN', 'ONGOING'].includes(e.status)
  )
);

const typeEmoji = {
  basketball: '\u{1F3C0}',
  football: '\u26BD',
  esports: '\u{1F3AE}',
  badminton: '\u{1F3F8}',
  track: '\u{1F3C3}',
};

function goToEvent(id) {
  router.push(`/event/${id}`);
}
</script>

<template>
  <div class="home">
    <!-- Hero -->
    <section class="hero">
      <h1 class="hero-title">EventChain</h1>
      <p class="hero-subtitle">校园赛事预测市场 &mdash; 预测即力量</p>
    </section>

    <div class="home-grid">
      <!-- Event cards -->
      <section class="events-section">
        <h2 class="section-title">热门赛事</h2>
        <div class="events-grid">
          <GlassCard
            v-for="event in hotEvents"
            :key="event.id"
            class="event-card"
            @click="goToEvent(event.id)"
          >
            <div class="event-card-header">
              <span class="event-type-emoji">{{ typeEmoji[event.type] || '\u{1F3C6}' }}</span>
              <span class="event-status glass-subtle">{{ event.status }}</span>
            </div>
            <h3 class="event-title">{{ event.title }}</h3>
            <div class="event-teams">
              {{ event.teams?.[0] }} <span class="vs">VS</span> {{ event.teams?.[1] }}
            </div>
            <ProbabilityBar
              v-if="event.odds"
              :prob-a="event.odds.probA"
              :label-a="event.predictionOptions?.[0] || 'A'"
              :label-b="event.predictionOptions?.[1] || 'B'"
              height="28px"
              style="margin-top: 12px"
            />
            <div class="event-card-footer">
              <SparkLine
                v-if="event.oddsHistory"
                :data="event.oddsHistory"
                :width="80"
                :height="24"
              />
              <span class="event-volume">
                {{ event.pool?.totalVolume || 0 }} 浙币参与
              </span>
            </div>
          </GlassCard>
        </div>
      </section>

      <!-- Sidebar -->
      <aside class="sidebar">
        <!-- Stats -->
        <GlassCard v-if="auth.isLoggedIn && userStore.profile" class="sidebar-card">
          <h3 class="sidebar-title">我的数据</h3>
          <div class="stats-row">
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.balance }}</span>
              <span class="stat-label">浙币</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ (userStore.profile.accuracyRate * 100).toFixed(1) }}%</span>
              <span class="stat-label">准确率</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.placedBets ?? 0 }}</span>
              <span class="stat-label">总下注</span>
            </div>
          </div>
        </GlassCard>

        <!-- Leaderboard -->
        <GlassCard class="sidebar-card">
          <h3 class="sidebar-title">预测之星</h3>
          <ol class="leaderboard-list">
            <li
              v-for="(entry, idx) in userStore.leaderboard.slice(0, 10)"
              :key="entry.userId"
              class="leaderboard-item"
            >
              <span class="lb-rank">{{ idx + 1 }}</span>
              <span class="lb-name">{{ entry.userId }}</span>
              <span class="lb-score">{{ (entry.accuracyRate * 100).toFixed(1) }}%</span>
            </li>
          </ol>
        </GlassCard>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.hero {
  text-align: center;
  padding: 48px 0 32px;
}

.hero-title {
  font-size: 48px;
  font-weight: 800;
  background: linear-gradient(135deg, var(--color-primary), #ec4899);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  font-size: 18px;
  color: var(--color-text-secondary);
  margin-top: 8px;
}

.home-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.events-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.event-card {
  cursor: pointer;
}

.event-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.event-type-emoji {
  font-size: 24px;
}

.event-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.event-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 4px;
}

.event-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.vs {
  color: var(--color-danger);
  font-weight: 700;
  margin: 0 4px;
}

.event-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

.event-volume {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

/* Sidebar */
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sidebar-card {
  padding: 20px;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 12px;
}

.stats-row {
  display: flex;
  justify-content: space-between;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.leaderboard-list {
  list-style: none;
  padding: 0;
}

.leaderboard-item {
  display: flex;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.leaderboard-item:last-child {
  border-bottom: none;
}

.lb-rank {
  width: 24px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-tertiary);
}

.lb-name {
  flex: 1;
  font-size: 14px;
}

.lb-score {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
}
</style>
