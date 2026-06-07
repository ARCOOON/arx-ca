import type {
  ApiEnvelope,
  CreateServiceAccountRequest,
  LoginRequest,
  LoginResponse,
  ServiceAccountResponse,
} from '../types/api'
import { apiClient } from './client'

export async function login(credentials: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post<ApiEnvelope<LoginResponse>>('/auth/login', credentials)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Login response did not include session data')
  }

  return payload.data
}

export async function createServiceAccount(
  request: CreateServiceAccountRequest,
): Promise<ServiceAccountResponse> {
  const response = await apiClient.post<ApiEnvelope<ServiceAccountResponse>>(
    '/auth/service-accounts',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Service account response did not include data')
  }

  return payload.data
}
