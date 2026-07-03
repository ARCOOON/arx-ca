import type { ApiEnvelope, SettingsConfigResponse } from '@/types/api'
import { apiClient } from './client'

export async function fetchSettingsConfig(): Promise<SettingsConfigResponse> {
  const response = await apiClient.get<ApiEnvelope<SettingsConfigResponse>>('/settings/config')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Settings config response did not include data')
  }
  return payload.data
}

export async function updateSettingsConfig(
  body: Partial<SettingsConfigResponse>,
): Promise<SettingsConfigResponse> {
  const response = await apiClient.put<ApiEnvelope<SettingsConfigResponse>>('/settings/config', body)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Settings config update response did not include data')
  }
  return payload.data
}
