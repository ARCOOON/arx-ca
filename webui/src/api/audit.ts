import type { ApiEnvelope, ListAuditLogsResponse } from '../types/api'
import { apiClient } from './client'

export interface ListAuditLogsParams {
  limit?: number
  offset?: number
  action?: string
  actor?: string
  ip?: string
  status?: number
}

export async function listAuditLogs(
  params: ListAuditLogsParams = {},
): Promise<ListAuditLogsResponse> {
  const query: Record<string, string | number> = {
    limit: params.limit ?? 50,
    offset: params.offset ?? 0,
  }

  if (params.action?.trim()) {
    query.action = params.action.trim()
  }
  if (params.actor?.trim()) {
    query.actor = params.actor.trim()
  }
  if (params.ip?.trim()) {
    query.ip = params.ip.trim()
  }
  if (params.status != null && params.status > 0) {
    query.status = params.status
  }

  const response = await apiClient.get<ApiEnvelope<ListAuditLogsResponse>>('/audit', {
    params: query,
  })
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Audit log response did not include data')
  }

  return payload.data
}
