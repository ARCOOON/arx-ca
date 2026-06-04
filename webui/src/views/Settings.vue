<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { resolveApiBaseURL } from '../api/client'
import { fetchHealth } from '../api/health'
import type { HealthReport } from '../types/api'
import { useAuthStore } from '../store/auth'
import { extractApiError } from '../utils/errors'

const authStore = useAuthStore()
const apiBaseUrl = resolveApiBaseURL()
const appOrigin = typeof window !== 'undefined' ? window.location.origin : ''
const health = ref<HealthReport | null>(null)
const sidebarCollapsed = ref(localStorage.getItem('arx_sidebar_collapsed') === 'true')
const isLoading = ref(true)
const errorMessage = ref('')

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    health.value = await fetchHealth()
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load server settings')
  } finally {
    isLoading.value = false
  }
})

function persistSidebarPreference(): void {
  localStorage.setItem('arx_sidebar_collapsed', String(sidebarCollapsed.value))
  window.location.reload()
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

    <section class="border border-zinc-800 bg-zinc-900/30">
      <header class="border-b border-zinc-800 px-4 py-2.5">
        <h2 class="text-sm font-semibold text-zinc-50">Session</h2>
      </header>
      <dl class="divide-y divide-zinc-800 text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">Roles</dt>
          <dd class="text-zinc-200">
            {{ authStore.roles.length > 0 ? authStore.roles.join(', ') : 'Administrator' }}
          </dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">JWT</dt>
          <dd class="font-mono text-[11px] text-zinc-400">
            {{ authStore.token ? `${authStore.token.slice(0, 24)}…` : 'Not authenticated' }}
          </dd>
        </div>
      </dl>
    </section>

    <section class="border border-zinc-800 bg-zinc-900/30">
      <header class="border-b border-zinc-800 px-4 py-2.5">
        <h2 class="text-sm font-semibold text-zinc-50">API client</h2>
      </header>
      <dl class="divide-y divide-zinc-800 text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">Base URL</dt>
          <dd class="break-all font-mono text-zinc-200">{{ apiBaseUrl }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">Origin</dt>
          <dd class="break-all font-mono text-zinc-200">{{ appOrigin }}</dd>
        </div>
      </dl>
    </section>

    <section class="border border-zinc-800 bg-zinc-900/30">
      <header class="border-b border-zinc-800 px-4 py-2.5">
        <h2 class="text-sm font-semibold text-zinc-50">Interface</h2>
      </header>
      <div class="flex items-center justify-between gap-4 px-4 py-3 text-xs">
        <div>
          <p class="text-zinc-300">Collapsed sidebar by default</p>
          <p class="mt-0.5 text-zinc-500">Stored in local storage; reload applies the layout.</p>
        </div>
        <label class="inline-flex items-center gap-2 text-zinc-400">
          <input v-model="sidebarCollapsed" type="checkbox" class="accent-emerald-500" />
          Collapsed
        </label>
      </div>
      <div class="border-t border-zinc-800 px-4 py-3">
        <button
          type="button"
          class="border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 hover:border-zinc-600"
          @click="persistSidebarPreference"
        >
          Save UI preference
        </button>
      </div>
    </section>

    <section v-if="!isLoading && health" class="border border-zinc-800 bg-zinc-900/30">
      <header class="border-b border-zinc-800 px-4 py-2.5">
        <h2 class="text-sm font-semibold text-zinc-50">Server</h2>
      </header>
      <dl class="divide-y divide-zinc-800 text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">API status</dt>
          <dd class="text-zinc-200">{{ health.api.status }} ({{ health.api.version }})</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-zinc-500">CA engine</dt>
          <dd class="text-zinc-200">{{ health.ca_backend.engine }} — {{ health.ca_backend.status }}</dd>
        </div>
      </dl>
    </section>
  </div>
</template>
