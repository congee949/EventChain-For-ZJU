import axios from 'axios';
import { ElMessage } from 'element-plus';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
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
      const message = body?.message || '请求失败';

      // Auto-logout on 401 only if user was logged in
      if (resp.status === 401 && localStorage.getItem('ec_token')) {
        localStorage.removeItem('ec_token');
        localStorage.removeItem('ec_user');
        window.location.href = '/login';
      }

      ElMessage.error(message);
      return Promise.reject(body);
    }

    ElMessage.error('网络连接失败，请稍后重试');
    return Promise.reject(error);
  }
);

export default api;
