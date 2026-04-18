<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
} from 'echarts/components';
import VChart from 'vue-echarts';
import { ElMessage, ElInputNumber, ElSlider, ElRadioGroup, ElRadioButton } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent]);

const route = useRoute();
const eventStore = useEventStore();
const predStore = usePredictionStore();
const userStore = useUserStore();
const auth = useAuthStore();

const eventId = computed(() => route.params.id);
const event = computed(() => eventStore.currentEvent);
const odds = computed(() => predStore.odds);

// Bet form — selectedOption holds the actual option name (e.g. "A队赢"),
// not just "A"/"B", so it can be passed straight to the chaincode.
const selectedOption = ref('');
const betAmount = ref(50);
const submitting = ref(false);

onMounted(async () => {
  await eventStore.fetchEvent(eventId.value);
  await predStore.fetchOdds(eventId.value);
  await predStore.fetchPool(eventId.value);
  // Always reset to optionA when an event loads — don't keep stale selection
  // from a previous event. Mirrors the watch(eventId) handler below.
  if (event.value?.predictionOptions?.[0]) {
    selectedOption.value = event.value.predictionOptions[0];
  }
});

watch(eventId, async (newId) => {
  if (newId) {
    await eventStore.fetchEvent(newId);
    await predStore.fetchOdds(newId);
    await predStore.fetchPool(newId);
    if (event.value?.predictionOptions?.[0]) {
      selectedOption.value = event.value.predictionOptions[0];
    }
  }
});

// Payout preview based on current AMM state
const payoutPreview = computed(() => {
  if (!predStore.pool || !betAmount.value) return 0;
  const pool = predStore.pool;
  const d = betAmount.value;

  // Compare against the actual optionA name (selectedOption is now the option name itself)
  const optionAName = event.value?.predictionOptions?.[0];
  if (selectedOption.value === optionAName) {
    const newPoolB = pool.poolB + d;
    const newPoolA = Math.floor(pool.k / newPoolB);
    const shares = Math.floor(pool.poolA - newPoolA);
    return shares;
  } else {
    const newPoolA = pool.poolA + d;
    const newPoolB = Math.floor(pool.k / newPoolA);
    const shares = Math.floor(pool.poolB - newPoolB);
    return shares;
  }
});

async function handleBet() {
  if (!auth.isLoggedIn) {
    ElMessage.warning('请先登录');
    return;
  }
  submitting.value = true;
  try {
    const result = await predStore.placeBet(eventId.value, selectedOption.value, betAmount.value);
    ElMessage.success(`下注成功！获得 ${result.shares.toFixed(2)} 份额`);
    await userStore.refreshBalance();
  } catch {
    // Error handled by interceptor
  } finally {
    submitting.value = false;
  }
}

// ECharts config for odds history
const chartOption = computed(() => {
  // Use event.oddsHistory if available, otherwise generate sample points
  const history = event.value?.oddsHistory || [];
  const times = history.map((h) => h.time || '');
  const probAData = history.map((h) => ((h.probA ?? 0.5) * 100).toFixed(1));
  const probBData = history.map((h) => ((h.probB ?? 0.5) * 100).toFixed(1));

  const optionA = event.value?.predictionOptions?.[0] || '选项 A';
  const optionB = event.value?.predictionOptions?.[1] || '选项 B';

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b', fontSize: 13 },
      formatter(params) {
        let html = `<div style="font-weight:600;margin-bottom:4px">${params[0].axisValue}</div>`;
        for (const p of params) {
          html += `<div>${p.marker} ${p.seriesName}: <b>${p.value}%</b></div>`;
        }
        return html;
      },
    },
    legend: {
      data: [optionA, optionB],
      bottom: 0,
      textStyle: { fontSize: 13 },
    },
    grid: {
      top: 20,
      right: 20,
      bottom: 40,
      left: 50,
      containLabel: false,
    },
    xAxis: {
      type: 'category',
      data: times,
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%', color: '#94a3b8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
    },
    series: [
      {
        name: optionA,
        type: 'line',
        data: probAData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#6366f1' },
        itemStyle: { color: '#6366f1' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(99,102,241,0.25)' },
              { offset: 1, color: 'rgba(99,102,241,0.02)' },
            ],
          },
        },
      },
      {
        name: optionB,
        type: 'line',
        data: probBData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#ec4899' },
        itemStyle: { color: '#ec4899' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(236,72,153,0.25)' },
              { offset: 1, color: 'rgba(236,72,153,0.02)' },
            ],
          },
        },
      },
    ],
  };
});

