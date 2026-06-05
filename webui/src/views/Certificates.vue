<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { downloadCRL, fetchCRLStatus, type CRLStatus } from '../api/crl'
import {
  fetchCertificateBySerial,
  generateCertificate,
  issueCertificate,
  listCertificates,
} from '../api/certificates'
import type { CertificateRecordDetail, CertificateSummary, KeyAlgorithm } from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import FlatToggle from '../components/ui/FlatToggle.vue'
import Modal from '../components/ui/Modal.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import TagInput from '../components/ui/TagInput.vue'
import {
  extractCommonName,
  resolveCertificateStatus,
  type CertificateLifecycleStatus,
} from '../utils/certificate'
import { downloadPemZip, downloadTextFile } from '../utils/download'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'

type IssueMode = 'csr' | 'native'

const certificates = ref<CertificateSummary[]>([])
const isLoading = ref(true)
const errorMessage = ref('')

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

const tableColumns = [
  { key: 'serial', label: 'Serial Number', cellClass: 'font-mono text-[11px]' },
  { key: 'commonName', label: 'Common Name' },
  { key: 'not_before', label: 'Issue Date' },
  { key: 'not_after', label: 'Expiry Date' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '', headerClass: 'w-28' },
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
    const response = await listCertificates()
    certificates.value = response.certificates
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load certificates')
  } finally {
    isLoading.value = false
  }
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
    await loadCertificates()
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
    const result = await generateCertificate({
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
    })

    const safeName = commonName.replace(/[^a-zA-Z0-9._-]+/g, '_')
    downloadPemZip(`${safeName}-bundle.zip`, result.certificate_pem, result.private_key_pem)

    issueSuccess.value =
      'Certificate and private key were downloaded as a ZIP archive. Store the key securely; it is not retained on the server.'
    nativeCommonName.value = ''
    nativeSansTags.value = []
    nativeOrganization.value = ''
    nativeOrganizationalUnit.value = ''
    nativeCountry.value = ''
    nativeState.value = ''
    nativeLocality.value = ''
    nativeServerAuth.value = false
    nativeClientAuth.value = false
    nativeAdvancedOpen.value = false
    await loadCertificates()
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
  await submitIssueCSR()
}

