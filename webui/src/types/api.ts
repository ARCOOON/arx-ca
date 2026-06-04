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
}

export interface IssueCertificateResponse {
  certificate_pem: string
  serial: string
  not_before: string
  not_after: string
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
