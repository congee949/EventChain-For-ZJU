<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import api, { idempotencyHeaders } from '../api/index.js';
import GlassCard from '../components/GlassCard.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useActivityV2Store } from '../stores/activityV2.js';
import { useAuthStore } from '../stores/auth.js';
import { activityStatusLabel, marketStatusLabel, roleLabel } from '../utils/display.js';

const auth = useAuthStore();
const finance = useFinanceStore();
const activityStore = useActivityV2Store();
const role = computed(() => auth.user?.role || '');
const marketForm = reactive({ marketId: '', eventId: '', categoryId: '', labels: '主队胜,平局,客队胜', closeAt: '', marketCap: 4000 });
const activityForm = reactive({ id: '', categoryId: '', title: '', capacity: 100, applicationCloseAt: '', startsAt: '', endsAt: '' });
const serviceForm = reactive({ offerId: '', categoryId: '', offerType: 'VENUE', title: '', price: 50, cancellationBps: 15000, guaranteeBudget: 500, bonusBudget: 300, inventory: 20 });
const action = reactive({ marketId: '', outcomeId: '', evidence: '', activityId: '', seed: '', activityStatus: '' });
const claimReport = ref(null);
const busy = ref(false);

const canCreate = computed(() => role.value === 'organizer');
const canManageMarket = computed(() => ['organizer', 'admin'].includes(role.value));
const canVerify = computed(() => ['verifier', 'admin'].includes(role.value));
const canOperate = computed(() => ['operator', 'admin'].includes(role.value));
const canArbitrate = computed(() => role.value === 'arbitrator');
const selectedMarket = computed(() => finance.markets.find((item) => item.marketId === action.marketId));
const selectedActivity = computed(() => activityStore.activities.find((item) => item.id === action.activityId));

const FINANCE_ID_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
const ACTIVITY_ID_PATTERN = /^[A-Za-z0-9_.:-]{1,128}$/;
const text = (value) => String(value ?? '').trim();
const marketOutcomes = () => marketForm.labels.split(',').map((label) => label.trim()).filter(Boolean).map((label, index) => ({ id: `o${index + 1}`, label }));
const isAmount = (value, { allowZero = false } = {}) => {
  const raw = text(value);
  return /^\d+(?:\.\d{1,6})?$/.test(raw) && Number.isFinite(Number(raw)) && (allowZero ? Number(raw) >= 0 : Number(raw) > 0);
};
const timeValue = (value) => value ? new Date(value).getTime() : Number.NaN;
const showValidation = (message) => { ElMessage.warning(message); return false; };
const toISO = (value) => new Date(value).toISOString();

function validateMarketForm() {
  const outcomes = marketOutcomes();
  const closeAt = timeValue(marketForm.closeAt);
  if (!FINANCE_ID_PATTERN.test(text(marketForm.marketId))) return '市场编号需以小写字母或数字开头，只能包含小写字母、数字、下划线和连字符，最长 64 位';
  if (!FINANCE_ID_PATTERN.test(text(marketForm.eventId))) return '关联活动编号需使用小写字母、数字、下划线或连字符，最长 64 位';
  if (!text(marketForm.categoryId)) return '请选择积分分类';
  if (outcomes.length < 2 || outcomes.length > 8) return '结果标签需填写 2–8 项，并用英文逗号分隔';
  if (outcomes.some((item) => item.label.length > 64)) return '每个结果标签最多 64 个字符';
  if (!Number.isFinite(closeAt)) return '请选择有效的锁盘时间';
  if (closeAt <= Date.now() + 60_000) return '锁盘时间必须至少晚于当前时间 1 分钟';
  if (marketForm.marketCap !== null && marketForm.marketCap !== '' && (!isAmount(marketForm.marketCap) || Number(marketForm.marketCap) < 2000)) return '市场总额上限不得低于 2000，且最多保留 6 位小数';
  return '';
}

