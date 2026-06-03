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
