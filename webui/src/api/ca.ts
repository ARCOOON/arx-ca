import { apiClient } from './client'
import type { ApiEnvelope, CAInfoResponse, CAProvisionersResponse } from '../types/api'

interface RootCertResponse {
  pem: string
}

interface IntermediateCertResponse {
  pem: string
}

export async function fetchRootCertPEM(): Promise<string> {
  const { data } = await apiClient.get<ApiEnvelope<RootCertResponse>>('/ca/root')
  if (data.error || !data.data?.pem) {
    throw new Error(data.error ?? 'Failed to load root CA certificate')
  }
  return data.data.pem
}

export async function fetchIntermediateCertPEM(): Promise<string> {
  const { data } = await apiClient.get<ApiEnvelope<IntermediateCertResponse>>('/public/ca/intermediate')
  if (data.error || !data.data?.pem) {
    throw new Error(data.error ?? 'Failed to load intermediate CA certificate')
  }
  return data.data.pem
}

export async function fetchCAInfo(): Promise<CAInfoResponse> {
  const { data } = await apiClient.get<ApiEnvelope<CAInfoResponse>>('/ca/info')
  if (data.error || !data.data) {
    throw new Error(data.error ?? 'Failed to load CA certificate information')
  }
  return data.data
}

export async function fetchCAProvisioners(): Promise<CAProvisionersResponse> {
  const { data } = await apiClient.get<ApiEnvelope<CAProvisionersResponse>>('/ca/provisioners')
  if (data.error || !data.data) {
    throw new Error(data.error ?? 'Failed to load CA provisioner configuration')
  }
  return data.data
}
