import { request } from './client'
import type { UpdaterChangelogResponse } from '@/types/api'

export function fetchCurrentChangelog(): Promise<UpdaterChangelogResponse> {
  return request<UpdaterChangelogResponse>('/updater/current-changelog')
}
