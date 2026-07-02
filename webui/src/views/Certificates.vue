<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { CertificateLifecycleStatus } from '../utils/certificate'
import { downloadCRL, fetchCRLStatus, type CRLStatus } from '../api/crl'
import {
  autoCertificate,
  downloadCertificateBundleFile,
  fetchCertificateBySerial,
  fetchCertificatePrivateKey,
  fetchCertificateStats,
  generateCertificateBundleFile,
  issueCertificate,
  issueCertificateWithToken,
  lintCertificate,
  listCertificates,
  rekeyCertificate,
  renewCertificate,
  revokeCertificate,
} from '../api/certificates'
import type {
  CertificateRecordDetail,
  CertificateStatsResponse,
  CertificateSummary,
  KeyAlgorithm,
  LintCertificateResponse,
} from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import FlatToggle from '../components/ui/FlatToggle.vue'
import Modal from '../components/ui/Modal.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import TagInput from '../components/ui/TagInput.vue'
import { extractCommonName, resolveCertificateStatus } from '../utils/certificate'
import { downloadCertificateBundleZip, downloadTextFile } from '../utils/download'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'
import { useAuthStore } from '../store/auth'
import { usePreferences } from '../composables/usePreferences'
import Filter from 'lucide-vue-next/dist/esm/icons/list-filter.js'
import ChevronDown from 'lucide-vue-next/dist/esm/icons/chevron-down.js'

type IssueMode = 'csr' | 'native' | 'token' | 'auto'

const REVOKE_REASONS: Array<{ label: string; code: number }> = [
  { label: 'Unspecified', code: 0 },
  { label: 'Key compromise', code: 1 },
  { label: 'CA compromise', code: 2 },
  { label: 'Affiliation changed', code: 3 },
  { label: 'Superseded', code: 4 },
  { label: 'Cessation of operation', code: 5 },
  { label: 'Certificate hold', code: 6 },
]

const authStore = useAuthStore()
const { showApiHints } = usePreferences()

const isSuperAdmin = computed(() => authStore.roles.includes('SuperAdmin'))

const certificates = ref<CertificateSummary[]>([])
const isLoading = ref(true)
const errorMessage = ref('')
const filtersOpen = ref(false)

const draftCommonName = ref('')
const draftSerialNumber = ref('')
const draftStatus = ref<CertificateLifecycleStatus | ''>('')

const appliedCommonName = ref('')
const appliedSerialNumber = ref('')
const appliedStatus = ref<CertificateLifecycleStatus | ''>('')

const hasActiveFilters = computed(
  () => Boolean(appliedCommonName.value || appliedSerialNumber.value || appliedStatus.value),
)

const certStats = ref<CertificateStatsResponse | null>(null)
const statsLoading = ref(true)
const statsError = ref('')

const crlStatus = ref<CRLStatus | null>(null)
const crlLoading = ref(true)
const crlDownloading = ref(false)
const crlError = ref('')

const issueModalOpen = ref(false)
const issueMode = ref<IssueMode>('csr')
const csrInput = ref('')
const ttlInput = ref('720h')
const isIssuing = ref(false)
const issueError = ref('')
const issueSuccess = ref('')

const nativeCommonName = ref('')
const nativeSansTags = ref<string[]>([])
const detailsModalOpen = ref(false)
const detailsLoading = ref(false)
const detailsError = ref('')
const certificateDetail = ref<CertificateRecordDetail | null>(null)
const keyRevealLoading = ref(false)
const keyRevealError = ref('')
const revealedPrivateKey = ref('')
const nativeTtlInput = ref('720h')
const nativeKeyAlgo = ref<KeyAlgorithm>('ECDSA256')
const nativeAdvancedOpen = ref(false)
const nativeOrganization = ref('')
const nativeOrganizationalUnit = ref('')
const nativeCountry = ref('')
const nativeState = ref('')
const nativeLocality = ref('')
const nativeServerAuth = ref(false)
const nativeClientAuth = ref(false)
const nativeDigitalSignature = ref(true)
const nativeKeyEncipherment = ref(true)

const tokenInput = ref('')
const tokenCsrInput = ref('')
const tokenTtlInput = ref('720h')

const autoCommonName = ref('')
const autoDnsSans = ref<string[]>([])
const autoIpSans = ref<string[]>([])
const autoTtlInput = ref('720h')

const revokeModalOpen = ref(false)
const revokeTargetSerial = ref('')
const revokeReasonCode = ref(0)
const revokeReasonText = ref('')
const revokeConfirmInput = ref('')
const revokeLoading = ref(false)
const revokeError = ref('')

const rekeyModalOpen = ref(false)
const rekeyCsrInput = ref('')
const rekeyLoading = ref(false)
const rekeyError = ref('')
const rekeySuccess = ref('')

const renewLoading = ref(false)
const renewError = ref('')

const lintLoading = ref(false)
const lintError = ref('')
const lintResult = ref<LintCertificateResponse | null>(null)

const tableColumns = [
  { key: 'serial', label: 'Serial Number', cellClass: 'font-mono text-[11px]' },
  { key: 'commonName', label: 'Common Name' },
  { key: 'not_before', label: 'Issue Date' },
  { key: 'not_after', label: 'Expiry Date' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '', headerClass: 'w-40' },
]

const tableRows = computed(() =>
  certificates.value.map((certificate) => ({
    ...certificate,
    commonName: extractCommonName(certificate.subject),
    status: resolveCertificateStatus(certificate),
  })),
)

