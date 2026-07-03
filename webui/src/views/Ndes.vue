<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Spinner from '@/components/ui/Spinner.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { fetchNdesStatus } from '@/api/ndes'
import type { NdesStatus } from '@/types/api'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const status = ref<NdesStatus | null>(null)

onMounted(async () => {
  try {
    status.value = await fetchNdesStatus()
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
        <h2 class="text-sm font-semibold text-foreground">NDES / AD CS</h2>
        <StatusBadge :status="status.enabled ? 'enabled' : 'disabled'" />
      </div>

      <div v-if="status.enabled" class="space-y-2 text-sm">
        <div class="flex gap-2">
          <span class="w-36 text-xs text-foreground-muted">SCEP Endpoint</span>
          <span class="font-mono text-xs text-foreground">{{ status.scep_endpoint ?? '—' }}</span>
        </div>
        <div class="flex gap-2">
          <span class="w-36 text-xs text-foreground-muted">Admin Endpoint</span>
          <span class="font-mono text-xs text-foreground">{{ status.admin_endpoint ?? '—' }}</span>
        </div>
        <div class="flex gap-2">
          <span class="w-36 text-xs text-foreground-muted">AD CS Compatible</span>
          <Badge :variant="status.adcs_compatible ? 'success' : 'secondary'">
            {{ status.adcs_compatible ? 'Yes' : 'No' }}
          </Badge>
        </div>
        <div v-if="status.connectors?.length" class="space-y-1">
          <p class="text-xs text-foreground-muted">Connectors</p>
          <div class="flex flex-wrap gap-1">
            <Badge v-for="c in status.connectors" :key="c" variant="outline" class="text-[10px]">{{ c }}</Badge>
          </div>
        </div>
      </div>
      <div v-else class="text-sm text-foreground-muted">
        NDES enrollment is not enabled in the current server configuration.
      </div>
    </Card>

    <div v-else class="rounded-lg border border-border bg-muted px-4 py-8 text-center text-sm text-foreground-muted">
      NDES status unavailable
    </div>
  </div>
</template>
