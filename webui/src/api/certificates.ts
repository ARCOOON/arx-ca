import type {
  ApiEnvelope,
  AutoCertificateRequest,
  AutoCertificateResponse,
  CertificatePrivateKeyResponse,
  CertificateRecordDetail,
  CertificateStatsResponse,
  GenerateCertificateRequest,
  GenerateCertificateResponse,
  IssueCertificateRequest,
  IssueCertificateResponse,
  IssueCertificateWithTokenRequest,
  LintCertificateRequest,
  LintCertificateResponse,
  ListCertificatesResponse,
  RekeyCertificateRequest,
  RenewCertificateRequest,
  RevokeCertificateRequest,
  RevokeCertificateResponse,
} from '../types/api'
import { apiClient } from './client'

export interface ListCertificatesParams {
  common_name?: string
  serial_number?: string
  status?: 'valid' | 'revoked' | 'expired' | ''
}

export async function listCertificates(
  params: ListCertificatesParams = {},
): Promise<ListCertificatesResponse> {
  const query: Record<string, string> = {}

  if (params.common_name?.trim()) {
    query.common_name = params.common_name.trim()
  }
  if (params.serial_number?.trim()) {
    query.serial_number = params.serial_number.trim()
  }
  if (params.status?.trim()) {
    query.status = params.status.trim()
  }

  const response = await apiClient.get<ApiEnvelope<ListCertificatesResponse>>('/certificates', {
    params: query,
  })
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

export async function revokeCertificate(
  request: RevokeCertificateRequest,
): Promise<RevokeCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<RevokeCertificateResponse>>(
    '/certificates/revoke',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate revocation response did not include data')
  }

  return payload.data
}

export async function lintCertificate(
  request: LintCertificateRequest,
): Promise<LintCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<LintCertificateResponse>>(
    '/certificates/lint',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate lint response did not include data')
  }

  return payload.data
}

export async function renewCertificate(
  request: RenewCertificateRequest,
): Promise<IssueCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<IssueCertificateResponse>>(
    '/certificates/renew',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate renewal response did not include data')
  }

  return payload.data
}

export async function rekeyCertificate(
  request: RekeyCertificateRequest,
): Promise<IssueCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<IssueCertificateResponse>>(
    '/certificates/rekey',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate rekey response did not include data')
  }

  return payload.data
}

export async function issueCertificateWithToken(
  request: IssueCertificateWithTokenRequest,
): Promise<IssueCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<IssueCertificateResponse>>(
    '/certificates/issue-with-token',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate issue-with-token response did not include data')
  }

  return payload.data
}

export async function autoCertificate(
  request: AutoCertificateRequest,
): Promise<AutoCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<AutoCertificateResponse>>(
    '/certificates/auto',
    request,
  )
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Auto certificate response did not include data')
  }

  return payload.data
}

export async function fetchCertificateStats(): Promise<CertificateStatsResponse> {
  const response = await apiClient.get<ApiEnvelope<CertificateStatsResponse>>('/certificates/stats')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('Certificate stats response did not include data')
  }

  return payload.data
}
