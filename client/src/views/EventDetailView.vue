<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import api from '../api/index.js';
import GlassCard from '../components/GlassCard.vue';
import CountdownTimer from '../components/CountdownTimer.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useActivityV2Store } from '../stores/activityV2.js';
import { useAuthStore } from '../stores/auth.js';
import ProbabilityBar from '../components/ProbabilityBar.vue';
import StatusPill from '../components/StatusPill.vue';

const route = useRoute();
const router = useRouter();
const finance = useFinanceStore();
const activityStore = useActivityV2Store();
const auth = useAuthStore();
const selectedOutcome = ref('');
const amount = ref(10);
const challengeReason = ref('');
const submitting = ref(false);
const market = computed(() => finance.currentMarket);
const activity = computed(() => activityStore.activities.find((item) => item.id === market.value?.eventId));
const category = computed(() => finance.categories.find((item) => item.id === market.value?.categoryId));
const walletBucket = computed(() => finance.wallet?.categories?.[market.value?.categoryId]?.[market.value?.stakeBucket === 'PAID' ? 'paid' : 'bonus']);
const marketAccepting = computed(() => market.value?.status === 'OPEN' && new Date(market.value.closeAt).getTime() > Date.now());
const probability = (id) => ((market.value?.outcomeProbabilityBps?.[id] || 0) / 100).toFixed(1);
const statusText = { OPEN: '预测开放', LOCKED: '已锁盘', RESULT_PROPOSED: '结果待确认', CHALLENGED: '争议仲裁中', PROVISIONAL_FINALIZED: '临时结算', FINALIZED: '已结算', PAUSED: '已暂停' };
const probabilityOutcomes = computed(() => (market.value?.outcomes || []).map((item) => ({ ...item, pct: Number(probability(item.id)) })));

async function load(id) {
  await Promise.all([finance.refresh(), activityStore.refresh(), finance.fetchMarket(id)]);
  selectedOutcome.value = market.value?.outcomes?.[0]?.id || '';
}
onMounted(() => load(route.params.id));
watch(() => route.params.id, (id) => id && load(id));

async function place() {
  if (!selectedOutcome.value) return ElMessage.warning('请选择预测结果');
  submitting.value = true;
  try {
    await finance.placePosition(market.value.marketId, selectedOutcome.value, amount.value);
    await finance.fetchMarket(market.value.marketId);
    ElMessage.success('仓位已写入私有账本');
  } finally { submitting.value = false; }
}
async function challenge() {
  if (challengeReason.value.trim().length < 4) return ElMessage.warning('请填写可核验的挑战理由');
  await api.post(`/finance/markets/${market.value.marketId}/challenge`, { reason: challengeReason.value });
  await finance.fetchMarket(market.value.marketId);
  ElMessage.success('挑战已提交并锁定保证金');
}
async function claim() {
  await finance.claim(market.value.marketId, market.value.settlementEpoch);
  ElMessage.success('Pending Claim 已生成，7 天后方可成熟');
}
</script>

