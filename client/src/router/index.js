import { createRouter, createWebHistory } from 'vue-router';
import { readStoredJSON } from '../utils/display.js';

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/event/:id',
    name: 'EventDetail',
    component: () => import('../views/EventDetailView.vue'),
    meta: { requiresAuth: true },
    props: true,
  },
  {
    path: '/tickets',
    name: 'TicketHall',
    component: () => import('../views/TicketHallView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/services',
    name: 'Services',
    component: () => import('../views/ServicesV2View.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/activities',
    redirect: '/tickets',
  },
  {
    path: '/wallet',
    name: 'Wallet',
    component: () => import('../views/WalletV2View.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/me',
    name: 'Profile',
    component: () => import('../views/ProfileView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/AdminView.vue'),
    meta: { requiresAuth: true, requiresRole: ['organizer', 'admin', 'operator', 'verifier', 'arbitrator'] },
  },
  {
    path: '/check-in',
    name: 'CheckIn',
    component: () => import('../views/CheckInView.vue'),
    meta: { requiresAuth: true, requiresRole: ['operator', 'admin'] },
  },
  {
    path: '/operations',
    redirect: '/admin',
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// Navigation guard
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('ec_token');
  const user = readStoredJSON('ec_user');

  if (to.meta.requiresAuth && !token) {
    return next({ name: 'Login', query: { redirect: to.fullPath } });
  }

  if (to.meta.requiresRole && user) {
    const allowed = to.meta.requiresRole;
    if (!allowed.includes(user.role)) {
      return next({ name: 'Home' });
    }
  }

  next();
});

export default router;
