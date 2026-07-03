<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Spinner from '@/components/ui/Spinner.vue'
import Dialog from '@/components/ui/Dialog.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { fetchCAProvisioners } from '@/api/ca'
import { fetchK8sStatus, generateProvisionerToken } from '@/api/provisioners'
import type { CAProvisionerDetail, K8sProvisionerStatus, ProvisionerTokenResponse } from '@/types/api'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const provisioners = ref<CAProvisionerDetail[]>([])
const k8sStatus = ref<K8sProvisionerStatus | null>(null)

const tokenOpen = ref(false)
const tokenCN = ref('')
const tokenProvisioner = ref('')
const tokenSans = ref('')
const tokenTtl = ref('1h')
const tokenLoading = ref(false)
const tokenResult = ref<ProvisionerTokenResponse | null>(null)

onMounted(async () => {
  try {
    const [prov, k8s] = await Promise.all([
      fetchCAProvisioners(),
      fetchK8sStatus().catch(() => null),
    ])
    provisioners.value = prov.provisioners
    k8sStatus.value = k8s
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
})

async function handleGenerateToken(): Promise<void> {
  tokenLoading.value = true
  tokenResult.value = null
  try {
    tokenResult.value = await generateProvisionerToken({
      common_name: tokenCN.value,
      provisioner: tokenProvisioner.value || undefined,
      dns_sans: tokenSans.value ? tokenSans.value.split(',').map((s) => s.trim()) : undefined,
      token_ttl: tokenTtl.value || undefined,
    })
    toast.success('Token generated')
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    tokenLoading.value = false
  }
}

function copy(text: string): void {
  navigator.clipboard.writeText(text).then(() => toast.success('Token copied')).catch(() => {})
}
</script>

<template>
  <div class="space-y-6">
    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <template v-else>
      <!-- Provisioners list -->
      <Card class="px-5 py-4 space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">CA Provisioners</h2>
          <Button size="sm" @click="() => { tokenResult = null; tokenOpen = true }">Generate Token</Button>
        </div>

        <div class="divide-y divide-border">
          <div
            v-for="p in provisioners"
            :key="p.name"
            class="flex items-center justify-between py-3"
          >
            <div>
              <p class="text-sm font-medium text-foreground">{{ p.name }}</p>
              <div class="flex gap-1.5 mt-1">
                <Badge variant="secondary" class="text-[10px]">{{ p.type }}</Badge>
                <Badge v-if="p.require_eab" variant="warning" class="text-[10px]">EAB Required</Badge>
              </div>
            </div>
            <div v-if="p.challenges?.length" class="flex gap-1">
              <Badge v-for="c in p.challenges" :key="c" variant="outline" class="text-[10px]">{{ c }}</Badge>
            </div>
          </div>

          <div v-if="provisioners.length === 0" class="py-8 text-center text-sm text-foreground-muted">
            No provisioners found
          </div>
        </div>
      </Card>

      <!-- Kubernetes status -->
      <Card v-if="k8sStatus" class="px-5 py-4 space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">Kubernetes Enrollment</h2>
          <StatusBadge :status="k8sStatus.enabled ? 'enabled' : 'disabled'" />
        </div>
        <div v-if="k8sStatus.enabled" class="grid grid-cols-2 gap-3 text-xs">
          <div>
            <p class="text-foreground-muted mb-0.5">Provisioner</p>
            <p class="text-foreground font-medium">{{ k8sStatus.provisioner ?? '—' }}</p>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">Review Mode</p>
            <p class="text-foreground font-medium">{{ k8sStatus.review_mode ?? '—' }}</p>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">Token Review API</p>
            <Badge :variant="k8sStatus.uses_token_review_api ? 'success' : 'secondary'">
              {{ k8sStatus.uses_token_review_api ? 'Yes' : 'No' }}
            </Badge>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">Public Keys</p>
            <Badge :variant="k8sStatus.has_public_keys ? 'success' : 'secondary'">
              {{ k8sStatus.has_public_keys ? 'Configured' : 'Missing' }}
            </Badge>
          </div>
        </div>
      </Card>
    </template>

    <!-- Token dialog -->
    <Dialog :open="tokenOpen" title="Generate Provisioner Token" @close="tokenOpen = false">
      <div class="space-y-3">
        <div class="space-y-1.5">
          <Label>Common Name</Label>
          <Input v-model="tokenCN" placeholder="service.example.com" />
        </div>
        <div class="space-y-1.5">
          <Label>Provisioner (optional)</Label>
          <Input v-model="tokenProvisioner" placeholder="default provisioner" />
        </div>
        <div class="space-y-1.5">
          <Label>DNS SANs (comma-separated, optional)</Label>
          <Input v-model="tokenSans" placeholder="api.example.com, www.example.com" />
        </div>
        <div class="space-y-1.5">
          <Label>TTL</Label>
          <Input v-model="tokenTtl" placeholder="1h" />
        </div>

        <div v-if="tokenResult" class="pt-2 space-y-2 rounded-md bg-muted p-3">
          <div class="flex items-center justify-between">
            <span class="text-xs text-foreground-muted">JWT Token</span>
            <button class="text-primary text-xs hover:underline" @click="copy(tokenResult.token)">Copy</button>
          </div>
          <p class="font-mono text-[10px] text-foreground break-all">{{ tokenResult.token }}</p>
          <div class="grid grid-cols-2 gap-2 text-xs mt-2">
            <div><p class="text-foreground-muted">Provisioner</p><p class="font-medium">{{ tokenResult.provisioner }}</p></div>
            <div><p class="text-foreground-muted">Expires In</p><p class="font-medium">{{ tokenResult.expires_in }}s</p></div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="tokenOpen = false">Close</Button>
        <Button :disabled="tokenLoading || !tokenCN" @click="handleGenerateToken">
          <Spinner v-if="tokenLoading" size="sm" />
          <span>{{ tokenLoading ? 'Generating…' : 'Generate' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>
