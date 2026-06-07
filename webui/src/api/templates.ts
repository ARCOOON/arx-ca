import type {
  ApiEnvelope,
  CreateCertificateTemplateRequest,
  CertificateTemplate,
  ListCertificateTemplatesResponse,
} from '../types/api'
import { apiClient } from './client'

export async function listTemplates(): Promise<ListCertificateTemplatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListCertificateTemplatesResponse>>('/templates')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Template list response did not include data')
  }

  return payload.data
}

export async function createTemplate(
  request: CreateCertificateTemplateRequest,
): Promise<CertificateTemplate> {
  const response = await apiClient.post<ApiEnvelope<CertificateTemplate>>('/templates', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Template creation response did not include data')
  }

  return payload.data
}
