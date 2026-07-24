import { request } from './client'
import type {
  CreateServiceAccountRequest,
  LoginRequest,
  LoginResponse,
  ServiceAccountResponse,
} from '@/types/api'

export function login(payload: LoginRequest): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/login', { method: 'POST', body: payload })
}

export function logoutSession(): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/logout', { method: 'POST' })
}

export function createServiceAccount(
  payload: CreateServiceAccountRequest,
): Promise<ServiceAccountResponse> {
  return request<ServiceAccountResponse>('/auth/service-accounts', {
    method: 'POST',
    body: payload,
  })
}