function validateActivityForm() {
  const closeAt = timeValue(activityForm.applicationCloseAt);
  const startsAt = timeValue(activityForm.startsAt);
  const endsAt = timeValue(activityForm.endsAt);
  if (!ACTIVITY_ID_PATTERN.test(text(activityForm.id))) return '活动编号需为 1–128 位字母、数字、下划线、连字符、点或冒号';
  if (!text(activityForm.title) || text(activityForm.title).length > 100) return '活动标题需填写 1–100 个字符';
  if (!text(activityForm.categoryId)) return '请选择积分分类';
  if (!Number.isInteger(activityForm.capacity) || activityForm.capacity < 1 || activityForm.capacity > 100000) return '活动容量必须是 1–100000 的整数';
  if (![closeAt, startsAt, endsAt].every(Number.isFinite)) return '请完整填写报名截止、开始和结束时间';
  if (closeAt <= Date.now()) return '报名截止时间必须晚于当前时间';
  if (startsAt <= closeAt || endsAt <= startsAt) return '时间需满足：报名截止 < 活动开始 < 活动结束';
  return '';
}

function validateServiceForm() {
  if (!FINANCE_ID_PATTERN.test(text(serviceForm.offerId))) return '商品编号需以小写字母或数字开头，只能包含小写字母、数字、下划线和连字符，最长 64 位';
  if (!text(serviceForm.title) || text(serviceForm.title).length > 100) return '服务标题需填写 1–100 个字符';
  if (!text(serviceForm.categoryId)) return '请选择积分分类';
  if (!['VENUE', 'EQUIPMENT', 'EVENT_PASS'].includes(serviceForm.offerType)) return '请选择有效的商品类型';
  if (!isAmount(serviceForm.price)) return '兑换价格必须大于 0，且最多保留 6 位小数';
  if (serviceForm.cancellationBps !== null && serviceForm.cancellationBps !== '' && (!Number.isInteger(serviceForm.cancellationBps) || serviceForm.cancellationBps < 10000 || serviceForm.cancellationBps > 15000)) return '取消赔付倍数必须是 10000–15000 的整数';
  if (serviceForm.guaranteeBudget !== null && serviceForm.guaranteeBudget !== '' && !isAmount(serviceForm.guaranteeBudget, { allowZero: true })) return 'A 保证金预算不得为负，且最多保留 6 位小数';
  if (serviceForm.bonusBudget !== null && serviceForm.bonusBudget !== '' && !isAmount(serviceForm.bonusBudget, { allowZero: true })) return '奖励预算不得为负，且最多保留 6 位小数';
  if (!Number.isInteger(serviceForm.inventory) || serviceForm.inventory < 1 || serviceForm.inventory > 1000000) return '库存必须是 1–1000000 的整数';
  return '';
}

function validateMarketAction(kind) {
  const market = selectedMarket.value;
  if (!market) return '请先选择要操作的市场';
  if (kind === 'result' || kind === 'vote') {
    const outcomeId = text(action.outcomeId);
    const allowed = (market.outcomes || []).map((item) => item.id);
    if (!outcomeId) return '请填写结果编号';
    if (kind === 'result' && !allowed.includes(outcomeId)) return `结果编号必须是：${allowed.join('、') || '该市场的有效结果'}`;
    if (kind === 'vote' && outcomeId !== 'void' && !allowed.includes(outcomeId)) return `仲裁结果必须是 ${allowed.join('、')} 或 void`;
  }
  if (kind === 'result' && !text(action.evidence)) return '提交结果前请填写证据或摘要';
  if (kind === 'result' && text(action.evidence).length > 256) return '证据或摘要最多 256 个字符';
  return '';
}

