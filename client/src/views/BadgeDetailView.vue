<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import BadgeArtwork from '../components/BadgeArtwork.vue';
import EcButton from '../components/EcButton.vue';
import GlassCard from '../components/GlassCard.vue';
import { useAuthStore } from '../stores/auth.js';
import { badgeErrorMessage, useBadgesStore } from '../stores/badges.js';

const route = useRoute();
const auth = useAuthStore();
const badges = useBadgesStore();
const confirmOpen = ref(false);
const confirmTitle = ref(null);
const confirmDialog = ref(null);
const claimTrigger = ref(null);
const actionMessage = ref('');

const seriesId = computed(() => String(route.params.seriesId || route.params.id || ''));
const item = computed(() => badges.currentSeries);
const view = computed(() => badges.currentView || {});
const award = computed(() => view.value.award || null);
const instance = computed(() => view.value.instance || null);
const eligibility = computed(() => view.value.eligibility || null);
const claimState = computed(() => badges.claimStates[seriesId.value] || { status: 'idle', message: '' });
const isBusy = computed(() => ['submitting', 'pending'].includes(claimState.value.status));
const isStudent = computed(() => auth.user?.role === 'student');
const isActive = computed(() => item.value?.status === 'ACTIVE');
const withinClaimWindow = computed(() => {
  if (!item.value) return false;
  const now = Date.now();
  return now >= new Date(item.value.claimOpenAt).getTime() && now <= new Date(item.value.claimCloseAt).getTime();
});
const isEligible = computed(() => eligibility.value?.status === 'CLAIMABLE' || view.value.eligible === true || view.value.claimable === true);
const canClaim = computed(() => isStudent.value && isEligible.value && isActive.value && withinClaimWindow.value && !award.value && !isBusy.value);
const serial = computed(() => award.value?.serialNumber ?? instance.value?.serialNumber);
const statusLabels = { DRAFT: '筹备中', ACTIVE: '开放领取', PAUSED: '暂时停领', CLOSED: '发行结束', FINALIZED: '已最终确认' };
const statusLabel = computed(() => statusLabels[item.value?.status] || item.value?.status || '未知状态');
const linkedActivities = computed(() => item.value?.activities || item.value?.activityLinks || item.value?.links || []);

