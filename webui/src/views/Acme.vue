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
    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-muted-foreground">Loading ACME configuration…</div>

    <template v-else-if="status">
      <section class="flex flex-wrap items-center gap-3">
        <StatusBadge
          :label="status.enabled ? 'Enabled' : 'Disabled'"
          :tone="status.enabled ? 'enabled' : 'disabled'"
        />
        <span v-if="showApiHints" class="text-xs text-muted-foreground">
          Discovery:
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/acme/status</code>
        </span>
      </section>

      <FlatToggle label="ACME server" :enabled="status.enabled" readonly />

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Endpoints</h2>
        </header>
        <dl class="ui-divide text-xs">
          <div class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Directory URL</dt>
            <dd class="break-all font-mono text-foreground/80">{{ directoryUrl }}</dd>
          </div>
          <div v-if="status.dns_name" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">DNS name</dt>
            <dd class="font-mono text-foreground/80">{{ status.dns_name }}</dd>
          </div>
          <div v-if="status.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Provisioner</dt>
            <dd class="font-mono text-foreground/80">{{ status.provisioner }}</dd>
          </div>
        </dl>
      </section>

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Policy</h2>
        </header>
        <div class="ui-divide space-y-0">
          <FlatToggle label="Require EAB" :enabled="status.require_eab" readonly />
          <FlatToggle label="Device attestation" :enabled="status.device_attest_enabled" readonly />
        </div>
        <div v-if="status.challenges?.length" class="border-t border-border px-4 py-3">
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Challenges</p>
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

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">External Account Binding (EAB)</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
            Mint credentials via
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/acme/eab-keys</code>
            for ACME account registration.
          </p>
        </header>
        <div class="space-y-3 px-4 py-3">
          <div v-if="eabError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
            {{ eabError }}
          </div>

          <div v-if="eabResult" class="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400 space-y-3" role="alert">
            <p class="font-semibold">Save the HMAC key now — it will not be shown again.</p>
            <dl class="space-y-2 text-xs">
              <div>
                <dt class="font-medium text-muted-foreground">Key ID</dt>
                <dd class="mt-0.5 break-all font-mono text-foreground">{{ eabResult.key_id }}</dd>
              </div>
              <div>
                <dt class="font-medium text-muted-foreground">Provisioner</dt>
                <dd class="mt-0.5 font-mono text-foreground">{{ eabResult.provisioner }}</dd>
              </div>
              <div v-if="eabResult.reference">
                <dt class="font-medium text-muted-foreground">Reference</dt>
                <dd class="mt-0.5 text-foreground">{{ eabResult.reference }}</dd>
              </div>
              <div>
                <dt class="font-medium text-muted-foreground">HMAC Key</dt>
                <dd class="mt-0.5 break-all font-mono text-sm text-foreground">{{ eabResult.hmac_key }}</dd>
              </div>
            </dl>
            <div class="flex flex-wrap gap-2">
              <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="copyHmacKey">
                {{ eabCopied ? 'Copied' : 'Copy HMAC Key' }}
              </button>
              <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="dismissEabResult">Dismiss</button>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="block text-xs font-medium text-foreground/80" for="eab-provisioner">
                Provisioner (optional)
              </label>
              <input
                id="eab-provisioner"
                v-model="eabProvisioner"
                type="text"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
                placeholder="acme"
                autocomplete="off"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground/80" for="eab-reference">
                Reference (optional)
              </label>
              <input
                id="eab-reference"
                v-model="eabReference"
                type="text"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
                placeholder="customer-123"
                autocomplete="off"
              />
            </div>
          </div>
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
            :disabled="eabGenerating || !status.enabled"
            @click="submitEabGeneration"
          >
            {{ eabGenerating ? 'Generating…' : 'Generate EAB Key' }}
          </button>
          <p v-if="!status.enabled" class="text-xs text-muted-foreground">
            ACME must be enabled in ca.json before EAB keys can be issued.
          </p>
        </div>
      </section>
    </template>
  </div>
</template>
