<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchAcmeStatus } from '../api/acme'
import type { AcmeStatus } from '../types/api'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { extractApiError } from '../utils/errors'

const status = ref<AcmeStatus | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const directoryUrl = computed(() => {
  if (status.value?.directory_url) {
    return status.value.directory_url
  }
  if (typeof window !== 'undefined') {
    return `${window.location.origin}/acme/directory`
  }
  return 'https://<host>:8443/acme/directory'
})

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    status.value = await fetchAcmeStatus()
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load ACME status')
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

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading ACME configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span class="text-xs ui-text-muted">
          Discovery:
          <code class="ui-code">GET /api/v1/acme/status</code>
        </span>
      </section>

      <FlatToggle label="ACME server" :enabled="status.enabled" readonly />

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Directory URL</dt>
            <dd class="break-all font-mono ui-text-secondary">{{ directoryUrl }}</dd>
          </div>
          <div v-if="status.dns_name" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">DNS name</dt>
            <dd class="font-mono ui-text-secondary">{{ status.dns_name }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Provisioner</dt>
            <dd class="font-mono ui-text-secondary">{{ status.provisioner }}</dd>
          </div>
        </dl>
      </section>

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Policy</h2>
        </header>
        <div class="ui-divide space-y-0">
          <FlatToggle label="Require EAB" :enabled="status.require_eab" readonly />
          <FlatToggle label="Device attestation" :enabled="status.device_attest_enabled" readonly />
        </div>
        <div v-if="status.challenges?.length" class="ui-border-t px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide ui-text-muted">Challenges</p>
          <div class="mt-2 flex flex-wrap gap-1.5">
            <StatusBadge
              v-for="challenge in status.challenges"
              :key="challenge"
              :label="challenge"
              tone="neutral"
            />
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
