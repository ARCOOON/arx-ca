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
  use_digital_signature?: boolean
  use_key_encipherment?: boolean
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
  use_digital_signature?: boolean
  use_key_encipherment?: boolean
}

export interface GenerateCertificateResponse {
  certificate_pem: string
  private_key_pem: string
  serial?: string
  not_before?: string
  not_after?: string
}

export interface CertificateRecordDetail {
  serial: string
  common_name: string
  subject: string
  dns_names?: string[]
  ip_addresses?: string[]
  not_before: string
  not_after: string
  requestor_id: string
  certificate_pem: string
  revoked: boolean
  revoked_at?: string
  reason_code?: number
  revocation_reason?: string
  has_escrowed_key?: boolean
}

export interface CertificatePrivateKeyResponse {
  serial: string
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

export interface NdesStatus {
  enabled: boolean
  scep_endpoint?: string
  admin_endpoint?: string
  connectors?: string[]
  adcs_compatible: boolean
}

export interface K8sProvisionerStatus {
  enabled: boolean
  provisioner?: string
  review_mode?: string
  has_public_keys: boolean
  uses_token_review_api: boolean
}

export interface CreateAcmeEabKeyRequest {
  provisioner?: string
  reference?: string
}

export interface AcmeEabKeyResponse {
  key_id: string
  provisioner: string
  hmac_key: string
  reference?: string
  created_at: string
}

export interface RevokeCertificateRequest {
  serial_number: string
  reason?: string
  reason_code: number
}

export interface RevokeCertificateResponse {
  serial: string
  revoked_at: string
}

export interface IssueCertificateWithTokenRequest {
  token: string
  csr: string
  ttl?: string
  template_id?: string
}

export interface AutoCertificateRequest {
  common_name: string
  dns_sans?: string[]
  ip_sans?: string[]
  ttl?: string
  template_id?: string
}

export interface AutoCertificateResponse {
  certificate_pem: string
  private_key_pem: string
  serial: string
  not_before: string
  not_after: string
}

export interface RenewCertificateRequest {
  certificate_pem?: string
  renew_token?: string
}

export interface RekeyCertificateRequest {
  certificate_pem?: string
  renew_token?: string
  csr: string
}

export interface LintCertificateRequest {
  certificate_pem: string
}

export interface CertificateLintFinding {
  lint: string
  source: string
  severity: string
  message?: string
}

export interface LintCertificateSummary {
  errors: number
  warnings: number
  notices: number
  fatals: number
}

export interface LintCertificateResponse {
  findings: CertificateLintFinding[]
  summary: LintCertificateSummary
}

export interface ProvisionerTokenRequest {
  provisioner?: string
  common_name: string
  dns_sans?: string[]
  ip_sans?: string[]
  token_ttl?: string
}

export interface ProvisionerTokenResponse {
  token: string
  provisioner: string
  provisioner_type: string
  expires_in: number
  audience: string
}

export interface CreateServiceAccountRequest {
  name: string
  roles?: string[]
}

export interface ServiceAccountResponse {
  id: string
  name: string
  roles: string[]
  api_key: string
  created_at: string
}

export interface CertificateTemplate {
  id: string
  name: string
  description?: string
  body: string
  created_at: string
  updated_at: string
}

export interface CreateCertificateTemplateRequest {
  name: string
  description?: string
  body: string
}

export interface ListCertificateTemplatesResponse {
  templates: CertificateTemplate[]
  total: number
}

export interface PublicCertificateSummary {
  serial: string
  subject: string
  not_before: string
  not_after: string
  revoked: boolean
}

export interface PublicListCertificatesResponse {
  certificates: PublicCertificateSummary[]
  total: number
}

export interface GenerateSshUserRequest {
  public_key: string
  principals: string[]
  ttl?: string
  provisioner?: string
}

export interface GenerateSshHostRequest {
  public_key: string
  principals: string[]
  ttl?: string
  provisioner?: string
}

export interface SignSshUserRequest {
  public_key: string
  principal?: string
  principals?: string[]
  ttl?: string
  token?: string
  provisioner?: string
}

export interface SignSshHostRequest {
  public_key: string
  hostname?: string
  principals?: string[]
  ttl?: string
  provisioner?: string
}

export interface InspectSshCertificateRequest {
  certificate: string
}

export interface SshCertificateResponse {
  certificate: string
  certificate_type: string
  key_id: string
  principals: string[]
  serial: number
  valid_after: string
  valid_before: string
}

export interface SshCertificateInspection {
  certificate_type: string
  key_id: string
  principals: string[]
  serial: number
  valid_after: string
  valid_before: string
  public_key_type: string
  critical_options?: Record<string, string>
  extensions?: Record<string, string>
  signature_key?: string
}

export interface SshRootKey {
  public_key: string
  key_type: string
  fingerprint: string
}

export interface SshRootsResponse {
  user_keys: SshRootKey[]
  host_keys: SshRootKey[]
}

export interface SshCertificateListItem {
  id: string
  serial: string
  cert_type: 'user' | 'host'
  principals: string[]
  fingerprint: string
  valid_after: string
  valid_before: string
}

export interface ListSshCertificatesResponse {
  certificates: SshCertificateListItem[]
  total: number
  limit: number
  offset: number
}

export interface CertificateStatsResponse {
  total_issued: number
  expiring_30d: number
  total_revoked: number
}

export interface SshStatsResponse {
  total_user_certs: number
  total_host_certs: number
  active_now: number
}

export interface AuditLogEntry {
  id: string
  timestamp: string
  request_id: string
  ip_address: string
  http_method: string
  endpoint: string
  status_code: number
  actor_type: string
  actor_id: string
  actor_roles?: string[]
  action: string
  provisioner?: string
  fingerprint?: string
  metadata?: Record<string, unknown>
}

export interface ListAuditLogsResponse {
  logs: AuditLogEntry[]
  total: number
  limit: number
  offset: number
}

export interface NotificationEntry {
  id: string
  timestamp: string
  action: string
  level: 'info' | 'critical'
  message: string
  is_read: boolean
  metadata?: Record<string, unknown>
}

export interface ListNotificationsResponse {
  notifications: NotificationEntry[]
  total: number
  unread_count: number
  limit: number
  offset: number
}

export interface MarkAllNotificationsReadResponse {
  updated: number
}

export interface ArchiveAllNotificationsResponse {
  archived: number
}

export interface WebhookResponse {
  id: string
  url: string
  name: string
  active: boolean
  subscribed_events: string[]
  has_secret_token: boolean
  created_at: string
  updated_at: string
}

export interface ListWebhooksResponse {
  webhooks: WebhookResponse[]
}

export interface WebhookEventOption {
  action: string
  label: string
  description: string
}

export interface WebhookEventsResponse {
  events: WebhookEventOption[]
}

export interface CreateWebhookRequest {
  url: string
  name: string
  secret_token?: string
  active?: boolean
  subscribed_events: string[]
}

export interface UpdateWebhookRequest {
  url: string
  name: string
  secret_token?: string
  active: boolean
  subscribed_events: string[]
}

export interface WebhookTestResponse {
  success: boolean
  status_code: number
  latency_ms: number
  error?: string
}
