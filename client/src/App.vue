<script setup>
import GlassNavbar from './components/GlassNavbar.vue';
import { useAuthStore } from './stores/auth.js';
import { useFinanceStore } from './stores/finance.js';

const auth = useAuthStore();
const finance = useFinanceStore();
auth.restoreSession();
if (auth.isLoggedIn) finance.refreshWallet().catch(() => {});
</script>

<template>
  <div class="app-frame" :class="{ 'app-frame--login': !auth.isLoggedIn }">
    <GlassNavbar />
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-frame{width:100%;min-height:100vh;background:var(--ec-paper);overflow:hidden;position:relative}
.app-frame--login{width:min(452px,calc(100% - 32px));min-height:0;margin:clamp(24px,8vh,68px) auto 24px;border:1px solid var(--ec-line);border-radius:var(--ec-r-panel);box-shadow:var(--ec-shadow-shell)}

.main-content {
  width:100%;
}
@media(max-width:700px){.app-frame--login{width:calc(100% - 24px);margin:24px auto;border-radius:var(--ec-r-panel)}}
</style>
