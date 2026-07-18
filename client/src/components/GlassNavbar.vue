<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { useFinanceStore } from '../stores/finance.js';
import Logo from './Logo.vue';
import LedgerStrip from './LedgerStrip.vue';

const router=useRouter(); const auth=useAuthStore(); const finance=useFinanceStore();
const displayName=computed(()=>auth.user?.name||auth.user?.userId||'');
const navLinks=[{label:'赛事市场',path:'/'},{label:'票务大厅',path:'/tickets'},{label:'服务兑换',path:'/services'},{label:'A / B 钱包',path:'/wallet'},{label:'我的',path:'/me'}];
function handleLogout(){auth.logout();router.push('/login')}
</script>
<template>
  <template v-if="auth.isLoggedIn">
    <nav class="masthead">
      <router-link to="/" class="brand"><Logo :size="22" /></router-link>
      <div class="nav-links">
        <router-link v-for="link in navLinks" :key="link.path" :to="link.path" class="nav-link" active-class="nav-link--active">{{ link.label }}</router-link>
        <router-link v-if="auth.user?.role && auth.user.role!=='student'" to="/admin" class="nav-link" active-class="nav-link--active">管理</router-link>
      </div>
      <div class="nav-right"><b>A {{ finance.aBalance }}</b><span>{{ displayName }}</span><button @click="handleLogout">退出</button></div>
    </nav>
    <LedgerStrip />
  </template>
</template>
<style scoped>
.masthead{height:60px;display:grid;grid-template-columns:200px 1fr 210px;align-items:center;padding:0 30px;border-bottom:2px solid var(--ec-ink);background:var(--ec-paper)}.brand{display:flex;text-decoration:none}.nav-links{display:flex;justify-content:center;gap:22px;min-width:0}.nav-link{color:var(--ec-muted);text-decoration:none;font:600 15px var(--ec-font-display);text-transform:uppercase;white-space:nowrap}.nav-link:hover,.nav-link--active{color:var(--ec-red)}.nav-right{display:flex;align-items:center;justify-content:flex-end;gap:14px;font:500 11px var(--ec-font-mono);white-space:nowrap}.nav-right b{color:var(--ec-red);font-size:12px}.nav-right span{color:var(--ec-muted)}.nav-right button{border:0;background:transparent;color:var(--ec-faint-2);font-size:10px;cursor:pointer}.nav-right button:hover{color:var(--ec-danger)}
@media(max-width:900px){.masthead{grid-template-columns:auto 1fr auto;padding:0 16px}.nav-links{justify-content:flex-start;overflow-x:auto;gap:13px;margin:0 14px}.nav-right span{display:none}}
@media(max-width:620px){.masthead{height:auto;min-height:58px;grid-template-columns:auto 1fr}.brand :deep(.logo){font-size:18px!important}.nav-links{grid-column:1/-1;grid-row:2;padding:8px 0 10px;margin:0}.nav-right{grid-column:2;grid-row:1}.nav-link{font-size:13px}}
</style>
