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
import { fetchAcmeStatus, createAcmeEabKey } from '@/api/acme'
import type { AcmeStatus, AcmeEabKeyResponse } from '@/types/api'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const status = ref<AcmeStatus | null>(null)

const eabOpen = ref(false)
const eabRef = ref('')
const eabProvisioner = ref('')
const eabLoading = ref(false)
const eabResult = ref<AcmeEabKeyResponse | null>(null)

onMounted(async () => {
  try {
    status.value = await fetchAcmeStatus()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
})

async function handleCreateEab(): Promise<void> {
  eabLoading.value = true
  eabResult.value = null
  try {
    eabResult.value = await createAcmeEabKey({
      provisioner: eabProvisioner.value || undefined,
      reference: eabRef.value || undefined,
    })
    toast.success('EAB key created')
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    eabLoading.value = false
  }
}

function copy(text: string): void {
  navigator.clipboard.writeText(text).then(() => toast.success('Copied')).catch(() => {})
}
</script>

<template>
  <div class="space-y-6 max-w-2xl">
    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <template v-else-if="status">
      <Card class="px-6 py-5 space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">ACME Protocol</h2>
          <StatusBadge :status="status.enabled ? 'enabled' : 'disabled'" />
        </div>

        <div v-if="status.enabled" class="space-y-3 text-sm">
          <InfoRow label="Directory URL" :value="status.directory_url ?? '—'" mono />
          <InfoRow label="Provisioner" :value="status.provisioner ?? '—'" />
          <InfoRow label="DNS Name" :value="status.dns_name ?? '—'" />
          <InfoRow label="Require EAB" :value="status.require_eab ? 'Yes' : 'No'" />
          <InfoRow label="Device Attestation" :value="status.device_attest_enabled ? 'Enabled' : 'Disabled'" />
          <div v-if="status.challenges?.length">
            <p class="text-xs text-foreground-muted mb-1.5">Supported Challenges</p>
            <div class="flex flex-wrap gap-1">
              <Badge v-for="c in status.challenges" :key="c" variant="outline">{{ c }}</Badge>
            </div>
          </div>
        </div>

        <div v-if="status.enabled && status.require_eab" class="pt-2">
          <Button size="sm" @click="eabOpen = true">Generate EAB Key</Button>
        </div>
      </Card>
    </template>

    <div v-else class="rounded-lg border border-border bg-muted px-4 py-8 text-center text-sm text-foreground-muted">
      ACME status unavailable
    </div>

    <!-- EAB key dialog -->
    <Dialog :open="eabOpen" title="Generate EAB Key" @close="eabOpen = false">
      <div class="space-y-3">
        <div class="space-y-1.5">
          <Label>Provisioner (optional)</Label>
          <Input v-model="eabProvisioner" placeholder="acme-provisioner" />
        </div>
        <div class="space-y-1.5">
          <Label>Reference (optional)</Label>
          <Input v-model="eabRef" placeholder="client identifier" />
        </div>

        <div v-if="eabResult" class="space-y-2 pt-2 rounded-md bg-muted p-3">
          <div class="text-xs space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-foreground-muted">Key ID</span>
              <button class="text-primary text-xs hover:underline" @click="copy(eabResult.key_id)">Copy</button>
            </div>
            <p class="font-mono text-foreground break-all">{{ eabResult.key_id }}</p>
            <div class="flex items-center justify-between mt-2">
              <span class="text-foreground-muted">HMAC Key</span>
              <button class="text-primary text-xs hover:underline" @click="copy(eabResult.hmac_key)">Copy</button>
            </div>
            <p class="font-mono text-foreground break-all">{{ eabResult.hmac_key }}</p>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="eabOpen = false">Close</Button>
        <Button :disabled="eabLoading" @click="handleCreateEab">
          <Spinner v-if="eabLoading" size="sm" />
          <span>{{ eabLoading ? 'Creating…' : 'Generate' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts">
const InfoRow = {
  props: { label: String, value: String, mono: Boolean },
  template: `
    <div class="flex items-start gap-2 text-sm">
      <span class="w-36 shrink-0 text-xs text-foreground-muted">{{ label }}</span>
      <span :class="['text-foreground', mono ? 'font-mono text-xs break-all' : '']">{{ value }}</span>
    </div>
  `,
}
export default { components: { InfoRow } }
</script>