<template>
  <div v-if="market" class="event-detail">
    <button class="back-link" @click="router.push('/')">← 返回赛事市场</button>
    <GlassCard class="event-header" padding="32px" :hoverable="false">
      <div class="header-top"><span>{{ category?.name || market.categoryId }}</span><StatusPill :status="market.status === 'OPEN' ? 'open' : 'closed'" :label="statusText[market.status] || market.status" /></div>
      <p class="market-id">{{ market.marketId }}</p>
      <h1>{{ activity?.title || market.eventId }}</h1>
      <p class="header-copy">{{ market.stakeBucket === 'BONUS' ? '仅使用不可兑回的 B_bonus' : '使用有 A 储备的 B_paid' }} · 公开概率来自五分钟取整快照</p>
      <div v-if="market.status === 'OPEN'" class="countdown-wrap"><span>距离锁盘</span><CountdownTimer :target-time="market.closeAt" /></div>
      <ProbabilityBar class="probability-grid" :outcomes="probabilityOutcomes" variant="split" />
    </GlassCard>

    <div class="detail-grid">
      <section class="detail-column">
        <GlassCard padding="24px" :hoverable="false">
          <div class="card-heading"><div><p>PARI-MUTUEL POOL</p><h2>奖金池状态</h2></div><span>{{ market.stakeBucket }}</span></div>
          <div class="pool-stats">
            <div><strong>{{ formatAmount(market.displayedPool) }}</strong><small>公开池规模</small></div>
            <div><strong>{{ formatAmount(market.marketCap) }}</strong><small>市场上限</small></div>
            <div><strong>{{ market.participantCount || '—' }}</strong><small>参与人数</small></div>
          </div>
          <p class="privacy-note">精确仓位、个人选择和实时流入不公开。页面不会通过排行榜暴露其他参与者。</p>
        </GlassCard>

        <GlassCard v-if="activity" padding="24px" :hoverable="false">
          <div class="card-heading"><div><p>CONNECTED ACTIVITY</p><h2>{{ activity.title }}</h2></div><span>{{ activity.status }}</span></div>
          <div class="activity-meta"><span>容量 {{ activity.capacity }}</span><span>申请 {{ activity.applicationCount }}</span><span>{{ new Date(activity.startsAt).toLocaleString('zh-CN') }}</span></div>
          <button class="soft-btn" @click="router.push('/tickets')">前往票务大厅</button>
        </GlassCard>

        <GlassCard padding="24px" :hoverable="false">
          <div class="card-heading"><div><p>SETTLEMENT SAFETY</p><h2>结算与纠错路径</h2></div></div>
          <div class="timeline"><div><b>1</b><span>组织者提交结果和证据摘要</span></div><div><b>2</b><span>独立验证者二次确认并开启挑战期</span></div><div><b>3</b><span>争议时由仲裁者投票；否则临时结算</span></div><div><b>4</b><span>Claim 等待 7 天后成熟，期间允许暂停和更正</span></div></div>
        </GlassCard>
      </section>

      <aside>
        <GlassCard v-if="auth.user?.role === 'student'" class="position-panel" padding="24px" :hoverable="false">
          <p class="panel-kicker">PRIVATE POSITION</p><h2>建立预测仓位</h2>
          <p class="available">可用 {{ market.stakeBucket === 'PAID' ? 'B_paid' : 'B_bonus' }}：<b>{{ formatAmount(walletBucket?.available) }}</b></p>
          <div class="outcome-buttons">
            <button v-for="outcome in market.outcomes" :key="outcome.id" :class="{ selected: selectedOutcome === outcome.id }" @click="selectedOutcome = outcome.id"><span>{{ outcome.label }}</span><b>{{ probability(outcome.id) }}%</b></button>
          </div>
          <label>投入数量</label><el-input-number v-model="amount" :min="0.000001" :precision="6" />
          <button class="submit-btn" :disabled="!marketAccepting || submitting" @click="place">{{ submitting ? '链上确认中…' : marketAccepting ? '确认建立仓位' : market.status === 'OPEN' ? '已到锁盘时间，等待链上锁定' : '市场当前不可参与' }}</button>
          <p class="fine-print">仓位不可转让；结算费仅在存在获胜方时收取。无人命中或市场作废时原路退款且不收费。</p>
        </GlassCard>

        <GlassCard v-else class="position-panel" padding="24px" :hoverable="false"><p class="panel-kicker">ROLE VIEW</p><h2>只读市场视图</h2><p class="fine-print">当前证书角色为 {{ auth.user?.role }}。只有 student 身份可以建立仓位或提出参与者挑战；管理动作请前往运营控制台。</p><button class="soft-btn" @click="router.push('/admin')">前往运营控制台</button></GlassCard>
        <GlassCard v-if="market.status === 'RESULT_PROPOSED' && auth.user?.role === 'student'" class="challenge-panel" padding="24px" :hoverable="false">
          <h3>对结果提出挑战</h3><p>提交理由摘要并锁定挑战保证金，避免无成本滥诉。</p><el-input v-model="challengeReason" type="textarea" :rows="3" placeholder="可核验的挑战理由" /><button class="danger-btn" @click="challenge">提交挑战</button>
        </GlassCard>
        <GlassCard v-if="market.status === 'PROVISIONAL_FINALIZED' && auth.user?.role === 'student'" class="challenge-panel" padding="24px" :hoverable="false">
          <h3>领取待成熟收益</h3><p>当前结算 epoch {{ market.settlementEpoch }}。Claim 会先进入 7 天等待期。</p><button class="soft-btn" @click="claim">生成 Pending Claim</button>
        </GlassCard>
      </aside>
    </div>
  </div>
  <div v-else class="loading-state">正在读取链上市场…</div>
</template>

