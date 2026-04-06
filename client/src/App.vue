<script setup>
import GlassNavbar from './components/GlassNavbar.vue';
import { useAuthStore } from './stores/auth.js';

const auth = useAuthStore();
auth.restoreSession();
</script>

<template>
  <div class="app-shell">
    <!-- Floating background orbs -->
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="orb orb-3"></div>

    <GlassNavbar />
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
}

.main-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 88px 24px 48px;
}
</style>