const crlStatusLabel = computed(() => {
  if (crlLoading.value) {
    return 'Checking…'
  }
  if (!crlStatus.value?.available) {
    return 'Unavailable'
  }
  if (crlStatus.value.expiresAt) {
    return `Available · next update ${crlStatus.value.expiresAt}`
  }
  return 'Available'
})

const crlStatusTone = computed((): 'valid' | 'revoked' | 'neutral' => {
  if (crlLoading.value) {
    return 'neutral'
  }
  return crlStatus.value?.available ? 'valid' : 'revoked'
})

const revokeSerialPrefix = computed(() =>
  revokeTargetSerial.value.replace(/\s+/g, '').slice(0, 8).toUpperCase(),
)

const canConfirmRevoke = computed(() => {
  const input = revokeConfirmInput.value.trim().toUpperCase()
  if (input === 'REVOKE') {
    return true
  }
  return input.length > 0 && input === revokeSerialPrefix.value
})

function statusTone(status: CertificateLifecycleStatus): 'valid' | 'revoked' | 'expired' {
  if (status === 'revoked') {
    return 'revoked'
  }
  if (status === 'expired') {
    return 'expired'
  }
  return 'valid'
}

function statusLabel(status: CertificateLifecycleStatus): string {
  if (status === 'revoked') {
    return 'Revoked'
  }
  if (status === 'expired') {
    return 'Expired'
  }
  return 'Valid'
}

async function loadCertificates(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const response = await listCertificates({
      common_name: appliedCommonName.value || undefined,
      serial_number: appliedSerialNumber.value || undefined,
      status: appliedStatus.value || undefined,
    })
    certificates.value = response.certificates
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load certificates')
  } finally {
    isLoading.value = false
  }
}

function applyFilters(): void {
  appliedCommonName.value = draftCommonName.value.trim()
  appliedSerialNumber.value = draftSerialNumber.value.trim()
  appliedStatus.value = draftStatus.value
  void loadCertificates()
}

function clearFilters(): void {
  draftCommonName.value = ''
  draftSerialNumber.value = ''
  draftStatus.value = ''
  appliedCommonName.value = ''
  appliedSerialNumber.value = ''
  appliedStatus.value = ''
  void loadCertificates()
}

async function loadStats(): Promise<void> {
  statsLoading.value = true
  statsError.value = ''

  try {
    certStats.value = await fetchCertificateStats()
  } catch (error) {
    statsError.value = extractApiError(error, 'Failed to load certificate statistics')
    certStats.value = null
  } finally {
    statsLoading.value = false
  }
}

async function refreshCertificateView(): Promise<void> {
  await Promise.all([loadCertificates(), loadStats()])
}

async function loadCRLStatus(): Promise<void> {
  crlLoading.value = true
  crlError.value = ''

  try {
    crlStatus.value = await fetchCRLStatus()
  } catch (error) {
    crlError.value = extractApiError(error, 'Failed to load CRL status')
    crlStatus.value = { available: false, expiresAt: null, format: 'pem' }
  } finally {
    crlLoading.value = false
  }
}

onMounted(() => {
  void loadCertificates()
  void loadStats()
  void loadCRLStatus()
})

function openIssueModal(): void {
  issueError.value = ''
  issueSuccess.value = ''
  issueMode.value = 'csr'
  issueModalOpen.value = true
}

function closeIssueModal(): void {
  if (isIssuing.value) {
    return
  }
  issueModalOpen.value = false
}

function setIssueMode(mode: IssueMode): void {
  issueMode.value = mode
  issueError.value = ''
  issueSuccess.value = ''
}

async function submitIssueCSR(): Promise<void> {
  issueError.value = ''
  issueSuccess.value = ''

  const csr = csrInput.value.trim()
  if (!csr.includes('BEGIN CERTIFICATE REQUEST')) {
    issueError.value = 'Paste a PEM-encoded certificate signing request.'
    return
  }

  isIssuing.value = true

  try {
    const result = await issueCertificate({
      csr,
      ttl: ttlInput.value.trim() || undefined,
    })
    issueSuccess.value = `Issued certificate serial ${result.serial}`
    csrInput.value = ''
    await refreshCertificateView()
  } catch (error) {
    issueError.value = extractApiError(error, 'Failed to issue certificate')
  } finally {
    isIssuing.value = false
  }
}

async function submitNativeGeneration(): Promise<void> {
  issueError.value = ''
  issueSuccess.value = ''

  const commonName = nativeCommonName.value.trim()
  if (!commonName) {
    issueError.value = 'Common Name is required.'
    return
  }

  const ttl = nativeTtlInput.value.trim()
  if (ttl && !/^\d+[smhdwMy]$/.test(ttl)) {
    issueError.value = 'TTL must be a duration such as 720h or 30d.'
    return
  }

  isIssuing.value = true

  try {
    const request = {
      common_name: commonName,
      sans: nativeSansTags.value.length > 0 ? nativeSansTags.value : undefined,
      ttl: ttl || undefined,
      key_algo: nativeKeyAlgo.value,
      organization: nativeOrganization.value.trim() || undefined,
      organizational_unit: nativeOrganizationalUnit.value.trim() || undefined,
      country: nativeCountry.value.trim() || undefined,
      state: nativeState.value.trim() || undefined,
      locality: nativeLocality.value.trim() || undefined,
      is_server_auth: nativeServerAuth.value || undefined,
      is_client_auth: nativeClientAuth.value || undefined,
      use_digital_signature: nativeDigitalSignature.value,
      use_key_encipherment: nativeKeyEncipherment.value,
    }

    const safeName = commonName.replace(/[^a-zA-Z0-9._-]+/g, '_')
    await generateCertificateBundleFile(request, `${safeName}-bundle.zip`)

    issueSuccess.value =
      'Certificate bundle downloaded. The private key is also escrowed on the server (AES-256-GCM encrypted) for SuperAdmin retrieval.'
    nativeCommonName.value = ''
    nativeSansTags.value = []
    nativeOrganization.value = ''
    nativeOrganizationalUnit.value = ''
    nativeCountry.value = ''
    nativeState.value = ''
    nativeLocality.value = ''
    nativeServerAuth.value = false
    nativeClientAuth.value = false
    nativeDigitalSignature.value = true
    nativeKeyEncipherment.value = true
    nativeAdvancedOpen.value = false
    await refreshCertificateView()
  } catch (error) {
    issueError.value = extractApiError(error, 'Failed to generate certificate')
  } finally {
    isIssuing.value = false
  }
}

