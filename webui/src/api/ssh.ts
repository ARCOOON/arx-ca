import type {
  ApiEnvelope,
  GenerateSshHostRequest,
  GenerateSshUserRequest,
  InspectSshCertificateRequest,
  ListSshCertificatesResponse,
  SignSshHostRequest,
  SignSshUserRequest,
  SshCertificateInspection,
  SshCertificateResponse,
  SshRootsResponse,
  SshStatsResponse,
} from '@/types/api'
import { apiClient } from './client'

function unwrap<T>(payload: ApiEnvelope<T>, label: string): T {
  if (payload.error) throw new Error(payload.error)
  if (!payload.data) throw new Error(`${label} response did not include data`)
  return payload.data
}

export async function fetchSshStats(): Promise<SshStatsResponse> {
  const response = await apiClient.get<ApiEnvelope<SshStatsResponse>>('/ssh/stats')
  return unwrap(response.data, 'SSH stats')
}

export async function fetchSshCertificates(params?: {
  limit?: number
  offset?: number
}): Promise<ListSshCertificatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListSshCertificatesResponse>>('/ssh/certificates', { params })
  return unwrap(response.data, 'SSH certificates')
}

export async function fetchSshRoots(): Promise<SshRootsResponse> {
  const response = await apiClient.get<ApiEnvelope<SshRootsResponse>>('/ssh/roots')
  return unwrap(response.data, 'SSH roots')
}

export async function generateSshUser(req: GenerateSshUserRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/generate/user', req)
  return unwrap(response.data, 'Generate SSH user certificate')
}

export async function generateSshHost(req: GenerateSshHostRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/generate/host', req)
  return unwrap(response.data, 'Generate SSH host certificate')
}

export async function signSshUser(req: SignSshUserRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/sign-user', req)
  return unwrap(response.data, 'Sign SSH user certificate')
}

export async function signSshHost(req: SignSshHostRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/sign-host', req)
  return unwrap(response.data, 'Sign SSH host certificate')
}

export async function inspectSshCertificate(req: InspectSshCertificateRequest): Promise<SshCertificateInspection> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateInspection>>('/ssh/inspect', req)
  return unwrap(response.data, 'Inspect SSH certificate')
}
