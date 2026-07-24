import { request, requestBlob, type QueryParams } from './client'
import type {
  AutoCertificateRequest,
  AutoCertificateResponse,
  CertificateRecordDetail,
  CertificateStatsResponse,
  IssueCertificateRequest,
  IssueCertificateResponse,
  IssueCertificateWithTokenRequest,
  ListCertificatesResponse,
  RevokeCertificateRequest,
  RevokeCertificateResponse,
} from '@/types/api'

export interface ListCertificatesParams {
  common_name?: string
  serial_number?: string
  status?: 'valid' | 'revoked' | 'expired'
}

export function listCertificates(
  params: ListCertificatesParams = {},
): Promise<ListCertificatesResponse> {
  return request<ListCertificatesResponse>('/certificates', { query: params as QueryParams })
}

export function fetchCertificateStats(): Promise<CertificateStatsResponse> {
  return request<CertificateStatsResponse>('/certificates/stats')
}

export function fetchCertificate(serial: string): Promise<CertificateRecordDetail> {
  return request<CertificateRecordDetail>(`/certificates/${encodeURIComponent(serial)}`)
}

export function issueCertificate(
  payload: IssueCertificateRequest,
): Promise<IssueCertificateResponse> {
  return request<IssueCertificateResponse>('/certificates/issue', {
    method: 'POST',
    body: payload,
  })
}

export function issueCertificateWithToken(
  payload: IssueCertificateWithTokenRequest,
): Promise<IssueCertificateResponse> {
  return request<IssueCertificateResponse>('/certificates/issue-with-token', {
    method: 'POST',
    body: payload,
  })
}

export function autoIssueCertificate(
  payload: AutoCertificateRequest,
): Promise<AutoCertificateResponse> {
  return request<AutoCertificateResponse>('/certificates/auto', {
    method: 'POST',
    body: payload,
  })
}

export function revokeCertificate(
  payload: RevokeCertificateRequest,
): Promise<RevokeCertificateResponse> {
  return request<RevokeCertificateResponse>('/certificates/revoke', {
    method: 'POST',
    body: payload,
  })
}

export function downloadCertificateBundle(serial: string): Promise<Blob> {
  return requestBlob(`/certificates/${encodeURIComponent(serial)}/bundle`)
}