async function submitIssue(): Promise<void> {
  if (issueMode.value === 'native') {
    await submitNativeGeneration()
    return
  }
  if (issueMode.value === 'token') {
    await submitIssueWithToken()
    return
  }
  if (issueMode.value === 'auto') {
    await submitAutoIssue()
    return
  }
  await submitIssueCSR()
}

async function openCertificateDetails(serial: string): Promise<void> {
  detailsModalOpen.value = true
  detailsLoading.value = true
  detailsError.value = ''
  certificateDetail.value = null
  keyRevealError.value = ''
  revealedPrivateKey.value = ''
  lintResult.value = null
  lintError.value = ''
  renewError.value = ''

  try {
    certificateDetail.value = await fetchCertificateBySerial(serial)
  } catch (error) {
    detailsError.value = extractApiError(error, 'Failed to load certificate details')
  } finally {
    detailsLoading.value = false
  }
}

function closeDetailsModal(): void {
  detailsModalOpen.value = false
  certificateDetail.value = null
  detailsError.value = ''
  keyRevealError.value = ''
  revealedPrivateKey.value = ''
}

async function revealPrivateKey(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.serial) {
    return
  }

  keyRevealLoading.value = true
  keyRevealError.value = ''

  try {
    const response = await fetchCertificatePrivateKey(detail.serial)
    revealedPrivateKey.value = response.private_key_pem
  } catch (error) {
    keyRevealError.value = extractApiError(error, 'Failed to retrieve private key')
  } finally {
    keyRevealLoading.value = false
  }
}

async function downloadEscrowedKey(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.serial) {
    return
  }

  keyRevealLoading.value = true
  keyRevealError.value = ''

  try {
    const response = await fetchCertificatePrivateKey(detail.serial)
    const safeName = (detail.common_name || detail.serial).replace(/[^a-zA-Z0-9._-]+/g, '_')
    downloadTextFile(`${safeName}.key`, response.private_key_pem)
  } catch (error) {
    keyRevealError.value = extractApiError(error, 'Failed to download private key')
  } finally {
    keyRevealLoading.value = false
  }
}

async function downloadEscrowedBundle(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.serial) {
    return
  }

  keyRevealLoading.value = true
  keyRevealError.value = ''

  try {
    const safeName = (detail.common_name || detail.serial).replace(/[^a-zA-Z0-9._-]+/g, '_')
    await downloadCertificateBundleFile(detail.serial, `${safeName}-bundle.zip`)
  } catch (error) {
    keyRevealError.value = extractApiError(error, 'Failed to download certificate bundle')
  } finally {
    keyRevealLoading.value = false
  }
}

function downloadCertificateCRT(): void {
  const detail = certificateDetail.value
  if (!detail?.certificate_pem) {
    return
  }
  const safeName = (detail.common_name || detail.serial).replace(/[^a-zA-Z0-9._-]+/g, '_')
  downloadTextFile(`${safeName}.crt`, detail.certificate_pem, 'application/x-x509-ca-cert')
}

async function handleDownloadCRL(format: 'der' | 'pem'): Promise<void> {
  crlDownloading.value = true
  crlError.value = ''

  try {
    await downloadCRL(format)
  } catch (error) {
    crlError.value = extractApiError(error, 'Failed to download CRL')
  } finally {
    crlDownloading.value = false
  }
}

function openRevokeModal(serial: string): void {
  revokeTargetSerial.value = serial
  revokeReasonCode.value = 0
  revokeReasonText.value = ''
  revokeConfirmInput.value = ''
  revokeError.value = ''
  revokeModalOpen.value = true
}

function closeRevokeModal(): void {
  if (revokeLoading.value) {
    return
  }
  revokeModalOpen.value = false
  revokeTargetSerial.value = ''
  revokeConfirmInput.value = ''
}

async function submitRevoke(): Promise<void> {
  if (!canConfirmRevoke.value) {
    revokeError.value = 'Type the first 8 characters of the serial number or REVOKE to continue.'
    return
  }

  revokeError.value = ''
  revokeLoading.value = true

  try {
    await revokeCertificate({
      serial_number: revokeTargetSerial.value,
      reason: revokeReasonText.value.trim() || undefined,
      reason_code: revokeReasonCode.value,
    })
    revokeModalOpen.value = false
    closeDetailsModal()
    await refreshCertificateView()
    await loadCRLStatus()
  } catch (error) {
    revokeError.value = extractApiError(error, 'Failed to revoke certificate')
  } finally {
    revokeLoading.value = false
  }
}

async function runLint(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.certificate_pem) {
    return
  }

  lintLoading.value = true
  lintError.value = ''
  lintResult.value = null

  try {
    lintResult.value = await lintCertificate({ certificate_pem: detail.certificate_pem })
  } catch (error) {
    lintError.value = extractApiError(error, 'Failed to lint certificate')
  } finally {
    lintLoading.value = false
  }
}

