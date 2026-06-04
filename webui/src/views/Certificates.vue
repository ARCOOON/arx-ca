<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { issueCertificate, listCertificates } from '../api/certificates'
import type { CertificateSummary } from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import Modal from '../components/ui/Modal.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import {
  extractCommonName,
  resolveCertificateStatus,
  type CertificateLifecycleStatus,
} from '../utils/certificate'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'

const certificates = ref<CertificateSummary[]>([])
const isLoading = ref(true)
const errorMessage = ref('')
const issueModalOpen = ref(false)
const csrInput = ref('')
const ttlInput = ref('720h')
const isIssuing = ref(false)
const issueError = ref('')
const issueSuccess = ref('')

const tableColumns = [
  { key: 'serial', label: 'Serial Number', cellClass: 'font-mono text-[11px]' },
  { key: 'commonName', label: 'Common Name' },
  { key: 'not_before', label: 'Issue Date' },
  { key: 'not_after', label: 'Expiry Date' },
  { key: 'status', label: 'Status' },
]

const tableRows = computed(() =>
  certificates.value.map((certificate) => ({
    ...certificate,
    commonName: extractCommonName(certificate.subject),
    status: resolveCertificateStatus(certificate),
  })),
)

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

onMounted(() => {
  void loadCertificates()
})

function openIssueModal(): void {
  issueError.value = ''
  issueSuccess.value = ''
  issueModalOpen.value = true
}

function closeIssueModal(): void {
  if (isIssuing.value) {
    return
  }
  issueModalOpen.value = false
}

async function submitIssue(): Promise<void> {
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
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-xs text-zinc-500">
          Inventory from
          <code class="border border-zinc-800 bg-zinc-900 px-1 text-emerald-300">GET /api/v1/certificates</code>
        </p>
      </div>
      <button
        type="button"
        class="border border-emerald-700 bg-emerald-900/40 px-3 py-1.5 text-xs font-medium text-emerald-200 transition hover:bg-emerald-900/70"
        @click="openIssueModal"
      >
        Issue Certificate
      </button>
    </div>

    <div
      v-if="errorMessage"
      class="border border-red-900/60 bg-red-950/40 px-3 py-2 text-sm text-red-300"
      role="alert"
    >
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
    </DataTable>

    <Modal :open="issueModalOpen" title="Issue Certificate" wide @close="closeIssueModal">
      <p class="mb-3 text-xs text-zinc-500">
        Signs a PEM CSR via
        <code class="border border-zinc-800 bg-zinc-950 px-1 text-emerald-300">POST /api/v1/certificates/issue</code>.
        The private key never leaves your client.
      </p>

      <div
        v-if="issueError"
        class="mb-3 border border-red-900/60 bg-red-950/40 px-3 py-2 text-xs text-red-300"
        role="alert"
      >
        {{ issueError }}
      </div>

      <div
        v-if="issueSuccess"
        class="mb-3 border border-emerald-900/60 bg-emerald-950/40 px-3 py-2 text-xs text-emerald-300"
        role="status"
      >
        {{ issueSuccess }}
      </div>

      <label class="block text-xs font-medium text-zinc-300" for="csr-input">Certificate signing request</label>
      <textarea
        id="csr-input"
        v-model="csrInput"
        rows="10"
        class="mt-1.5 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono text-[11px] text-zinc-100 outline-none focus:border-emerald-600"
        placeholder="-----BEGIN CERTIFICATE REQUEST-----&#10;...&#10;-----END CERTIFICATE REQUEST-----"
        spellcheck="false"
      />

      <label class="mt-3 block text-xs font-medium text-zinc-300" for="ttl-input">TTL (optional)</label>
      <input
        id="ttl-input"
        v-model="ttlInput"
        type="text"
        class="mt-1.5 w-full max-w-xs border border-zinc-700 bg-zinc-950 px-3 py-2 text-xs text-zinc-100 outline-none focus:border-emerald-600"
        placeholder="720h"
      />

      <template #footer>
        <button
          type="button"
          class="border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 hover:border-zinc-600"
          :disabled="isIssuing"
          @click="closeIssueModal"
        >
          Cancel
        </button>
        <button
          type="button"
          class="border border-emerald-700 bg-emerald-900/40 px-3 py-1.5 text-xs font-medium text-emerald-200 hover:bg-emerald-900/70 disabled:opacity-50"
          :disabled="isIssuing"
          @click="submitIssue"
        >
          {{ isIssuing ? 'Issuing…' : 'Sign CSR' }}
        </button>
      </template>
    </Modal>
  </div>
</template>
