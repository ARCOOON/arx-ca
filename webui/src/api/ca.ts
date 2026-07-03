import type { ApiEnvelope, CAInfoResponse, CAProvisionersResponse } from '@/types/api'
import { apiClient } from './client'

export async function fetchCAInfo(): Promise<CAInfoResponse> {
  const response = await apiClient.get<ApiEnvelope<CAInfoResponse>>('/ca/info')
  const payload = response.data
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error('CA info response did not include data')
  return payload.data
}

export async function fetchCAProvisioners(): Promise<CAProvisionersResponse> {
  const response = await apiClient.get<ApiEnvelope<CAProvisionersResponse>>('/ca/provisioners')
  const payload = response.data
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error('CA provisioners response did not include data')
  return payload.data
}

export async function fetchCARootPEM(): Promise<string> {
  const response = await apiClient.get<string>('/ca/root', {
    responseType: 'text',
    headers: { Accept: 'application/x-pem-file' },
  })
  return response.data
}
