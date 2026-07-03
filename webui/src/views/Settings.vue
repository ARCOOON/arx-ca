<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchSettingsConfig, updateSettingsConfig } from '@/api/config'
import { resolveApiBaseURL } from '@/api/client'
import { fetchHealth } from '@/api/health'
import { listPublicCertificates } from '@/api/public'
import type { HealthReport } from '@/types/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useAuthStore } from '@/store/auth'
import { extractApiError } from '@/utils/errors'
import { formatBytes } from '@/utils/format'

const authStore = useAuthStore()
const apiBaseUrl = resolveApiBaseURL()
const appOrigin = typeof window !== 'undefined' ? window.location.origin : ''

const health = ref<HealthReport | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')
const publicCertTotal = ref<number | null>(null)
const publicError = ref('')

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
  try {
    const response = await listPublicCertificates()
    publicCertTotal.value = response.total
  } catch (error) {
    publicError.value = extractApiError(error, 'Failed to load public certificate count')
  }
}
</script>

<template>
  <div class="space-y-4">
    <Alert v-if="errorMessage" variant="destructive" class="rounded-lg">
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>

    <section class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader>
          <CardTitle class="text-sm">Session</CardTitle>
          <CardDescription>Current operator context and API endpoint.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <div class="flex flex-wrap gap-2">
            <Badge v-for="role in authStore.roles" :key="role" variant="secondary" class="rounded-md">
              {{ role }}
            </Badge>
            <Badge v-if="authStore.roles.length === 0" variant="outline" class="rounded-md">No roles</Badge>
          </div>
          <p><span class="text-muted-foreground">API base:</span> {{ apiBaseUrl }}</p>
          <p v-if="health"><span class="text-muted-foreground">Server version:</span> {{ health.api.binary_version }}</p>
          <p v-if="health"><span class="text-muted-foreground">Heap:</span> {{ formatBytes(health.memory.heap_alloc_bytes) }}</p>
        </CardContent>
      </Card>

      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader>
          <CardTitle class="text-sm">Public API</CardTitle>
          <CardDescription>Read-only certificate inventory endpoint.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-2 text-sm">
          <p class="break-all font-mono text-xs">{{ publicListUrl }}</p>
          <p v-if="publicCertTotal !== null" class="text-muted-foreground">{{ publicCertTotal }} public certificates</p>
          <p v-if="publicError" class="text-destructive">{{ publicError }}</p>
        </CardContent>
      </Card>
    </section>

    <Card v-if="canManageSettings" class="rounded-lg border border-border shadow-none">
      <CardHeader>
        <CardTitle class="text-sm">Auto-Updater</CardTitle>
        <CardDescription>
          Configure background release polling and post-update changelog prompts.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <p v-if="updaterLoading" class="text-sm text-muted-foreground">Loading updater configuration…</p>

        <template v-else>
          <div class="flex items-center justify-between rounded-lg border border-border bg-muted/30 px-3 py-2">
            <div>
              <p class="text-sm font-medium">Enabled</p>
              <p class="text-xs text-muted-foreground">Poll GitHub releases on the configured channel.</p>
            </div>
            <Switch v-model:checked="updaterEnabled" />
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label>Release channel</Label>
              <Select v-model="updaterChannel">
                <SelectTrigger class="rounded-lg">
                  <SelectValue placeholder="Select channel" />
                </SelectTrigger>
                <SelectContent class="rounded-lg">
                  <SelectItem v-for="channel in UPDATE_CHANNELS" :key="channel" :value="channel">
                    {{ channel }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label for="check-interval">Check interval</Label>
              <Input id="check-interval" v-model="updaterCheckInterval" placeholder="1h" class="rounded-lg" />
            </div>
          </div>

          <div class="flex items-center justify-between rounded-lg border border-border bg-muted/30 px-3 py-2">
            <div>
              <p class="text-sm font-medium">Notify only</p>
              <p class="text-xs text-muted-foreground">
                When off, the server downloads and applies updates automatically.
              </p>
            </div>
            <Switch v-model:checked="updaterNotifyOnly" />
          </div>

          <div class="flex items-center justify-between rounded-lg border border-border bg-muted/30 px-3 py-2">
            <div>
              <p class="text-sm font-medium">Show changelog after update</p>
              <p class="text-xs text-muted-foreground">
                Prompt administrators once when the binary version changes.
              </p>
            </div>
            <Switch v-model:checked="updaterViewChangelogAfterUpdate" />
          </div>

          <Alert v-if="updaterError" variant="destructive" class="rounded-lg">
            <AlertDescription>{{ updaterError }}</AlertDescription>
          </Alert>

          <Alert v-if="updaterSaved" class="rounded-lg border-success/30 bg-success/10 text-success-foreground">
            <AlertDescription>Auto-updater settings saved.</AlertDescription>
          </Alert>

          <Button class="rounded-lg" :disabled="updaterSaving" @click="saveUpdaterSettings">
            {{ updaterSaving ? 'Saving…' : 'Save updater settings' }}
          </Button>
        </template>
      </CardContent>
    </Card>

    <Card v-else class="rounded-lg border border-border shadow-none">
      <CardHeader>
        <CardTitle class="text-sm">Auto-Updater</CardTitle>
      </CardHeader>
      <CardContent>
        <p class="text-sm text-muted-foreground">
          SuperAdmin or CA-Admin role required to manage updater configuration.
        </p>
      </CardContent>
    </Card>
  </div>
</template>
