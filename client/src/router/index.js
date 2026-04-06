import { createRouter, createWebHistory } from 'vue-router';

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/event/:id',
    name: 'EventDetail',
    component: () => import('../views/EventDetailView.vue'),
    props: true,
  },
  {
    path: '/tickets',
    name: 'TicketHall',
    component: () => import('../views/TicketHallView.vue'),
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
    meta: { requiresAuth: true, requiresRole: ['organizer', 'admin'] },
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
  const user = JSON.parse(localStorage.getItem('ec_user') || 'null');

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
