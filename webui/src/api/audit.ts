import type { ApiEnvelope, ListAuditLogsResponse } from '@/types/api'
import { apiClient } from './client'

export async function fetchAuditLogs(params?: {
  limit?: number
  offset?: number
  actor_id?: string
  action?: string
}): Promise<ListAuditLogsResponse> {
  const response = await apiClient.get<ApiEnvelope<ListAuditLogsResponse>>('/audit', { params })
  const payload = response.data
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error('Audit logs response did not include data')
  return payload.data
}
