<script setup>
import { onMounted, computed } from 'vue';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { RadarChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import VChart from 'vue-echarts';
import { useUserStore } from '../stores/user.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

use([CanvasRenderer, RadarChart, TitleComponent, TooltipComponent, LegendComponent]);

const userStore = useUserStore();
const predStore = usePredictionStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    userStore.fetchProfile(),
    predStore.fetchMyBets(),
    predStore.fetchMyScore(),
    ticketStore.fetchMyTickets(),
  ]);
});

const profile = computed(() => userStore.profile);

// Radar chart — accuracy by event type
const radarOption = computed(() => {
  // Group bets by event type and compute per-type accuracy
  const typeMap = {};
  const types = ['basketball', 'football', 'esports', 'badminton', 'track'];
  const typeLabels = {
    basketball: '篮球',
    football: '足球',
    esports: '电竞',
    badminton: '羽毛球',
    track: '田径',
  };

  for (const t of types) {
    typeMap[t] = { total: 0, correct: 0 };
  }

  for (const bet of predStore.myBets) {
    const t = bet.eventType || 'basketball';
    if (typeMap[t]) {
      typeMap[t].total++;
      if (bet.won) typeMap[t].correct++;
    }
  }

  const indicators = types.map((t) => ({
    name: typeLabels[t] || t,
    max: 100,
  }));

  const values = types.map((t) => {
    const { total, correct } = typeMap[t];
    return total > 0 ? Math.round((correct / total) * 100) : 0;
  });

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b' },
    },
    radar: {
      indicator: indicators,
      shape: 'polygon',
      axisName: {
        color: '#64748b',
        fontSize: 13,
      },
      splitArea: {
        areaStyle: {
          color: [
            'rgba(99,102,241,0.03)',
            'rgba(99,102,241,0.06)',
            'rgba(99,102,241,0.09)',
            'rgba(99,102,241,0.12)',
            'rgba(99,102,241,0.15)',
          ],
        },
      },
      splitLine: {
        lineStyle: { color: 'rgba(0,0,0,0.06)' },
      },
      axisLine: {
        lineStyle: { color: 'rgba(0,0,0,0.08)' },
      },
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            value: values,
            name: '准确率',
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: '#6366f1', width: 2 },
            itemStyle: { color: '#6366f1' },
            areaStyle: { color: 'rgba(99,102,241,0.2)' },
          },
        ],
      },
    ],
  };
});

// Achievement badges
const achievements = computed(() => {
  const list = [];
  const p = profile.value;
  if (!p) return list;

  if (p.totalBets >= 1) list.push({ label: '初出茅庐', desc: '完成第一次预测', icon: '\u{1F3AF}' });
  if (p.totalBets >= 50) list.push({ label: '预测达人', desc: '完成 50 次预测', icon: '\u{1F525}' });
  if (p.accuracyRate >= 0.8 && p.totalBets >= 10) list.push({ label: '神算子', desc: '准确率超过 80%', icon: '\u{1F52E}' });
  if (p.balance >= 5000) list.push({ label: '富甲一方', desc: '余额超过 5000', icon: '\u{1F4B0}' });

  return list;
});
</script>

<template>
  <div class="profile-page">
    <h1 class="page-title">个人中心</h1>

    <div class="profile-grid" v-if="profile">
      <!-- Balance + stats -->
      <GlassCard class="balance-card" padding="32px">
        <div class="balance-header">
          <h2 class="balance-title">{{ profile.name }}</h2>
          <span class="user-role glass-subtle">{{ profile.role }}</span>
        </div>
        <div class="stats-row">
          <div class="stat-item">
            <span class="stat-value primary">{{ profile.balance }}</span>
            <span class="stat-label">浙币余额</span>
          </div>
          <div class="stat-item">
            <span class="stat-value success">{{ (profile.accuracyRate * 100).toFixed(1) }}%</span>
            <span class="stat-label">预测准确率</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.totalBets }}</span>
            <span class="stat-label">总预测次数</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.correctBets }}</span>
            <span class="stat-label">正确次数</span>
          </div>
        </div>
      </GlassCard>

      <!-- Radar chart -->
      <GlassCard class="radar-card" padding="24px">
        <h2 class="card-title">分项准确率</h2>
        <VChart
          :option="radarOption"
          style="height: 300px; width: 100%"
          autoresize
        />
      </GlassCard>

      <!-- Bet history -->
      <GlassCard class="history-card" padding="24px">
        <h2 class="card-title">预测记录</h2>
        <div class="bet-list">
          <div
            v-for="bet in predStore.myBets"
            :key="bet.betID || bet.eventID + bet.timestamp"
            class="bet-item glass-subtle"
          >
            <div class="bet-info">
              <span class="bet-event">{{ bet.eventID }}</span>
              <span class="bet-option">{{ bet.option }}</span>
            </div>
            <div class="bet-meta">
              <span class="bet-amount">{{ bet.amount }} 浙币</span>
              <span class="bet-shares">{{ bet.shares?.toFixed(2) }} 份额</span>
              <span
                class="bet-result"
                :class="bet.won ? 'won' : bet.won === false ? 'lost' : 'pending'"
              >
                {{ bet.won ? '胜' : bet.won === false ? '负' : '待定' }}
              </span>
            </div>
          </div>
        </div>
        <p v-if="!predStore.myBets.length" class="empty-text">暂无预测记录</p>
      </GlassCard>

      <!-- Achievements -->
      <GlassCard class="achievements-card" padding="24px">
        <h2 class="card-title">成就徽章</h2>
        <div class="badge-grid">
          <div
            v-for="badge in achievements"
            :key="badge.label"
            class="badge-item glass-subtle"
          >
            <span class="badge-icon">{{ badge.icon }}</span>
            <span class="badge-label">{{ badge.label }}</span>
            <span class="badge-desc">{{ badge.desc }}</span>
          </div>
        </div>
        <p v-if="!achievements.length" class="empty-text">继续努力，解锁成就吧</p>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.profile-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

/* Balance card spans full width */
.balance-card {
  grid-column: 1 / -1;
}

.balance-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.balance-title {
  font-size: 24px;
  font-weight: 800;
}

.user-role {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.stats-row {
  display: flex;
  justify-content: space-around;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
}

.stat-value.primary { color: var(--color-primary); }
.stat-value.success { color: var(--color-success); }

.stat-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* History card spans full width */
.history-card {
  grid-column: 1 / -1;
}

.bet-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.bet-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.bet-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bet-event {
  font-weight: 600;
}

.bet-option {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.bet-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
}

.bet-amount { color: var(--color-text-secondary); }
.bet-shares { color: var(--color-text-tertiary); }

.bet-result {
  font-weight: 700;
  padding: 2px 10px;
  border-radius: 6px;
}

.bet-result.won { color: var(--color-success); background: rgba(34, 197, 94, 0.1); }
.bet-result.lost { color: var(--color-danger); background: rgba(239, 68, 68, 0.1); }
.bet-result.pending { color: var(--color-warning); background: rgba(245, 158, 11, 0.1); }

/* Achievements */
.achievements-card {
  grid-column: 1 / -1;
}

.badge-grid {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.badge-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 20px;
  min-width: 120px;
  text-align: center;
  gap: 4px;
}

.badge-icon {
  font-size: 32px;
}

.badge-label {
  font-size: 14px;
  font-weight: 700;
}

.badge-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  text-align: center;
  padding: 20px 0;
}
</style>
