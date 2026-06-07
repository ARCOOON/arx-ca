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
} from '../types/api'
import { apiClient } from './client'

export async function generateSshUser(request: GenerateSshUserRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/generate/user', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH user certificate response did not include data')
  }

  return payload.data
}

export async function generateSshHost(request: GenerateSshHostRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/generate/host', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH host certificate response did not include data')
  }

  return payload.data
}

export async function signSshUser(request: SignSshUserRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/sign-user', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH user certificate response did not include data')
  }

  return payload.data
}

export async function signSshHost(request: SignSshHostRequest): Promise<SshCertificateResponse> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateResponse>>('/ssh/sign-host', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH host certificate response did not include data')
  }

  return payload.data
}

export async function inspectSshCertificate(
  request: InspectSshCertificateRequest,
): Promise<SshCertificateInspection> {
  const response = await apiClient.post<ApiEnvelope<SshCertificateInspection>>('/ssh/inspect', request)
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH inspection response did not include data')
  }

  return payload.data
}

export async function fetchSshRoots(): Promise<SshRootsResponse> {
  const response = await apiClient.get<ApiEnvelope<SshRootsResponse>>('/ssh/roots')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH roots response did not include data')
  }

  return payload.data
}

export async function listSshCertificates(
  limit = 50,
  offset = 0,
): Promise<ListSshCertificatesResponse> {
  const response = await apiClient.get<ApiEnvelope<ListSshCertificatesResponse>>('/ssh/certificates', {
    params: { limit, offset },
  })
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH certificate list response did not include data')
  }

  return payload.data
}

export async function fetchSshStats(): Promise<SshStatsResponse> {
  const response = await apiClient.get<ApiEnvelope<SshStatsResponse>>('/ssh/stats')
  const payload = response.data

  if (payload.error) {
    throw new Error(payload.error)
  }

  if (!payload.data) {
    throw new Error('SSH stats response did not include data')
  }

  return payload.data
}
