import { request } from './client'
import type { HealthReport } from '@/types/api'

export function fetchHealth(): Promise<HealthReport> {
  return request<HealthReport>('/health')
}
