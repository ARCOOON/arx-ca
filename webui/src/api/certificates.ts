import type {
  ApiEnvelope,
  CertificateRecordDetail,
  CertificateStatsResponse,
  GenerateCertificateRequest,
  GenerateCertificateResponse,
  IssueCertificateRequest,
  IssueCertificateResponse,
  LintCertificateRequest,
  LintCertificateResponse,
  ListCertificatesResponse,
  RevokeCertificateRequest,
  RevokeCertificateResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchCertificates(params?: {
  limit?: number
  offset?: number
  serial?: string
}): Promise<ListCertificatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListCertificatesResponse>>('/certificates', { params })
  return unwrap(response.data, 'List certificates')
}

export async function fetchCertificate(serial: string): Promise<CertificateRecordDetail> {
  const response = await apiClient.get<ApiEnvelope<CertificateRecordDetail>>(`/certificates/${serial}`)
  return unwrap(response.data, 'Certificate detail')
}

export async function fetchCertificateStats(): Promise<CertificateStatsResponse> {
  const response = await apiClient.get<ApiEnvelope<CertificateStatsResponse>>('/certificates/stats')
  return unwrap(response.data, 'Certificate stats')
}

export async function issueCertificate(req: IssueCertificateRequest): Promise<IssueCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<IssueCertificateResponse>>('/certificates/issue', req)
  return unwrap(response.data, 'Issue certificate')
}

export async function generateCertificate(req: GenerateCertificateRequest): Promise<GenerateCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<GenerateCertificateResponse>>('/certificates/generate', req)
  return unwrap(response.data, 'Generate certificate')
}

export async function revokeCertificate(req: RevokeCertificateRequest): Promise<RevokeCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<RevokeCertificateResponse>>('/certificates/revoke', req)
  return unwrap(response.data, 'Revoke certificate')
}

export async function lintCertificate(req: LintCertificateRequest): Promise<LintCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<LintCertificateResponse>>('/certificates/lint', req)
  return unwrap(response.data, 'Lint certificate')
}
