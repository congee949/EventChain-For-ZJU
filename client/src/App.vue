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
.app-frame{width:min(1180px,calc(100% - 32px));min-height:calc(100vh - 56px);margin:28px auto;background:var(--ec-paper);border-radius:var(--ec-r-shell);box-shadow:var(--ec-shadow-shell);overflow:hidden;position:relative}
.app-frame--login{width:min(452px,calc(100% - 32px));min-height:0;margin-top:68px;border-radius:var(--ec-r-panel)}

.main-content {
  width:100%;
}
@media(max-width:700px){.app-frame{width:100%;margin:0;min-height:100vh;border-radius:0}.app-frame--login{width:calc(100% - 24px);margin:24px auto;border-radius:var(--ec-r-panel)}}
</style>
