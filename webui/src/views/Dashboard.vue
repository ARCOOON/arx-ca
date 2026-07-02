<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { downloadCRL, fetchCRLStatus, type CRLStatus } from '../api/crl'
import { downloadCABundle, fetchCAInfo, fetchCAProvisioners } from '../api/ca'
import { fetchHealth } from '../api/health'
import { listCertificates } from '../api/certificates'
import type { CAInfoResponse, CAProvisionerDetail, HealthReport } from '../types/api'
import { extractApiError } from '../utils/errors'
import { formatBytes } from '../utils/format'
import {
  downloadCertificate,
  formatCertDate,
  formatUsageList,
  parseBackendDetails,
  shortenFingerprint,
} from '../utils/ca'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { usePreferences } from '../composables/usePreferences'

const { showApiHints } = usePreferences()

const health = ref<HealthReport | null>(null)
const caInfo = ref<CAInfoResponse | null>(null)
const caProvisioners = ref<CAProvisionerDetail[]>([])
const certificateTotal = ref<number | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')
const chainDownloading = ref(false)
const chainError = ref('')
const crlStatus = ref<CRLStatus | null>(null)
const crlDownloading = ref(false)
const crlError = ref('')

const backendDetails = computed(() => parseBackendDetails(health.value?.ca_backend.message))

const crlStatusLabel = computed(() => {
  if (!crlStatus.value) {
    return 'Unknown'
  }
  if (!crlStatus.value.available) {
    return 'Unavailable'
  }
  if (crlStatus.value.expiresAt) {
    return `Available · next update ${crlStatus.value.expiresAt}`
  }
  return 'Available'
})

const crlStatusTone = computed((): 'valid' | 'revoked' | 'neutral' => {
  return crlStatus.value?.available ? 'valid' : 'revoked'
})

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const [healthReport, certificateList, caInfoReport, provisionersReport, crlReport] = await Promise.all([
      fetchHealth(),
      listCertificates(),
      fetchCAInfo(),
      fetchCAProvisioners(),
      fetchCRLStatus(),
    ])
    health.value = healthReport
    certificateTotal.value = certificateList.total
    caInfo.value = caInfoReport
    caProvisioners.value = provisionersReport.provisioners
    crlStatus.value = crlReport
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load dashboard metrics')
  } finally {
    isLoading.value = false
  }
})

