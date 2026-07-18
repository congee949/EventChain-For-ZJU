<script setup>
import { onMounted, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import GlassCard from '../components/GlassCard.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useAuthStore } from '../stores/auth.js';

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
  <div class="v2-page"><header class="v2-hero"><p class="eyebrow">B_paid UTILITY</p><h1>器材、场馆与活动兑换</h1><p>服务只收 B_paid。组织者必须预存取消赔付保证金和延期 B_bonus 预算。</p></header>
    <div class="v2-grid">
      <GlassCard v-for="offer in finance.offers" :key="offer.offerId" :hoverable="false">
        <div class="v2-row"><span class="v2-tag">{{ offer.offerType }}</span><span>余量 {{ offer.inventory }}</span></div>
        <h2>{{ offer.title }}</h2><div class="price">{{ formatAmount(offer.pricePaidB) }} <small>B_paid</small></div>
        <p class="muted">取消赔付 {{ (offer.cancellationBps / 10000).toFixed(2) }}× · 保证金余 {{ formatAmount(offer.guaranteeAvailableA) }} A</p>
        <div v-if="auth.user?.role === 'student'" class="v2-actions"><el-date-picker v-model="schedules[offer.offerId]" type="datetime" placeholder="预约时间" /><el-button type="primary" @click="order(offer)">兑换</el-button></div><p v-else class="muted">当前角色为 {{ auth.user?.role }}；只有学生账户可以兑换服务。</p>
      </GlassCard>
    </div><el-empty v-if="!finance.loading && !finance.offers.length" description="尚无服务" />
  </div>
</template>
