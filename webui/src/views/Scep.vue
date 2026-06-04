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
      class="ui-alert-error"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading SCEP configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span class="text-xs ui-text-muted">
          Discovery:
          <code class="ui-code">GET /api/v1/scep/status</code>
        </span>
      </section>

      <FlatToggle label="SCEP enrollment" :enabled="status.enabled" readonly />

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Base URL</dt>
            <dd class="break-all font-mono ui-text-secondary">{{ baseUrl }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Provisioner</dt>
            <dd class="font-mono ui-text-secondary">{{ status.provisioner }}</dd>
          </div>
          <div v-if="status.challenge_hint" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Challenge</dt>
            <dd class="ui-text-secondary">{{ status.challenge_hint }}</dd>
          </div>
        </dl>
      </section>
    </template>
  </div>
</template>
