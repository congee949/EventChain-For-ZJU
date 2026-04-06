<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { useUserStore } from '../stores/user.js';
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const router = useRouter();
const auth = useAuthStore();
const userStore = useUserStore();
const { elementRef } = useGlassHighlight();

const isLoggedIn = computed(() => auth.isLoggedIn);
const displayName = computed(() => auth.user?.name || auth.user?.userId || '');
const balance = computed(() => userStore.balance);

function handleLogout() {
  auth.logout();
  router.push('/login');
}

const navLinks = [
  { label: '首页', path: '/' },
  { label: '票务大厅', path: '/tickets' },
  { label: '我的', path: '/me' },
];
</script>

<template>
  <nav ref="elementRef" class="glass-dense navbar">
    <div class="navbar-inner">
      <router-link to="/" class="brand">
        <span class="brand-icon">&#9830;</span>
        <span class="brand-text">EventChain</span>
      </router-link>

      <div class="nav-links">
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          class="nav-link"
          active-class="nav-link--active"
        >
          {{ link.label }}
        </router-link>

        <router-link
          v-if="auth.user?.role === 'organizer' || auth.user?.role === 'admin'"
          to="/admin"
          class="nav-link"
          active-class="nav-link--active"
        >
          管理
        </router-link>
      </div>

      <div class="nav-right">
        <template v-if="isLoggedIn">
          <div class="balance-badge glass-subtle">
            <span class="balance-icon">&#9733;</span>
            <span class="balance-amount">{{ balance }}</span>
          </div>
          <div class="user-info">
            <span class="user-name">{{ displayName }}</span>
            <button class="logout-btn" @click="handleLogout">退出</button>
          </div>
        </template>
        <template v-else>
          <router-link to="/login" class="login-btn">登录</router-link>
        </template>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  position: fixed;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  width: calc(100% - 48px);
  max-width: 1200px;
  z-index: 1000;
  padding: 0 24px;
  border-radius: var(--radius-lg);
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--color-text);
  font-weight: 700;
  font-size: 18px;
}

.brand-icon {
  font-size: 22px;
  color: var(--color-primary);
}

.nav-links {
  display: flex;
  gap: 8px;
}

.nav-link {
  text-decoration: none;
  color: var(--color-text-secondary);
  padding: 6px 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 500;
  transition: all var(--transition-base);
}

.nav-link:hover {
  color: var(--color-text);
  background: rgba(255, 255, 255, 0.2);
}

.nav-link--active {
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.balance-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-warning);
}

.balance-icon {
  font-size: 16px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text);
}

.logout-btn {
  background: none;
  border: none;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  margin-left: 4px;
  border-radius: 6px;
  transition: all var(--transition-base);
}

.logout-btn:hover {
  color: var(--color-danger);
  background: rgba(239, 68, 68, 0.08);
}

.login-btn {
  text-decoration: none;
  padding: 6px 20px;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  transition: background var(--transition-base);
}

.login-btn:hover {
  background: var(--color-primary-dark);
}
</style>
