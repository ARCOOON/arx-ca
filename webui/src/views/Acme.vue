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
      class="border border-red-900/60 bg-red-950/40 px-3 py-2 text-sm text-red-300"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-zinc-500">Loading ACME configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span class="text-xs text-zinc-500">
          Discovery:
          <code class="border border-zinc-800 bg-zinc-900 px-1 text-emerald-300">GET /api/v1/acme/status</code>
        </span>
      </section>

      <FlatToggle label="ACME server" :enabled="status.enabled" readonly />

      <section class="border border-zinc-800 bg-zinc-900/30">
        <header class="border-b border-zinc-800 px-4 py-2.5">
          <h2 class="text-sm font-semibold text-zinc-50">Endpoints</h2>
        </header>
        <dl class="divide-y divide-zinc-800 text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">Directory URL</dt>
            <dd class="break-all font-mono text-zinc-200">{{ directoryUrl }}</dd>
          </div>
          <div v-if="status.dns_name" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">DNS name</dt>
            <dd class="font-mono text-zinc-200">{{ status.dns_name }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-zinc-500">Provisioner</dt>
            <dd class="font-mono text-zinc-200">{{ status.provisioner }}</dd>
          </div>
        </dl>
      </section>

      <section class="border border-zinc-800 bg-zinc-900/30">
        <header class="border-b border-zinc-800 px-4 py-2.5">
          <h2 class="text-sm font-semibold text-zinc-50">Policy</h2>
        </header>
        <div class="space-y-0 divide-y divide-zinc-800">
          <FlatToggle label="Require EAB" :enabled="status.require_eab" readonly />
          <FlatToggle label="Device attestation" :enabled="status.device_attest_enabled" readonly />
        </div>
        <div v-if="status.challenges?.length" class="border-t border-zinc-800 px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">Challenges</p>
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
