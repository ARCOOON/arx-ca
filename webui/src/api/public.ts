import type { ApiEnvelope, PublicListCertificatesResponse } from '../types/api'
import { apiClient } from './client'

export async function listPublicCertificates(): Promise<PublicListCertificatesResponse> {
  const response = await apiClient.get<ApiEnvelope<PublicListCertificatesResponse>>(
    '/public/certificates',
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Public certificate list response did not include data')
  }

  return payload.data
}

export async function fetchPublicCertificatePem(serial: string): Promise<string> {
  const encoded = encodeURIComponent(serial)
  const response = await apiClient.get<ApiEnvelope<{ certificate_pem: string }>>(
    `/public/certificates/${encoded}`,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data?.certificate_pem) {
    throw new Error('Public certificate response did not include PEM data')
  }

  return payload.data.certificate_pem
}