async function handleDownloadCAChain(): Promise<void> {
  chainDownloading.value = true
  chainError.value = ''

  try {
    await downloadCABundle()
  } catch (error) {
    chainError.value = extractApiError(error, 'Failed to download CA bundle')
  } finally {
    chainDownloading.value = false
  }
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

function backendTone(status: string): 'valid' | 'revoked' | 'neutral' {
  if (status === 'healthy') {
    return 'valid'
  }
  if (status === 'unhealthy' || status === 'degraded') {
    return 'revoked'
  }
  return 'neutral'
}
</script>

<template>
  <div class="space-y-4">
    <div
      v-if="errorMessage"
      class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-muted-foreground">Loading server status…</div>

    <template v-else-if="health">
      <section class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
        <article class="bg-card border-border-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Uptime</p>
          <p class="mt-1 text-lg font-semibold text-foreground">{{ health.uptime.human }}</p>
          <p class="text-xs text-muted-foreground">{{ health.uptime.seconds }} seconds</p>
        </article>

        <article class="bg-card border-border-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">API</p>
          <div class="mt-2 flex items-center gap-2">
            <StatusBadge :label="health.api.status" :tone="backendTone(health.api.status)" />
            <span class="text-xs text-muted-foreground">v{{ health.api.version }}</span>
          </div>
        </article>

        <article class="bg-card border-border-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">CA Backend</p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <StatusBadge :label="health.ca_backend.status" :tone="backendTone(health.ca_backend.status)" />
            <span class="text-xs text-muted-foreground">{{ health.ca_backend.engine }}</span>
          </div>
          <dl
            v-if="backendDetails.length > 0"
            class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs"
          >
            <template v-for="detail in backendDetails" :key="detail.label">
              <dt class="text-muted-foreground">{{ detail.label }}</dt>
              <dd class="truncate font-mono text-foreground/80" :title="detail.value">
                {{ detail.value }}
              </dd>
            </template>
          </dl>
        </article>

        <article class="bg-card border-border-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Certificates</p>
          <p class="mt-1 text-lg font-semibold text-foreground">
            {{ certificateTotal ?? '—' }}
          </p>
          <p class="text-xs text-muted-foreground">Issued in database</p>
        </article>
      </section>

      <section class="bg-card border-border-muted">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Runtime</h2>
        </header>
        <div class="grid grid-cols-1 gap-px md:grid-cols-2 lg:grid-cols-4">
          <div class="px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Heap in use</p>
            <p class="mt-1 text-sm text-foreground/80">{{ formatBytes(health.memory.heap_inuse_bytes) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Goroutines</p>
            <p class="mt-1 text-sm text-foreground/80">{{ health.memory.goroutines }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-muted-foreground">GC cycles</p>
            <p class="mt-1 text-sm text-foreground/80">{{ health.memory.num_gc }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Engine initialized</p>
            <p class="mt-1 text-sm text-foreground/80">{{ health.ca_backend.initialized ? 'Yes' : 'No' }}</p>
          </div>
        </div>
      </section>

      <section class="bg-card border-border-muted">
        <header class="border-b border-border flex flex-wrap items-center justify-between gap-2 px-4 py-2.5">
          <div>
            <h2 class="text-sm font-semibold text-foreground">Certificate Revocation List</h2>
            <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
              Public endpoint
              <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/crl</code>
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
        </header>
        <div class="flex flex-wrap items-center gap-2 px-4 py-3">
          <StatusBadge :label="crlStatusLabel" :tone="crlStatusTone" />
        </div>
        <p v-if="crlError" class="px-4 pb-3 text-xs" role="alert">
          {{ crlError }}
        </p>
      </section>

      <section v-if="caInfo" class="bg-card border-border-muted">
        <header class="border-b border-border flex flex-wrap items-center justify-between gap-2 px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Certificate Authorities</h2>
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
            :disabled="chainDownloading"
            @click="handleDownloadCAChain"
          >
            {{ chainDownloading ? 'Downloading…' : 'Download CA Bundle (.zip)' }}
          </button>
        </header>
        <p v-if="chainError" class="px-4 pt-2 text-xs" role="alert">
          {{ chainError }}
        </p>
        <div class="grid grid-cols-1 gap-px lg:grid-cols-2">
          <article
            v-for="entry in [
              { label: 'Root CA', cert: caInfo.root, filename: 'root_ca.crt' },
              { label: 'Intermediate CA', cert: caInfo.intermediate, filename: 'intermediate_ca.crt' },
            ]"
            :key="entry.label"
            class="px-4 py-3"
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-[10px] uppercase tracking-wide text-muted-foreground">{{ entry.label }}</p>
                <p class="mt-1 truncate text-sm font-medium text-foreground" :title="entry.cert.subject.common_name">
                  {{ entry.cert.subject.common_name }}
                </p>
              </div>
              <button
                type="button"
                class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 shrink-0"
                @click="downloadCertificate(entry.filename, entry.cert.pem)"
              >
                Download .crt
              </button>
            </div>
            <dl class="mt-3 space-y-1.5 text-xs">
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Expires</dt>
                <dd class="text-foreground/80">{{ formatCertDate(entry.cert.not_after) }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Serial</dt>
                <dd
                  class="truncate font-mono text-foreground/80"
                  :title="entry.cert.serial_number"
                >
                  {{ entry.cert.serial_number || '—' }}
                </dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Signature</dt>
                <dd class="text-foreground/80">{{ entry.cert.signature_algorithm || '—' }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Key usage</dt>
                <dd class="text-foreground/80">{{ formatUsageList(entry.cert.key_usages) }}</dd>
              </div>
              <div
                v-if="entry.cert.ext_key_usages && entry.cert.ext_key_usages.length > 0"
                class="grid grid-cols-[5.5rem_1fr] gap-2"
              >
                <dt class="text-muted-foreground">Ext key usage</dt>
                <dd class="text-foreground/80">{{ formatUsageList(entry.cert.ext_key_usages) }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Fingerprint</dt>
                <dd
                  class="truncate font-mono text-foreground/80"
                  :title="entry.cert.fingerprint"
                >
                  {{ shortenFingerprint(entry.cert.fingerprint) }}
                </dd>
              </div>
            </dl>
          </article>
        </div>
      </section>

      <section class="bg-card border-border-muted">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Active Provisioners</h2>
        </header>
        <div
          v-if="caProvisioners.length === 0"
          class="px-4 py-3 text-xs text-muted-foreground"
        >
          No provisioners configured in ca.json.
        </div>
        <div
          v-else
          class="grid grid-cols-1 gap-px md:grid-cols-2 xl:grid-cols-3"
        >
          <article
            v-for="prov in caProvisioners"
            :key="`${prov.type}-${prov.name}`"
            class="px-4 py-3"
          >
            <div class="flex items-center gap-2">
              <StatusBadge :label="prov.type" tone="neutral" />
              <p class="truncate text-sm font-medium text-foreground" :title="prov.name">
                {{ prov.name }}
              </p>
            </div>
            <dl
              v-if="prov.type === 'ACME'"
              class="mt-3 space-y-1.5 text-xs"
            >
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">EAB required</dt>
                <dd class="text-foreground/80">{{ prov.require_eab ? 'Yes' : 'No' }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Challenges</dt>
                <dd class="text-foreground/80">{{ formatUsageList(prov.challenges) }}</dd>
              </div>
            </dl>
            <dl
              v-else-if="prov.type === 'SCEP'"
              class="mt-3 space-y-1.5 text-xs"
            >
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="text-muted-foreground">Challenge</dt>
                <dd class="text-foreground/80">{{ prov.challenge || 'not configured' }}</dd>
              </div>
            </dl>
          </article>
        </div>
      </section>
    </template>
  </div>
</template>
