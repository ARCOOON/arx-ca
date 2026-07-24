import { request, type QueryParams } from './client'
import type {
  GenerateSshHostRequest,
  GenerateSshUserRequest,
  InspectSshCertificateRequest,
  ListSshCertificatesResponse,
  SshCertificateInspection,
  SshCertificateResponse,
  SshRootsResponse,
  SshStatsResponse,
} from '@/types/api'

export function fetchSshStats(): Promise<SshStatsResponse> {
  return request<SshStatsResponse>('/ssh/stats')
}

export interface ListSshCertificatesParams {
  limit?: number
  offset?: number
}

export function listSshCertificates(
  params: ListSshCertificatesParams = {},
): Promise<ListSshCertificatesResponse> {
  return request<ListSshCertificatesResponse>('/ssh/certificates', { query: params as QueryParams })
}

export function fetchSshRoots(): Promise<SshRootsResponse> {
  return request<SshRootsResponse>('/ssh/roots')
}

export function generateSshUserCertificate(
  payload: GenerateSshUserRequest,
): Promise<SshCertificateResponse> {
  return request<SshCertificateResponse>('/ssh/generate/user', {
    method: 'POST',
    body: payload,
  })
}

export function generateSshHostCertificate(
  payload: GenerateSshHostRequest,
): Promise<SshCertificateResponse> {
  return request<SshCertificateResponse>('/ssh/generate/host', {
    method: 'POST',
    body: payload,
  })
}

export function inspectSshCertificate(
  payload: InspectSshCertificateRequest,
): Promise<SshCertificateInspection> {
  return request<SshCertificateInspection>('/ssh/inspect', {
    method: 'POST',
    body: payload,
  })
}
