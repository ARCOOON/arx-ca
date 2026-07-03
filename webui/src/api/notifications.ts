import type {
  ApiEnvelope,
  ArchiveAllNotificationsResponse,
  ListNotificationsResponse,
  MarkAllNotificationsReadResponse,
} from '@/types/api'
import { apiClient } from './client'

export interface ListNotificationsParams {
  limit?: number
  offset?: number
  unread?: boolean
}

export async function listNotifications(
  params: ListNotificationsParams = {},
): Promise<ListNotificationsResponse> {
  const response = await apiClient.get<ApiEnvelope<ListNotificationsResponse>>('/notifications', {
    params: {
      limit: params.limit ?? 50,
      offset: params.offset ?? 0,
      ...(params.unread ? { unread: 'true' } : {}),
    },
  })
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Notification response did not include data')
  }

  return payload.data
}

export async function markNotificationRead(id: string): Promise<void> {
  const response = await apiClient.post<ApiEnvelope<{ id: string; status: string }>>(
    `/notifications/${encodeURIComponent(id)}/read`,
  )
  if (response.data.error) {
    throw new Error(response.data.error)
  }
}

export async function markAllNotificationsRead(): Promise<MarkAllNotificationsReadResponse> {
  const response = await apiClient.post<ApiEnvelope<MarkAllNotificationsReadResponse>>(
    '/notifications/read-all',
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Mark all read response did not include data')
  }

  return payload.data
}

export async function deleteNotification(id: string): Promise<void> {
  const response = await apiClient.delete<ApiEnvelope<{ id: string; status: string }>>(
    `/notifications/${encodeURIComponent(id)}`,
  )
  if (response.data.error) {
    throw new Error(response.data.error)
  }
}

export async function archiveAllNotifications(): Promise<ArchiveAllNotificationsResponse> {
  const response = await apiClient.post<ApiEnvelope<ArchiveAllNotificationsResponse>>(
    '/notifications/archive-all',
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Archive all notifications response did not include data')
  }

  return payload.data
}
