import type {
  ApiEnvelope,
  CreateServiceAccountRequest,
  K8sProvisionerStatus,
  ProvisionerTokenRequest,
  ProvisionerTokenResponse,
  ServiceAccountResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchK8sStatus(): Promise<K8sProvisionerStatus> {
  const response = await apiClient.get<ApiEnvelope<K8sProvisionerStatus>>('/k8s/status')
  return unwrap(response.data, 'K8s status')
}

export async function generateProvisionerToken(req: ProvisionerTokenRequest): Promise<ProvisionerTokenResponse> {
  const response = await apiClient.post<ApiEnvelope<ProvisionerTokenResponse>>('/provisioners/token', req)
  return unwrap(response.data, 'Provisioner token')
}

export async function createServiceAccount(req: CreateServiceAccountRequest): Promise<ServiceAccountResponse> {
  const response = await apiClient.post<ApiEnvelope<ServiceAccountResponse>>('/auth/service-accounts', req)
  return unwrap(response.data, 'Service account')
}
