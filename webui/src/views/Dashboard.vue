<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchCAInfo, fetchCAProvisioners } from '../api/ca'
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

const health = ref<HealthReport | null>(null)
const caInfo = ref<CAInfoResponse | null>(null)
const caProvisioners = ref<CAProvisionerDetail[]>([])
const certificateTotal = ref<number | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const backendDetails = computed(() => parseBackendDetails(health.value?.ca_backend.message))

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const [healthReport, certificateList, caInfoReport, provisionersReport] = await Promise.all([
      fetchHealth(),
      listCertificates(),
      fetchCAInfo(),
      fetchCAProvisioners(),
    ])
    health.value = healthReport
    certificateTotal.value = certificateList.total
    caInfo.value = caInfoReport
    caProvisioners.value = provisionersReport.provisioners
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load dashboard metrics')
  } finally {
    isLoading.value = false
  }
})

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
      class="ui-alert-error"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading server status…</div>

    <template v-else-if="health">
      <section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <article class="ui-surface-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide ui-text-muted">Uptime</p>
          <p class="mt-1 text-lg font-semibold ui-text-primary">{{ health.uptime.human }}</p>
          <p class="text-xs ui-text-muted">{{ health.uptime.seconds }} seconds</p>
        </article>

        <article class="ui-surface-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide ui-text-muted">API</p>
          <div class="mt-2 flex items-center gap-2">
            <StatusBadge :label="health.api.status" :tone="backendTone(health.api.status)" />
            <span class="text-xs ui-text-muted">v{{ health.api.version }}</span>
          </div>
        </article>

        <article class="ui-surface-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide ui-text-muted">CA Backend</p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <StatusBadge :label="health.ca_backend.status" :tone="backendTone(health.ca_backend.status)" />
            <span class="text-xs ui-text-muted">{{ health.ca_backend.engine }}</span>
          </div>
          <dl
            v-if="backendDetails.length > 0"
            class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs"
          >
            <template v-for="detail in backendDetails" :key="detail.label">
              <dt class="ui-text-muted">{{ detail.label }}</dt>
              <dd class="truncate font-mono ui-text-secondary" :title="detail.value">
                {{ detail.value }}
              </dd>
            </template>
          </dl>
        </article>

        <article class="ui-surface-muted px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide ui-text-muted">Certificates</p>
          <p class="mt-1 text-lg font-semibold ui-text-primary">
            {{ certificateTotal ?? '—' }}
          </p>
          <p class="text-xs ui-text-muted">Issued in database</p>
        </article>
      </section>

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Runtime</h2>
        </header>
        <div class="grid gap-px sm:grid-cols-2 lg:grid-cols-4" style="background-color: var(--border-subtle)">
          <div class="px-4 py-3" style="background-color: var(--bg-inset)">
            <p class="text-[10px] uppercase tracking-wide ui-text-muted">Heap in use</p>
            <p class="mt-1 text-sm ui-text-secondary">{{ formatBytes(health.memory.heap_inuse_bytes) }}</p>
          </div>
          <div class="px-4 py-3" style="background-color: var(--bg-inset)">
            <p class="text-[10px] uppercase tracking-wide ui-text-muted">Goroutines</p>
            <p class="mt-1 text-sm ui-text-secondary">{{ health.memory.goroutines }}</p>
          </div>
          <div class="px-4 py-3" style="background-color: var(--bg-inset)">
            <p class="text-[10px] uppercase tracking-wide ui-text-muted">GC cycles</p>
            <p class="mt-1 text-sm ui-text-secondary">{{ health.memory.num_gc }}</p>
          </div>
          <div class="px-4 py-3" style="background-color: var(--bg-inset)">
            <p class="text-[10px] uppercase tracking-wide ui-text-muted">Engine initialized</p>
            <p class="mt-1 text-sm ui-text-secondary">{{ health.ca_backend.initialized ? 'Yes' : 'No' }}</p>
          </div>
        </div>
      </section>

      <section v-if="caInfo" class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Certificate Authorities</h2>
        </header>
        <div class="grid gap-px lg:grid-cols-2" style="background-color: var(--border-subtle)">
          <article
            v-for="entry in [
              { label: 'Root CA', cert: caInfo.root, filename: 'root_ca.crt' },
              { label: 'Intermediate CA', cert: caInfo.intermediate, filename: 'intermediate_ca.crt' },
            ]"
            :key="entry.label"
            class="px-4 py-3"
            style="background-color: var(--bg-inset)"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-[10px] uppercase tracking-wide ui-text-muted">{{ entry.label }}</p>
                <p class="mt-1 truncate text-sm font-medium ui-text-primary" :title="entry.cert.subject.common_name">
                  {{ entry.cert.subject.common_name }}
                </p>
              </div>
              <button
                type="button"
                class="ui-btn-secondary shrink-0"
                @click="downloadCertificate(entry.filename, entry.cert.pem)"
              >
                Download .crt
              </button>
            </div>
            <dl class="mt-3 space-y-1.5 text-xs">
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Expires</dt>
                <dd class="ui-text-secondary">{{ formatCertDate(entry.cert.not_after) }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Serial</dt>
                <dd
                  class="truncate font-mono ui-text-secondary"
                  :title="entry.cert.serial_number"
                >
                  {{ entry.cert.serial_number || '—' }}
                </dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Signature</dt>
                <dd class="ui-text-secondary">{{ entry.cert.signature_algorithm || '—' }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Key usage</dt>
                <dd class="ui-text-secondary">{{ formatUsageList(entry.cert.key_usages) }}</dd>
              </div>
              <div
                v-if="entry.cert.ext_key_usages && entry.cert.ext_key_usages.length > 0"
                class="grid grid-cols-[5.5rem_1fr] gap-2"
              >
                <dt class="ui-text-muted">Ext key usage</dt>
                <dd class="ui-text-secondary">{{ formatUsageList(entry.cert.ext_key_usages) }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Fingerprint</dt>
                <dd
                  class="truncate font-mono ui-text-secondary"
                  :title="entry.cert.fingerprint"
                >
                  {{ shortenFingerprint(entry.cert.fingerprint) }}
                </dd>
              </div>
            </dl>
          </article>
        </div>
      </section>

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Active Provisioners</h2>
        </header>
        <div
          v-if="caProvisioners.length === 0"
          class="px-4 py-3 text-xs ui-text-muted"
        >
          No provisioners configured in ca.json.
        </div>
        <div
          v-else
          class="grid gap-px sm:grid-cols-2 xl:grid-cols-3"
          style="background-color: var(--border-subtle)"
        >
          <article
            v-for="prov in caProvisioners"
            :key="`${prov.type}-${prov.name}`"
            class="px-4 py-3"
            style="background-color: var(--bg-inset)"
          >
            <div class="flex items-center gap-2">
              <StatusBadge :label="prov.type" tone="neutral" />
              <p class="truncate text-sm font-medium ui-text-primary" :title="prov.name">
                {{ prov.name }}
              </p>
            </div>
            <dl
              v-if="prov.type === 'ACME'"
              class="mt-3 space-y-1.5 text-xs"
            >
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">EAB required</dt>
                <dd class="ui-text-secondary">{{ prov.require_eab ? 'Yes' : 'No' }}</dd>
              </div>
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Challenges</dt>
                <dd class="ui-text-secondary">{{ formatUsageList(prov.challenges) }}</dd>
              </div>
            </dl>
            <dl
              v-else-if="prov.type === 'SCEP'"
              class="mt-3 space-y-1.5 text-xs"
            >
              <div class="grid grid-cols-[5.5rem_1fr] gap-2">
                <dt class="ui-text-muted">Challenge</dt>
                <dd class="ui-text-secondary">{{ prov.challenge || 'not configured' }}</dd>
              </div>
            </dl>
          </article>
        </div>
      </section>
    </template>
  </div>
</template>
