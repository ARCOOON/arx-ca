<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { createServiceAccount } from '../api/auth'
import { fetchRootCertPEM, fetchIntermediateCertPEM } from '../api/ca'
import { resolveApiBaseURL } from '../api/client'
import { fetchHealth } from '../api/health'
import { listPublicCertificates } from '../api/public'
import type { HealthReport, ServiceAccountResponse } from '../types/api'
import { useNotificationLayout, type NotificationLayoutStyle } from '../composables/useNotificationLayout'
import { usePreferences } from '../composables/usePreferences'
import { useAuthStore } from '../store/auth'
import { copyToClipboard } from '../utils/clipboard'
import { downloadTextFile } from '../utils/download'
import { extractApiError } from '../utils/errors'
import { formatBytes } from '../utils/format'

const authStore = useAuthStore()
const { showApiHints, setShowApiHints } = usePreferences()
const { layoutStyle: notificationLayoutStyle, setLayoutStyle: setNotificationLayoutStyle } =
  useNotificationLayout()

const CANONICAL_ROLES = ['SuperAdmin', 'CA-Admin', 'Operator', 'Auditor'] as const

const notificationLayoutOptions: { value: NotificationLayoutStyle; label: string; description: string }[] = [
  {
    value: 'drawer',
    label: 'Drawer',
    description: 'Slide-out panel anchored to the right edge of the viewport.',
  },
  {
    value: 'overlay',
    label: 'Overlay',
    description: 'Floating card dropdown below the notification bell in the top bar.',
  },
]
const apiBaseUrl = resolveApiBaseURL()
const appOrigin = typeof window !== 'undefined' ? window.location.origin : ''
const health = ref<HealthReport | null>(null)
const sidebarCollapsed = ref(localStorage.getItem('arx_sidebar_collapsed') === 'true')
const isLoading = ref(true)
const errorMessage = ref('')

const publicCertTotal = ref<number | null>(null)
const publicLoading = ref(false)
const publicError = ref('')

const saName = ref('')
const saRoles = ref('')
const saCreating = ref(false)
const saError = ref('')
const saResult = ref<ServiceAccountResponse | null>(null)
const saCopied = ref(false)

const rootDownloading = ref(false)
const intermediateDownloading = ref(false)
const certDownloadError = ref('')

const isSuperAdmin = computed(() => authStore.roles.includes('SuperAdmin'))

const publicListUrl = computed(() => `${appOrigin}${apiBaseUrl}/public/certificates`)

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

  void loadPublicCertCount()
})

async function loadPublicCertCount(): Promise<void> {
  publicLoading.value = true
  publicError.value = ''

  try {
    const response = await listPublicCertificates()
    publicCertTotal.value = response.total
  } catch (error) {
    publicError.value = extractApiError(error, 'Failed to load public certificate count')
  } finally {
    publicLoading.value = false
  }
}

function persistSidebarPreference(): void {
  localStorage.setItem('arx_sidebar_collapsed', String(sidebarCollapsed.value))
  window.location.reload()
}

async function downloadRootCert(): Promise<void> {
  rootDownloading.value = true
  certDownloadError.value = ''

  try {
    const pem = await fetchRootCertPEM()
    downloadTextFile('root_ca.pem', pem)
  } catch (error) {
    certDownloadError.value = extractApiError(error, 'Failed to download root CA certificate')
  } finally {
    rootDownloading.value = false
  }
}

async function downloadIntermediateCert(): Promise<void> {
  intermediateDownloading.value = true
  certDownloadError.value = ''

  try {
    const pem = await fetchIntermediateCertPEM()
    downloadTextFile('intermediate_ca.pem', pem)
  } catch (error) {
    certDownloadError.value = extractApiError(error, 'Failed to download intermediate CA certificate')
  } finally {
    intermediateDownloading.value = false
  }
}

