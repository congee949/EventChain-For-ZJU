<script setup>
import { ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const mode = ref('login'); // 'login' | 'register'
const form = ref({ studentID: '', password: '', name: '' });
const submitting = ref(false);
const errorMsg = ref('');

const buttonLabel = computed(() => (mode.value === 'login' ? '登录' : '注册'));

async function handleSubmit() {
  errorMsg.value = '';
  submitting.value = true;

  try {
    if (mode.value === 'register') {
      if (!form.value.name) {
        errorMsg.value = '请输入姓名';
        return;
      }
      await auth.register(form.value.studentID, form.value.password, form.value.name);
    } else {
      await auth.login(form.value.studentID, form.value.password);
    }
    // Redirect to the page they tried to access, or home
    const redirect = route.query.redirect || '/';
    router.push(redirect);
  } catch (err) {
    errorMsg.value = err?.message || '操作失败';
  } finally {
    submitting.value = false;
  }
}

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login';
  errorMsg.value = '';
}
</script>

<template>
  <div class="login-page">
    <GlassCard class="login-card" padding="40px">
      <h2 class="login-title">
        {{ mode === 'login' ? '欢迎回来' : '加入 EventChain' }}
      </h2>
      <p class="login-subtitle">
        {{ mode === 'login' ? '登录你的账号' : '注册即送 1000 浙币' }}
      </p>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">学号</label>
          <input
            v-model="form.studentID"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入学号"
            required
          />
        </div>

        <div v-if="mode === 'register'" class="form-group">
          <label class="form-label">姓名</label>
          <input
            v-model="form.name"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入姓名"
          />
        </div>

        <div class="form-group">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            class="form-input glass-subtle"
            placeholder="请输入密码"
            required
          />
        </div>

        <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>

        <button
          type="submit"
          class="submit-btn"
          :disabled="submitting"
        >
          {{ submitting ? '处理中...' : buttonLabel }}
        </button>
      </form>

      <p class="toggle-text">
        {{ mode === 'login' ? '还没有账号？' : '已有账号？' }}
        <a class="toggle-link" @click.prevent="toggleMode">
          {{ mode === 'login' ? '立即注册' : '去登录' }}
        </a>
      </p>
    </GlassCard>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: calc(100vh - 120px);
}

.login-card {
  width: 100%;
  max-width: 420px;
}

.login-title {
  font-size: 24px;
  font-weight: 800;
  text-align: center;
}

.login-subtitle {
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 14px;
  margin-top: 4px;
  margin-bottom: 28px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 10px 14px;
  border: none;
  outline: none;
  font-size: 15px;
  color: var(--color-text);
  border-radius: var(--radius-sm);
}

.form-input::placeholder {
  color: var(--color-text-tertiary);
}

.form-input:focus {
  box-shadow: 0 0 0 2px var(--color-primary-light);
}

.error-text {
  color: var(--color-danger);
  font-size: 13px;
  text-align: center;
}

.submit-btn {
  margin-top: 8px;
  padding: 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.submit-btn:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-text {
  text-align: center;
  margin-top: 20px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.toggle-link {
  color: var(--color-primary);
  font-weight: 600;
  cursor: pointer;
}

.toggle-link:hover {
  text-decoration: underline;
}
</style>