const statusLabel = {
  CREATED: '已创建',
  PREDICTION_OPEN: '预测中',
  TICKET_OPEN: '购票中',
  ONGOING: '进行中',
  SETTLED: '已结算',
};
</script>

<template>
  <div v-if="event" class="event-detail">
    <!-- Header -->
    <GlassCard class="event-header" padding="32px">
      <div class="header-top">
        <span class="event-type">{{ event.type }}</span>
        <span class="status-badge glass-subtle">{{ statusLabel[event.status] || event.status }}</span>
      </div>
      <h1 class="event-title">{{ event.title }}</h1>
      <div class="teams-display">
        <span class="team team-a">{{ event.teams?.[0] }}</span>
        <span class="vs-badge">VS</span>
        <span class="team team-b">{{ event.teams?.[1] }}</span>
      </div>
      <ProbabilityBar
        v-if="odds"
        :prob-a="odds.probA"
        :label-a="event.predictionOptions?.[0] || 'A'"
        :label-b="event.predictionOptions?.[1] || 'B'"
        height="36px"
        style="margin-top: 20px"
      />
    </GlassCard>

    <div class="detail-grid">
      <!-- Odds chart -->
      <GlassCard class="chart-card" padding="24px">
        <h2 class="card-title">概率走势</h2>
        <VChart
          v-if="event.oddsHistory && event.oddsHistory.length > 0"
          :option="chartOption"
          style="height: 320px; width: 100%"
          autoresize
        />
        <p v-else class="empty-chart-text">暂无走势数据，首次下注后开始记录</p>
      </GlassCard>

      <!-- Bet panel -->
      <GlassCard class="bet-panel" padding="24px">
        <h2 class="card-title">下注预测</h2>

        <div class="bet-options">
          <ElRadioGroup v-model="selectedOption" size="large">
            <ElRadioButton :value="event.predictionOptions?.[0] || 'A'">
              {{ event.predictionOptions?.[0] || 'A' }}
            </ElRadioButton>
            <ElRadioButton :value="event.predictionOptions?.[1] || 'B'">
              {{ event.predictionOptions?.[1] || 'B' }}
            </ElRadioButton>
          </ElRadioGroup>
        </div>

        <div class="bet-amount">
          <label class="form-label">投注金额（浙币）</label>
          <ElSlider v-model="betAmount" :min="1" :max="500" :step="10" show-input />
        </div>

        <div class="payout-preview glass-subtle">
          <div class="preview-row">
            <span class="preview-label">投注</span>
            <span class="preview-value">{{ betAmount }} 浙币</span>
          </div>
          <div class="preview-row">
            <span class="preview-label">预计份额</span>
            <span class="preview-value highlight">{{ payoutPreview }}</span>
          </div>
        </div>

        <button
          class="bet-btn"
          :disabled="submitting || event.status !== 'PREDICTION_OPEN'"
          @click="handleBet"
        >
          {{ submitting ? '提交中...' : event.status === 'PREDICTION_OPEN' ? '确认下注' : '预测未开放' }}
        </button>

        <!-- Pool info -->
        <div v-if="predStore.pool" class="pool-info">
          <span>总投注量: {{ predStore.pool.totalVolume }} 浙币</span>
        </div>
      </GlassCard>
    </div>
  </div>

  <div v-else class="loading-state">
    <p>加载中...</p>
  </div>
</template>

<style scoped>
.event-detail {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.event-header {
  text-align: center;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.event-type {
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-secondary);
  letter-spacing: 0.05em;
}

.status-badge {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
}

.event-title {
  font-size: 28px;
  font-weight: 800;
}

.teams-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 8px;
  font-size: 20px;
}

.team {
  font-weight: 700;
}

.team-a { color: var(--color-primary); }
.team-b { color: #ec4899; }

.vs-badge {
  font-size: 14px;
  font-weight: 800;
  color: var(--color-danger);
  padding: 4px 10px;
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.08);
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* Bet panel */
.bet-options {
  margin-bottom: 20px;
}

.bet-amount {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.payout-preview {
  padding: 16px;
  margin-bottom: 16px;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
}

.preview-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.preview-value {
  font-size: 14px;
  font-weight: 600;
}

.preview-value.highlight {
  color: var(--color-primary);
  font-size: 16px;
}

.bet-btn {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--color-primary), #8b5cf6);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-base);
}

.bet-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.3);
}

.bet-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pool-info {
  text-align: center;
  margin-top: 12px;
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.empty-chart-text {
  text-align: center;
  padding: 80px 0;
  color: var(--color-text-tertiary);
  font-size: 14px;
}

.loading-state {
  text-align: center;
  padding: 80px 0;
  color: var(--color-text-tertiary);
}
</style>
