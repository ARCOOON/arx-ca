import axios from 'axios'
import { useAuthStore } from '../store/auth'

const DEFAULT_API_BASE_URL = 'https://localhost:8443/api/v1'

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL,
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
