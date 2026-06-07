import type {
  ApiEnvelope,
  CreateWebhookRequest,
  ListWebhooksResponse,
  UpdateWebhookRequest,
  WebhookEventsResponse,
  WebhookResponse,
  WebhookTestResponse,
} from '../types/api'
import { apiClient } from './client'

export async function listWebhooks(): Promise<ListWebhooksResponse> {
  const response = await apiClient.get<ApiEnvelope<ListWebhooksResponse>>('/webhooks')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Webhook list response did not include data')
  }
  return payload.data
}

export async function listWebhookEvents(): Promise<WebhookEventsResponse> {
  const response = await apiClient.get<ApiEnvelope<WebhookEventsResponse>>('/webhooks/events')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Webhook events response did not include data')
  }
  return payload.data
}

export async function createWebhook(body: CreateWebhookRequest): Promise<WebhookResponse> {
  const response = await apiClient.post<ApiEnvelope<WebhookResponse>>('/webhooks', body)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Webhook creation response did not include data')
  }
  return payload.data
}

export async function updateWebhook(id: string, body: UpdateWebhookRequest): Promise<WebhookResponse> {
  const response = await apiClient.put<ApiEnvelope<WebhookResponse>>(`/webhooks/${id}`, body)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Webhook update response did not include data')
  }
  return payload.data
}

export async function deleteWebhook(id: string): Promise<void> {
  const response = await apiClient.delete<ApiEnvelope<{ deleted: string }>>(`/webhooks/${id}`)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
}

export async function testWebhook(id: string): Promise<WebhookTestResponse> {
  const response = await apiClient.post<ApiEnvelope<WebhookTestResponse>>(`/webhooks/${id}/test`)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Webhook test response did not include data')
  }
  return payload.data
}
