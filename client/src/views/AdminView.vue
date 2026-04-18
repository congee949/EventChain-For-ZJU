<script setup>
import { ref, onMounted, computed } from 'vue';
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElInputNumber,
  ElButton,
  ElTable,
  ElTableColumn,
  ElTag,
  ElMessage,
  ElMessageBox,
} from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await eventStore.fetchEvents();
});

// --- Create Event Form ---
const createForm = ref({
  eventID: '',
  title: '',
  type: 'basketball',
  teamA: '',
  teamB: '',
  ticketTotal: 100,
  optionA: '',
  optionB: '',
});
const creating = ref(false);

const eventTypes = [
  { value: 'basketball', label: '篮球' },
  { value: 'football', label: '足球' },
  { value: 'esports', label: '电竞' },
  { value: 'badminton', label: '羽毛球' },
  { value: 'track', label: '田径' },
];

async function handleCreate() {
  const f = createForm.value;
  if (!f.eventID || !f.title || !f.teamA || !f.teamB || !f.optionA || !f.optionB) {
    ElMessage.warning('请填写所有必填字段');
    return;
  }

  creating.value = true;
  try {
    await eventStore.createEvent({
      eventID: f.eventID,
      title: f.title,
      type: f.type,
      teams: [f.teamA, f.teamB],
      ticketTotal: f.ticketTotal,
      predictionOptions: [f.optionA, f.optionB],
    });
    ElMessage.success('赛事创建成功');
    // Reset form
    createForm.value = {
      eventID: '', title: '', type: 'basketball', teamA: '', teamB: '',
      ticketTotal: 100, optionA: '', optionB: '',
    };
  } catch {
    // Handled by interceptor
  } finally {
    creating.value = false;
  }
}

// --- Event Manager ---
const statusFlow = {
  CREATED: 'PREDICTION_OPEN',
  PREDICTION_OPEN: 'TICKET_OPEN',
  TICKET_OPEN: 'ONGOING',
};

const statusTagType = {
  CREATED: 'info',
  PREDICTION_OPEN: 'warning',
  TICKET_OPEN: '',
  ONGOING: 'success',
  SETTLED: 'danger',
};

async function advanceStatus(event) {
  const nextStatus = statusFlow[event.status];
  if (!nextStatus) return;

  try {
    await ElMessageBox.confirm(
      `确认将「${event.title}」状态推进为 ${nextStatus}？`,
      '确认操作'
    );
    await eventStore.updateStatus(event.id, nextStatus);
    ElMessage.success('状态已更新');
  } catch {
    // Cancelled or error
  }
}

async function settleEvent(event) {
  try {
    const { value: outcome } = await ElMessageBox.prompt(
      '请输入赛事结果（选项名称）',
      '录入结果',
      { inputPlaceholder: event.predictionOptions?.join(' 或 ') }
    );
    if (!outcome) return;
    await eventStore.recordResult(event.id, outcome);
    ElMessage.success('结算完成');
  } catch {
    // Cancelled or error
  }
}

async function runLottery(event) {
  try {
    const { value: ticketCountStr } = await ElMessageBox.prompt(
      `请输入抽签人数（总票数：${event.ticketTotal}）`,
      '执行抽签',
      { inputValue: String(event.ticketTotal), inputPattern: /^[1-9]\d*$/, inputErrorMessage: '请输入正整数' }
    );
    await ticketStore.runLottery(event.id, Number(ticketCountStr));
    ElMessage.success('抽签完成');
  } catch {
    // Cancelled or error
  }
}

// --- System stats ---
const totalEvents = computed(() => eventStore.events.length);
const activeEvents = computed(() =>
  eventStore.events.filter((e) => !['SETTLED', 'CREATED'].includes(e.status)).length
);
</script>

<template>
  <div class="admin-page">
    <h1 class="page-title">管理控制台</h1>

    <div class="admin-grid">
      <!-- System stats -->
      <GlassCard class="stats-card" padding="24px">
        <h2 class="card-title">系统概览</h2>
        <div class="admin-stats">
          <div class="admin-stat">
            <span class="admin-stat-value">{{ totalEvents }}</span>
            <span class="admin-stat-label">总赛事</span>
          </div>
          <div class="admin-stat">
            <span class="admin-stat-value">{{ activeEvents }}</span>
            <span class="admin-stat-label">进行中</span>
          </div>
        </div>
      </GlassCard>

      <!-- Create event form -->
      <GlassCard class="create-card" padding="28px">
        <h2 class="card-title">创建赛事</h2>
        <ElForm label-position="top" :model="createForm">
          <ElFormItem label="赛事 ID">
            <ElInput v-model="createForm.eventID" placeholder="例：evt005（字母数字_-，1-64位）" />
          </ElFormItem>

          <ElFormItem label="赛事名称">
            <ElInput v-model="createForm.title" placeholder="例：院际篮球决赛" />
          </ElFormItem>

          <ElFormItem label="赛事类型">
            <ElSelect v-model="createForm.type" style="width: 100%">
              <ElOption
                v-for="t in eventTypes"
                :key="t.value"
                :label="t.label"
                :value="t.value"
              />
            </ElSelect>
          </ElFormItem>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="队伍 A">
              <ElInput v-model="createForm.teamA" placeholder="队伍名称" />
            </ElFormItem>
            <ElFormItem label="队伍 B">
              <ElInput v-model="createForm.teamB" placeholder="队伍名称" />
            </ElFormItem>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="预测选项 A">
              <ElInput v-model="createForm.optionA" placeholder="例：教院胜" />
            </ElFormItem>
            <ElFormItem label="预测选项 B">
              <ElInput v-model="createForm.optionB" placeholder="例：丹青胜" />
            </ElFormItem>
          </div>

          <ElFormItem label="票务总量">
            <ElInputNumber v-model="createForm.ticketTotal" :min="1" :max="10000" style="width: 100%" />
          </ElFormItem>

          <ElButton
            type="primary"
            :loading="creating"
            style="width: 100%; margin-top: 8px"
            @click="handleCreate"
          >
            创建赛事
          </ElButton>
        </ElForm>
      </GlassCard>

      <!-- Event manager table -->
      <GlassCard class="manager-card" padding="24px">
        <h2 class="card-title">赛事管理</h2>
        <ElTable :data="eventStore.events" stripe style="width: 100%">
          <ElTableColumn prop="title" label="赛事名称" min-width="180" />
          <ElTableColumn prop="type" label="类型" width="80" />
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }">
              <ElTag :type="statusTagType[row.status] || 'info'" size="small">
                {{ row.status }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="280">
            <template #default="{ row }">
              <ElButton
                v-if="statusFlow[row.status]"
                type="primary"
                size="small"
                @click="advanceStatus(row)"
              >
                推进状态
              </ElButton>
              <ElButton
                v-if="row.status === 'ONGOING'"
                type="warning"
                size="small"
                @click="settleEvent(row)"
              >
                录入结果
              </ElButton>
              <ElButton
                v-if="row.status === 'TICKET_OPEN'"
                type="success"
                size="small"
                @click="runLottery(row)"
              >
                执行抽签
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.admin-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.stats-card {
  grid-column: 1 / -1;
}

.manager-card {
  grid-column: 1 / -1;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

.admin-stats {
  display: flex;
  gap: 48px;
}

.admin-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.admin-stat-value {
  font-size: 36px;
  font-weight: 800;
  color: var(--color-primary);
}

.admin-stat-label {
  font-size: 14px;
  color: var(--color-text-tertiary);
}
</style>
