export interface ApiEnvelope<T> {
  error: string | null
  data: T | null
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: string
  token_type: string
  roles?: string[]
}

export interface CertificateSummary {
  serial: string
  subject: string
  dns_names?: string[]
  ip_addresses?: string[]
  not_before: string
  not_after: string
  revoked: boolean
  provisioner_id?: string
  provisioner?: string
}

export interface ListCertificatesResponse {
  certificates: CertificateSummary[]
  total: number
}

export interface IssueCertificateRequest {
  csr: string
  ttl?: string
  template_id?: string
  organization?: string
  organizational_unit?: string
  country?: string
  state?: string
  locality?: string
  is_server_auth?: boolean
  is_client_auth?: boolean
}

export interface IssueCertificateResponse {
  certificate_pem: string
  serial: string
  not_before: string
  not_after: string
}

export type KeyAlgorithm = 'RSA2048' | 'ECDSA256'

export interface GenerateCertificateRequest {
  common_name: string
  sans?: string[]
  ttl?: string
  key_algo: KeyAlgorithm
  organization?: string
  organizational_unit?: string
  country?: string
  state?: string
  locality?: string
  is_server_auth?: boolean
  is_client_auth?: boolean
}

export interface GenerateCertificateResponse {
  certificate_pem: string
  private_key_pem: string
}

export interface HealthReport {
  uptime: {
    seconds: number
    human: string
  }
  memory: {
    alloc_bytes: number
    total_alloc_bytes: number
    sys_bytes: number
    heap_alloc_bytes: number
    heap_inuse_bytes: number
    heap_objects: number
    stack_inuse_bytes: number
    num_gc: number
    last_gc_unix: number
    goroutines: number
  }
  api: {
    status: string
    version: string
  }
  ca_backend: {
    status: string
    message?: string
    engine: string
    initialized: boolean
  }
}

export interface CASubjectInfo {
  common_name: string
  organization?: string[]
  organizational_unit?: string[]
  country?: string[]
  province?: string[]
  locality?: string[]
  street_address?: string[]
  postal_code?: string[]
  serial_number?: string
}

export interface CACertificateInfo {
  subject: CASubjectInfo
  issuer: CASubjectInfo
  not_before: string
  not_after: string
  serial_number: string
  signature_algorithm: string
  key_usages?: string[]
  ext_key_usages?: string[]
  fingerprint: string
  pem: string
}

export interface CAProvisionerDetail {
  name: string
  type: string
  require_eab?: boolean
  challenges?: string[]
  challenge?: string
}

export interface CAProvisionersResponse {
  provisioners: CAProvisionerDetail[]
  total: number
}

export interface CAInfoResponse {
  root: CACertificateInfo
  intermediate: CACertificateInfo
}

export interface AcmeStatus {
  enabled: boolean
  directory_url?: string
  provisioner?: string
  challenges?: string[]
  dns_name?: string
  require_eab: boolean
  device_attest_enabled: boolean
  attestation_formats?: string[]
}

export interface ScepStatus {
  enabled: boolean
  base_url?: string
  provisioner?: string
  challenge_hint?: string
}
