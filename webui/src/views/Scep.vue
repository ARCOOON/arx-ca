<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchScepStatus } from '../api/scep'
import type { ScepStatus } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { extractApiError } from '../utils/errors'

const { showApiHints } = usePreferences()

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
      class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-muted-foreground">Loading SCEP configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span v-if="showApiHints" class="text-xs text-muted-foreground">
          Discovery:
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/scep/status</code>
        </span>
      </section>

      <FlatToggle label="SCEP enrollment" :enabled="status.enabled" readonly />

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Base URL</dt>
            <dd class="break-all font-mono text-foreground/80">{{ baseUrl }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Provisioner</dt>
            <dd class="font-mono text-foreground/80">{{ status.provisioner }}</dd>
          </div>
          <div v-if="status.challenge_hint" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Challenge</dt>
            <dd class="text-foreground/80">{{ status.challenge_hint }}</dd>
          </div>
        </dl>
      </section>
    </template>
  </div>
</template>