function validateActivityAction(kind) {
  const activity = selectedActivity.value;
  if (!activity) return '请先选择要操作的活动';
  if (kind === 'commit' || kind === 'draw') {
    const seedLength = action.seed.length;
    if (seedLength < 32 || seedLength > 256) return '独立种子需包含 32–256 个字符';
  }
  if (kind === 'commit' && activity.status !== 'DRAFT') return '只有草稿状态的活动可以承诺种子';
  if (kind === 'draw' && activity.status !== 'APPLICATION_OPEN') return '只有报名中的活动可以揭示种子并抽签';
  if (kind === 'draw' && timeValue(activity.applicationCloseAt) > Date.now()) return '报名截止后才能揭示种子并抽签';
  if (kind === 'status') {
    if (!action.activityStatus) return '请选择目标状态';
    const validTransition = action.activityStatus === 'CANCELLED'
      ? !['COMPLETED', 'CANCELLED'].includes(activity.status)
      : (activity.status === 'DRAWN' && action.activityStatus === 'ONGOING')
        || (activity.status === 'ONGOING' && action.activityStatus === 'COMPLETED');
    if (!validTransition) return `不能将“${activityStatusLabel(activity.status)}”更新为“${activityStatusLabel(action.activityStatus)}”`;
  }
  return '';
}

async function refresh() {
  await Promise.all([finance.refresh(), activityStore.refresh()]);
  marketForm.categoryId ||= finance.categories[0]?.id || '';
  activityForm.categoryId ||= marketForm.categoryId;
  serviceForm.categoryId ||= marketForm.categoryId;
}

onMounted(refresh);

async function run(fn, success = '链上操作成功') {
  busy.value = true;
  try {
    const result = await fn();
    ElMessage.success(success);
    await refresh();
    return result;
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message);
    throw error;
  } finally {
    busy.value = false;
  }
}

async function createMarket() {
  const error = validateMarketForm();
  if (error) return showValidation(error);
  await run(() => api.post('/finance/markets', {
    marketId: text(marketForm.marketId), eventId: text(marketForm.eventId), categoryId: marketForm.categoryId,
    outcomes: marketOutcomes(), closeAt: toISO(marketForm.closeAt), stakeBucket: 'BONUS', marketCap: marketForm.marketCap,
  }), 'B_bonus 市场已创建');
}

async function createActivity() {
  const error = validateActivityForm();
  if (error) return showValidation(error);
  await run(() => api.post('/activities', {
    id: text(activityForm.id), categoryId: activityForm.categoryId, title: text(activityForm.title), capacity: activityForm.capacity,
    applicationCloseAt: toISO(activityForm.applicationCloseAt), startsAt: toISO(activityForm.startsAt), endsAt: toISO(activityForm.endsAt),
  }), '活动草稿已创建');
}

async function createService() {
  const error = validateServiceForm();
  if (error) return showValidation(error);
  await run(() => api.post('/finance/offers', {
    offerId: text(serviceForm.offerId), categoryId: serviceForm.categoryId, offerType: serviceForm.offerType,
    title: text(serviceForm.title), price: serviceForm.price, cancellationBps: serviceForm.cancellationBps,
    guaranteeBudget: serviceForm.guaranteeBudget, bonusBudget: serviceForm.bonusBudget, inventory: serviceForm.inventory,
  }, { headers: idempotencyHeaders('offer') }), '服务商品已创建并锁定赔付预算');
}

async function runMarketAction(kind, url, body = {}) {
  const error = validateMarketAction(kind);
  if (error) return showValidation(error);
  return run(() => api.post(url, body, { headers: idempotencyHeaders('ops') }));
}

async function runActivityAction(kind, fn) {
  const error = validateActivityAction(kind);
  if (error) return showValidation(error);
  return run(fn);
}

async function processClaims() {
  claimReport.value = await run(() => api.post('/finance/admin/claims/process', {}, { headers: idempotencyHeaders('ops') }));
}
</script>

