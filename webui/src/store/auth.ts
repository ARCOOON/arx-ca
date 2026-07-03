import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import router from '@/router'

const TOKEN_STORAGE_KEY = 'arx_auth_token'
const ROLES_STORAGE_KEY = 'arx_auth_roles'

function readStoredToken(): string | null {
  return localStorage.getItem(TOKEN_STORAGE_KEY)
}

function readStoredRoles(): string[] {
  const raw = localStorage.getItem(ROLES_STORAGE_KEY)
  if (!raw) {
    return []
  }

  try {
    const parsed = JSON.parse(raw) as unknown
    return Array.isArray(parsed) ? parsed.filter((role): role is string => typeof role === 'string') : []
  } catch {
    return []
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readStoredToken())
  const roles = ref<string[]>(readStoredRoles())

  const isAuthenticated = computed(() => token.value !== null && token.value.length > 0)

  function setSession(newToken: string, newRoles: string[] = []): void {
    token.value = newToken
    roles.value = newRoles
    localStorage.setItem(TOKEN_STORAGE_KEY, newToken)
    localStorage.setItem(ROLES_STORAGE_KEY, JSON.stringify(newRoles))
  }

  async function logout(): Promise<void> {
    token.value = null
    roles.value = []
    localStorage.removeItem(TOKEN_STORAGE_KEY)
    localStorage.removeItem(ROLES_STORAGE_KEY)

    await router.push({ name: 'login' })
  }

  return {
    token,
    roles,
    isAuthenticated,
    setSession,
    logout,
  }
})
