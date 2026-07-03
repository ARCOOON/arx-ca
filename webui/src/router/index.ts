import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true },
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
          path: 'acme',
          name: 'acme',
          component: () => import('@/views/Acme.vue'),
          meta: { title: 'ACME', subtitle: 'Automated certificate enrollment protocol' },
        },
        {
          path: 'scep',
          name: 'scep',
          component: () => import('@/views/Scep.vue'),
          meta: { title: 'SCEP', subtitle: 'Simple Certificate Enrollment Protocol' },
        },
        {
          path: 'ndes',
          name: 'ndes',
          component: () => import('@/views/Ndes.vue'),
          meta: { title: 'NDES', subtitle: 'Network Device Enrollment Service' },
        },
        {
          path: 'provisioners',
          name: 'provisioners',
          component: () => import('@/views/Provisioners.vue'),
          meta: { title: 'Provisioners', subtitle: 'Provisioner tokens and enrollment status' },
        },
        {
          path: 'templates',
          name: 'templates',
          component: () => import('@/views/Templates.vue'),
          meta: { title: 'Templates', subtitle: 'Certificate issuance template management' },
        },
        {
          path: 'ssh',
          name: 'ssh',
          component: () => import('@/views/Ssh.vue'),
          meta: { title: 'SSH CA', subtitle: 'SSH user and host certificate operations' },
        },
        {
          path: 'audit',
          name: 'audit',
          component: () => import('@/views/Audit.vue'),
          meta: { title: 'Audit Log', subtitle: 'Immutable forensic trail of critical API operations' },
        },
        {
          path: 'webhooks',
          name: 'webhooks',
          component: () => import('@/views/Webhooks.vue'),
          meta: { title: 'Webhooks', subtitle: 'Outbound notifications for high-criticality audit events' },
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
  const isPublic = to.meta.public === true

  if (!isPublic && !authStore.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && authStore.isAuthenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
