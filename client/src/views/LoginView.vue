<script setup>
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import Logo from '../components/Logo.vue';
import EcButton from '../components/EcButton.vue';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const form = ref({ studentID: '', password: '' });
const submitting = ref(false);
const errorMsg = ref('');

async function handleSubmit() {
  errorMsg.value = '';
  if (!form.value.studentID || !form.value.password) {
    errorMsg.value = '请输入学号和密码';
    return;
  }
  submitting.value = true;

  try {
    await auth.login(form.value.studentID, form.value.password);
    // Redirect to the page they tried to access, or home
    const redirect = route.query.redirect || '/';
    router.push(redirect);
  } catch (err) {
    errorMsg.value = err?.message || '操作失败';
  } finally {
    submitting.value = false;
  }
}

</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="top-rule"></div>
      <div class="login-content">
      <Logo class="login-logo" :size="24" />
      <p class="login-kicker">封闭积分账户</p>
      <h2 class="login-title">
        欢迎回来
      </h2>
      <p class="login-subtitle">
        登录你的封闭积分账户；课程名单由管理员预置。
      </p>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">学号</label>
          <input
            v-model="form.studentID"
            type="text"
            class="form-input"
            placeholder="3220100001"
            autocomplete="username"
          />
        </div>

        <div class="form-group">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            class="form-input"
            placeholder="••••••••"
            autocomplete="current-password"
          />
        </div>

        <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>

        <EcButton type="submit" class="submit-btn" :disabled="submitting">
          {{ submitting ? '处理中...' : '登录' }}
        </EcButton>
      </form>

      <p class="toggle-text">没有账号？请联系课程管理员加入名单。</p></div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 0;
}

.login-card {
  width: 100%;
  max-width: 452px;
  background:var(--ec-paper);
}
.top-rule{height:6px;background:var(--ec-red)}.login-content{padding:40px 40px 34px}.login-logo{display:flex;justify-content:center;margin-bottom:26px}.login-kicker{text-align:center;color:var(--ec-red);font:500 10px var(--ec-font-mono);letter-spacing:.18em;margin-bottom:9px}

.login-title {
  font-size: 34px;
  font-weight: 800;
  text-align: center;
}

.login-subtitle {
  text-align: center;
  color: var(--ec-muted);
  font-size: 13.5px;
  line-height:1.6;
  margin-top: 8px;
  margin-bottom: 28px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font:500 10px var(--ec-font-mono);
  letter-spacing:.14em;
  color:var(--ec-muted);
}

.form-input {
  padding: 10px 14px;
  border: 1px solid var(--ec-line);
  outline: none;
  font-size: 15px;
  color: var(--ec-ink);
  border-radius:var(--ec-r-field);
  background:#fff;
}

.form-input::placeholder {
  color: var(--color-text-tertiary);
}

.form-input:focus {
  border-color:var(--ec-red);
}

.error-text {
  color: var(--color-danger);
  font-size: 13px;
  text-align: center;
}

.submit-btn{width:100%;margin-top:4px}

.toggle-text {
  text-align: center;
  margin-top: 20px;
  font-size:12px;
  color:var(--ec-faint);
}

@media(max-width:520px){.login-content{padding:34px 24px 30px}}
</style>
