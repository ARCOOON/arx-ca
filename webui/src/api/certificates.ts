import type {
  ApiEnvelope,
  IssueCertificateRequest,
  IssueCertificateResponse,
  ListCertificatesResponse,
} from '../types/api'
import { apiClient } from './client'

export async function listCertificates(): Promise<ListCertificatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListCertificatesResponse>>('/certificates')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate list response did not include data')
  }

  return payload.data
}

export async function issueCertificate(
  request: IssueCertificateRequest,
): Promise<IssueCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<IssueCertificateResponse>>(
    '/certificates/issue',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate issue response did not include data')
  }

  return payload.data
}
