<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { createServiceAccount } from '../api/auth'
import { fetchRootCertPEM, fetchIntermediateCertPEM } from '../api/ca'
import { fetchSettingsConfig, updateSettingsConfig } from '../api/config'
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
import { Switch } from '@/components/ui/switch'

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
const canManageSettings = computed(
  () => authStore.roles.includes('SuperAdmin') || authStore.roles.includes('CA-Admin'),
)

const UPDATE_CHANNELS = ['main', 'stable', 'develop', 'beta'] as const

const updaterEnabled = ref(true)
const updaterChannel = ref('main')
const updaterNotifyOnly = ref(true)
const updaterCheckInterval = ref('1h')
const updaterViewChangelogAfterUpdate = ref(true)
const updaterLoading = ref(false)
const updaterSaving = ref(false)
const updaterError = ref('')
const updaterSaved = ref(false)

const updateActionOptions = [
  { value: true, label: 'Notify Only', description: 'Emit operator notifications when a newer release is available.' },
  {
    value: false,
    label: 'Download & Apply',
    description: 'Download the release, swap the binary, and gracefully restart the server.',
  },
] as const

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

  if (canManageSettings.value) {
    void loadUpdaterSettings()
  }
})

async function loadUpdaterSettings(): Promise<void> {
  updaterLoading.value = true
  updaterError.value = ''
  updaterSaved.value = false

  try {
    const config = await fetchSettingsConfig()
    updaterEnabled.value = config.updater.enabled
    updaterChannel.value = config.updater.channel || 'main'
    updaterNotifyOnly.value = config.updater.notify_only
    updaterCheckInterval.value = config.updater.check_interval || '1h'
    updaterViewChangelogAfterUpdate.value = config.updater.view_changelog_after_update ?? true
  } catch (error) {
    updaterError.value = extractApiError(error, 'Failed to load system configuration')
  } finally {
    updaterLoading.value = false
  }
}

