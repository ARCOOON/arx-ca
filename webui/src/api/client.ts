import axios from 'axios'
import { useAuthStore } from '../store/auth'

const DEFAULT_API_BASE_URL = 'https://localhost:8443/api/v1'

/**
 * Resolves the REST API base URL for the Vue app.
 * - Explicit VITE_API_BASE_URL wins (build-time override).
 * - In the browser, defaults to same-origin /api/v1 (drop-in WebUI + API proxy).
 * - Falls back to DEFAULT_API_BASE_URL for non-browser contexts.
 */
export function resolveApiBaseURL(): string {
  const envBase = import.meta.env.VITE_API_BASE_URL
  if (typeof envBase === 'string' && envBase.trim() !== '') {
    return envBase.trim().replace(/\/$/, '')
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}/api/v1`
  }

  return DEFAULT_API_BASE_URL
}

export const apiClient = axios.create({
  baseURL: resolveApiBaseURL(),
  headers: {
    Accept: 'application/json',
    'Content-Type': 'application/json',
  },
  timeout: 30_000,
})

apiClient.interceptors.request.use((config) => {
  const authStore = useAuthStore()
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status
    const requestUrl = error.config?.url ?? ''

    if (status === 401 && !requestUrl.includes('/auth/login')) {
      const authStore = useAuthStore()
      await authStore.logout()
    }

    return Promise.reject(error)
  },
)
