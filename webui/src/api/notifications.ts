import { request, type QueryParams } from './client'
import type {
  ArchiveAllNotificationsResponse,
  ListNotificationsResponse,
  MarkAllNotificationsReadResponse,
} from '@/types/api'

export interface ListNotificationsParams {
  limit?: number
  offset?: number
  unread?: boolean
}

export function listNotifications(
  params: ListNotificationsParams = {},
): Promise<ListNotificationsResponse> {
  return request<ListNotificationsResponse>('/notifications', { query: params as QueryParams })
}

export function markNotificationRead(id: string): Promise<{ id: string; status: string }> {
  return request<{ id: string; status: string }>(
    `/notifications/${encodeURIComponent(id)}/read`,
    { method: 'POST' },
  )
}

export function markAllNotificationsRead(): Promise<MarkAllNotificationsReadResponse> {
  return request<MarkAllNotificationsReadResponse>('/notifications/read-all', {
    method: 'POST',
  })
}

export function deleteNotification(id: string): Promise<{ id: string; status: string }> {
  return request<{ id: string; status: string }>(`/notifications/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function archiveAllNotifications(): Promise<ArchiveAllNotificationsResponse> {
  return request<ArchiveAllNotificationsResponse>('/notifications/archive-all', {
    method: 'POST',
  })
}
