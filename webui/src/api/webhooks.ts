import type {
  ApiEnvelope,
  CreateWebhookRequest,
  ListWebhooksResponse,
  UpdateWebhookRequest,
  WebhookEventsResponse,
  WebhookResponse,
  WebhookTestResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchWebhooks(): Promise<ListWebhooksResponse> {
  const response = await apiClient.get<ApiEnvelope<ListWebhooksResponse>>('/webhooks')
  return unwrap(response.data, 'Webhooks')
}

export async function fetchWebhookEvents(): Promise<WebhookEventsResponse> {
  const response = await apiClient.get<ApiEnvelope<WebhookEventsResponse>>('/webhooks/events')
  return unwrap(response.data, 'Webhook events')
}

export async function createWebhook(req: CreateWebhookRequest): Promise<WebhookResponse> {
  const response = await apiClient.post<ApiEnvelope<WebhookResponse>>('/webhooks', req)
  return unwrap(response.data, 'Create webhook')
}

export async function updateWebhook(id: string, req: UpdateWebhookRequest): Promise<WebhookResponse> {
  const response = await apiClient.put<ApiEnvelope<WebhookResponse>>(`/webhooks/${id}`, req)
  return unwrap(response.data, 'Update webhook')
}

export async function deleteWebhook(id: string): Promise<void> {
  await apiClient.delete(`/webhooks/${id}`)
}

export async function testWebhook(id: string): Promise<WebhookTestResponse> {
  const response = await apiClient.post<ApiEnvelope<WebhookTestResponse>>(`/webhooks/${id}/test`)
  return unwrap(response.data, 'Test webhook')
}
