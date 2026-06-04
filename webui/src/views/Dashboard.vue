<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchHealth } from '../api/health'
import { listCertificates } from '../api/certificates'
import type { HealthReport } from '../types/api'
import { extractApiError } from '../utils/errors'
import { formatBytes } from '../utils/format'
import StatusBadge from '../components/ui/StatusBadge.vue'

const health = ref<HealthReport | null>(null)
const certificateTotal = ref<number | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const [healthReport, certificateList] = await Promise.all([
      fetchHealth(),
      listCertificates(),
    ])
    health.value = healthReport
    certificateTotal.value = certificateList.total
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
          <p v-if="health.ca_backend.message" class="mt-1 text-xs ui-text-muted">
            {{ health.ca_backend.message }}
          </p>
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
    </template>
  </div>
</template>
