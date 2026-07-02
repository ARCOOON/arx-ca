<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createProvisionerToken, fetchK8sStatus } from '../api/provisioners'
import type { K8sProvisionerStatus, ProvisionerTokenResponse } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import FlatToggle from '../components/ui/FlatToggle.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import TagInput from '../components/ui/TagInput.vue'
import { copyToClipboard } from '../utils/clipboard'
import { extractApiError } from '../utils/errors'

const { showApiHints } = usePreferences()

const k8sStatus = ref<K8sProvisionerStatus | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')

const tokenProvisioner = ref('')
const tokenCommonName = ref('')
const tokenDnsSans = ref<string[]>([])
const tokenIpSans = ref<string[]>([])
const tokenTtl = ref('5m')
const tokenGenerating = ref(false)
const tokenError = ref('')
const tokenResult = ref<ProvisionerTokenResponse | null>(null)
const tokenCopied = ref(false)

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    k8sStatus.value = await fetchK8sStatus()
    if (k8sStatus.value.provisioner) {
      tokenProvisioner.value = k8sStatus.value.provisioner
    }
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load provisioner status')
  } finally {
    isLoading.value = false
  }
})

async function submitTokenGeneration(): Promise<void> {
  tokenError.value = ''
  tokenResult.value = null
  tokenCopied.value = false

  const commonName = tokenCommonName.value.trim()
  if (!commonName) {
    tokenError.value = 'Common Name is required.'
    return
  }

  tokenGenerating.value = true

  try {
    tokenResult.value = await createProvisionerToken({
      provisioner: tokenProvisioner.value.trim() || undefined,
      common_name: commonName,
      dns_sans: tokenDnsSans.value.length > 0 ? tokenDnsSans.value : undefined,
      ip_sans: tokenIpSans.value.length > 0 ? tokenIpSans.value : undefined,
      token_ttl: tokenTtl.value.trim() || undefined,
    })
  } catch (error) {
    tokenError.value = extractApiError(error, 'Failed to mint provisioner token')
  } finally {
    tokenGenerating.value = false
  }
}

async function copyToken(): Promise<void> {
  if (!tokenResult.value?.token) {
    return
  }
  await copyToClipboard(tokenResult.value.token)
  tokenCopied.value = true
}

function dismissTokenResult(): void {
  tokenResult.value = null
  tokenCopied.value = false
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm text-muted-foreground">Loading provisioner configuration…</div>

    <template v-else>
      <section v-if="k8sStatus" class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Kubernetes Provisioner</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
            Status from
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/k8s/status</code>
          </p>
        </header>
        <div class="flex flex-wrap items-center gap-3 px-4 py-3">
          <StatusBadge
            :label="k8sStatus.enabled ? 'Enabled' : 'Disabled'"
            :tone="k8sStatus.enabled ? 'enabled' : 'disabled'"
          />
        </div>
        <dl class="ui-divide text-xs">
          <div v-if="k8sStatus.provisioner" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Provisioner</dt>
            <dd class="font-mono text-foreground/80">{{ k8sStatus.provisioner }}</dd>
          </div>
          <div v-if="k8sStatus.review_mode" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="text-muted-foreground">Review mode</dt>
            <dd class="text-foreground/80">{{ k8sStatus.review_mode }}</dd>
          </div>
          <div class="px-4 py-3 space-y-2">
            <FlatToggle label="Public keys configured" :enabled="k8sStatus.has_public_keys" readonly />
            <FlatToggle label="Uses TokenReview API" :enabled="k8sStatus.uses_token_review_api" readonly />
          </div>
        </dl>
      </section>

      <section class="bg-card border-border">
        <header class="border-b border-border px-4 py-2.5">
          <h2 class="text-sm font-semibold text-foreground">Mint Provisioner Token</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
            Single-use JWK signing token via
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/provisioners/token</code>
          </p>
        </header>
        <div class="space-y-3 px-4 py-3">
          <div v-if="tokenError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
            {{ tokenError }}
          </div>

          <div v-if="tokenResult" class="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400 space-y-3" role="alert">
            <p class="font-semibold">Save the token now — it will not be shown again.</p>
            <dl class="space-y-2 text-xs">
              <div>
                <dt class="font-medium text-muted-foreground">Provisioner</dt>
                <dd class="mt-0.5 font-mono text-foreground">
                  {{ tokenResult.provisioner }} ({{ tokenResult.provisioner_type }})
                </dd>
              </div>
              <div>
                <dt class="font-medium text-muted-foreground">Expires in</dt>
                <dd class="mt-0.5 text-foreground">{{ tokenResult.expires_in }} seconds</dd>
              </div>
              <div>
                <dt class="font-medium text-muted-foreground">Audience</dt>
                <dd class="mt-0.5 break-all font-mono text-foreground">{{ tokenResult.audience }}</dd>
              </div>
              <div>
                <dt class="font-medium text-muted-foreground">Token</dt>
                <dd class="mt-0.5 break-all font-mono text-[10px] text-foreground">{{ tokenResult.token }}</dd>
              </div>
            </dl>
            <div class="flex flex-wrap gap-2">
              <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="copyToken">
                {{ tokenCopied ? 'Copied' : 'Copy Token' }}
              </button>
              <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="dismissTokenResult">Dismiss</button>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="block text-xs font-medium text-foreground/80" for="token-provisioner">
                Provisioner (optional)
              </label>
              <input
                id="token-provisioner"
                v-model="tokenProvisioner"
                type="text"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
                autocomplete="off"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground/80" for="token-ttl">
                Token TTL
              </label>
              <input id="token-ttl" v-model="tokenTtl" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" placeholder="5m" />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-foreground/80" for="token-cn">Common Name</label>
            <input
              id="token-cn"
              v-model="tokenCommonName"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              placeholder="pod.example.svc"
              autocomplete="off"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-foreground/80">DNS SANs</label>
            <TagInput v-model="tokenDnsSans" placeholder="api.example.com" />
          </div>

          <div>
            <label class="block text-xs font-medium text-foreground/80">IP SANs</label>
            <TagInput v-model="tokenIpSans" placeholder="10.0.0.1" />
          </div>

          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
            :disabled="tokenGenerating"
            @click="submitTokenGeneration"
          >
            {{ tokenGenerating ? 'Minting…' : 'Mint Token' }}
          </button>
        </div>
      </section>
    </template>
  </div>
</template>
