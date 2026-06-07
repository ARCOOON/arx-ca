<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { createAcmeEabKey, fetchAcmeStatus } from '../api/acme'
import type { AcmeEabKeyResponse, AcmeStatus } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { copyToClipboard } from '../utils/clipboard'
import { extractApiError } from '../utils/errors'

const { showApiHints } = usePreferences()

const status = ref<AcmeStatus | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const eabProvisioner = ref('')
const eabReference = ref('')
const eabGenerating = ref(false)
const eabError = ref('')
const eabResult = ref<AcmeEabKeyResponse | null>(null)
const eabCopied = ref(false)

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
    if (status.value.provisioner) {
      eabProvisioner.value = status.value.provisioner
    }
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load ACME status')
  } finally {
    isLoading.value = false
  }
})

async function submitEabGeneration(): Promise<void> {
  eabError.value = ''
  eabResult.value = null
  eabCopied.value = false
  eabGenerating.value = true

  try {
    eabResult.value = await createAcmeEabKey({
      provisioner: eabProvisioner.value.trim() || undefined,
      reference: eabReference.value.trim() || undefined,
    })
  } catch (error) {
    eabError.value = extractApiError(error, 'Failed to generate EAB key')
  } finally {
    eabGenerating.value = false
  }
}

async function copyHmacKey(): Promise<void> {
  if (!eabResult.value?.hmac_key) {
    return
  }
  await copyToClipboard(eabResult.value.hmac_key)
  eabCopied.value = true
}

function dismissEabResult(): void {
  eabResult.value = null
  eabCopied.value = false
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading ACME configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span v-if="showApiHints" class="text-xs ui-text-muted">
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

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">External Account Binding (EAB)</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs ui-text-muted">
            Mint credentials via
            <code class="ui-code">POST /api/v1/acme/eab-keys</code>
            for ACME account registration.
          </p>
        </header>
        <div class="space-y-3 px-4 py-3">
          <div v-if="eabError" class="ui-alert-error text-xs" role="alert">
            {{ eabError }}
          </div>

          <div v-if="eabResult" class="ui-alert-warning space-y-3" role="alert">
            <p class="font-semibold">Save the HMAC key now — it will not be shown again.</p>
            <dl class="space-y-2 text-xs">
              <div>
                <dt class="font-medium ui-text-muted">Key ID</dt>
                <dd class="mt-0.5 break-all font-mono ui-text-primary">{{ eabResult.key_id }}</dd>
              </div>
              <div>
                <dt class="font-medium ui-text-muted">Provisioner</dt>
                <dd class="mt-0.5 font-mono ui-text-primary">{{ eabResult.provisioner }}</dd>
              </div>
              <div v-if="eabResult.reference">
                <dt class="font-medium ui-text-muted">Reference</dt>
                <dd class="mt-0.5 ui-text-primary">{{ eabResult.reference }}</dd>
              </div>
              <div>
                <dt class="font-medium ui-text-muted">HMAC Key</dt>
                <dd class="mt-0.5 break-all font-mono text-sm ui-text-primary">{{ eabResult.hmac_key }}</dd>
              </div>
            </dl>
            <div class="flex flex-wrap gap-2">
              <button type="button" class="ui-btn-primary" @click="copyHmacKey">
                {{ eabCopied ? 'Copied' : 'Copy HMAC Key' }}
              </button>
              <button type="button" class="ui-btn-secondary" @click="dismissEabResult">Dismiss</button>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="block text-xs font-medium ui-text-secondary" for="eab-provisioner">
                Provisioner (optional)
              </label>
              <input
                id="eab-provisioner"
                v-model="eabProvisioner"
                type="text"
                class="ui-input mt-1.5"
                placeholder="acme"
                autocomplete="off"
              />
            </div>
            <div>
              <label class="block text-xs font-medium ui-text-secondary" for="eab-reference">
                Reference (optional)
              </label>
              <input
                id="eab-reference"
                v-model="eabReference"
                type="text"
                class="ui-input mt-1.5"
                placeholder="customer-123"
                autocomplete="off"
              />
            </div>
          </div>
          <button
            type="button"
            class="ui-btn-primary"
            :disabled="eabGenerating || !status.enabled"
            @click="submitEabGeneration"
          >
            {{ eabGenerating ? 'Generating…' : 'Generate EAB Key' }}
          </button>
          <p v-if="!status.enabled" class="text-xs ui-text-muted">
            ACME must be enabled in ca.json before EAB keys can be issued.
          </p>
        </div>
      </section>
    </template>
  </div>
</template>
