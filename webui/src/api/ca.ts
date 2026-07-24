import { request, requestBlob } from './client'
import type {
  CAInfoResponse,
  CAPemResponse,
  CAProvisionersResponse,
} from '@/types/api'

export function fetchCAInfo(): Promise<CAInfoResponse> {
  return request<CAInfoResponse>('/ca/info')
}

export function fetchCAProvisioners(): Promise<CAProvisionersResponse> {
  return request<CAProvisionersResponse>('/ca/provisioners')
}

export function fetchRootCAPem(): Promise<CAPemResponse> {
  return request<CAPemResponse>('/ca/root')
}

export function fetchIntermediateCAPem(): Promise<CAPemResponse> {
  return request<CAPemResponse>('/public/ca/intermediate')
}

export function downloadCAChain(): Promise<Blob> {
  return requestBlob('/ca/chain')
}
