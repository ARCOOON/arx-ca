import type { ApiEnvelope, LoginRequest, LoginResponse } from '@/types/api'
import { apiClient } from './client'

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post<ApiEnvelope<LoginResponse>>('/auth/login', req)
  const payload = response.data
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error('Login response did not include data')
  return payload.data
}

export async function logout(): Promise<void> {
  await apiClient.post('/auth/logout').catch(() => undefined)
}
