<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RefreshCcwDot, Download, KeyRound, UserPlus, Copy, Loader2, Save } from '@lucide/vue'
import { toast } from 'vue-sonner'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchSettingsConfig, updateSettingsConfig } from '@/api/settings'
import { fetchRootCAPem, fetchIntermediateCAPem } from '@/api/ca'
import { createServiceAccount } from '@/api/auth'
import type { SettingsConfigResponse, UpdaterConfigView } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import { copyToClipboard, downloadBlob } from '@/lib/download'
import { extractApiError } from '@/lib/errors'

const authStore = useAuthStore()
const canManageUpdater = computed(() => authStore.hasRole('SuperAdmin', 'CA-Admin'))
const isSuperAdmin = computed(() => authStore.hasRole('SuperAdmin'))

const isLoading = ref(true)
const config = ref<SettingsConfigResponse | null>(null)
const updater = ref<UpdaterConfigView>({
  enabled: false,
  channel: 'stable',
  notify_only: true,
  check_interval: '24h',
  view_changelog_after_update: true,
})
const savingUpdater = ref(false)

async function loadConfig(): Promise<void> {
  isLoading.value = true
  try {
    if (canManageUpdater.value) {
      config.value = await fetchSettingsConfig()
      updater.value = { ...config.value.updater }
    }
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to load settings'))
  } finally {
    isLoading.value = false
  }
}

async function saveUpdater(): Promise<void> {
  savingUpdater.value = true
  try {
    const updated = await updateSettingsConfig({ updater: { ...updater.value } })
    updater.value = { ...updated.updater }
    toast.success('Auto-updater settings saved')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to save settings'))
  } finally {
    savingUpdater.value = false
  }
}

/* ------------------------------ CA downloads ----------------------------- */

async function downloadRoot(): Promise<void> {
  try {
    const { pem } = await fetchRootCAPem()
    downloadBlob(pem, 'arx-root-ca.pem', 'application/x-pem-file')
    toast.success('Root CA downloaded')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to download root CA'))
  }
}

async function downloadIntermediate(): Promise<void> {
  try {
    const { pem } = await fetchIntermediateCAPem()
    downloadBlob(pem, 'arx-intermediate-ca.pem', 'application/x-pem-file')
    toast.success('Intermediate CA downloaded')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to download intermediate CA'))
  }
}

/* --------------------------- Service accounts ---------------------------- */

const accountName = ref('')
const accountBusy = ref(false)
const issuedApiKey = ref('')

async function createAccount(): Promise<void> {
  if (!accountName.value.trim()) {
    toast.error('A service account name is required.')
    return
  }
  accountBusy.value = true
  try {
    const account = await createServiceAccount({ name: accountName.value.trim() })
    issuedApiKey.value = account.api_key
    accountName.value = ''
    toast.success('Service account created')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to create service account'))
  } finally {
    accountBusy.value = false
  }
}

async function copyApiKey(): Promise<void> {
  if (await copyToClipboard(issuedApiKey.value)) {
    toast.success('API key copied')
  } else {
    toast.error('Clipboard unavailable')
  }
}

onMounted(loadConfig)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
      <p class="text-sm text-muted-foreground">Manage automatic updates, CA material, and access.</p>
    </div>

    <!-- Session -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">Session</CardTitle>
        <CardDescription>Your current roles determine available actions.</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="role in authStore.roles.length ? authStore.roles : ['Operator']"
            :key="role"
            class="rounded-md bg-accent px-2.5 py-1 text-xs font-medium"
          >{{ role }}</span>
        </div>
      </CardContent>
    </Card>

    <!-- Auto-updater -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <RefreshCcwDot class="size-4 text-primary" /> Auto-updater
        </CardTitle>
        <CardDescription>Control how new Arx CA releases are detected and applied.</CardDescription>
      </CardHeader>
      <CardContent>
        <p v-if="!canManageUpdater" class="text-sm text-muted-foreground">
          You need the SuperAdmin or CA-Admin role to manage the auto-updater.
        </p>
        <div v-else-if="isLoading" class="space-y-4">
          <Skeleton class="h-10 w-full" />
          <Skeleton class="h-10 w-full" />
        </div>
        <div v-else class="space-y-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <Label class="text-sm">Enable auto-updates</Label>
              <p class="text-xs text-muted-foreground">Periodically check for and apply new releases.</p>
            </div>
            <Switch v-model="updater.enabled" />
          </div>
          <Separator />
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label>Release channel</Label>
              <Select v-model="updater.channel">
                <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="stable">Stable</SelectItem>
                  <SelectItem value="beta">Beta</SelectItem>
                  <SelectItem value="edge">Edge</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label for="interval">Check interval</Label>
              <Input id="interval" v-model="updater.check_interval" placeholder="e.g. 24h" />
            </div>
          </div>
          <Separator />
          <div class="flex items-center justify-between gap-4">
            <div>
              <Label class="text-sm">Notify only</Label>
              <p class="text-xs text-muted-foreground">Surface available updates without installing them.</p>
            </div>
            <Switch v-model="updater.notify_only" />
          </div>
          <div class="flex items-center justify-between gap-4">
            <div>
              <Label class="text-sm">Show changelog after update</Label>
              <p class="text-xs text-muted-foreground">Display release notes the first time a new version runs.</p>
            </div>
            <Switch v-model="updater.view_changelog_after_update" />
          </div>
          <div class="flex justify-end">
            <Button :disabled="savingUpdater" @click="saveUpdater">
              <Loader2 v-if="savingUpdater" class="size-4 animate-spin" />
              <Save v-else class="size-4" />
              Save changes
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- CA material -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <KeyRound class="size-4 text-primary" /> CA material
        </CardTitle>
        <CardDescription>Download PEM-encoded CA certificates for trust distribution.</CardDescription>
      </CardHeader>
      <CardContent class="flex flex-wrap gap-3">
        <Button variant="outline" @click="downloadRoot">
          <Download class="size-4" /> Root CA (.pem)
        </Button>
        <Button variant="outline" @click="downloadIntermediate">
          <Download class="size-4" /> Intermediate CA (.pem)
        </Button>
      </CardContent>
    </Card>

    <!-- Service accounts -->
    <Card v-if="isSuperAdmin">
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <UserPlus class="size-4 text-primary" /> Service accounts
        </CardTitle>
        <CardDescription>Create API keys for machine-to-machine access.</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="min-w-[200px] flex-1 space-y-1.5">
            <Label for="acct">Name</Label>
            <Input id="acct" v-model="accountName" placeholder="ci-pipeline" />
          </div>
          <Button :disabled="accountBusy" @click="createAccount">
            <Loader2 v-if="accountBusy" class="size-4 animate-spin" /> Create
          </Button>
        </div>
        <div
          v-if="issuedApiKey"
          class="space-y-2 rounded-md border border-warning/40 bg-warning/10 p-3"
        >
          <p class="text-xs font-medium text-foreground">
            Copy this API key now — it will not be shown again.
          </p>
          <div class="flex items-center gap-2">
            <code class="min-w-0 flex-1 break-all rounded bg-background px-2 py-1 text-xs">{{ issuedApiKey }}</code>
            <Button variant="secondary" size="icon-sm" @click="copyApiKey">
              <Copy class="size-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
