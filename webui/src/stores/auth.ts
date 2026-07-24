import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { logoutSession } from '@/api/auth'

const TOKEN_KEY = 'arx_auth_token'
const ROLES_KEY = 'arx_auth_roles'

function readStoredToken(): string | null {
  if (typeof localStorage === 'undefined') {
    return null
  }
  return localStorage.getItem(TOKEN_KEY)
}

function readStoredRoles(): string[] {
  if (typeof localStorage === 'undefined') {
    return []
  }
  try {
    const raw = localStorage.getItem(ROLES_KEY)
    return raw ? (JSON.parse(raw) as string[]) : []
  } catch {
    return []
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readStoredToken())
  const roles = ref<string[]>(readStoredRoles())

  const isAuthenticated = computed(() => token.value !== null && token.value.length > 0)

  function hasRole(...candidates: string[]): boolean {
    return candidates.some((role) => roles.value.includes(role))
  }

  function setSession(newToken: string, newRoles: string[] = []): void {
    token.value = newToken
    roles.value = newRoles
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(ROLES_KEY, JSON.stringify(newRoles))
  }

  function clearSession(): void {
    token.value = null
    roles.value = []
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(ROLES_KEY)
  }

  async function logout(): Promise<void> {
    try {
      await logoutSession()
    } catch {
      // Best-effort server-side session invalidation; always clear locally.
    }
    clearSession()

    const { default: router } = await import('@/router')
    if (router.currentRoute.value.name !== 'login') {
      await router.push({ name: 'login' })
    }
  }

  return { token, roles, isAuthenticated, hasRole, setSession, clearSession, logout }
})
