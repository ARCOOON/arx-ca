import type {
  ApiEnvelope,
  ArchiveAllNotificationsResponse,
  ListNotificationsResponse,
  MarkAllNotificationsReadResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchNotifications(params?: {
  limit?: number
  offset?: number
}): Promise<ListNotificationsResponse> {
  const response = await apiClient.get<ApiEnvelope<ListNotificationsResponse>>('/notifications', { params })
  return unwrap(response.data, 'Notifications')
}

export async function markAllRead(): Promise<MarkAllNotificationsReadResponse> {
  const response = await apiClient.post<ApiEnvelope<MarkAllNotificationsReadResponse>>('/notifications/read-all')
  return unwrap(response.data, 'Mark all read')
}

export async function archiveAll(): Promise<ArchiveAllNotificationsResponse> {
  const response = await apiClient.post<ApiEnvelope<ArchiveAllNotificationsResponse>>('/notifications/archive-all')
  return unwrap(response.data, 'Archive all')
}

export async function markOneRead(id: string): Promise<void> {
  await apiClient.post(`/notifications/${id}/read`)
}

export async function deleteNotification(id: string): Promise<void> {
  await apiClient.delete(`/notifications/${id}`)
}

export function createNotificationStream(onMessage: (data: string) => void): EventSource {
  const base = apiClient.defaults.baseURL ?? ''
  const es = new EventSource(`${base}/notifications/stream`, { withCredentials: true })
  es.onmessage = (e) => onMessage(e.data as string)
  return es
}
