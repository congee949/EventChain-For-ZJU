<script setup>
import { onMounted, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import GlassCard from '../components/GlassCard.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useAuthStore } from '../stores/auth.js';
import CategoryWallet from '../components/CategoryWallet.vue';
import { roleLabel } from '../utils/display.js';

const finance = useFinanceStore();
const auth = useAuthStore();
const state = reactive({ categoryId: '', amount: 10, toAccountId: '' });
onMounted(async () => { await finance.refresh(); state.categoryId = finance.categories[0]?.id || ''; });
async function act(fn, message) { await fn(); ElMessage.success(message); }
</script>

<template>
  <div class="v2-page">
    <header class="v2-hero"><p class="eyebrow">私有积分钱包</p><h1>A / B 钱包</h1><p>A 是封闭积分；B_paid 有 A 储备并可兑回，B_bonus 来自签到/补偿且不可兑回。</p></header>
    <GlassCard :hoverable="false">
      <div class="balance-a"><span>A 可用</span><strong>{{ finance.aBalance }}</strong><small>冻结 {{ formatAmount(finance.wallet?.aReserved) }}</small></div>
    </GlassCard>
    <div class="v2-grid">
      <CategoryWallet v-for="category in finance.categories" :key="category.id" :sport="category.name" :code="category.id" :paid="formatAmount(finance.wallet?.categories?.[category.id]?.paid?.available)" :paid-locked="formatAmount(finance.wallet?.categories?.[category.id]?.paid?.locked)" :bonus="formatAmount(finance.wallet?.categories?.[category.id]?.bonus?.available)" :bonus-locked="formatAmount(finance.wallet?.categories?.[category.id]?.bonus?.locked)" />
    </div>
    <GlassCard :hoverable="false">
      <h2>兑换与转让</h2>
      <div v-if="auth.user?.role === 'student'" class="v2-actions wrap">
        <el-select v-model="state.categoryId" placeholder="运动类别"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id" /></el-select>
        <el-input-number v-model="state.amount" :min="0.000001" :precision="6" />
        <el-button type="primary" @click="act(() => finance.convert(state.categoryId, state.amount), 'A 已兑换为 B_paid')">A → B_paid</el-button>
        <el-button @click="act(() => finance.redeem(state.categoryId, state.amount), 'B_paid 已扣除 0.5% 手续费并兑回 A')">B_paid → A</el-button>
        <el-input v-model="state.toAccountId" placeholder="收款方匿名账户编号" />
        <el-button @click="act(() => finance.transfer(state.categoryId, state.toAccountId, state.amount), '转让已提交')">转让 B_paid</el-button>
      </div>
      <p v-else class="muted">当前角色为{{ roleLabel(auth.user?.role) }}；兑换、兑回和转让仅对学生积分账户开放。</p>
      <p class="muted">默认：每日 500 B_paid、最多 5 个收款方、转入后冷却 24 小时；B_paid 365 天到期自动退 A，B_bonus 90 天到期销毁。</p>
    </GlassCard>
  </div>
</template>
