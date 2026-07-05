<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createProvisionerToken, fetchK8sStatus } from '../api/provisioners'
import type { K8sProvisionerStatus, ProvisionerTokenResponse } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import Button from '@/components/ui/Button.vue'
import FlatToggle from '../components/ui/FlatToggle.vue'
import Input from '@/components/ui/Input.vue'
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
    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <div v-if="isLoading" class="text-sm ui-text-muted">Loading provisioner configuration…</div>

    <template v-else>
      <section v-if="k8sStatus" class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Kubernetes Provisioner</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs ui-text-muted">
            Status from
            <code class="ui-code">GET /api/v1/k8s/status</code>
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
            <dt class="ui-text-muted">Provisioner</dt>
            <dd class="font-mono ui-text-secondary">{{ k8sStatus.provisioner }}</dd>
          </div>
          <div v-if="k8sStatus.review_mode" class="grid gap-2 px-4 py-3 sm:grid-cols-[10rem_1fr]">
            <dt class="ui-text-muted">Review mode</dt>
            <dd class="ui-text-secondary">{{ k8sStatus.review_mode }}</dd>
          </div>
          <div class="px-4 py-3 space-y-2">
            <FlatToggle label="Public keys configured" :enabled="k8sStatus.has_public_keys" readonly />
            <FlatToggle label="Uses TokenReview API" :enabled="k8sStatus.uses_token_review_api" readonly />
          </div>
        </dl>
      </section>

      <section class="ui-surface-muted">
        <header class="ui-border-b px-4 py-2.5">
          <h2 class="text-sm font-semibold ui-text-primary">Mint Provisioner Token</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs ui-text-muted">
            Single-use JWK signing token via
            <code class="ui-code">POST /api/v1/provisioners/token</code>
          </p>
        </header>
        <div class="space-y-3 px-4 py-3">
          <div v-if="tokenError" class="ui-alert-error text-xs" role="alert">
            {{ tokenError }}
          </div>

          <div v-if="tokenResult" class="ui-alert-warning space-y-3" role="alert">
            <p class="font-semibold">Save the token now — it will not be shown again.</p>
            <dl class="space-y-2 text-xs">
              <div>
                <dt class="font-medium ui-text-muted">Provisioner</dt>
                <dd class="mt-0.5 font-mono ui-text-primary">
                  {{ tokenResult.provisioner }} ({{ tokenResult.provisioner_type }})
                </dd>
              </div>
              <div>
                <dt class="font-medium ui-text-muted">Expires in</dt>
                <dd class="mt-0.5 ui-text-primary">{{ tokenResult.expires_in }} seconds</dd>
              </div>
              <div>
                <dt class="font-medium ui-text-muted">Audience</dt>
                <dd class="mt-0.5 break-all font-mono ui-text-primary">{{ tokenResult.audience }}</dd>
              </div>
              <div>
                <dt class="font-medium ui-text-muted">Token</dt>
                <dd class="mt-0.5 break-all font-mono text-[10px] ui-text-primary">{{ tokenResult.token }}</dd>
              </div>
            </dl>
            <div class="flex flex-wrap gap-2">
              <Button @click="copyToken">
                {{ tokenCopied ? 'Copied' : 'Copy Token' }}
              </Button>
              <Button variant="secondary" @click="dismissTokenResult">Dismiss</Button>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="block text-xs font-medium ui-text-secondary" for="token-provisioner">
                Provisioner (optional)
              </label>
              <Input
                id="token-provisioner"
                v-model="tokenProvisioner"
                class="mt-1.5"
                autocomplete="off"
              />
            </div>
            <div>
              <label class="block text-xs font-medium ui-text-secondary" for="token-ttl">
                Token TTL
              </label>
              <Input id="token-ttl" v-model="tokenTtl" class="mt-1.5" placeholder="5m" />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="token-cn">Common Name</label>
            <Input
              id="token-cn"
              v-model="tokenCommonName"
              class="mt-1.5"
              placeholder="pod.example.svc"
              autocomplete="off"
            />
          </div>

          <div>
            <label class="block text-xs font-medium ui-text-secondary">DNS SANs</label>
            <TagInput v-model="tokenDnsSans" placeholder="api.example.com" />
          </div>

          <div>
            <label class="block text-xs font-medium ui-text-secondary">IP SANs</label>
            <TagInput v-model="tokenIpSans" placeholder="10.0.0.1" />
          </div>

          <Button :disabled="tokenGenerating" @click="submitTokenGeneration">
            {{ tokenGenerating ? 'Minting…' : 'Mint Token' }}
          </Button>
        </div>
      </section>
    </template>
  </div>
</template>
