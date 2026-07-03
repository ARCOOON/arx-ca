import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true, title: 'Login' },
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppShell.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/Dashboard.vue'),
          meta: { title: 'Dashboard', subtitle: 'General statistics and server status' },
        },
        {
          path: 'certificates',
          name: 'certificates',
          component: () => import('@/views/Certificates.vue'),
          meta: { title: 'Certificates', subtitle: 'Issued certificate inventory and CSR signing' },
        },
        {
          path: 'ssh',
          name: 'ssh',
          component: () => import('@/views/Ssh.vue'),
          meta: { title: 'SSH CA', subtitle: 'SSH user and host certificate operations' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/Settings.vue'),
          meta: { title: 'Settings', subtitle: 'Session, API client, and interface preferences' },
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const authStore = useAuthStore()
  const isPublicRoute = to.meta.public === true
  const authenticated = authStore.isAuthenticated

  if (!isPublicRoute && !authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.name === 'login' && authenticated) {
    return { name: 'dashboard' }
  }

  return true
})

export default router
