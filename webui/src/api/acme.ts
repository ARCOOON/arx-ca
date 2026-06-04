import type { AcmeStatus, ApiEnvelope } from '../types/api'
import { apiClient } from './client'

export async function fetchAcmeStatus(): Promise<AcmeStatus> {
  const response = await apiClient.get<ApiEnvelope<AcmeStatus>>('/acme/status')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('ACME status response did not include data')
  }

  return payload.data
}