<template>
  <div class="admin-page">
    <section class="admin-hero">
      <div><p>证书角色控制</p><h1>运营控制台</h1><span>当前角色：<b>{{ roleLabel(role) }}</b>。按钮可见性只是交互提示，最终权限由 Fabric 证书属性和链码再次校验。</span></div>
      <div class="network-state"><i></i><span>第二版网络</span><b>财务链码 · 活动链码</b></div>
    </section>
    <div class="overview-grid">
      <GlassCard :hoverable="false"><strong>{{ finance.markets.length }}</strong><span>预测市场</span></GlassCard>
      <GlassCard :hoverable="false"><strong>{{ activityStore.activities.length }}</strong><span>活动</span></GlassCard>
      <GlassCard :hoverable="false"><strong>{{ finance.offers.length }}</strong><span>服务商品</span></GlassCard>
      <GlassCard :hoverable="false"><strong>{{ roleLabel(role) }}</strong><span>证书角色</span></GlassCard>
    </div>

    <section v-if="canCreate" class="section">
      <div class="section-heading"><div><p>创建</p><h2>创建业务对象</h2></div><span><i class="required">*</i> 为必填字段；保证金与预算仍由链码复核</span></div>
      <div class="form-grid">
        <GlassCard :hoverable="false"><h3>预测市场</h3><div class="form-stack">
          <div class="form-field"><label>市场编号 <i class="required">*</i></label><el-input v-model="marketForm.marketId" placeholder="如 football_final_01"/><small>唯一链上编号；小写字母或数字开头，最长 64 位。</small></div>
          <div class="form-field"><label>关联活动编号 <i class="required">*</i></label><el-input v-model="marketForm.eventId" placeholder="如 event_2026_01"/><small>用于把市场与线下活动关联，不要求活动已在本页创建。</small></div>
          <div class="form-field"><label>积分分类 <i class="required">*</i></label><el-select v-model="marketForm.categoryId" placeholder="请选择积分分类"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><small>参与者将使用该分类下的 B_bonus 建立仓位。</small></div>
          <div class="form-field"><label>结果标签 <i class="required">*</i></label><el-input v-model="marketForm.labels" placeholder="主队胜,平局,客队胜"/><small>填写 2–8 项，以英文逗号分隔；系统依次生成 o1、o2…。</small></div>
          <div class="form-field"><label>锁盘时间 <i class="required">*</i></label><el-date-picker v-model="marketForm.closeAt" type="datetime" placeholder="选择锁盘时间"/><small>必须至少晚于当前时间 1 分钟，此后不能再建立仓位。</small></div>
          <div class="form-field"><label>市场总额上限 <span class="optional">可选</span></label><el-input-number v-model="marketForm.marketCap" :min="2000"/><small>默认 4000 B_bonus；创建时会锁定总额 10% 的 A 保证金（最低 200 A）。</small></div>
          <el-button type="primary" :loading="busy" @click="createMarket">创建 B_bonus 市场</el-button>
        </div></GlassCard>

        <GlassCard :hoverable="false"><h3>活动与抽签</h3><div class="form-stack">
          <div class="form-field"><label>活动编号 <i class="required">*</i></label><el-input v-model="activityForm.id" placeholder="如 lecture_2026_01"/><small>唯一链上编号；可使用字母、数字、下划线、连字符、点和冒号。</small></div>
          <div class="form-field"><label>活动标题 <i class="required">*</i></label><el-input v-model="activityForm.title" maxlength="100" show-word-limit placeholder="活动名称"/><small>面向学生展示，长度为 1–100 个字符。</small></div>
          <div class="form-field"><label>积分分类 <i class="required">*</i></label><el-select v-model="activityForm.categoryId" placeholder="请选择积分分类"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><small>决定签到奖励发放到哪个 B_bonus 分类。</small></div>
          <div class="form-field"><label>中签容量 <i class="required">*</i></label><el-input-number v-model="activityForm.capacity" :min="1" :max="100000" :precision="0"/><small>抽签最多选出的参与人数，范围 1–100000。</small></div>
          <div class="form-field"><label>报名截止 <i class="required">*</i></label><el-date-picker v-model="activityForm.applicationCloseAt" type="datetime" placeholder="选择报名截止时间"/><small>必须晚于当前时间；截止后才能揭示种子并抽签。</small></div>
          <div class="form-field"><label>开始时间 <i class="required">*</i></label><el-date-picker v-model="activityForm.startsAt" type="datetime" placeholder="选择活动开始时间"/><small>必须晚于报名截止时间。</small></div>
          <div class="form-field"><label>结束时间 <i class="required">*</i></label><el-date-picker v-model="activityForm.endsAt" type="datetime" placeholder="选择活动结束时间"/><small>必须晚于活动开始时间。</small></div>
          <el-button type="primary" :loading="busy" @click="createActivity">创建活动草稿</el-button>
        </div></GlassCard>

        <GlassCard :hoverable="false"><h3>服务兑换商品</h3><div class="form-stack">
          <div class="form-field"><label>商品编号 <i class="required">*</i></label><el-input v-model="serviceForm.offerId" placeholder="如 venue_booking_01"/><small>唯一链上编号；小写字母或数字开头，最长 64 位。</small></div>
          <div class="form-field"><label>服务标题 <i class="required">*</i></label><el-input v-model="serviceForm.title" maxlength="100" show-word-limit placeholder="服务名称"/><small>面向学生展示，长度为 1–100 个字符。</small></div>
          <div class="form-field"><label>积分分类 <i class="required">*</i></label><el-select v-model="serviceForm.categoryId" placeholder="请选择积分分类"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><small>学生使用该分类下的 B_paid 兑换服务。</small></div>
          <div class="form-field"><label>商品类型 <i class="required">*</i></label><el-select v-model="serviceForm.offerType"><el-option label="场馆" value="VENUE"/><el-option label="器材" value="EQUIPMENT"/><el-option label="活动通行" value="EVENT_PASS"/></el-select><small>用于区分场馆、器材和活动通行权益。</small></div>
          <div class="form-field"><label>兑换价格 <i class="required">*</i></label><el-input-number v-model="serviceForm.price" :min="0.000001"/><small>单位为 B_paid，必须大于 0，最多 6 位小数。</small></div>
          <div class="form-field"><label>取消赔付倍数 <span class="optional">可选</span></label><el-input-number v-model="serviceForm.cancellationBps" :min="10000" :max="15000" :step="100" :precision="0"/><small>10000 = 1 倍退款，15000 = 1.5 倍退款。</small></div>
          <div class="form-field"><label>A 保证金预算 <span class="optional">可选</span></label><el-input-number v-model="serviceForm.guaranteeBudget" :min="0"/><small>创建时从组织者 A 余额预留，用于取消等赔付。</small></div>
          <div class="form-field"><label>B_bonus 奖励预算 <span class="optional">可选</span></label><el-input-number v-model="serviceForm.bonusBudget" :min="0"/><small>创建时从本分类锁定，用于服务补偿奖励。</small></div>
          <div class="form-field"><label>库存 <i class="required">*</i></label><el-input-number v-model="serviceForm.inventory" :min="1" :max="1000000" :precision="0"/><small>最多可创建的服务订单数，范围 1–1000000。</small></div>
          <el-button type="primary" :loading="busy" @click="createService">创建并预存保证金</el-button>
        </div></GlassCard>
      </div>
    </section>

    <section class="section">
      <div class="section-heading"><div><p>状态控制</p><h2>状态、结果与结算</h2></div><span>操作前会检查当前选择及该动作所需字段</span></div>
      <div class="action-grid">
        <GlassCard v-if="canOperate" :hoverable="false"><h3>活动签到核验</h3><p class="helper">调用摄像头扫描学生的 30 秒动态二维码，核销票据并发放签到奖励。</p><router-link class="checkin-entry" to="/check-in">打开签到扫码 →</router-link></GlassCard>
        <GlassCard :hoverable="false"><h3>市场动作</h3><div class="form-stack">
          <div class="form-field"><label>目标市场 <i class="required">*</i></label><el-select v-model="action.marketId" filterable placeholder="选择市场"><el-option v-for="m in finance.markets" :key="m.marketId" :label="`${m.marketId} · ${marketStatusLabel(m.status)}`" :value="m.marketId"/></el-select><small>开放、锁盘、确认和结算只需选择市场。</small></div>
          <div class="form-field"><label>结果编号 <span class="conditional">提交结果/仲裁必填</span></label><el-input v-model="action.outcomeId" placeholder="如 o1；仲裁作废填 void"/><small>提交结果须填写市场内的结果编号；仲裁投票还可填写 void。</small></div>
          <div class="form-field"><label>证据或摘要 <span class="conditional">提交结果必填</span></label><el-input v-model="action.evidence" type="textarea" :rows="2" maxlength="256" show-word-limit placeholder="填写结果依据或证据摘要"/><small>服务器会将文本转换为 SHA-256 摘要后上链。</small></div>
          <div class="button-grid">
            <el-button v-if="canManageMarket" :disabled="busy" @click="runMarketAction('open', `/finance/markets/${action.marketId}/open`)">开放</el-button>
            <el-button v-if="canManageMarket" :disabled="busy" @click="runMarketAction('lock', `/finance/markets/${action.marketId}/lock`)">锁盘</el-button>
            <el-button v-if="canCreate" :disabled="busy" @click="runMarketAction('result', `/finance/markets/${action.marketId}/result`, { outcomeId: text(action.outcomeId), evidence: text(action.evidence) })">提交结果</el-button>
            <el-button v-if="canVerify" :disabled="busy" @click="runMarketAction('confirm', `/finance/markets/${action.marketId}/confirm`)">二次确认</el-button>
            <el-button v-if="canOperate" :disabled="busy" @click="runMarketAction('finalize', `/finance/markets/${action.marketId}/finalize`)">临时结算</el-button>
            <el-button v-if="canArbitrate" :disabled="busy" @click="runMarketAction('vote', `/finance/markets/${action.marketId}/vote`, { outcomeId: text(action.outcomeId) })">仲裁投票</el-button>
          </div>
        </div></GlassCard>
        <GlassCard :hoverable="false"><h3>抽签与活动状态</h3><div class="form-stack">
          <div class="form-field"><label>目标活动 <i class="required">*</i></label><el-select v-model="action.activityId" filterable placeholder="选择活动"><el-option v-for="a in activityStore.activities" :key="a.id" :label="`${a.title} · ${activityStatusLabel(a.status)}`" :value="a.id"/></el-select><small>承诺种子、抽签和状态更新均以此活动为目标。</small></div>
          <div class="form-field"><label>独立种子 <span class="conditional">承诺/抽签必填</span></label><el-input v-model="action.seed" show-password placeholder="输入 32–256 个字符"/><small>验证员先承诺哈希，组织者在报名截止后用完全相同的原文揭示。</small></div>
          <div class="form-field"><label>目标状态 <span class="conditional">更新状态必填</span></label><el-select v-model="action.activityStatus" placeholder="选择目标状态"><el-option v-for="s in ['ONGOING', 'COMPLETED', 'CANCELLED']" :key="s" :label="activityStatusLabel(s)" :value="s"/></el-select><small>正常路径为“已抽签 → 进行中 → 已结束”；未结束活动也可取消。</small></div>
          <div class="button-grid">
            <el-button v-if="canVerify" :disabled="busy" @click="runActivityAction('commit', () => api.post(`/activities/${action.activityId}/draw-commitment`, { seed: action.seed }))">承诺种子</el-button>
            <el-button v-if="canCreate" :disabled="busy" @click="runActivityAction('draw', () => api.post(`/activities/${action.activityId}/draw`, { seed: action.seed }))">揭示并抽签</el-button>
            <el-button v-if="canCreate || role === 'admin'" :disabled="busy" @click="runActivityAction('status', () => api.put(`/activities/${action.activityId}/status`, { status: action.activityStatus }))">更新状态</el-button>
          </div>
        </div></GlassCard>
        <GlassCard v-if="canOperate" :hoverable="false"><h3>分批收益处理</h3><p class="helper">每批扫描 100 个参与者、并发 5，账本版本冲突自动重试。收益先进入 7 天待成熟状态。</p><el-button type="primary" :loading="busy" @click="processClaims">处理所有待结算批次</el-button><pre v-if="claimReport">{{ JSON.stringify(claimReport, null, 2) }}</pre></GlassCard>
      </div>
    </section>

    <section class="section"><div class="section-heading"><div><p>链上对象</p><h2>当前对象</h2></div></div><div class="object-table"><div v-for="m in finance.markets" :key="m.marketId"><b>{{ m.marketId }}</b><span>{{ m.categoryId }}</span><em>{{ marketStatusLabel(m.status) }}</em><span>池 {{ formatAmount(m.displayedPool) }}</span></div><div v-for="a in activityStore.activities" :key="a.id"><b>{{ a.title }}</b><span>{{ a.categoryId }}</span><em>{{ activityStatusLabel(a.status) }}</em><span>{{ a.applicationCount }}/{{ a.capacity }}</span></div></div></section>
  </div>
