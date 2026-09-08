import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../api/index.js';
import { readStoredJSON } from '../utils/display.js';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(null);
  const user = ref(null); // { userId, name, role }

  const isLoggedIn = computed(() => !!token.value);

  // Restore session from localStorage on app init
  function restoreSession() {
    const savedToken = localStorage.getItem('ec_token');
    const savedUser = readStoredJSON('ec_user');
    if (savedToken && savedUser) {
      token.value = savedToken;
      user.value = savedUser;
    } else if (savedToken) {
      localStorage.removeItem('ec_token');
    }
  }

  function setSession(jwt, userData) {
    token.value = jwt;
    user.value = userData;
    localStorage.setItem('ec_token', jwt);
    localStorage.setItem('ec_user', JSON.stringify(userData));
  }

  async function register(studentID, password, name) {
    const data = await api.post('/auth/register', { studentID, password, name });
    setSession(data.token, {
      userId: data.userId,
      name: data.name || name,
      role: 'student',
    });
    return data;
  }

  async function login(studentID, password) {
    const data = await api.post('/auth/login', { studentID, password });
    setSession(data.token, {
      userId: data.userId,
      name: data.name,
      role: data.role,
    });
    return data;
  }

  function logout() {
    token.value = null;
    user.value = null;
    localStorage.removeItem('ec_token');
    localStorage.removeItem('ec_user');
  }

  return { token, user, isLoggedIn, restoreSession, register, login, logout };
});
