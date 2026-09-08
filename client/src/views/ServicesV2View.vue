<script setup>
import { onMounted, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import GlassCard from '../components/GlassCard.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useAuthStore } from '../stores/auth.js';
import { offerTypeLabel, roleLabel } from '../utils/display.js';

const finance = useFinanceStore();
const auth = useAuthStore();
const schedules = reactive({});
onMounted(finance.refresh);
async function order(offer) {
  const value = schedules[offer.offerId];
  if (!value) return ElMessage.warning('请选择预约时间');
  await finance.order(offer.offerId, new Date(value).toISOString());
  ElMessage.success('订单已创建，取消保证金已预先锁定');
}
</script>

<template>
  <div class="v2-page"><header class="v2-hero"><p class="eyebrow">B_paid 服务兑换</p><h1>器材、场馆与活动兑换</h1><p>服务只收 B_paid。组织者必须预存取消赔付保证金和延期 B_bonus 预算。</p></header>
    <section class="service-section">
      <div class="service-heading"><div><p>当前可兑换</p><h2>选择一项服务</h2></div><span>订单和取消规则由链码统一校验</span></div>
      <div class="service-grid">
      <GlassCard v-for="offer in finance.offers" :key="offer.offerId" class="service-card" :hoverable="false">
        <div class="v2-row"><span class="v2-tag">{{ offerTypeLabel(offer.offerType) }}</span><span>余量 {{ offer.inventory }}</span></div>
        <h2>{{ offer.title }}</h2><div class="price">{{ formatAmount(offer.pricePaidB) }} <small>B_paid</small></div>
        <p class="muted">取消赔付 {{ (offer.cancellationBps / 10000).toFixed(2) }}× · 保证金余 {{ formatAmount(offer.guaranteeAvailableA) }} A</p>
        <div v-if="auth.user?.role === 'student'" class="v2-actions"><el-date-picker v-model="schedules[offer.offerId]" type="datetime" placeholder="预约时间" /><el-button type="primary" @click="order(offer)">兑换</el-button></div><p v-else class="muted">当前角色为{{ roleLabel(auth.user?.role) }}；只有学生账户可以兑换服务。</p>
      </GlassCard>
      </div>
      <el-empty v-if="!finance.loading && !finance.offers.length" description="尚无服务" />
    </section>
  </div>
</template>

<style scoped>
.service-section{display:grid;gap:16px}.service-heading{display:flex;align-items:end;justify-content:space-between;padding-bottom:10px;border-bottom:2px solid var(--ec-ink)}.service-heading p{color:var(--ec-red);font:700 10px var(--ec-font-mono);letter-spacing:.12em;margin-bottom:4px}.service-heading h2{font-size:24px}.service-heading span{color:var(--ec-faint);font:500 10px var(--ec-font-mono)}.service-grid{display:grid;grid-template-columns:1fr;gap:14px}.service-card{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,280px);column-gap:32px;align-items:center}.service-card .v2-row,.service-card>h2,.service-card>.price,.service-card>.muted{grid-column:1}.service-card .v2-row{grid-row:1}.service-card>h2{grid-row:2}.service-card>.price{grid-row:3}.service-card>.muted{grid-row:4}.service-card .v2-actions{grid-column:2;grid-row:1 / span 4;align-self:center}.service-card>p.muted:last-child{grid-column:2;grid-row:1 / span 4;align-self:center;margin:0}.service-card .v2-actions .el-date-editor{min-width:0;flex:1}.service-card .v2-actions .el-button{flex:0 0 auto}@media(max-width:720px){.service-heading{display:block}.service-heading span{display:block;margin-top:7px}.service-card{display:block}.service-card .v2-actions{margin-top:18px}.service-card>p.muted:last-child{margin-top:18px}}
</style>