async function openCertificateDetails(serial: string): Promise<void> {
  detailsModalOpen.value = true
  detailsLoading.value = true
  detailsError.value = ''
  certificateDetail.value = null

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
</script>

<template>
  <div class="space-y-4">
    <section class="ui-surface-muted px-4 py-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold ui-text-primary">Certificate Revocation List</h2>
          <p class="mt-1 text-xs ui-text-muted">
            Published via
            <code class="ui-code">GET /api/v1/crl</code>
            (alias of
            <code class="ui-code">/api/v1/ca/crl</code>).
          </p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <StatusBadge :label="crlStatusLabel" :tone="crlStatusTone" />
          </div>
          <p v-if="crlError" class="mt-2 text-xs" style="color: var(--danger-text)" role="alert">
            {{ crlError }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="ui-btn-secondary"
            :disabled="crlDownloading"
            @click="handleDownloadCRL('pem')"
          >
            {{ crlDownloading ? 'Downloading…' : 'Download CRL (PEM)' }}
          </button>
          <button
            type="button"
            class="ui-btn-secondary"
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
        <p class="text-xs ui-text-muted">
          Inventory from
          <code class="ui-code">GET /api/v1/certificates</code>
        </p>
      </div>
      <button type="button" class="ui-btn-primary" @click="openIssueModal">Issue Certificate</button>
    </div>

    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <DataTable
      :columns="tableColumns"
      :rows="tableRows"
      :row-key="(row) => row.serial"
      :loading="isLoading"
      empty-message="No certificates have been issued yet."
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
        <button type="button" class="ui-btn-secondary text-[11px]" @click="openCertificateDetails(row.serial)">
          View Details
        </button>
      </template>
    </DataTable>

    <Modal :open="detailsModalOpen" title="Certificate Details" wide @close="closeDetailsModal">
      <div v-if="detailsLoading" class="text-sm ui-text-muted">Loading certificate record…</div>
      <div v-else-if="detailsError" class="ui-alert-error text-xs" role="alert">
        {{ detailsError }}
      </div>
      <template v-else-if="certificateDetail">
        <dl class="grid gap-3 text-xs sm:grid-cols-2">
          <div>
            <dt class="font-medium ui-text-muted">Serial</dt>
            <dd class="mt-0.5 font-mono ui-text-primary">{{ certificateDetail.serial }}</dd>
          </div>
          <div>
            <dt class="font-medium ui-text-muted">Requestor ID</dt>
            <dd class="mt-0.5 font-mono ui-text-primary">{{ certificateDetail.requestor_id }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium ui-text-muted">Subject</dt>
            <dd class="mt-0.5 ui-text-primary">{{ certificateDetail.subject }}</dd>
          </div>
          <div>
            <dt class="font-medium ui-text-muted">Issued At</dt>
            <dd class="mt-0.5 ui-text-primary">{{ formatDateTime(certificateDetail.not_before) }}</dd>
          </div>
          <div>
            <dt class="font-medium ui-text-muted">Expires</dt>
            <dd class="mt-0.5 ui-text-primary">{{ formatDateTime(certificateDetail.not_after) }}</dd>
          </div>
          <div v-if="certificateDetail.dns_names?.length" class="sm:col-span-2">
            <dt class="font-medium ui-text-muted">DNS SANs</dt>
            <dd class="mt-0.5 ui-text-primary">{{ certificateDetail.dns_names.join(', ') }}</dd>
          </div>
          <div v-if="certificateDetail.ip_addresses?.length" class="sm:col-span-2">
            <dt class="font-medium ui-text-muted">IP SANs</dt>
            <dd class="mt-0.5 ui-text-primary">{{ certificateDetail.ip_addresses.join(', ') }}</dd>
          </div>
        </dl>
        <div class="mt-4">
          <p class="text-xs font-medium ui-text-secondary">Certificate (PEM)</p>
          <pre class="ui-inset mt-1.5 max-h-48 overflow-auto p-3 font-mono text-[10px] ui-text-secondary">{{ certificateDetail.certificate_pem }}</pre>
        </div>
      </template>

      <template #footer>
        <button type="button" class="ui-btn-secondary" @click="closeDetailsModal">Close</button>
        <button
          type="button"
          class="ui-btn-primary"
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
          class="ui-tab"
          :class="{ 'ui-tab-active': issueMode === 'csr' }"
          @click="setIssueMode('csr')"
        >
          Paste CSR
        </button>
        <button
          type="button"
          class="ui-tab"
          :class="{ 'ui-tab-active': issueMode === 'native' }"
          @click="setIssueMode('native')"
        >
          Native Generation
        </button>
      </div>

      <p v-if="issueMode === 'csr'" class="mb-3 text-xs ui-text-muted">
        Signs a PEM CSR via
        <code class="ui-code">POST /api/v1/certificates/issue</code>. The private key never leaves
        your client.
      </p>
      <p v-else class="mb-3 text-xs ui-text-muted">
        Generates a key pair and signs the certificate via
        <code class="ui-code">POST /api/v1/certificates/generate</code>. The private key is
        returned once and downloaded as a ZIP archive.
      </p>

      <div v-if="issueError" class="mb-3 ui-alert-error text-xs" role="alert">
        {{ issueError }}
      </div>

      <div v-if="issueSuccess" class="mb-3 ui-alert-success" role="status">
        {{ issueSuccess }}
      </div>

      <template v-if="issueMode === 'csr'">
        <label class="block text-xs font-medium ui-text-secondary" for="csr-input">
          Certificate signing request
        </label>
        <textarea
          id="csr-input"
          v-model="csrInput"
          rows="10"
          class="ui-textarea mt-1.5"
          placeholder="-----BEGIN CERTIFICATE REQUEST-----&#10;...&#10;-----END CERTIFICATE REQUEST-----"
          spellcheck="false"
        />

        <label class="mt-3 block text-xs font-medium ui-text-secondary" for="ttl-input">
          TTL (optional)
        </label>
        <input id="ttl-input" v-model="ttlInput" type="text" class="ui-input mt-1.5 max-w-xs" placeholder="720h" />
      </template>

      <template v-else>
        <label class="block text-xs font-medium ui-text-secondary" for="cn-input">Common Name</label>
        <input
          id="cn-input"
          v-model="nativeCommonName"
          type="text"
          class="ui-input mt-1.5"
          placeholder="www.example.com"
          autocomplete="off"
        />

        <label class="mt-3 block text-xs font-medium ui-text-secondary" for="sans-input">
          Subject Alternative Names
        </label>
        <TagInput
          id="sans-input"
          v-model="nativeSansTags"
          placeholder="api.example.com or 203.0.113.10"
        />
        <p class="mt-1 text-[11px] ui-text-muted">
          Press Enter, Space, or comma to add each DNS name or IP address.
        </p>

        <div class="mt-3 grid gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="key-algo-select">
              Key algorithm
            </label>
            <select id="key-algo-select" v-model="nativeKeyAlgo" class="ui-input mt-1.5">
              <option value="RSA2048">RSA 2048</option>
              <option value="ECDSA256">ECDSA P-256</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="native-ttl-input">
              TTL / validity
            </label>
            <input
              id="native-ttl-input"
              v-model="nativeTtlInput"
              type="text"
              class="ui-input mt-1.5"
              placeholder="720h"
            />
          </div>
        </div>

        <details class="mt-4 ui-inset" :open="nativeAdvancedOpen" @toggle="nativeAdvancedOpen = ($event.target as HTMLDetailsElement).open">
          <summary class="cursor-pointer px-4 py-3 text-xs font-medium ui-text-secondary">
            Advanced Subject Options
          </summary>
          <div class="space-y-3 border-t border-[var(--border-subtle)] px-4 py-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <label class="block text-xs font-medium ui-text-secondary" for="org-input">Organization (O)</label>
                <input id="org-input" v-model="nativeOrganization" type="text" class="ui-input mt-1.5" autocomplete="organization" />
              </div>
              <div>
                <label class="block text-xs font-medium ui-text-secondary" for="ou-input">Organizational Unit (OU)</label>
                <input id="ou-input" v-model="nativeOrganizationalUnit" type="text" class="ui-input mt-1.5" />
              </div>
              <div>
                <label class="block text-xs font-medium ui-text-secondary" for="country-input">Country (C)</label>
                <input id="country-input" v-model="nativeCountry" type="text" class="ui-input mt-1.5" maxlength="2" placeholder="US" />
              </div>
              <div>
                <label class="block text-xs font-medium ui-text-secondary" for="state-input">State / Province (ST)</label>
                <input id="state-input" v-model="nativeState" type="text" class="ui-input mt-1.5" />
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium ui-text-secondary" for="locality-input">Locality (L)</label>
              <input id="locality-input" v-model="nativeLocality" type="text" class="ui-input mt-1.5" />
            </div>
          </div>
        </details>

        <div class="mt-3 space-y-2">
          <p class="text-xs font-medium ui-text-secondary">Extended Key Usage</p>
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
      </template>

      <template #footer>
        <button type="button" class="ui-btn-secondary" :disabled="isIssuing" @click="closeIssueModal">
          Cancel
        </button>
        <button type="button" class="ui-btn-primary" :disabled="isIssuing" @click="submitIssue">
          {{
            isIssuing
              ? 'Working…'
              : issueMode === 'csr'
                ? 'Sign CSR'
                : 'Generate & Download'
          }}
        </button>
      </template>
    </Modal>
  </div>
</template>
