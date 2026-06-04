import { apiClient } from './client'
import type { ApiEnvelope, CAInfoResponse } from '../types/api'

export async function fetchCAInfo(): Promise<CAInfoResponse> {
  const { data } = await apiClient.get<ApiEnvelope<CAInfoResponse>>('/ca/info')
  if (data.error || !data.data) {
    throw new Error(data.error ?? 'Failed to load CA certificate information')
  }
  return data.data
}
