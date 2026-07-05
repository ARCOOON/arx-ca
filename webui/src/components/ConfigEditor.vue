<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchSettingsConfig, updateSettingsConfig } from '../api/config'
import type { SettingsConfigResponse } from '../types/api'
import { extractApiError } from '../utils/errors'
import Button from './ui/Button.vue'

const props = defineProps<{
  canEdit: boolean
}>()

const loading = ref(true)
const saving = ref(false)
const errorMessage = ref('')
const saved = ref(false)
const activeSection = ref('server')
const config = ref<SettingsConfigResponse | null>(null)

const sections = [
  { id: 'server', label: 'Server' },
  { id: 'database', label: 'Database' },
  { id: 'ca', label: 'CA' },
  { id: 'ca_bootstrap', label: 'CA Bootstrap' },
  { id: 'security', label: 'Security' },
  { id: 'bootstrap', label: 'Bootstrap' },
  { id: 'telemetry', label: 'Telemetry' },
  { id: 'service', label: 'Service' },
  { id: 'webui', label: 'WebUI' },
  { id: 'updater', label: 'Updater' },
] as const

const draft = ref<SettingsConfigResponse | null>(null)

const acmeEnabled = computed({
  get: () => {
    const enabled = (draft.value?.ca as Record<string, unknown>)?.provisioners as Record<string, unknown> | undefined
    const acme = enabled?.acme as Record<string, unknown> | undefined
    return acme?.enabled === true
  },
  set: (value: boolean) => {
    if (!draft.value) return
    const ca = draft.value.ca as Record<string, unknown>
    const provisioners = (ca.provisioners ?? {}) as Record<string, unknown>
    const acme = (provisioners.acme ?? {}) as Record<string, unknown>
    acme.enabled = value
    provisioners.acme = acme
    ca.provisioners = provisioners
    draft.value.ca = ca
  },
})

const scepEnabled = computed({
  get: () => {
    const provisioners = (draft.value?.ca as Record<string, unknown>)?.provisioners as Record<string, unknown> | undefined
    const scep = provisioners?.scep as Record<string, unknown> | undefined
    return scep?.enabled === true
  },
  set: (value: boolean) => {
    if (!draft.value) return
    const ca = draft.value.ca as Record<string, unknown>
    const provisioners = (ca.provisioners ?? {}) as Record<string, unknown>
    const scep = (provisioners.scep ?? {}) as Record<string, unknown>
    scep.enabled = value
    provisioners.scep = scep
    ca.provisioners = provisioners
    draft.value.ca = ca
  },
})

onMounted(() => {
  void loadConfig()
})

function cloneConfig<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

async function loadConfig(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    config.value = await fetchSettingsConfig()
    draft.value = cloneConfig(config.value)
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load configuration')
  } finally {
    loading.value = false
  }
}

function sectionData(id: string): Record<string, unknown> {
  if (!draft.value) return {}
  return (draft.value as Record<string, unknown>)[id] as Record<string, unknown>
}

function updateField(section: string, key: string, value: unknown): void {
  if (!draft.value) return
  const target = (draft.value as Record<string, unknown>)[section] as Record<string, unknown>
  target[key] = value
}

function updateNested(section: string, nested: string, key: string, value: unknown): void {
  if (!draft.value) return
  const target = (draft.value as Record<string, unknown>)[section] as Record<string, unknown>
  const child = (target[nested] ?? {}) as Record<string, unknown>
  child[key] = value
  target[nested] = child
}

function firewallAllowlistText(): string {
  const security = sectionData('security')
  const firewall = (security.firewall ?? {}) as Record<string, unknown>
  const list = firewall.allowlist
  return Array.isArray(list) ? list.join('\n') : ''
}

function setFirewallAllowlist(raw: string): void {
  const lines = raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
  updateNested('security', 'firewall', 'allowlist', lines)
}

function setSectionFromJSON(section: string, raw: string): void {
  if (!draft.value) return
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    ;(draft.value as Record<string, unknown>)[section] = parsed
  } catch {
    errorMessage.value = 'Invalid JSON in section editor'
  }
}

