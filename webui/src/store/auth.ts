import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import router from '@/router'

const TOKEN_KEY = 'arx_auth_token'
const ROLES_KEY = 'arx_auth_roles'

function readToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

function readRoles(): string[] {
  const raw = localStorage.getItem(ROLES_KEY)
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw) as unknown
    return Array.isArray(parsed) ? parsed.filter((r): r is string => typeof r === 'string') : []
  } catch {
    return []
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readToken())
  const roles = ref<string[]>(readRoles())

  const isAuthenticated = computed(() => token.value !== null && token.value.length > 0)

  function setSession(newToken: string, newRoles: string[] = []): void {
    token.value = newToken
    roles.value = newRoles
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(ROLES_KEY, JSON.stringify(newRoles))
  }

  async function logout(): Promise<void> {
    token.value = null
    roles.value = []
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(ROLES_KEY)
    await router.push({ name: 'login' })
  }

  return { token, roles, isAuthenticated, setSession, logout }
})