</template>

<style scoped>
.admin-page{padding-bottom:48px}.admin-hero{display:flex;justify-content:space-between;align-items:end;padding:40px 8px 28px}.admin-hero p,.section-heading p{color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.18em}.admin-hero h1{font-size:clamp(38px,6vw,60px);letter-spacing:-.05em}.admin-hero span{display:block;color:var(--color-text-secondary);margin-top:9px}.network-state{text-align:right}.network-state i{display:inline-block;width:8px;height:8px;border-radius:50%;background:var(--color-success);margin-right:7px}.network-state span,.network-state b{font-size:11px}.overview-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:32px}.overview-grid>div{padding:18px!important}.overview-grid strong,.overview-grid span{display:block}.overview-grid strong{font-size:27px}.overview-grid span{color:var(--color-text-tertiary);font-size:11px;margin-top:4px}.section{margin-bottom:34px}.section-heading{display:flex;justify-content:space-between;align-items:end;margin-bottom:15px}.section-heading h2{font-size:23px;margin-top:3px}.section-heading>span{color:var(--color-text-tertiary);font-size:11px}.form-grid,.action-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.form-grid h3,.action-grid h3{margin-bottom:14px}.form-stack{display:grid;gap:14px}.form-field{display:grid;gap:5px}.form-field label{font-size:12px;font-weight:750;color:var(--color-text-primary)}.form-field small{min-height:30px;color:var(--color-text-tertiary);font-size:10px;line-height:1.5}.required{color:var(--ec-red);font-style:normal}.optional,.conditional{display:inline-block;margin-left:4px;padding:1px 5px;border-radius:4px;background:var(--ec-inset);color:var(--color-text-tertiary);font-size:9px;font-weight:600;vertical-align:1px}.conditional{color:var(--color-primary)}.form-stack :deep(.el-date-editor),.form-stack :deep(.el-input-number),.form-stack :deep(.el-select){width:100%}.button-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}.helper{color:var(--color-text-secondary);font-size:12px;line-height:1.6;margin-bottom:16px}.checkin-entry{display:inline-flex;padding:10px 14px;border-radius:10px;background:var(--ec-red);color:#fff;text-decoration:none;font:700 14px var(--ec-font-display)}.checkin-entry:hover{color:#fff;background:var(--ec-orange-hi)}pre{max-height:230px;background:rgba(15,23,42,.06);padding:12px;border-radius:10px;margin-top:12px}.object-table{display:grid;gap:7px}.object-table>div{display:grid;grid-template-columns:2fr 1fr 1fr 1fr;gap:12px;padding:12px 15px;border-radius:11px;background:rgba(255,255,255,.23);font-size:12px}.object-table em{font-style:normal;color:#087f5b;font-weight:700}.object-table span{color:var(--color-text-secondary)}@media(max-width:980px){.form-grid,.action-grid{grid-template-columns:1fr 1fr}.overview-grid{grid-template-columns:1fr 1fr}}@media(max-width:650px){.admin-hero{align-items:start}.network-state{display:none}.form-grid,.action-grid{grid-template-columns:1fr}.object-table>div{grid-template-columns:1fr 1fr}}
.admin-page{padding:0 30px 48px}.admin-hero{padding-left:0;padding-right:0;border-bottom:2px solid var(--ec-ink);margin-bottom:24px}.overview-grid>div,.form-grid>div,.action-grid>div{background:var(--ec-card);border-color:var(--ec-line)}.overview-grid strong{font-family:var(--ec-font-display)}.object-table>div{background:var(--ec-inset);border-radius:var(--ec-r-field)}@media(max-width:650px){.admin-page{padding:0 16px 40px}.section-heading>span{max-width:48%;text-align:right}}
</style>