async function saveSection(): Promise<void> {
  if (!draft.value || !props.canEdit) return
  saving.value = true
  errorMessage.value = ''
  saved.value = false
  try {
    const section = activeSection.value
    const patch: Partial<SettingsConfigResponse> = {
      [section]: (draft.value as Record<string, unknown>)[section],
    } as Partial<SettingsConfigResponse>
    const updated = await updateSettingsConfig(patch)
    config.value = updated
    draft.value = cloneConfig(updated)
    saved.value = true
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to save configuration')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="ui-surface-muted">
    <header class="ui-border-b px-4 py-2.5">
      <h2 class="text-sm font-semibold ui-text-primary">Runtime Configuration</h2>
      <p class="mt-1 text-xs ui-text-muted">
        Edit server.toml values. Changes apply at runtime where supported (log level, firewall, provisioners).
      </p>
    </header>

    <div v-if="loading" class="px-4 py-6 text-sm ui-text-muted">Loading configuration…</div>
    <div v-else-if="errorMessage" class="px-4 py-3 ui-alert-error text-xs" role="alert">{{ errorMessage }}</div>
    <div v-else-if="draft" class="grid gap-0 lg:grid-cols-[12rem_1fr]">
      <nav class="ui-border-b lg:ui-border-b-0 lg:ui-border-r p-2">
        <button
          v-for="section in sections"
          :key="section.id"
          type="button"
          class="mb-1 w-full rounded-[var(--radius-control)] px-3 py-2 text-left text-xs transition-colors"
          :class="
            activeSection === section.id
              ? 'bg-[var(--accent-surface)] font-medium ui-text-primary'
              : 'ui-text-muted hover:bg-[var(--bg-surface)]'
          "
          @click="activeSection = section.id"
        >
          {{ section.label }}
        </button>
      </nav>

      <div class="space-y-4 p-4">
        <div v-if="saved" class="ui-alert-success text-xs" role="status">Configuration saved.</div>

        <template v-if="activeSection === 'server'">
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-xs ui-text-secondary">
              Host
              <input class="ui-input mt-1 w-full" :value="sectionData('server').host" @input="updateField('server', 'host', ($event.target as HTMLInputElement).value)" />
            </label>
            <label class="text-xs ui-text-secondary">
              Port
              <input class="ui-input mt-1 w-full" type="number" :value="sectionData('server').port" @input="updateField('server', 'port', Number(($event.target as HTMLInputElement).value))" />
            </label>
            <label class="text-xs ui-text-secondary">
              Log Level
              <input class="ui-input mt-1 w-full" :value="sectionData('server').log_level" @input="updateField('server', 'log_level', ($event.target as HTMLInputElement).value)" />
            </label>
          </div>
        </template>

        <template v-else-if="activeSection === 'ca'">
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-xs ui-text-secondary sm:col-span-2">
              Provisioner Name
              <input class="ui-input mt-1 w-full" :value="sectionData('ca').provisioner_name" @input="updateField('ca', 'provisioner_name', ($event.target as HTMLInputElement).value)" />
            </label>
            <label class="text-xs ui-text-secondary">
              Max TTL
              <input class="ui-input mt-1 w-full" :value="sectionData('ca').max_ttl" @input="updateField('ca', 'max_ttl', ($event.target as HTMLInputElement).value)" />
            </label>
          </div>
          <div class="ui-inset mt-4 space-y-3 p-4">
            <h3 class="text-xs font-semibold ui-text-primary">Provisioners</h3>
            <label class="flex items-center gap-2 text-xs ui-text-secondary">
              <input v-model="acmeEnabled" type="checkbox" :disabled="!canEdit" />
              ACME enabled
            </label>
            <label class="flex items-center gap-2 text-xs ui-text-secondary">
              <input v-model="scepEnabled" type="checkbox" :disabled="!canEdit" />
              SCEP enabled
            </label>
          </div>
        </template>

        <template v-else-if="activeSection === 'security'">
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-xs ui-text-secondary">
              Token Expiration (hours)
              <input class="ui-input mt-1 w-full" type="number" :value="sectionData('security').token_expiration_hours" @input="updateField('security', 'token_expiration_hours', Number(($event.target as HTMLInputElement).value))" />
            </label>
            <label class="text-xs ui-text-secondary">
              Cookie SameSite
              <input class="ui-input mt-1 w-full" :value="sectionData('security').cookie_same_site" @input="updateField('security', 'cookie_same_site', ($event.target as HTMLInputElement).value)" />
            </label>
          </div>
          <div class="ui-inset mt-4 space-y-3 p-4">
            <h3 class="text-xs font-semibold ui-text-primary">API Firewall</h3>
            <label class="flex items-center gap-2 text-xs ui-text-secondary">
              <input
                type="checkbox"
                :checked="((sectionData('security').firewall as Record<string, unknown>)?.enabled as boolean) === true"
                :disabled="!canEdit"
                @change="updateNested('security', 'firewall', 'enabled', ($event.target as HTMLInputElement).checked)"
              />
              Enable IP allowlist (blocks all non-matching clients)
            </label>
            <label class="block text-xs ui-text-secondary">
              Allowlist (one IP or CIDR per line)
              <textarea
                class="ui-textarea mt-1 w-full font-mono text-[11px]"
                rows="6"
                :value="firewallAllowlistText()"
                :disabled="!canEdit"
                @input="setFirewallAllowlist(($event.target as HTMLTextAreaElement).value)"
              />
            </label>
          </div>
        </template>

        <template v-else>
          <p class="text-xs ui-text-muted">
            Edit raw JSON fields for <strong>{{ activeSection }}</strong> via the structured server API.
          </p>
          <textarea
            class="ui-textarea w-full font-mono text-[11px]"
            rows="14"
            :value="JSON.stringify(sectionData(activeSection), null, 2)"
            :disabled="!canEdit"
            @input="setSectionFromJSON(activeSection, ($event.target as HTMLTextAreaElement).value)"
          />
        </template>

        <div v-if="canEdit" class="flex justify-end">
          <Button :disabled="saving" @click="saveSection">
            {{ saving ? 'Saving…' : 'Save Section' }}
          </Button>
        </div>
      </div>
    </div>
  </section>
</template>
