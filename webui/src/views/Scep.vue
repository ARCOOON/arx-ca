<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchScepStatus } from '../api/scep'
import type { ScepStatus } from '../types/api'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { extractApiError } from '../utils/errors'

const status = ref<ScepStatus | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const baseUrl = computed(() => {
  if (status.value?.base_url) {
    return status.value.base_url
  }
  if (typeof window !== 'undefined') {
    return `${window.location.origin}/scep/scep`
  }
  return 'https://<host>:8443/scep/scep'
})

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    status.value = await fetchScepStatus()
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load SCEP status')
  } finally {
    isLoading.value = false
  }
})
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

    <div v-if="isLoading" class="text-sm text-zinc-500">Loading SCEP configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span class="text-xs text-zinc-500">
          Discovery:
          <code class="border border-zinc-800 bg-zinc-900 px-1 text-emerald-300">GET /api/v1/scep/status</code>
        </span>
      </section>

      <FlatToggle label="SCEP enrollment" :enabled="status.enabled" readonly />

      <section class="border border-zinc-800 bg-zinc-900/30">
        <header class="border-b border-zinc-800 px-4 py-2.5">
          <h2 class="text-sm font-semibold text-zinc-50">Endpoints</h2>
        </header>
        <dl class="divide-y divide-zinc-800 text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">Base URL</dt>
            <dd class="break-all font-mono text-zinc-200">{{ baseUrl }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">Provisioner</dt>
            <dd class="font-mono text-zinc-200">{{ status.provisioner }}</dd>
          </div>
          <div v-if="status.challenge_hint" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">Challenge</dt>
            <dd class="text-zinc-200">{{ status.challenge_hint }}</dd>
          </div>
        </dl>
      </section>
    </template>
  </div>
</template>
