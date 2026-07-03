import type {
  ApiEnvelope,
  CertificateTemplate,
  CreateCertificateTemplateRequest,
  ListCertificateTemplatesResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchTemplates(): Promise<ListCertificateTemplatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListCertificateTemplatesResponse>>('/templates')
  return unwrap(response.data, 'Templates')
}

export async function createTemplate(req: CreateCertificateTemplateRequest): Promise<CertificateTemplate> {
  const response = await apiClient.post<ApiEnvelope<CertificateTemplate>>('/templates', req)
  return unwrap(response.data, 'Create template')
}
