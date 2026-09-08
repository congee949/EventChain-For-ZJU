<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { useFinanceStore } from '../stores/finance.js';
import Logo from './Logo.vue';
import LedgerStrip from './LedgerStrip.vue';
import { roleLabel } from '../utils/display.js';

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
        <router-link v-if="['operator','admin'].includes(auth.user?.role)" to="/check-in" class="nav-link" active-class="nav-link--active">签到核验</router-link>
        <router-link v-if="auth.user?.role && auth.user.role!=='student'" to="/admin" class="nav-link" active-class="nav-link--active">运营台</router-link>
      </div>
      <div class="nav-right"><b>A {{ finance.aBalance }}</b><span>{{ displayName }} · {{ roleLabel(auth.user?.role) }}</span><button @click="handleLogout">退出</button></div>
    </nav>
    <LedgerStrip />
  </template>
</template>
<style scoped>
.masthead{position:relative;min-height:60px;display:flex;align-items:center;justify-content:center;padding:0 clamp(16px,3vw,40px);border-bottom:1px solid var(--ec-line);background:var(--ec-paper)}.brand{position:absolute;left:clamp(16px,3vw,40px);display:flex;text-decoration:none}.nav-links{display:flex;justify-content:center;gap:clamp(14px,2vw,26px);min-width:0;max-width:calc(100vw - 260px);height:100%;align-items:center}.nav-link{position:relative;color:var(--ec-muted);text-decoration:none;font:700 14px var(--ec-font-display);white-space:nowrap}.nav-link:hover,.nav-link--active{color:var(--ec-red)}.nav-link--active::after{content:'';position:absolute;left:0;right:0;bottom:-19px;height:3px;background:var(--ec-red)}.nav-right{position:absolute;right:clamp(16px,3vw,40px);display:flex;align-items:center;justify-content:flex-end;gap:clamp(8px,1.5vw,14px);font:500 10px var(--ec-font-mono);white-space:nowrap}.nav-right b{padding:4px 8px;border-radius:var(--ec-r-chip);background:rgba(0,113,227,.08);color:var(--ec-red);font-size:12px}.nav-right span{color:var(--ec-muted)}.nav-right button{border:0;background:transparent;color:var(--ec-faint-2);font-size:10px;cursor:pointer}.nav-right button:hover{color:var(--ec-danger)}
@media(max-width:900px){.nav-links{justify-content:flex-start;overflow-x:auto;gap:13px}.nav-right span{display:none}}
@media(max-width:620px){.masthead{height:auto;min-height:58px;display:grid;grid-template-columns:auto 1fr;padding:0 16px}.brand{position:static}.nav-links{position:static;grid-column:1/-1;grid-row:2;max-width:none;justify-content:flex-start;padding:8px 0 10px;margin:0}.nav-right{position:static;grid-column:2;grid-row:1}.brand :deep(.logo){font-size:18px!important}.nav-link{font-size:13px}.nav-link--active::after{bottom:-10px}}
</style>