async function submitServiceAccount(): Promise<void> {
  saError.value = ''
  saResult.value = null
  saCopied.value = false

  const name = saName.value.trim()
  if (!name) {
    saError.value = 'Service account name is required.'
    return
  }

  saCreating.value = true

  try {
    const roles = saRoles.value
      .split(',')
      .map((role) => role.trim())
      .filter(Boolean)

    saResult.value = await createServiceAccount({
      name,
      roles: roles.length > 0 ? roles : undefined,
    })
    saName.value = ''
    saRoles.value = ''
  } catch (error) {
    saError.value = extractApiError(error, 'Failed to create service account')
  } finally {
    saCreating.value = false
  }
}

async function copyApiKey(): Promise<void> {
  if (!saResult.value?.api_key) {
    return
  }
  await copyToClipboard(saResult.value.api_key)
  saCopied.value = true
}

function dismissServiceAccountResult(): void {
  saResult.value = null
  saCopied.value = false
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="errorMessage" class="ui-alert-error" role="alert">
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
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Uptime</dt>
          <dd class="ui-text-secondary">{{ health.uptime.human }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Heap in use</dt>
          <dd class="ui-text-secondary">{{ formatBytes(health.memory.heap_inuse_bytes) }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Goroutines</dt>
          <dd class="ui-text-secondary">{{ health.memory.goroutines }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Engine initialized</dt>
          <dd class="ui-text-secondary">{{ health.ca_backend.initialized ? 'Yes' : 'No' }}</dd>
        </div>
      </dl>
    </section>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">CA Certificates</h2>
        <p class="mt-0.5 text-xs ui-text-muted">Public PEM downloads for trust store configuration.</p>
      </header>
      <div class="flex flex-wrap gap-2 px-4 py-3">
        <button type="button" class="ui-btn-secondary" :disabled="rootDownloading" @click="downloadRootCert">
          {{ rootDownloading ? 'Downloading…' : 'Download Root CA (.pem)' }}
        </button>
        <button
          type="button"
          class="ui-btn-secondary"
          :disabled="intermediateDownloading"
          @click="downloadIntermediateCert"
        >
          {{ intermediateDownloading ? 'Downloading…' : 'Download Intermediate CA (.pem)' }}
        </button>
      </div>
      <p v-if="certDownloadError" class="px-4 pb-3 text-xs" style="color: var(--danger-text)" role="alert">
        {{ certDownloadError }}
      </p>
    </section>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">Public API</h2>
        <p class="mt-0.5 text-xs ui-text-muted">Unauthenticated read-only endpoints for agents and clients.</p>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Certificate list</dt>
          <dd class="break-all font-mono ui-text-secondary">{{ publicListUrl }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Certificate by serial</dt>
          <dd class="break-all font-mono ui-text-secondary">
            {{ publicListUrl }}/{serial}
          </dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">OCSP responder</dt>
          <dd class="break-all font-mono ui-text-secondary">{{ appOrigin }}/ocsp</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="ui-text-muted">Published certs</dt>
          <dd class="ui-text-secondary">
            <span v-if="publicLoading">Loading…</span>
            <span v-else-if="publicError" style="color: var(--danger-text)">{{ publicError }}</span>
            <span v-else>{{ publicCertTotal ?? '—' }} visible</span>
          </dd>
        </div>
      </dl>
    </section>

    <section v-if="isSuperAdmin" class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">Service Accounts</h2>
        <p v-if="showApiHints" class="mt-0.5 text-xs ui-text-muted">
          Create API keys via
          <code class="ui-code">POST /api/v1/auth/service-accounts</code>
        </p>
      </header>
      <div class="space-y-3 px-4 py-3">
        <div v-if="saError" class="ui-alert-error text-xs" role="alert">{{ saError }}</div>

        <div v-if="saResult" class="ui-alert-warning space-y-3" role="alert">
          <p class="font-semibold">Save the API key now — it will not be shown again.</p>
          <dl class="space-y-2 text-xs">
            <div>
              <dt class="font-medium ui-text-muted">Name</dt>
              <dd class="mt-0.5 ui-text-primary">{{ saResult.name }}</dd>
            </div>
            <div>
              <dt class="font-medium ui-text-muted">ID</dt>
              <dd class="mt-0.5 font-mono ui-text-primary">{{ saResult.id }}</dd>
            </div>
            <div>
              <dt class="font-medium ui-text-muted">Roles</dt>
              <dd class="mt-0.5 ui-text-primary">{{ saResult.roles.join(', ') || 'default' }}</dd>
            </div>
            <div>
              <dt class="font-medium ui-text-muted">API Key</dt>
              <dd class="mt-0.5 break-all font-mono text-sm ui-text-primary">{{ saResult.api_key }}</dd>
            </div>
          </dl>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="ui-btn-primary" @click="copyApiKey">
              {{ saCopied ? 'Copied' : 'Copy API Key' }}
            </button>
            <button type="button" class="ui-btn-secondary" @click="dismissServiceAccountResult">Dismiss</button>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="sa-name">Name</label>
            <input id="sa-name" v-model="saName" type="text" class="ui-input mt-1.5" autocomplete="off" />
          </div>
          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="sa-roles">
              Roles (comma-separated, optional)
            </label>
            <input
              id="sa-roles"
              v-model="saRoles"
              type="text"
              list="available-roles"
              class="ui-input mt-1.5"
              placeholder="Operator"
              autocomplete="off"
            />
            <datalist id="available-roles">
              <option v-for="role in CANONICAL_ROLES" :key="role" :value="role" />
            </datalist>
          </div>
        </div>
        <button type="button" class="ui-btn-primary" :disabled="saCreating" @click="submitServiceAccount">
          {{ saCreating ? 'Creating…' : 'Create Service Account' }}
        </button>
      </div>
    </section>

    <section class="ui-surface-muted">
      <header class="ui-border-b px-4 py-2.5">
        <h2 class="text-sm font-semibold ui-text-primary">UI Preferences</h2>
        <p class="mt-0.5 text-xs ui-text-muted">Operator layout and notification presentation settings.</p>
      </header>

      <div class="ui-divide text-xs">
        <div class="space-y-3 px-4 py-3">
          <div>
            <p class="font-medium ui-text-secondary">Notification Center Layout</p>
            <p class="mt-0.5 ui-text-muted">
              Choose between a full-height side drawer or a compact overlay near the bell icon. Applies
              immediately.
            </p>
          </div>
          <div
            class="grid gap-2 sm:grid-cols-2"
            role="radiogroup"
            aria-label="Notification Center Layout"
          >
            <button
              v-for="option in notificationLayoutOptions"
              :key="option.value"
              type="button"
              class="rounded-[var(--radius-control)] border px-3 py-2.5 text-left transition-all duration-300"
              :class="
                notificationLayoutStyle === option.value
                  ? 'border-[var(--accent-border)] bg-[var(--accent-surface)]'
                  : 'border-[var(--border-color)] bg-[var(--bg-surface)] hover:border-[var(--border-subtle)]'
              "
              role="radio"
              :aria-checked="notificationLayoutStyle === option.value"
              @click="setNotificationLayoutStyle(option.value)"
            >
              <span class="block font-medium ui-text-primary">{{ option.label }}</span>
              <span class="mt-0.5 block ui-text-muted">{{ option.description }}</span>
            </button>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="ui-text-secondary">Show Developer API Hints</p>
            <p class="mt-0.5 ui-text-muted">
              Display route badges and documentation references across all views.
            </p>
          </div>
          <button
            type="button"
            class="ui-theme-toggle shrink-0"
            :data-active="showApiHints"
            :aria-pressed="showApiHints"
            aria-label="Show Developer API Hints"
            @click="setShowApiHints(!showApiHints)"
          >
            <span class="ui-theme-toggle-thumb" />
          </button>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
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
      </div>

      <div class="ui-border-t px-4 py-3">
        <button type="button" class="ui-btn-secondary" @click="persistSidebarPreference">
          Save sidebar preference
        </button>
      </div>
    </section>
  </div>
</template>
