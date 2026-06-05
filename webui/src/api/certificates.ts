import type {
  ApiEnvelope,
  CertificatePrivateKeyResponse,
  CertificateRecordDetail,
  GenerateCertificateRequest,
  GenerateCertificateResponse,
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

export async function generateCertificate(
  request: GenerateCertificateRequest,
): Promise<GenerateCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<GenerateCertificateResponse>>(
    '/certificates/generate',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate generation response did not include data')
  }

  return payload.data
}

export async function generateCertificateBundleFile(
  request: GenerateCertificateRequest,
  filename: string,
): Promise<void> {
  const response = await apiClient.post<ArrayBuffer>('/certificates/generate?format=zip', request, {
    headers: { Accept: 'application/zip' },
    responseType: 'arraybuffer',
  })

  const blob = new Blob([response.data], { type: 'application/zip' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

export async function fetchCertificateBySerial(serial: string): Promise<CertificateRecordDetail> {
  const encoded = encodeURIComponent(serial)
  const response = await apiClient.get<ApiEnvelope<CertificateRecordDetail>>(`/certificates/${encoded}`)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate detail response did not include data')
  }

  return payload.data
}

export async function fetchCertificatePrivateKey(serial: string): Promise<CertificatePrivateKeyResponse> {
  const encoded = encodeURIComponent(serial)
  const response = await apiClient.get<ApiEnvelope<CertificatePrivateKeyResponse>>(
    `/certificates/${encoded}/key`,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data?.private_key_pem) {
    throw new Error('Private key response did not include key material')
  }

  return payload.data
}

export async function downloadCertificateBundleFile(serial: string, filename: string): Promise<void> {
  const encoded = encodeURIComponent(serial)
  const response = await apiClient.get<ArrayBuffer>(`/certificates/${encoded}/bundle`, {
    responseType: 'arraybuffer',
  })

  const blob = new Blob([response.data], { type: 'application/zip' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
