import type { ApiEnvelope, HealthReport } from '@/types/api'
import { apiClient } from './client'

export async function fetchHealth(): Promise<HealthReport> {
  const response = await apiClient.get<ApiEnvelope<HealthReport>>('/health')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Health response did not include data')
  }

  return payload.data
}
