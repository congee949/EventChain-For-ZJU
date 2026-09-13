<script setup>
import { computed, onMounted } from 'vue';
import GlassCard from '../components/GlassCard.vue';
import { useAuthStore } from '../stores/auth.js';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useActivityV2Store } from '../stores/activityV2.js';
import { applicationStatusLabel, roleLabel } from '../utils/display.js';
import SeasonBadgeCard from '../components/SeasonBadgeCard.vue';
import { useBadgesStore } from '../stores/badges.js';
import { isVisibleBadgeSeries, isVisibleActivity } from '../utils/badgeVisibility.js';

const auth = useAuthStore();
const finance = useFinanceStore();
const activities = useActivityV2Store();
const badges = useBadgesStore();
onMounted(async () => { await Promise.all([finance.refresh(), activities.refresh(), badges.loadAwards().catch(() => [])]); await activities.hydrateMine(); });
const categoryRows = computed(() => finance.categories.map((category) => ({ ...category, wallet: finance.wallet?.categories?.[category.id] })));
const myApplications = computed(() => activities.activities.filter((item) => isVisibleActivity(item) && activities.applications[item.id]).map((item) => ({ activity: item, application: activities.applications[item.id] })));
const ticketCount = computed(() => Object.values(activities.tickets).filter(Boolean).length);
const paidTotal = computed(() => categoryRows.value.reduce((sum, item) => sum + Number(item.wallet?.paid?.available || 0), 0));
const bonusTotal = computed(() => categoryRows.value.reduce((sum, item) => sum + Number(item.wallet?.bonus?.available || 0), 0));
const shortAccount = computed(() => finance.wallet?.accountId ? `${finance.wallet.accountId.slice(0, 12)}…${finance.wallet.accountId.slice(-8)}` : '—');
const visibleAwards = computed(() => badges.awards.filter((entry) => isVisibleBadgeSeries(entry.series)));
</script>