function formatDate(value, dateOnly = false) {
  if (!value) return '—';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '—';
  return new Intl.DateTimeFormat('zh-CN', dateOnly
    ? { year: 'numeric', month: 'long', day: 'numeric' }
    : { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(parsed);
}

function openConfirmation() {
  confirmOpen.value = true;
  nextTick(() => confirmTitle.value?.focus());
}

function closeConfirmation() {
  if (!isBusy.value) {
    confirmOpen.value = false;
    nextTick(() => claimTrigger.value?.$el?.focus?.() || claimTrigger.value?.focus?.());
  }
}

function trapDialogFocus(event) {
  if (event.key === 'Escape') {
    event.preventDefault();
    closeConfirmation();
    return;
  }
  if (event.key !== 'Tab') return;
  const focusable = [...(confirmDialog.value?.querySelectorAll('button,[href],[tabindex]:not([tabindex="-1"])') || [])]
    .filter((element) => !element.disabled);
  if (!focusable.length) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

async function submitClaim() {
  confirmOpen.value = false;
  actionMessage.value = '';
  try {
    const result = await badges.claim(seriesId.value);
    actionMessage.value = result?.award ? '领取成功，徽章已写入链上。' : badges.claimStates[seriesId.value]?.message || '';
  } catch (cause) {
    actionMessage.value = badgeErrorMessage(cause);
  }
}

async function load() {
  actionMessage.value = '';
  try {
    if (isStudent.value) {
      await badges.loadDetail(seriesId.value);
    } else {
      const [loadedSeries, activities] = await Promise.all([
        badges.loadSeries(seriesId.value),
        badges.loadSeriesLinks(seriesId.value),
      ]);
      badges.currentSeries = { ...loadedSeries, activities };
      badges.currentView = null;
    }
    if (!award.value && claimState.value.status === 'pending') {
      badges.resumePending(seriesId.value).catch((cause) => {
        actionMessage.value = badgeErrorMessage(cause);
      });
    }
  } catch { /* The page renders the store error. */ }
}

watch(seriesId, load);
onMounted(load);
</script>

<template>
  <div class="badge-detail-page">
    <div v-if="badges.detailLoading" class="detail-state" role="status" aria-live="polite">正在核对系列和你的领取状态…</div>
    <div v-else-if="badges.error || !item" class="detail-state detail-state--error" role="alert">
      <h1>徽章详情暂时不可用</h1><p>{{ badges.error || '未找到该系列。' }}</p><EcButton variant="outline" @click="load">重新加载</EcButton>
    </div>
    <template v-else>
      <header class="detail-header">
        <RouterLink to="/badges" class="back-link">← 返回赛季徽章</RouterLink>
        <div><p class="eyebrow">VERIFIABLE SEASON MEMENTO</p><h1>{{ item.title }}</h1><p>{{ item.description }}</p></div>
        <span class="series-status" :class="`is-${item.status?.toLowerCase()}`">{{ statusLabel }}</span>
      </header>

      <section class="detail-layout">
        <div class="art-column">
          <BadgeArtwork :src="item.assetUri" :sha256="item.assetSha256" :alt="`${item.title}图案`" />
          <div class="non-financial" role="note"><strong>纪念凭证</strong><span>不可交易 · 不可兑换 · 无现金价值</span></div>
        </div>

        <div class="detail-column">
          <GlassCard v-if="award" class="ownership-card" :hoverable="false" padding="24px">
            <p class="eyebrow">MY BADGE</p><h2>已领取</h2>
            <div class="serial" aria-label="你的徽章编号">No. {{ String(serial).padStart(4, '0') }} <small>/ {{ item.maxSupply }}</small></div>
            <p>编号只代表链上登记顺序，不代表名次、稀有等级或价值。</p>
          </GlassCard>

          <GlassCard v-else class="claim-card" :hoverable="false" padding="24px">
            <p class="eyebrow">FREE CLAIM</p><h2>领取状态</h2>
            <p v-if="isEligible">你已通过链上活动事实取得领取资格。</p>
            <p v-else>当前账户尚无可领取资格。资格只会由有效活动签到生成。</p>
            <p v-if="!isStudent" class="claim-hint">只有 student 证书可以领取。</p>
            <p v-else-if="item.status === 'PAUSED'" class="claim-hint">系列暂时停领，已取得的资格不会因此消失。</p>
            <p v-else-if="['CLOSED','FINALIZED'].includes(item.status)" class="claim-hint">该系列领取已经结束。</p>
            <p v-else-if="!withinClaimWindow" class="claim-hint">当前不在领取时间窗内。</p>
            <EcButton v-if="canClaim" ref="claimTrigger" class="claim-button" @click="openConfirmation">免费领取</EcButton>
            <div v-if="isBusy" class="pending-box" role="status" aria-live="polite"><span aria-hidden="true">◷</span><p>{{ claimState.message }}</p></div>
            <p v-if="claimState.status === 'error'" class="action-error" role="alert">{{ claimState.message }}</p>
          </GlassCard>

          <GlassCard class="supply-card" :hoverable="false" padding="24px">
            <p class="eyebrow">SUPPLY AUDIT</p><h2>发行记录</h2>
            <div class="supply-values"><div><strong>{{ Number(item.issuedCount || 0).toLocaleString('zh-CN') }}</strong><span>历史已发行</span></div><div><strong>{{ Number(item.maxSupply || 0).toLocaleString('zh-CN') }}</strong><span>固定发行上限</span></div></div>
            <p>该计数用于审计发行是否超过激活时冻结的上限，不表示价格、稀有等级或投资收益。</p>
          </GlassCard>
        </div>
      </section>

      <section class="audit-grid" aria-label="链上验证信息">
        <GlassCard :hoverable="false"><p class="eyebrow">SERIES SOURCE</p><h2>系列来源</h2><dl><div><dt>系列 ID</dt><dd>{{ item.seriesId }}</dd></div><div><dt>赛季</dt><dd>{{ item.seasonId }}</dd></div><div><dt>发行组织</dt><dd>{{ item.issuerOrganization }}</dd></div><div><dt>领取窗口</dt><dd>{{ formatDate(item.claimOpenAt) }} — {{ formatDate(item.claimCloseAt) }}</dd></div></dl></GlassCard>
        <GlassCard :hoverable="false"><p class="eyebrow">CONTENT PROOF</p><h2>内容哈希</h2><dl><div><dt>图案 SHA-256</dt><dd class="hash">{{ item.assetSha256 }}</dd></div><div><dt>元数据 SHA-256</dt><dd class="hash">{{ item.metadataSha256 }}</dd></div><div><dt>资格规则 SHA-256</dt><dd class="hash">{{ item.eligibilityPolicyHash }}</dd></div></dl></GlassCard>
        <GlassCard v-if="instance" :hoverable="false"><p class="eyebrow">INSTANCE PROOF</p><h2>编号验证</h2><dl><div><dt>公开实例 ID</dt><dd class="hash">{{ instance.instanceId }}</dd></div><div><dt>编号</dt><dd>No. {{ String(instance.serialNumber).padStart(4, '0') }} / {{ item.maxSupply }}</dd></div><div><dt>发行批次日期</dt><dd>{{ formatDate(instance.issuanceBatchDate, true) }}</dd></div></dl></GlassCard>
        <GlassCard v-if="linkedActivities.length" :hoverable="false"><p class="eyebrow">ELIGIBILITY</p><h2>关联活动</h2><ul class="activity-links"><li v-for="link in linkedActivities" :key="link.activityId || link.id"><strong>{{ link.title || link.activityId || link.id }}</strong><span>有效签到 · 资格上限 {{ link.eligibilityQuota ?? link.capacity ?? '—' }}</span></li></ul></GlassCard>
      </section>

      <p class="sr-live" aria-live="polite">{{ actionMessage || claimState.message }}</p>
    </template>

    <div v-if="confirmOpen" class="dialog-backdrop" @click.self="closeConfirmation">
      <section ref="confirmDialog" class="claim-dialog" role="dialog" aria-modal="true" aria-labelledby="claim-dialog-title" @keydown="trapDialogFocus">
        <h2 id="claim-dialog-title" ref="confirmTitle" tabindex="-1">确认免费领取</h2>
        <p>领取不会扣除 A 或 B，不影响 P 仓位，也不会公开你的真实身份。徽章不可交易、不可兑换且无现金价值。</p>
        <div class="dialog-actions"><EcButton variant="ghost" @click="closeConfirmation">返回</EcButton><EcButton @click="submitClaim">确认领取</EcButton></div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.badge-detail-page{display:grid;gap:22px;padding:30px}.detail-header{display:grid;grid-template-columns:1fr auto;gap:18px;padding:8px 0 22px;border-bottom:2px solid var(--ec-ink)}
.back-link{grid-column:1/-1;width:max-content;font-size:12px;font-weight:700;text-decoration:none}.detail-header h1{max-width:800px;font-size:clamp(38px,6vw,60px);letter-spacing:-.025em}.detail-header div>p:last-child{max-width:680px;margin-top:12px;color:var(--ec-muted)}
.series-status{align-self:start;padding:7px 10px;border-radius:var(--ec-r-pill);background:rgba(110,97,82,.1);color:var(--ec-muted);font:700 10px var(--ec-font-mono)}.series-status.is-active{background:rgba(72,132,63,.1);color:var(--ec-open)}.series-status.is-paused{background:rgba(154,103,0,.1);color:var(--ec-pending)}
.detail-layout{display:grid;grid-template-columns:minmax(260px,390px) minmax(0,1fr);gap:20px}.art-column,.detail-column{display:grid;align-content:start;gap:14px}.non-financial{display:grid;gap:3px;padding:15px;border-radius:var(--ec-r-field);background:var(--ec-dark);color:var(--ec-cream)}.non-financial strong{font:800 17px var(--ec-font-display);text-transform:uppercase}.non-financial span{color:var(--ec-cream-2);font:700 10px var(--ec-font-mono)}
.ownership-card{background:rgba(72,132,63,.07)}.ownership-card h2,.claim-card h2,.supply-card h2,.audit-grid h2{font-size:27px}.serial{margin:15px 0 7px;font:800 clamp(38px,6vw,58px)/1 var(--ec-font-display)}.serial small{color:var(--ec-faint);font-size:.42em}.ownership-card>p:last-child,.claim-card>p,.supply-card>p{color:var(--ec-muted);font-size:12px}
.claim-button{width:100%;margin-top:18px}.claim-hint{margin-top:10px;padding:10px;border-radius:var(--ec-r-field);background:var(--ec-inset)}.pending-box{display:flex;align-items:flex-start;gap:10px;margin-top:15px;padding:12px;border-radius:var(--ec-r-field);background:rgba(154,103,0,.1);color:var(--ec-pending)}.pending-box span{font-size:20px}.pending-box p{font-size:12px}.action-error{margin-top:12px;color:var(--ec-danger)!important}
.supply-values{display:grid;grid-template-columns:1fr 1fr;gap:9px;margin:16px 0}.supply-values div{padding:13px;border-radius:var(--ec-r-field);background:var(--ec-inset)}.supply-values strong,.supply-values span{display:block}.supply-values strong{font:800 34px var(--ec-font-display)}.supply-values span{color:var(--ec-faint);font:700 9px var(--ec-font-mono);text-transform:uppercase}
.audit-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.audit-grid dl{display:grid;gap:10px;margin-top:17px}.audit-grid dl div{padding-bottom:9px;border-bottom:1px solid var(--ec-line)}.audit-grid dt{color:var(--ec-faint);font:700 9px var(--ec-font-mono);letter-spacing:.06em;text-transform:uppercase}.audit-grid dd{margin-top:3px;font-size:12px;overflow-wrap:anywhere}.hash{font:10px/1.55 var(--ec-font-mono)}.activity-links{display:grid;gap:8px;margin-top:17px;list-style:none}.activity-links li{display:grid;gap:2px;padding:10px;border-radius:var(--ec-r-field);background:var(--ec-inset)}.activity-links strong{font-size:12px}.activity-links span{color:var(--ec-muted);font-size:10px}
.detail-state{display:grid;justify-items:center;gap:13px;padding:72px 20px;text-align:center;color:var(--ec-muted)}.detail-state h1{color:var(--ec-ink);font-size:36px}.detail-state--error{color:var(--ec-danger)}
.dialog-backdrop{position:fixed;z-index:50;inset:0;display:grid;place-items:center;padding:16px;background:rgba(12,8,5,.68)}.claim-dialog{width:min(470px,100%);padding:26px;border:1px solid var(--ec-line);border-radius:var(--ec-r-panel);background:var(--ec-paper);box-shadow:var(--ec-shadow-shell)}.claim-dialog h2{font-size:31px}.claim-dialog p{margin-top:13px;color:var(--ec-muted);font-size:13px}.dialog-actions{display:flex;justify-content:flex-end;gap:9px;margin-top:22px}.sr-live{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap}
@media(max-width:760px){.badge-detail-page{padding:20px 16px 40px}.detail-header{grid-template-columns:1fr}.series-status{justify-self:start}.detail-layout,.audit-grid{grid-template-columns:1fr}.art-column{max-width:420px;width:100%;justify-self:center}.dialog-actions{display:grid;grid-template-columns:1fr 1fr}.dialog-actions :deep(button){width:100%;min-height:44px}}
@media(max-width:390px){.supply-values{grid-template-columns:1fr}.serial{font-size:40px}.claim-dialog{padding:20px}}
@media(prefers-reduced-motion:reduce){.badge-detail-page *{animation:none!important;transition:none!important}}
</style>
