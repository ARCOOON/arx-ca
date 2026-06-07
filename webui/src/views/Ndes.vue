<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchNdesStatus } from '../api/ndes'
import type { NdesStatus } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { extractApiError } from '../utils/errors'

const { showApiHints } = usePreferences()

const status = ref<NdesStatus | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const scepEndpoint = computed(() => {
  if (status.value?.scep_endpoint) {
    return status.value.scep_endpoint
  }
  if (typeof window !== 'undefined') {
    return `${window.location.origin}/certsrv/mscep/mscep.dll`
  }
  return 'https://<host>:8443/certsrv/mscep/mscep.dll'
})

const adminEndpoint = computed(() => {
  if (status.value?.admin_endpoint) {
    return status.value.admin_endpoint
  }
  if (typeof window !== 'undefined') {
    return `${window.location.origin}/certsrv/mscep_admin/mscep_admin.dll`
  }
  return 'https://<host>:8443/certsrv/mscep_admin/mscep_admin.dll'
})

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    status.value = await fetchNdesStatus()
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load NDES status')
  } finally {
    isLoading.value = false
  }
})
</script>

<template>
  <div class="space-y-4">
    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading NDES configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <StatusBadge
          v-if="status.adcs_compatible"
          label="AD CS compatible"
          tone="valid"
        />
        <span v-if="showApiHints" class="text-xs ui-text-muted">
          Discovery:
          <code class="ui-code">GET /api/v1/ndes/status</code>
        </span>
      </section>

      <FlatToggle label="NDES enrollment" :enabled="status.enabled" readonly />

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">SCEP endpoint</dt>
            <dd class="break-all font-mono ui-text-secondary">{{ scepEndpoint }}</dd>
          </div>
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Admin endpoint</dt>
            <dd class="break-all font-mono ui-text-secondary">{{ adminEndpoint }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="status.connectors?.length" class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Connectors</h2>
        </header>
        <ul class="ui-divide px-4 py-3 text-xs">
          <li
            v-for="connector in status.connectors"
            :key="connector"
            class="py-1.5 font-mono ui-text-secondary"
          >
            {{ connector }}
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>
