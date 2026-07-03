import type { AcmeEabKeyResponse, AcmeStatus, ApiEnvelope, CreateAcmeEabKeyRequest } from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchAcmeStatus(): Promise<AcmeStatus> {
  const response = await apiClient.get<ApiEnvelope<AcmeStatus>>('/acme/status')
  return unwrap(response.data, 'ACME status')
}

export async function createAcmeEabKey(req: CreateAcmeEabKeyRequest): Promise<AcmeEabKeyResponse> {
  const response = await apiClient.post<ApiEnvelope<AcmeEabKeyResponse>>('/acme/eab-keys', req)
  return unwrap(response.data, 'ACME EAB key')
}
