import type {
  ApiEnvelope,
  K8sProvisionerStatus,
  ProvisionerTokenRequest,
  ProvisionerTokenResponse,
} from '../types/api'
import { apiClient } from './client'

export async function fetchK8sStatus(): Promise<K8sProvisionerStatus> {
  const response = await apiClient.get<ApiEnvelope<K8sProvisionerStatus>>('/k8s/status')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Kubernetes provisioner status response did not include data')
  }

  return payload.data
}

export async function createProvisionerToken(
  request: ProvisionerTokenRequest,
): Promise<ProvisionerTokenResponse> {
  const response = await apiClient.post<ApiEnvelope<ProvisionerTokenResponse>>(
    '/provisioners/token',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Provisioner token response did not include data')
  }

  return payload.data
}
