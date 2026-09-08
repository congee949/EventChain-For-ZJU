import axios from 'axios';
import { ElMessage } from 'element-plus';

export function readableApiError(rawMessage) {
  const message = String(rawMessage || '').trim();
  if (/UNAVAILABLE|No connection established|ECONNREFUSED|failed to connect/i.test(message)) {
    return '区块链网络暂不可用，请确认 Fabric 节点已经启动';
  }
  if (/DEADLINE_EXCEEDED|deadline exceeded|timeout/i.test(message)) {
    return '链上确认超时，请稍后重试并先检查操作结果';
  }
  if (/ENDORSEMENT_POLICY_FAILURE|failed to collect enough transaction endorsements/i.test(message)) {
    return '链上背书未通过，请确认相关组织节点均已启动';
  }
  return message || '请求失败';
}

function showError(message) {
  ElMessage({ type: 'error', message, grouping: true });
}

const api = axios.create({
  baseURL: '/api/v2',
  timeout: 60000,
  headers: { 'Content-Type': 'application/json' },
});

// ---- Request interceptor: attach JWT ----
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ec_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// ---- Response interceptor: unwrap data, handle errors ----
api.interceptors.response.use(
  (response) => {
    // Successful responses have { error: false, data: ... }
    return response.data?.data !== undefined ? response.data.data : response.data;
  },
  (error) => {
    const resp = error.response;
    if (resp) {
      const body = resp.data;
      const message = readableApiError(body?.message);

      // Auto-logout on 401 only if user was logged in
      if (resp.status === 401 && localStorage.getItem('ec_token')) {
        localStorage.removeItem('ec_token');
        localStorage.removeItem('ec_user');
        window.location.href = '/login';
      }

      if (!error.config?.silent) showError(message);
      return Promise.reject(body);
    }

    if (!error.config?.silent) showError('无法连接后端服务，请确认服务端已经启动');
    return Promise.reject(error);
  }
);

export default api;

export function idempotencyHeaders(prefix = 'op') {
  const random = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return { 'Idempotency-Key': `${prefix}:${random}` };
}
