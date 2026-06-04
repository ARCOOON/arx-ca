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
      class="ui-alert-error"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">Session</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Roles</dt>
          <dd class="ui-text-secondary">
            {{ authStore.roles.length > 0 ? authStore.roles.join(', ') : 'Administrator' }}
          </dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">JWT</dt>
          <dd class="font-mono text-[11px] ui-text-muted">
            {{ authStore.token ? `${authStore.token.slice(0, 24)}…` : 'Not authenticated' }}
          </dd>
        </div>
      </dl>
    </section>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">API client</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Base URL</dt>
          <dd class="break-all font-mono ui-text-secondary">{{ apiBaseUrl }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Origin</dt>
          <dd class="break-all font-mono ui-text-secondary">{{ appOrigin }}</dd>
        </div>
      </dl>
    </section>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">Interface</h2>
      </header>
      <div class="flex items-center justify-between gap-4 px-4 py-3 text-xs">
        <div>
          <p class="ui-text-secondary">Collapsed sidebar by default</p>
          <p class="mt-0.5 ui-text-muted">Stored in local storage; reload applies the layout.</p>
        </div>
        <button
          type="button"
          class="ui-theme-toggle shrink-0"
          :data-active="sidebarCollapsed"
          :aria-pressed="sidebarCollapsed"
          aria-label="Collapsed sidebar by default"
          @click="sidebarCollapsed = !sidebarCollapsed"
        >
          <span class="ui-theme-toggle-thumb" />
        </button>
      </div>
      <div class="ui-border-t px-4 py-3">
        <button type="button" class="ui-btn-secondary" @click="persistSidebarPreference">
          Save UI preference
        </button>
      </div>
    </section>

    <section v-if="!isLoading && health" class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">Server</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">API status</dt>
          <dd class="ui-text-secondary">{{ health.api.status }} ({{ health.api.version }})</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">CA engine</dt>
          <dd class="ui-text-secondary">{{ health.ca_backend.engine }} — {{ health.ca_backend.status }}</dd>
        </div>
      </dl>
    </section>
  </div>
</template>
