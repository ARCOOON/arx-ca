<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Spinner from '@/components/ui/Spinner.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { fetchScepStatus } from '@/api/scep'
import type { ScepStatus } from '@/types/api'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const status = ref<ScepStatus | null>(null)

onMounted(async () => {
  try {
    status.value = await fetchScepStatus()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="space-y-6 max-w-2xl">
    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <Card v-else-if="status" class="px-6 py-5 space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-semibold text-foreground">SCEP Protocol</h2>
        <StatusBadge :status="status.enabled ? 'enabled' : 'disabled'" />
      </div>

      <div v-if="status.enabled" class="space-y-2 text-sm">
        <div class="flex gap-2">
          <span class="w-32 text-xs text-foreground-muted">Base URL</span>
          <span class="font-mono text-xs text-foreground">{{ status.base_url ?? '—' }}</span>
        </div>
        <div class="flex gap-2">
          <span class="w-32 text-xs text-foreground-muted">Provisioner</span>
          <span class="text-foreground">{{ status.provisioner ?? '—' }}</span>
        </div>
        <div v-if="status.challenge_hint" class="flex gap-2">
          <span class="w-32 text-xs text-foreground-muted">Challenge</span>
          <span class="font-mono text-xs text-foreground">{{ status.challenge_hint }}</span>
        </div>
      </div>
      <div v-else class="text-sm text-foreground-muted">
        SCEP enrollment is not enabled in the current server configuration.
      </div>
    </Card>

    <div v-else class="rounded-lg border border-border bg-muted px-4 py-8 text-center text-sm text-foreground-muted">
      SCEP status unavailable
    </div>
  </div>
</template>