async function runRenew(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.certificate_pem) {
    return
  }

  renewLoading.value = true
  renewError.value = ''

  try {
    const result = await renewCertificate({ certificate_pem: detail.certificate_pem })
    renewError.value = ''
    certificateDetail.value = {
      ...detail,
      serial: result.serial,
      certificate_pem: result.certificate_pem,
      not_before: result.not_before,
      not_after: result.not_after,
    }
    await refreshCertificateView()
  } catch (error) {
    renewError.value = extractApiError(error, 'Failed to renew certificate')
  } finally {
    renewLoading.value = false
  }
}

function openRekeyModal(): void {
  rekeyCsrInput.value = ''
  rekeyError.value = ''
  rekeySuccess.value = ''
  rekeyModalOpen.value = true
}

function closeRekeyModal(): void {
  if (rekeyLoading.value) {
    return
  }
  rekeyModalOpen.value = false
}

async function submitRekey(): Promise<void> {
  const detail = certificateDetail.value
  if (!detail?.certificate_pem) {
    return
  }

  const csr = rekeyCsrInput.value.trim()
  if (!csr.includes('BEGIN CERTIFICATE REQUEST')) {
    rekeyError.value = 'Paste a PEM-encoded certificate signing request.'
    return
  }

  rekeyLoading.value = true
  rekeyError.value = ''
  rekeySuccess.value = ''

  try {
    const result = await rekeyCertificate({
      certificate_pem: detail.certificate_pem,
      csr,
    })
    rekeySuccess.value = `Rekeyed certificate serial ${result.serial}`
    certificateDetail.value = {
      ...detail,
      serial: result.serial,
      certificate_pem: result.certificate_pem,
      not_before: result.not_before,
      not_after: result.not_after,
    }
    await refreshCertificateView()
  } catch (error) {
    rekeyError.value = extractApiError(error, 'Failed to rekey certificate')
  } finally {
    rekeyLoading.value = false
  }
}

async function submitIssueWithToken(): Promise<void> {
  issueError.value = ''
  issueSuccess.value = ''

  const token = tokenInput.value.trim()
  const csr = tokenCsrInput.value.trim()

  if (!token) {
    issueError.value = 'Provisioner token is required.'
    return
  }
  if (!csr.includes('BEGIN CERTIFICATE REQUEST')) {
    issueError.value = 'Paste a PEM-encoded certificate signing request.'
    return
  }

  isIssuing.value = true

  try {
    const result = await issueCertificateWithToken({
      token,
      csr,
      ttl: tokenTtlInput.value.trim() || undefined,
    })
    issueSuccess.value = `Issued certificate serial ${result.serial}`
    tokenCsrInput.value = ''
    await refreshCertificateView()
  } catch (error) {
    issueError.value = extractApiError(error, 'Failed to issue certificate with token')
  } finally {
    isIssuing.value = false
  }
}