<template>
  <div class="v2-page profile-page">
    <section class="profile-hero"><div class="avatar">{{ auth.user?.name?.slice(0,1) || '用' }}</div><div><p>私有账户</p><h1>{{ auth.user?.name }}</h1><span>{{ roleLabel(auth.user?.role) }} · {{ shortAccount }}</span></div><span class="privacy-badge">学号未写入链上</span></section>
    <div class="profile-grid">
      <GlassCard class="balance-card" padding="28px" :hoverable="false">
        <div class="card-title"><div><p>资产概览</p><h2>我的积分组合</h2></div><router-link to="/wallet">管理钱包 →</router-link></div>
        <div class="stats-row"><div><strong>{{ finance.aBalance }}</strong><span>A 可用</span></div><div><strong>{{ formatAmount(paidTotal) }}</strong><span>B_paid</span></div><div><strong>{{ formatAmount(bonusTotal) }}</strong><span>B_bonus</span></div><div><strong>{{ ticketCount }}</strong><span>有效票据</span></div></div>
      </GlassCard>

      <GlassCard class="category-card" padding="24px" :hoverable="false">
        <div class="card-title"><div><p>分类钱包</p><h2>分类余额</h2></div></div>
        <div class="category-list"><div v-for="row in categoryRows" :key="row.id"><span><b>{{ row.name }}</b></span><span><b>{{ formatAmount(row.wallet?.paid?.available) }}</b><small>B_paid</small></span><span><b>{{ formatAmount(row.wallet?.bonus?.available) }}</b><small>B_bonus</small></span></div></div>
      </GlassCard>

      <GlassCard class="privacy-card" padding="24px" :hoverable="false">
        <div class="card-title"><div><p>隐私边界</p><h2>哪些信息不会公开</h2></div></div>
        <ul><li><b>真实身份：</b>链上使用随机匿名账户编号，登录学号加密保存在服务端。</li><li><b>精确仓位：</b>结果选择和投入金额位于平台与学生组织的私有集合。</li><li><b>资金追踪：</b>市场公开快照经过五分钟延迟和取整，不提供个人排行榜。</li><li><b>现实边界：</b>组织级隐私并非零知识；授权节点管理员仍属于信任边界。</li></ul>
      </GlassCard>

      <GlassCard class="applications-card" padding="24px" :hoverable="false">
        <div class="card-title"><div><p>我的活动</p><h2>票务申请</h2></div><router-link to="/tickets">票务大厅 →</router-link></div>
        <div class="application-list"><div v-for="entry in myApplications" :key="entry.activity.id"><span><b>{{ entry.activity.title }}</b><small>{{ new Date(entry.activity.startsAt).toLocaleString('zh-CN') }}</small></span><em :class="entry.application.status.toLowerCase()">{{ applicationStatusLabel(entry.application.status) }}</em></div><p v-if="!myApplications.length">暂无申请记录</p></div>
      </GlassCard>

      <GlassCard class="applications-card" padding="24px" :hoverable="false">
        <div class="card-title"><div><p>赛季纪念</p><h2>我的赛季徽章</h2></div><router-link to="/badges">查看全部系列 →</router-link></div>
        <div v-if="visibleAwards.length" class="my-badges"><SeasonBadgeCard v-for="entry in visibleAwards" :key="entry.series?.seriesId || entry.award?.instanceId" :series="entry.series" /></div>
        <p v-else class="empty-badges">尚未领取赛季徽章。完成指定活动签到后，可在系列详情中免费领取。</p>
      </GlassCard>

      <GlassCard class="rules-card" padding="24px" :hoverable="false">
        <div class="card-title"><div><p>生命周期</p><h2>积分生命周期</h2></div></div>
        <div class="rule-grid"><div><b>365 天</b><span>B_paid 到期后按储备退回 A</span></div><div><b>90 天</b><span>B_bonus 到期销毁，不可兑回</span></div><div><b>24 小时</b><span>新收到 B_paid 的转让冷却期</span></div><div><b>7 天</b><span>预测收益成熟等待期</span></div></div>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.profile-page{padding-bottom:48px}.profile-hero{display:flex;align-items:center;gap:18px;padding:18px 0 30px;border-bottom:2px solid var(--ec-ink);margin-bottom:22px}.avatar{display:grid;place-items:center;width:72px;height:72px;border-left:5px solid var(--ec-red);border-radius:var(--ec-r-card);background:var(--ec-inset);color:var(--ec-red);font:800 34px var(--ec-font-display)}.profile-hero p,.card-title p{color:var(--ec-red);font:700 10px var(--ec-font-mono);letter-spacing:.12em}.profile-hero h1{font-size:38px}.profile-hero div>span{color:var(--ec-faint);font:500 10px var(--ec-font-mono)}.privacy-badge{margin-left:auto;padding:7px 12px;border-radius:var(--ec-r-pill);background:rgba(36,138,91,.1);color:var(--ec-open);font:700 10px var(--ec-font-mono)}.profile-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}.balance-card{grid-column:1/-1}.card-title{display:flex;justify-content:space-between;align-items:start}.card-title h2{margin-top:4px}.card-title a{color:var(--ec-red);text-decoration:none;font-size:12px;font-weight:700}.stats-row{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-top:24px}.stats-row div,.category-list>div,.application-list>div,.rule-grid div{padding:15px;border-radius:var(--ec-r-field);background:var(--ec-inset)}.stats-row strong,.stats-row span{display:block}.stats-row strong{font:800 30px var(--ec-font-display)}.stats-row span{color:var(--ec-faint);font:500 9px var(--ec-font-mono);margin-top:5px}.category-list,.application-list{display:grid;gap:8px;margin-top:18px}.category-list>div{display:grid;grid-template-columns:1fr auto auto;gap:22px;padding:11px}.category-list span b,.category-list span small,.application-list span b,.application-list span small{display:block}.category-list small,.application-list small{color:var(--ec-faint);font:500 9px var(--ec-font-mono);margin-top:3px}.privacy-card{background:var(--ec-inset);color:var(--ec-ink);border-color:var(--ec-line)}.privacy-card .card-title h2{color:var(--ec-ink)}.privacy-card ul{display:grid;gap:11px;margin:18px 0 0;padding-left:18px}.privacy-card li{color:var(--ec-muted);font-size:12px;line-height:1.55}.applications-card,.rules-card{grid-column:1/-1}.application-list>div{display:flex;align-items:center;justify-content:space-between;padding:11px}.application-list em{font:700 10px var(--ec-font-mono);font-style:normal}.application-list em.pending{color:var(--ec-pending)}.application-list em.won{color:var(--ec-open)}.application-list em.lost{color:var(--ec-faint)}.application-list>p{color:var(--ec-faint);text-align:center}.rule-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px;margin-top:18px}.rule-grid b,.rule-grid span{display:block}.rule-grid b{color:var(--ec-red);font:800 22px var(--ec-font-display)}.rule-grid span{color:var(--ec-muted);font-size:11px;line-height:1.5;margin-top:5px}@media(max-width:760px){.profile-page{padding-bottom:40px}.profile-grid{grid-template-columns:1fr}.privacy-card,.category-card{grid-column:1}.stats-row,.rule-grid{grid-template-columns:1fr 1fr}.privacy-badge{display:none}}@media(max-width:480px){.stats-row,.rule-grid{grid-template-columns:1fr}.category-list>div{grid-template-columns:1fr 1fr}.category-list>div>span:first-child{grid-column:1/-1}}
.my-badges{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin-top:18px}.empty-badges{margin-top:18px;padding:24px;border:1px dashed var(--ec-line);border-radius:var(--ec-r-field);color:var(--ec-muted);text-align:center}@media(max-width:760px){.my-badges{grid-template-columns:1fr}}
</style>
