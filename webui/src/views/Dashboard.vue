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
      class="border border-red-900/60 bg-red-950/40 px-3 py-2 text-sm text-red-300"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-zinc-500">Loading server status…</div>

    <template v-else-if="health">
      <section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <article class="border border-zinc-800 bg-zinc-900/30 px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">Uptime</p>
          <p class="mt-1 text-lg font-semibold text-zinc-50">{{ health.uptime.human }}</p>
          <p class="text-xs text-zinc-500">{{ health.uptime.seconds }} seconds</p>
        </article>

        <article class="border border-zinc-800 bg-zinc-900/30 px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">API</p>
          <div class="mt-2 flex items-center gap-2">
            <StatusBadge :label="health.api.status" :tone="backendTone(health.api.status)" />
            <span class="text-xs text-zinc-500">v{{ health.api.version }}</span>
          </div>
        </article>

        <article class="border border-zinc-800 bg-zinc-900/30 px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">CA Backend</p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <StatusBadge :label="health.ca_backend.status" :tone="backendTone(health.ca_backend.status)" />
            <span class="text-xs text-zinc-500">{{ health.ca_backend.engine }}</span>
          </div>
          <p v-if="health.ca_backend.message" class="mt-1 text-xs text-zinc-500">
            {{ health.ca_backend.message }}
          </p>
        </article>

        <article class="border border-zinc-800 bg-zinc-900/30 px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">Certificates</p>
          <p class="mt-1 text-lg font-semibold text-zinc-50">
            {{ certificateTotal ?? '—' }}
          </p>
          <p class="text-xs text-zinc-500">Issued in database</p>
        </article>
      </section>

      <section class="border border-zinc-800 bg-zinc-900/30">
        <header class="border-b border-zinc-800 px-4 py-2.5">
          <h2 class="text-sm font-semibold text-zinc-50">Runtime</h2>
        </header>
        <div class="grid gap-px bg-zinc-800 sm:grid-cols-2 lg:grid-cols-4">
          <div class="bg-zinc-950 px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-zinc-500">Heap in use</p>
            <p class="mt-1 text-sm text-zinc-200">{{ formatBytes(health.memory.heap_inuse_bytes) }}</p>
          </div>
          <div class="bg-zinc-950 px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-zinc-500">Goroutines</p>
            <p class="mt-1 text-sm text-zinc-200">{{ health.memory.goroutines }}</p>
          </div>
          <div class="bg-zinc-950 px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-zinc-500">GC cycles</p>
            <p class="mt-1 text-sm text-zinc-200">{{ health.memory.num_gc }}</p>
          </div>
          <div class="bg-zinc-950 px-4 py-3">
            <p class="text-[10px] uppercase tracking-wide text-zinc-500">Engine initialized</p>
            <p class="mt-1 text-sm text-zinc-200">{{ health.ca_backend.initialized ? 'Yes' : 'No' }}</p>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
