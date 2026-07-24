import { request } from './client'
import type { SettingsConfigResponse } from '@/types/api'

export function fetchSettingsConfig(): Promise<SettingsConfigResponse> {
  return request<SettingsConfigResponse>('/settings/config')
}

export function updateSettingsConfig(
  patch: Partial<SettingsConfigResponse>,
): Promise<SettingsConfigResponse> {
  return request<SettingsConfigResponse>('/settings/config', {
    method: 'PUT',
    body: patch,
  })
}
