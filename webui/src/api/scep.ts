import type { ApiEnvelope, ScepStatus } from '../types/api'
import { apiClient } from './client'

export async function fetchScepStatus(): Promise<ScepStatus> {
  const response = await apiClient.get<ApiEnvelope<ScepStatus>>('/scep/status')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SCEP status response did not include data')
  }

  return payload.data
}
