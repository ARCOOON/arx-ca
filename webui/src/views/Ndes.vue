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
    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-muted-foreground">Loading NDES configuration…</div>

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
        <span v-if="showApiHints" class="text-xs text-muted-foreground">
          Discovery:
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/ndes/status</code>
        </span>
      </section>

      <FlatToggle label="NDES enrollment" :enabled="status.enabled" readonly />

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">SCEP endpoint</dt>
            <dd class="break-all font-mono text-foreground/80">{{ scepEndpoint }}</dd>
          </div>
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Admin endpoint</dt>
            <dd class="break-all font-mono text-foreground/80">{{ adminEndpoint }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="status.connectors?.length" class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Connectors</h2>
        </header>
        <ul class="ui-divide px-4 py-3 text-xs">
          <li
            v-for="connector in status.connectors"
            :key="connector"
            class="py-1.5 font-mono text-foreground/80"
          >
            {{ connector }}
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>
