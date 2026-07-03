import type { ApiEnvelope, NdesStatus } from '@/types/api'
import { apiClient } from './client'

export async function fetchNdesStatus(): Promise<NdesStatus> {
  const response = await apiClient.get<ApiEnvelope<NdesStatus>>('/ndes/status')
  const payload = response.data
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error('NDES status response did not include data')
  return payload.data
}