async function submitAutoIssue(): Promise<void> {
  issueError.value = ''
  issueSuccess.value = ''

  const commonName = autoCommonName.value.trim()
  if (!commonName) {
    issueError.value = 'Common Name is required.'
    return
  }

  isIssuing.value = true

  try {
    const result = await autoCertificate({
      common_name: commonName,
      dns_sans: autoDnsSans.value.length > 0 ? autoDnsSans.value : undefined,
      ip_sans: autoIpSans.value.length > 0 ? autoIpSans.value : undefined,
      ttl: autoTtlInput.value.trim() || undefined,
    })

    const safeName = commonName.replace(/[^a-zA-Z0-9._-]+/g, '_')
    downloadCertificateBundleZip(`${safeName}-auto.zip`, {
      certificatePem: result.certificate_pem,
      privateKeyPem: result.private_key_pem,
    })

    issueSuccess.value = `Auto-issued certificate serial ${result.serial}. Bundle downloaded.`
    autoCommonName.value = ''
    autoDnsSans.value = []
    autoIpSans.value = []
    await refreshCertificateView()
  } catch (error) {
    issueError.value = extractApiError(error, 'Failed to auto-issue certificate')
  } finally {
    isIssuing.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <section class="grid grid-cols-1 gap-4 md:grid-cols-3 mb-6">
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Total Issued</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (certStats?.total_issued ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Certificates in the CA store</p>
      </article>
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Expiring (&lt; 30d)</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (certStats?.expiring_30d ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Active certificates nearing expiry</p>
      </article>
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Revoked</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (certStats?.total_revoked ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Revoked certificates in the CA store</p>
      </article>
    </section>

    <p v-if="statsError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md px-3 py-2 text-xs" role="alert">
      {{ statsError }}
    </p>

    <section class="bg-card border-border-muted px-4 py-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-foreground">Certificate Revocation List</h2>
          <p v-if="showApiHints" class="mt-1 text-xs text-muted-foreground">
            Published via
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/crl</code>
            (alias of
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">/api/v1/ca/crl</code>).
          </p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <StatusBadge :label="crlStatusLabel" :tone="crlStatusTone" />
          </div>
          <p v-if="crlError" class="mt-2 text-xs" role="alert">
            {{ crlError }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
            :disabled="crlDownloading"
            @click="handleDownloadCRL('pem')"
          >
            {{ crlDownloading ? 'Downloading…' : 'Download CRL (PEM)' }}
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
            :disabled="crlDownloading"
            @click="handleDownloadCRL('der')"
          >
            Download CRL (DER)
          </button>
        </div>
      </div>
    </section>

    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p v-if="showApiHints" class="text-xs text-muted-foreground">
          Inventory from
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/certificates</code>
        </p>
      </div>
      <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="openIssueModal">Issue Certificate</button>
    </div>

    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <section class="bg-card border-border-muted border-b border-border px-4 py-3">
      <button
        type="button"
        class="flex w-full items-center gap-2 text-left text-xs font-medium text-foreground/80"
        :aria-expanded="filtersOpen"
        @click="filtersOpen = !filtersOpen"
      >
        <Filter class="h-3.5 w-3.5" aria-hidden="true" />
        <span>Search</span>
        <span v-if="hasActiveFilters" class="text-muted-foreground">(active)</span>
        <ChevronDown
          class="ml-auto h-4 w-4 transition-transform"
          :class="{ 'rotate-180': filtersOpen }"
          aria-hidden="true"
        />
      </button>

      <div
        v-show="filtersOpen"
        class="mt-3 grid gap-3 rounded-md border border-border p-3 sm:grid-cols-3"
      >
        <div>
          <label class="block text-xs font-medium text-foreground/80" for="cert-filter-cn">
            Common Name
          </label>
          <input
            id="cert-filter-cn"
            v-model="draftCommonName"
            type="text"
            class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full"
            placeholder="www.example.com"
            autocomplete="off"
            @keydown.enter.prevent="applyFilters"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-foreground/80" for="cert-filter-serial">
            Serial Number
          </label>
          <input
            id="cert-filter-serial"
            v-model="draftSerialNumber"
            type="text"
            class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full font-mono text-[11px]"
            placeholder="1234567890"
            autocomplete="off"
            @keydown.enter.prevent="applyFilters"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-foreground/80" for="cert-filter-status">
            Status
          </label>
          <select id="cert-filter-status" v-model="draftStatus" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full">
            <option value="">All statuses</option>
            <option value="valid">Valid</option>
            <option value="expired">Expired</option>
            <option value="revoked">Revoked</option>
          </select>
        </div>

        <div class="flex flex-wrap items-end gap-2 sm:col-span-3">
          <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="isLoading" @click="applyFilters">
            Apply Search
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
            :disabled="isLoading || !hasActiveFilters"
            @click="clearFilters"
          >
            Clear
          </button>
        </div>
      </div>
    </section>

    <DataTable
      :columns="tableColumns"
      :rows="tableRows"
      :row-key="(row) => row.serial"
      :loading="isLoading"
      empty-message="No certificates match the current filters."
    >
      <template #cell-not_before="{ row }">
        {{ formatDateTime(row.not_before) }}
      </template>
      <template #cell-not_after="{ row }">
        {{ formatDateTime(row.not_after) }}
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :label="statusLabel(row.status)" :tone="statusTone(row.status)" />
      </template>
      <template #cell-actions="{ row }">
        <div class="flex flex-wrap gap-1">
          <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 text-[11px]" @click="openCertificateDetails(row.serial)">
            Details
          </button>
          <button
            v-if="row.status !== 'revoked'"
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md bg-destructive px-3 py-2 text-sm font-medium text-destructive-foreground shadow-none transition-colors hover:bg-destructive/90 disabled:pointer-events-none disabled:opacity-50 text-[11px]"
            @click="openRevokeModal(row.serial)"
          >
            Revoke
          </button>
        </div>
      </template>
    </DataTable>

    <Modal :open="detailsModalOpen" title="Certificate Details" wide @close="closeDetailsModal">
      <div v-if="detailsLoading" class="text-sm text-muted-foreground">Loading certificate record…</div>
      <div v-else-if="detailsError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
        {{ detailsError }}
      </div>
      <template v-else-if="certificateDetail">
        <dl class="grid gap-3 text-xs sm:grid-cols-2">
          <div>
            <dt class="font-medium text-muted-foreground">Serial</dt>
            <dd class="mt-0.5 font-mono text-foreground">{{ certificateDetail.serial }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Requestor ID</dt>
            <dd class="mt-0.5 font-mono text-foreground">{{ certificateDetail.requestor_id }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Subject</dt>
            <dd class="mt-0.5 text-foreground">{{ certificateDetail.subject }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Issued At</dt>
            <dd class="mt-0.5 text-foreground">{{ formatDateTime(certificateDetail.not_before) }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Expires</dt>
            <dd class="mt-0.5 text-foreground">{{ formatDateTime(certificateDetail.not_after) }}</dd>
          </div>
          <div v-if="certificateDetail.revoked">
            <dt class="font-medium text-muted-foreground">Status</dt>
            <dd class="mt-0.5">
              <StatusBadge label="Revoked" tone="revoked" />
            </dd>
          </div>
          <div v-if="certificateDetail.revoked_at">
            <dt class="font-medium text-muted-foreground">Revoked At</dt>
            <dd class="mt-0.5 text-foreground">{{ formatDateTime(certificateDetail.revoked_at) }}</dd>
          </div>
          <div v-if="certificateDetail.reason_code != null">
            <dt class="font-medium text-muted-foreground">Reason Code</dt>
            <dd class="mt-0.5 text-foreground">{{ certificateDetail.reason_code }}</dd>
          </div>
          <div v-if="certificateDetail.revocation_reason" class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Revocation Reason</dt>
            <dd class="mt-0.5 text-foreground">{{ certificateDetail.revocation_reason }}</dd>
          </div>
          <div v-if="certificateDetail.dns_names?.length" class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">DNS SANs</dt>
            <dd class="mt-0.5 text-foreground">{{ certificateDetail.dns_names.join(', ') }}</dd>
          </div>
          <div v-if="certificateDetail.ip_addresses?.length" class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">IP SANs</dt>
            <dd class="mt-0.5 text-foreground">{{ certificateDetail.ip_addresses.join(', ') }}</dd>
          </div>
        </dl>
        <div class="mt-4">
          <p class="text-xs font-medium text-foreground/80">Certificate (PEM)</p>
          <pre class="rounded-md border border-input bg-muted/30 mt-1.5 max-h-48 overflow-auto p-3 font-mono text-[10px] text-foreground/80">{{ certificateDetail.certificate_pem }}</pre>
        </div>

        <div v-if="renewError" class="mt-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">{{ renewError }}</div>
        <div v-if="lintError" class="mt-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">{{ lintError }}</div>
        <div v-if="lintResult" class="mt-3 rounded-md border border-input bg-muted/30 p-3 text-xs">
          <p class="font-medium text-foreground/80">
            Lint summary:
            {{ lintResult.summary.fatals }} fatal,
            {{ lintResult.summary.errors }} error,
            {{ lintResult.summary.warnings }} warning,
            {{ lintResult.summary.notices }} notice
          </p>
          <ul v-if="lintResult.findings.length" class="mt-2 max-h-32 space-y-1 overflow-auto">
            <li v-for="(finding, index) in lintResult.findings" :key="index" class="text-muted-foreground">
              [{{ finding.severity }}] {{ finding.lint }}: {{ finding.message }}
            </li>
          </ul>
        </div>
        <div
          v-if="isSuperAdmin && certificateDetail.has_escrowed_key"
          class="mt-4 rounded border border-border p-3"
        >
          <p class="text-xs font-medium text-foreground/80">Escrowed Private Key</p>
          <p class="mt-1 text-[11px] text-muted-foreground">
            SuperAdmin access only. Key material is decrypted server-side using the CA master password.
          </p>
          <div v-if="keyRevealError" class="mt-2 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
            {{ keyRevealError }}
          </div>
          <div v-if="revealedPrivateKey" class="mt-3">
            <pre class="rounded-md border border-input bg-muted/30 max-h-48 overflow-auto p-3 font-mono text-[10px] text-foreground/80">{{ revealedPrivateKey }}</pre>
          </div>
        </div>
      </template>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="closeDetailsModal">Close</button>
        <button
          v-if="certificateDetail && !certificateDetail.revoked"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md bg-destructive px-3 py-2 text-sm font-medium text-destructive-foreground shadow-none transition-colors hover:bg-destructive/90 disabled:pointer-events-none disabled:opacity-50"
          @click="openRevokeModal(certificateDetail.serial)"
        >
          Revoke
        </button>
        <button
          v-if="certificateDetail?.certificate_pem"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          :disabled="lintLoading"
          @click="runLint"
        >
          {{ lintLoading ? 'Linting…' : 'Lint Certificate' }}
        </button>
        <button
          v-if="certificateDetail?.certificate_pem && !certificateDetail.revoked"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          :disabled="renewLoading"
          @click="runRenew"
        >
          {{ renewLoading ? 'Renewing…' : 'Renew' }}
        </button>
        <button
          v-if="certificateDetail?.certificate_pem && !certificateDetail.revoked"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          @click="openRekeyModal"
        >
          Rekey
        </button>
        <button
          v-if="isSuperAdmin && certificateDetail?.has_escrowed_key"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          :disabled="keyRevealLoading"
          @click="revealPrivateKey"
        >
          {{ keyRevealLoading ? 'Working…' : 'Reveal Private Key' }}
        </button>
        <button
          v-if="isSuperAdmin && certificateDetail?.has_escrowed_key"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          :disabled="keyRevealLoading"
          @click="downloadEscrowedKey"
        >
          Download Key
        </button>
        <button
          v-if="isSuperAdmin && certificateDetail?.has_escrowed_key"
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
          :disabled="keyRevealLoading"
          @click="downloadEscrowedBundle"
        >
          Download Bundle
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
          :disabled="!certificateDetail?.certificate_pem"
          @click="downloadCertificateCRT"
        >
          Download Certificate (.crt)
        </button>
      </template>
    </Modal>

    <Modal :open="issueModalOpen" title="Issue Certificate" wide @close="closeIssueModal">
      <div class="mb-4 flex flex-wrap gap-2">
        <button
          type="button"
          class="rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          :class="{ 'border-primary bg-primary/15 text-foreground': issueMode === 'csr' }"
          @click="setIssueMode('csr')"
        >
          Paste CSR
        </button>
        <button
          type="button"
          class="rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          :class="{ 'border-primary bg-primary/15 text-foreground': issueMode === 'native' }"
          @click="setIssueMode('native')"
        >
          Native Generation
        </button>
        <button
          type="button"
          class="rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          :class="{ 'border-primary bg-primary/15 text-foreground': issueMode === 'token' }"
          @click="setIssueMode('token')"
        >
          Provisioner Token
        </button>
        <button
          v-if="isSuperAdmin"
          type="button"
          class="rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          :class="{ 'border-primary bg-primary/15 text-foreground': issueMode === 'auto' }"
          @click="setIssueMode('auto')"
        >
          Auto Issue
        </button>
      </div>

      <p v-if="issueMode === 'csr'" class="mb-3 text-xs text-muted-foreground">
        Signs a PEM CSR<template v-if="showApiHints">
          via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/issue</code></template>.
        The private key never leaves your client.
      </p>
      <p v-else-if="issueMode === 'token'" class="mb-3 text-xs text-muted-foreground">
        Signs a CSR using a provisioner token<template v-if="showApiHints">
          via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/issue-with-token</code></template>.
      </p>
      <p v-else-if="issueMode === 'auto'" class="mb-3 text-xs text-muted-foreground">
        Generates key pair and certificate in one step<template v-if="showApiHints">
          via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/auto</code></template>
        (SuperAdmin).
      </p>
      <p v-else class="mb-3 text-xs text-muted-foreground">
        Generates a key pair and signs the certificate<template v-if="showApiHints">
          via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/generate</code></template>.
        The private key is returned immediately in a ZIP bundle (`certificate.crt`, `certificate.pem`, `private.key`) and escrowed encrypted at rest for SuperAdmin retrieval. Download the CA chain separately from the Dashboard.
      </p>

      <div v-if="issueError" class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
        {{ issueError }}
      </div>

      <div v-if="issueSuccess" class="mb-3 rounded-lg border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-foreground" role="status">
        {{ issueSuccess }}
      </div>

      <template v-if="issueMode === 'csr'">
        <label class="block text-xs font-medium text-foreground/80" for="csr-input">
          Certificate signing request
        </label>
        <textarea
          id="csr-input"
          v-model="csrInput"
          rows="10"
          class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
          placeholder="-----BEGIN CERTIFICATE REQUEST-----&#10;...&#10;-----END CERTIFICATE REQUEST-----"
          spellcheck="false"
        />

        <label class="mt-3 block text-xs font-medium text-foreground/80" for="ttl-input">
          TTL (optional)
        </label>
        <input id="ttl-input" v-model="ttlInput" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 max-w-xs" placeholder="720h" />
      </template>

      <template v-else-if="issueMode === 'native'">
        <label class="block text-xs font-medium text-foreground/80" for="cn-input">Common Name</label>
        <input
          id="cn-input"
          v-model="nativeCommonName"
          type="text"
          class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
          placeholder="www.example.com"
          autocomplete="off"
        />

        <label class="mt-3 block text-xs font-medium text-foreground/80" for="sans-input">
          Subject Alternative Names
        </label>
        <TagInput
          id="sans-input"
          v-model="nativeSansTags"
          placeholder="api.example.com or 203.0.113.10"
        />
        <p class="mt-1 text-[11px] text-muted-foreground">
          Press Enter, Space, or comma to add each DNS name or IP address.
        </p>

        <div class="mt-3 grid gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium text-foreground/80" for="key-algo-select">
              Key algorithm
            </label>
            <select id="key-algo-select" v-model="nativeKeyAlgo" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5">
              <option value="RSA2048">RSA 2048</option>
              <option value="ECDSA256">ECDSA P-256</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium text-foreground/80" for="native-ttl-input">
              TTL / validity
            </label>
            <input
              id="native-ttl-input"
              v-model="nativeTtlInput"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              placeholder="720h"
            />
          </div>
        </div>

        <details class="mt-4 rounded-md border border-input bg-muted/30" :open="nativeAdvancedOpen" @toggle="nativeAdvancedOpen = ($event.target as HTMLDetailsElement).open">
          <summary class="cursor-pointer px-4 py-3 text-xs font-medium text-foreground/80">
            Advanced Subject Options
          </summary>
          <div class="space-y-3 border-t border-border px-4 py-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <label class="block text-xs font-medium text-foreground/80" for="org-input">Organization (O)</label>
                <input id="org-input" v-model="nativeOrganization" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" autocomplete="organization" />
              </div>
              <div>
                <label class="block text-xs font-medium text-foreground/80" for="ou-input">Organizational Unit (OU)</label>
                <input id="ou-input" v-model="nativeOrganizationalUnit" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" />
              </div>
              <div>
                <label class="block text-xs font-medium text-foreground/80" for="country-input">Country (C)</label>
                <input id="country-input" v-model="nativeCountry" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" maxlength="2" placeholder="US" />
              </div>
              <div>
                <label class="block text-xs font-medium text-foreground/80" for="state-input">State / Province (ST)</label>
                <input id="state-input" v-model="nativeState" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" />
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground/80" for="locality-input">Locality (L)</label>
              <input id="locality-input" v-model="nativeLocality" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" />
            </div>
          </div>
        </details>

        <div class="mt-3 space-y-3">
          <p class="text-xs leading-relaxed text-muted-foreground">
            Recommended key-usage profiles for common issuance paths:
            <span class="mt-1.5 block">
              <span class="font-medium text-foreground/80">Webserver / ACME:</span>
              Digital Signature, Key Encipherment, Server Authentication.
            </span>
            <span class="mt-1 block">
              <span class="font-medium text-foreground/80">MDM / SCEP:</span>
              Digital Signature, Client Authentication.
            </span>
          </p>

          <div class="space-y-2">
            <p class="text-xs font-medium text-foreground/80">Standard Key Usage</p>
            <FlatToggle
              label="Digital Signature"
              :enabled="nativeDigitalSignature"
              @toggle="nativeDigitalSignature = !nativeDigitalSignature"
            />
            <FlatToggle
              label="Key Encipherment"
              :enabled="nativeKeyEncipherment"
              @toggle="nativeKeyEncipherment = !nativeKeyEncipherment"
            />
          </div>

          <div class="space-y-2">
            <p class="text-xs font-medium text-foreground/80">Extended Key Usage</p>
            <FlatToggle
              label="Server Authentication"
              :enabled="nativeServerAuth"
              @toggle="nativeServerAuth = !nativeServerAuth"
            />
            <FlatToggle
              label="Client Authentication"
              :enabled="nativeClientAuth"
              @toggle="nativeClientAuth = !nativeClientAuth"
            />
          </div>
        </div>
      </template>

      <template v-else-if="issueMode === 'token'">
        <label class="block text-xs font-medium text-foreground/80" for="token-input">Provisioner token</label>
        <input id="token-input" v-model="tokenInput" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 font-mono text-[11px]" autocomplete="off" />

        <label class="mt-3 block text-xs font-medium text-foreground/80" for="token-csr-input">
          Certificate signing request
        </label>
        <textarea
          id="token-csr-input"
          v-model="tokenCsrInput"
          rows="10"
          class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
          placeholder="-----BEGIN CERTIFICATE REQUEST-----"
          spellcheck="false"
        />

        <label class="mt-3 block text-xs font-medium text-foreground/80" for="token-ttl-input">
          TTL (optional)
        </label>
        <input id="token-ttl-input" v-model="tokenTtlInput" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 max-w-xs" placeholder="720h" />
      </template>

      <template v-else-if="issueMode === 'auto'">
        <label class="block text-xs font-medium text-foreground/80" for="auto-cn-input">Common Name</label>
        <input id="auto-cn-input" v-model="autoCommonName" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" autocomplete="off" />

        <label class="mt-3 block text-xs font-medium text-foreground/80">DNS SANs</label>
        <TagInput v-model="autoDnsSans" placeholder="api.example.com" />

        <label class="mt-3 block text-xs font-medium text-foreground/80">IP SANs</label>
        <TagInput v-model="autoIpSans" placeholder="10.0.0.1" />

        <label class="mt-3 block text-xs font-medium text-foreground/80" for="auto-ttl-input">
          TTL (optional)
        </label>
        <input id="auto-ttl-input" v-model="autoTtlInput" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 max-w-xs" placeholder="720h" />
      </template>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="isIssuing" @click="closeIssueModal">
          Cancel
        </button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="isIssuing" @click="submitIssue">
          {{
            isIssuing
              ? 'Working…'
              : issueMode === 'csr'
                ? 'Sign CSR'
                : issueMode === 'token'
                  ? 'Sign with Token'
                  : issueMode === 'auto'
                    ? 'Auto Issue & Download'
                    : 'Generate & Download'
          }}
        </button>
      </template>
    </Modal>

    <Modal :open="revokeModalOpen" title="Revoke Certificate" @close="closeRevokeModal">
      <div class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
        This action is irreversible. The certificate will be added to the CRL and clients must reject it.
      </div>

      <p class="mb-3 text-xs text-muted-foreground">
        Permanently revokes serial
        <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">{{ revokeTargetSerial }}</code><template v-if="showApiHints">
          via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/revoke</code></template>.
      </p>

      <div v-if="revokeError" class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">{{ revokeError }}</div>

      <label class="block text-xs font-medium text-foreground/80" for="revoke-confirm-input">
        Confirmation
      </label>
      <input
        id="revoke-confirm-input"
        v-model="revokeConfirmInput"
        type="text"
        class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 font-mono"
        :placeholder="`Type ${revokeSerialPrefix || 'serial prefix'} or REVOKE`"
        autocomplete="off"
        spellcheck="false"
      />
      <p class="mt-1 text-[11px] text-muted-foreground">
        Enter the first 8 characters of the serial number (<code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">{{ revokeSerialPrefix }}</code>)
        or type <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">REVOKE</code> to enable confirmation.
      </p>

      <label class="mt-3 block text-xs font-medium text-foreground/80" for="revoke-reason-code">Reason code</label>
      <select id="revoke-reason-code" v-model.number="revokeReasonCode" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5">
        <option v-for="reason in REVOKE_REASONS" :key="reason.code" :value="reason.code">
          {{ reason.label }} ({{ reason.code }})
        </option>
      </select>

      <label class="mt-3 block text-xs font-medium text-foreground/80" for="revoke-reason-text">
        Reason text (optional)
      </label>
      <input id="revoke-reason-text" v-model="revokeReasonText" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" autocomplete="off" />

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="revokeLoading" @click="closeRevokeModal">
          Cancel
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md bg-destructive px-3 py-2 text-sm font-medium text-destructive-foreground shadow-none transition-colors hover:bg-destructive/90 disabled:pointer-events-none disabled:opacity-50"
          :disabled="revokeLoading || !canConfirmRevoke"
          @click="submitRevoke"
        >
          {{ revokeLoading ? 'Revoking…' : 'Confirm Revocation' }}
        </button>
      </template>
    </Modal>

    <Modal :open="rekeyModalOpen" title="Rekey Certificate" wide @close="closeRekeyModal">
      <p v-if="showApiHints" class="mb-3 text-xs text-muted-foreground">
        Submit a new CSR via
        <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/certificates/rekey</code>.
      </p>

      <div v-if="rekeyError" class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">{{ rekeyError }}</div>
      <div v-if="rekeySuccess" class="mb-3 rounded-lg border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-foreground text-xs" role="status">{{ rekeySuccess }}</div>

      <label class="block text-xs font-medium text-foreground/80" for="rekey-csr-input">
        New certificate signing request
      </label>
      <textarea
        id="rekey-csr-input"
        v-model="rekeyCsrInput"
        rows="10"
        class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
        placeholder="-----BEGIN CERTIFICATE REQUEST-----"
        spellcheck="false"
      />

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="rekeyLoading" @click="closeRekeyModal">
          Cancel
        </button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="rekeyLoading" @click="submitRekey">
          {{ rekeyLoading ? 'Rekeying…' : 'Submit Rekey' }}
        </button>
      </template>
    </Modal>
  </div>
</template>
