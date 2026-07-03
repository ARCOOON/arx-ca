<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Switch from '@/components/ui/Switch.vue'
import Select from '@/components/ui/Select.vue'
import Separator from '@/components/ui/Separator.vue'
import Spinner from '@/components/ui/Spinner.vue'
import Badge from '@/components/ui/Badge.vue'
import { fetchSettingsConfig, updateSettingsConfig } from '@/api/config'
import type { SettingsConfigResponse, UpdaterConfigView } from '@/types/api'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const saving = ref(false)
const config = ref<SettingsConfigResponse | null>(null)

const updater = ref<UpdaterConfigView>({
  enabled: false,
  channel: 'stable',
  notify_only: false,
  check_interval: '12h',
  view_changelog_after_update: true,
})

onMounted(async () => {
  try {
    config.value = await fetchSettingsConfig()
    updater.value = { ...config.value.updater }
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
})

async function saveUpdaterConfig(): Promise<void> {
  if (!config.value) return
  saving.value = true
  try {
    const updated = await updateSettingsConfig({ updater: updater.value })
    config.value = updated
    toast.success('Updater configuration saved')
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-2xl">
    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <template v-else-if="config">
      <!-- Auto-Updater -->
      <Card class="px-6 py-5 space-y-5">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-sm font-semibold text-foreground">Auto-Updater</h2>
            <p class="mt-0.5 text-xs text-foreground-muted">Configure automatic binary update behavior</p>
          </div>
          <Badge :variant="updater.enabled ? 'success' : 'secondary'">
            {{ updater.enabled ? 'Enabled' : 'Disabled' }}
          </Badge>
        </div>

        <Separator />

        <div class="space-y-5">
          <div class="flex items-center justify-between">
            <div>
              <Label>Enable Auto-Updater</Label>
              <p class="mt-0.5 text-xs text-foreground-muted">Automatically check for and apply updates</p>
            </div>
            <Switch v-model="updater.enabled" />
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label>Notify Only</Label>
              <p class="mt-0.5 text-xs text-foreground-muted">Notify without auto-applying (manual restart required)</p>
            </div>
            <Switch v-model="updater.notify_only" :disabled="!updater.enabled" />
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label>Show Changelog After Update</Label>
              <p class="mt-0.5 text-xs text-foreground-muted">Display what's new after a version change is detected</p>
            </div>
            <Switch v-model="updater.view_changelog_after_update" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label for="channel">Release Channel</Label>
              <Select id="channel" v-model="updater.channel" :disabled="!updater.enabled">
                <option value="stable">Stable</option>
                <option value="beta">Beta</option>
                <option value="edge">Edge</option>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label for="interval">Check Interval</Label>
              <Input
                id="interval"
                v-model="updater.check_interval"
                placeholder="12h"
                :disabled="!updater.enabled"
              />
            </div>
          </div>
        </div>

        <div class="flex justify-end pt-2">
          <Button :disabled="saving" @click="saveUpdaterConfig">
            <Spinner v-if="saving" size="sm" />
            <span>{{ saving ? 'Saving…' : 'Save Changes' }}</span>
          </Button>
        </div>
      </Card>

      <!-- Server config summary (read-only) -->
      <Card class="px-6 py-5 space-y-4">
        <div>
          <h2 class="text-sm font-semibold text-foreground">Server Configuration</h2>
          <p class="mt-0.5 text-xs text-foreground-muted">Read-only view of the active server configuration</p>
        </div>
        <Separator />
        <div class="rounded-md bg-muted p-4 overflow-x-auto">
          <pre class="text-[10px] font-mono text-foreground-muted whitespace-pre-wrap">{{ JSON.stringify({ server: config.server, database: config.database, security: config.security }, null, 2) }}</pre>
        </div>
      </Card>
    </template>
  </div>
</template>