async function saveUpdaterSettings(): Promise<void> {
  updaterSaving.value = true
  updaterError.value = ''
  updaterSaved.value = false

  const interval = updaterCheckInterval.value.trim()
  if (!interval) {
    updaterError.value = 'Check interval is required (e.g. 1h, 24h).'
    updaterSaving.value = false
    return
  }

  try {
    const config = await updateSettingsConfig({
      updater: {
        enabled: updaterEnabled.value,
        channel: updaterChannel.value,
        notify_only: updaterNotifyOnly.value,
        check_interval: interval,
        view_changelog_after_update: updaterViewChangelogAfterUpdate.value,
      },
    })
    updaterEnabled.value = config.updater.enabled
    updaterChannel.value = config.updater.channel
    updaterNotifyOnly.value = config.updater.notify_only
    updaterCheckInterval.value = config.updater.check_interval
    updaterViewChangelogAfterUpdate.value = config.updater.view_changelog_after_update
    updaterSaved.value = true
  } catch (error) {
    updaterError.value = extractApiError(error, 'Failed to save auto-updater settings')
  } finally {
    updaterSaving.value = false
  }
}

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
    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">Session</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Roles</dt>
          <dd class="text-foreground/80">
            {{ authStore.roles.length > 0 ? authStore.roles.join(', ') : 'Administrator' }}
          </dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">JWT</dt>
          <dd class="font-mono text-[11px] text-muted-foreground">
            {{ authStore.token ? `${authStore.token.slice(0, 24)}…` : 'Not authenticated' }}
          </dd>
        </div>
      </dl>
    </section>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">API client</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Base URL</dt>
          <dd class="break-all font-mono text-foreground/80">{{ apiBaseUrl }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Origin</dt>
          <dd class="break-all font-mono text-foreground/80">{{ appOrigin }}</dd>
        </div>
      </dl>
    </section>

    <section v-if="!isLoading && health" class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">Server</h2>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">API status</dt>
          <dd class="text-foreground/80">{{ health.api.status }} ({{ health.api.version }})</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">CA engine</dt>
          <dd class="text-foreground/80">{{ health.ca_backend.engine }} — {{ health.ca_backend.status }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Uptime</dt>
          <dd class="text-foreground/80">{{ health.uptime.human }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Heap in use</dt>
          <dd class="text-foreground/80">{{ formatBytes(health.memory.heap_inuse_bytes) }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Goroutines</dt>
          <dd class="text-foreground/80">{{ health.memory.goroutines }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Engine initialized</dt>
          <dd class="text-foreground/80">{{ health.ca_backend.initialized ? 'Yes' : 'No' }}</dd>
        </div>
      </dl>
    </section>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">CA Certificates</h2>
        <p class="mt-0.5 text-xs text-muted-foreground">Public PEM downloads for trust store configuration.</p>
      </header>
      <div class="flex flex-wrap gap-2 px-4 py-3">
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="rootDownloading" @click="downloadRootCert">
          {{ rootDownloading ? 'Downloading…' : 'Download Root CA (.pem)' }}
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
          :disabled="intermediateDownloading"
          @click="downloadIntermediateCert"
        >
          {{ intermediateDownloading ? 'Downloading…' : 'Download Intermediate CA (.pem)' }}
        </button>
      </div>
      <p v-if="certDownloadError" class="px-4 pb-3 text-xs" role="alert">
        {{ certDownloadError }}
      </p>
    </section>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">Public API</h2>
        <p class="mt-0.5 text-xs text-muted-foreground">Unauthenticated read-only endpoints for agents and clients.</p>
      </header>
      <dl class="ui-divide text-xs">
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Certificate list</dt>
          <dd class="break-all font-mono text-foreground/80">{{ publicListUrl }}</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Certificate by serial</dt>
          <dd class="break-all font-mono text-foreground/80">
            {{ publicListUrl }}/{serial}
          </dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">OCSP responder</dt>
          <dd class="break-all font-mono text-foreground/80">{{ appOrigin }}/ocsp</dd>
        </div>
        <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
          <dt class="text-muted-foreground">Published certs</dt>
          <dd class="text-foreground/80">
            <span v-if="publicLoading">Loading…</span>
            <span v-else-if="publicError">{{ publicError }}</span>
            <span v-else>{{ publicCertTotal ?? '—' }} visible</span>
          </dd>
        </div>
      </dl>
    </section>

    <section v-if="isSuperAdmin" class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">Service Accounts</h2>
        <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
          Create API keys via
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/auth/service-accounts</code>
        </p>
      </header>
      <div class="space-y-3 px-4 py-3">
        <div v-if="saError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">{{ saError }}</div>

        <div v-if="saResult" class="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400 space-y-3" role="alert">
          <p class="font-semibold">Save the API key now — it will not be shown again.</p>
          <dl class="space-y-2 text-xs">
            <div>
              <dt class="font-medium text-muted-foreground">Name</dt>
              <dd class="mt-0.5 text-foreground">{{ saResult.name }}</dd>
            </div>
            <div>
              <dt class="font-medium text-muted-foreground">ID</dt>
              <dd class="mt-0.5 font-mono text-foreground">{{ saResult.id }}</dd>
            </div>
            <div>
              <dt class="font-medium text-muted-foreground">Roles</dt>
              <dd class="mt-0.5 text-foreground">{{ saResult.roles.join(', ') || 'default' }}</dd>
            </div>
            <div>
              <dt class="font-medium text-muted-foreground">API Key</dt>
              <dd class="mt-0.5 break-all font-mono text-sm text-foreground">{{ saResult.api_key }}</dd>
            </div>
          </dl>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="copyApiKey">
              {{ saCopied ? 'Copied' : 'Copy API Key' }}
            </button>
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="dismissServiceAccountResult">Dismiss</button>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium text-foreground/80" for="sa-name">Name</label>
            <input id="sa-name" v-model="saName" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" autocomplete="off" />
          </div>
          <div>
            <label class="block text-xs font-medium text-foreground/80" for="sa-roles">
              Roles (comma-separated, optional)
            </label>
            <input
              id="sa-roles"
              v-model="saRoles"
              type="text"
              list="available-roles"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              placeholder="Operator"
              autocomplete="off"
            />
            <datalist id="available-roles">
              <option v-for="role in CANONICAL_ROLES" :key="role" :value="role" />
            </datalist>
          </div>
        </div>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="saCreating" @click="submitServiceAccount">
          {{ saCreating ? 'Creating…' : 'Create Service Account' }}
        </button>
      </div>
    </section>

    <section v-if="canManageSettings" class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">Auto-Updater</h2>
        <p class="mt-0.5 text-xs text-muted-foreground">
          Background GitHub release checker. Changes are written to
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">server.yaml</code>
          on save.
        </p>
        <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET/PUT /api/v1/settings/config</code>
        </p>
      </header>

      <div class="ui-divide text-xs">
        <div v-if="updaterError" class="px-4 py-3">
          <p class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">{{ updaterError }}</p>
        </div>
        <div v-if="updaterSaved" class="px-4 py-3">
          <p class="rounded-lg border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-foreground" role="status">Auto-updater settings saved to server.yaml.</p>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="font-medium text-foreground/80">Enable Auto-Updater</p>
            <p class="mt-0.5 text-muted-foreground">Poll GitHub releases on the configured channel and interval.</p>
          </div>
          <Switch
            :checked="updaterEnabled"
            :disabled="updaterLoading || updaterSaving"
            aria-label="Enable Auto-Updater"
            @update:checked="updaterEnabled = $event"
          />
        </div>

        <div class="grid gap-3 px-4 py-3 sm:grid-cols-2">
          <div>
            <label class="block font-medium text-foreground/80" for="updater-channel">Update Channel</label>
            <select
              id="updater-channel"
              v-model="updaterChannel"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              :disabled="updaterLoading || updaterSaving"
            >
              <option v-for="channel in UPDATE_CHANNELS" :key="channel" :value="channel">
                {{ channel }}
              </option>
            </select>
          </div>
          <div>
            <label class="block font-medium text-foreground/80" for="updater-interval">Check Interval</label>
            <input
              id="updater-interval"
              v-model="updaterCheckInterval"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 font-mono"
              placeholder="1h"
              :disabled="updaterLoading || updaterSaving"
              autocomplete="off"
            />
            <p class="mt-1 text-muted-foreground">Go duration string, e.g. <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">30m</code>, <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">24h</code>.</p>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="font-medium text-foreground/80">View Changelog After Update</p>
            <p class="mt-0.5 text-muted-foreground">
              Show release notes once on the first administrator visit after a version change.
            </p>
          </div>
          <Switch
            :checked="updaterViewChangelogAfterUpdate"
            :disabled="updaterLoading || updaterSaving"
            aria-label="View Changelog After Update"
            @update:checked="updaterViewChangelogAfterUpdate = $event"
          />
        </div>

        <div class="space-y-3 px-4 py-3">
          <div>
            <p class="font-medium text-foreground/80">Action on Update</p>
            <p class="mt-0.5 text-muted-foreground">Choose notification-only mode or automatic binary replacement.</p>
          </div>
          <div
            class="grid gap-2 sm:grid-cols-2"
            role="radiogroup"
            aria-label="Action on Update"
          >
            <button
              v-for="option in updateActionOptions"
              :key="String(option.value)"
              type="button"
              class="rounded-md border px-3 py-2.5 text-left transition-all duration-300"
              :class="
                updaterNotifyOnly === option.value
                  ? 'border-primary/30 bg-primary/10'
                  : 'border-border bg-muted hover:border-border'
              "
              role="radio"
              :aria-checked="updaterNotifyOnly === option.value"
              :disabled="updaterLoading || updaterSaving"
              @click="updaterNotifyOnly = option.value"
            >
              <span class="block font-medium text-foreground">{{ option.label }}</span>
              <span class="mt-0.5 block text-muted-foreground">{{ option.description }}</span>
            </button>
          </div>
        </div>
      </div>

      <div class="border-t border-border px-4 py-3">
        <button
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
          :disabled="updaterLoading || updaterSaving"
          @click="saveUpdaterSettings"
        >
          {{ updaterSaving ? 'Saving…' : updaterLoading ? 'Loading…' : 'Save Auto-Updater Settings' }}
        </button>
      </div>
    </section>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">UI Preferences</h2>
        <p class="mt-0.5 text-xs text-muted-foreground">Operator layout and notification presentation settings.</p>
      </header>

      <div class="ui-divide text-xs">
        <div class="space-y-3 px-4 py-3">
          <div>
            <p class="font-medium text-foreground/80">Notification Center Layout</p>
            <p class="mt-0.5 text-muted-foreground">
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
              class="rounded-md border px-3 py-2.5 text-left transition-all duration-300"
              :class="
                notificationLayoutStyle === option.value
                  ? 'border-primary/30 bg-primary/10'
                  : 'border-border bg-muted hover:border-border'
              "
              role="radio"
              :aria-checked="notificationLayoutStyle === option.value"
              @click="setNotificationLayoutStyle(option.value)"
            >
              <span class="block font-medium text-foreground">{{ option.label }}</span>
              <span class="mt-0.5 block text-muted-foreground">{{ option.description }}</span>
            </button>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="text-foreground/80">Show Developer API Hints</p>
            <p class="mt-0.5 text-muted-foreground">
              Display route badges and documentation references across all views.
            </p>
          </div>
          <Switch
            :checked="showApiHints"
            aria-label="Show Developer API Hints"
            @update:checked="setShowApiHints($event)"
          />
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="text-foreground/80">Collapsed sidebar by default</p>
            <p class="mt-0.5 text-muted-foreground">Stored in local storage; reload applies the layout.</p>
          </div>
          <Switch
            :checked="sidebarCollapsed"
            aria-label="Collapsed sidebar by default"
            @update:checked="sidebarCollapsed = $event"
          />
        </div>
      </div>

      <div class="border-t border-border px-4 py-3">
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="persistSidebarPreference">
          Save sidebar preference
        </button>
      </div>
    </section>
  </div>
</template>