<style scoped>
.event-detail { display: grid; gap: 20px; padding:26px 30px 42px; }.back-link { width: fit-content; border: 0; background: transparent; color: var(--ec-red); cursor: pointer; font:600 15px var(--ec-font-display);text-transform:uppercase; }
.event-header { text-align: center; }.header-top { display: flex; justify-content: space-between; color: var(--color-text-secondary); font-size: 12px; font-weight: 700; text-transform: uppercase; }.status-badge { padding: 5px 12px; color: #087f5b; }.market-id { color: var(--color-primary); font-weight: 800; letter-spacing: .12em; font-size: 11px; margin-top: 14px; }.event-header h1 { font-size: clamp(32px,5vw,54px); letter-spacing: -.04em; margin: 5px 0 8px; }.header-copy { color: var(--color-text-secondary); }
.countdown-wrap { display:flex;justify-content:center;align-items:center;gap:10px;margin-top:14px;color:var(--color-text-tertiary);font-size:12px; }
.probability-grid{max-width:540px;margin:26px auto 0}
.detail-grid { display: grid; grid-template-columns: minmax(0,1fr) 360px; gap: 20px; align-items: start; }.detail-column, aside { display: grid; gap: 20px; }.card-heading { display: flex; justify-content: space-between; align-items: start; }.card-heading p,.panel-kicker { color: var(--color-primary); font-size: 10px; font-weight: 800; letter-spacing: .16em; }.card-heading h2 { margin-top: 3px; }.card-heading > span { font-size: 11px; font-weight: 800; color: var(--color-primary); }
.pool-stats { display: grid; grid-template-columns: repeat(3,1fr); gap: 10px; margin-top: 22px; }.pool-stats div { background:var(--ec-inset); border-radius:var(--ec-r-field); padding: 15px; }.pool-stats strong,.pool-stats small { display: block; }.pool-stats strong {font:800 27px var(--ec-font-display)}.pool-stats small { color: var(--ec-faint); margin-top: 5px;font:500 9px var(--ec-font-mono);text-transform:uppercase }.privacy-note,.fine-print,.challenge-panel p { color: var(--ec-muted); font-size: 12px; line-height: 1.65; margin-top: 15px; }
.activity-meta { display: flex; flex-wrap: wrap; gap: 14px; color:var(--ec-muted); margin: 17px 0;font:500 10px var(--ec-font-mono) }.timeline { display: grid; gap: 11px; margin-top: 18px; }.timeline div { display: grid; grid-template-columns: 28px 1fr; align-items: center; gap: 10px; }.timeline b { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 50%; background:var(--ec-ink); color:var(--ec-orange);font-family:var(--ec-font-mono) }.timeline span { color:var(--ec-ink-2); font-size: 13px; }
.position-panel{background:var(--ec-card-white);border-color:var(--ec-ink)}.position-panel h2 { font-size: 24px; margin-top: 4px; }.available { color:var(--ec-muted); margin: 8px 0 18px; }.outcome-buttons { display: grid; gap: 8px; }.outcome-buttons button { display: flex; justify-content: space-between; border: 1px solid var(--ec-line); border-radius: 11px; padding: 12px; background:transparent; cursor: pointer; color: inherit; }.outcome-buttons button.selected { border-color:var(--ec-red);background:rgba(232,72,44,.08)}.position-panel label { display: block; margin: 18px 0 8px;font:500 10px var(--ec-font-mono);letter-spacing:.14em;text-transform:uppercase}.position-panel :deep(.el-input-number) { width: 100%; }.submit-btn,.soft-btn,.danger-btn { width: 100%; border: 0; border-radius: 11px; padding: 12px;font:700 15px var(--ec-font-display);text-transform:uppercase;cursor: pointer; margin-top: 14px; }.submit-btn { color:var(--ec-ink);background:var(--ec-orange)}.submit-btn:disabled { opacity:.45; cursor:not-allowed; }.soft-btn { color:var(--ec-cream);background:var(--ec-ink)}.danger-btn { color:var(--ec-danger);background:rgba(180,35,24,.08)}.loading-state { text-align:center;padding:100px;color:var(--ec-faint); }
@media (max-width: 850px) { .detail-grid { grid-template-columns: 1fr; }.pool-stats { grid-template-columns: 1fr 1fr; } }
@media (max-width: 520px) { .pool-stats { grid-template-columns: 1fr; }.event-header { padding: 22px !important; } }
</style>
