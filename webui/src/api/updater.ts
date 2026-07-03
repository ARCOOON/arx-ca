import type { ApiEnvelope, UpdaterChangelogResponse } from '@/types/api'
import { apiClient } from './client'

export async function fetchCurrentChangelog(): Promise<UpdaterChangelogResponse> {
  const response = await apiClient.get<ApiEnvelope<UpdaterChangelogResponse>>('/updater/current-changelog')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }
  if (!payload.data) {
    throw new Error('Changelog response did not include data')
  }
  return payload.data
}
