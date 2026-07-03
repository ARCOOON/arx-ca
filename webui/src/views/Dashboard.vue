<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Spinner from '@/components/ui/Spinner.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { fetchHealth } from '@/api/health'
import { fetchCAInfo } from '@/api/ca'
import { fetchCertificateStats } from '@/api/certificates'
import { fetchSshStats } from '@/api/ssh'
import type { HealthReport, CAInfoResponse, CertificateStatsResponse, SshStatsResponse } from '@/types/api'
import { formatBytes, formatDate } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'

const loading = ref(true)
const error = ref<string | null>(null)

const health = ref<HealthReport | null>(null)
const caInfo = ref<CAInfoResponse | null>(null)
const certStats = ref<CertificateStatsResponse | null>(null)
const sshStats = ref<SshStatsResponse | null>(null)

onMounted(async () => {
  try {
    const [h, ca, cs, ss] = await Promise.all([
      fetchHealth(),
      fetchCAInfo().catch(() => null),
      fetchCertificateStats().catch(() => null),
      fetchSshStats().catch(() => null),
    ])
    health.value = h
    caInfo.value = ca
    certStats.value = cs
    sshStats.value = ss
  } catch (err) {
    error.value = extractErrorMessage(err)
  } finally {
    loading.value = false
  }
})

const caStatus = computed(() => {
  if (!health.value) return 'unknown'
  return health.value.ca_backend.initialized ? 'online' : 'offline'
})
</script>

<template>
  <div class="space-y-6">
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-24">
      <Spinner size="lg" />
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ error }}
    </div>

    <template v-else-if="health">
      <!-- Status row -->
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Card class="px-4 py-4">
          <p class="text-xs text-foreground-muted mb-1">API Status</p>
          <StatusBadge :status="health.api.status" class="text-sm" />
        </Card>
        <Card class="px-4 py-4">
          <p class="text-xs text-foreground-muted mb-1">CA Backend</p>
          <StatusBadge :status="caStatus" class="text-sm" />
        </Card>
        <Card class="px-4 py-4">
          <p class="text-xs text-foreground-muted mb-1">Uptime</p>
          <p class="text-sm font-semibold text-foreground">{{ health.uptime.human }}</p>
        </Card>
        <Card class="px-4 py-4">
          <p class="text-xs text-foreground-muted mb-1">Version</p>
          <p class="text-sm font-semibold text-foreground font-mono">
            {{ health.api.binary_version || health.api.version }}
          </p>
        </Card>
      </div>

      <!-- Certificate & SSH stats -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Card class="px-5 py-5">
          <div class="flex items-start justify-between">
            <div>
              <p class="text-xs text-foreground-muted mb-2">Total Issued</p>
              <p class="text-2xl font-bold text-foreground tabular-nums">
                {{ certStats?.total_issued ?? '—' }}
              </p>
            </div>
            <Badge variant="default">X.509</Badge>
          </div>
        </Card>
        <Card class="px-5 py-5">
          <div class="flex items-start justify-between">
            <div>
              <p class="text-xs text-foreground-muted mb-2">Expiring in 30d</p>
              <p class="text-2xl font-bold tabular-nums"
                :class="(certStats?.expiring_30d ?? 0) > 0 ? 'text-warning' : 'text-foreground'"
              >
                {{ certStats?.expiring_30d ?? '—' }}
              </p>
            </div>
            <Badge :variant="(certStats?.expiring_30d ?? 0) > 0 ? 'warning' : 'secondary'">Expiring</Badge>
          </div>
        </Card>
        <Card class="px-5 py-5">
          <div class="flex items-start justify-between">
            <div>
              <p class="text-xs text-foreground-muted mb-2">Revoked</p>
              <p class="text-2xl font-bold text-foreground tabular-nums">
                {{ certStats?.total_revoked ?? '—' }}
              </p>
            </div>
            <Badge variant="outline">CRL</Badge>
          </div>
        </Card>
      </div>

      <!-- SSH stats -->
      <div v-if="sshStats" class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">SSH User Certs</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ sshStats.total_user_certs }}</p>
        </Card>
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">SSH Host Certs</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ sshStats.total_host_certs }}</p>
        </Card>
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">SSH Active Now</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ sshStats.active_now }}</p>
        </Card>
      </div>

      <!-- CA Info -->
      <div v-if="caInfo" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Card class="px-5 py-5 space-y-3">
          <div class="flex items-center justify-between">
            <p class="text-sm font-semibold text-foreground">Root CA</p>
            <Badge variant="secondary">Root</Badge>
          </div>
          <div class="space-y-1.5 text-xs">
            <Row label="CN" :value="caInfo.root.subject.common_name" />
            <Row label="Algorithm" :value="caInfo.root.signature_algorithm" />
            <Row label="Not After" :value="formatDate(caInfo.root.not_after)" />
            <Row label="Fingerprint" :value="caInfo.root.fingerprint" mono />
          </div>
        </Card>
        <Card class="px-5 py-5 space-y-3">
          <div class="flex items-center justify-between">
            <p class="text-sm font-semibold text-foreground">Intermediate CA</p>
            <Badge variant="info">Intermediate</Badge>
          </div>
          <div class="space-y-1.5 text-xs">
            <Row label="CN" :value="caInfo.intermediate.subject.common_name" />
            <Row label="Algorithm" :value="caInfo.intermediate.signature_algorithm" />
            <Row label="Not After" :value="formatDate(caInfo.intermediate.not_after)" />
            <Row label="Fingerprint" :value="caInfo.intermediate.fingerprint" mono />
          </div>
        </Card>
      </div>

      <!-- Memory metrics -->
      <Card class="px-5 py-5 space-y-3">
        <p class="text-sm font-semibold text-foreground">Runtime Memory</p>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 text-xs">
          <div>
            <p class="text-foreground-muted mb-0.5">Heap Alloc</p>
            <p class="font-semibold text-foreground">{{ formatBytes(health.memory.heap_alloc_bytes) }}</p>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">Heap In-use</p>
            <p class="font-semibold text-foreground">{{ formatBytes(health.memory.heap_inuse_bytes) }}</p>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">Goroutines</p>
            <p class="font-semibold text-foreground">{{ health.memory.goroutines }}</p>
          </div>
          <div>
            <p class="text-foreground-muted mb-0.5">GC Cycles</p>
            <p class="font-semibold text-foreground">{{ health.memory.num_gc }}</p>
          </div>
        </div>
      </Card>
    </template>
  </div>
</template>

<script lang="ts">
const Row = {
  props: { label: String, value: String, mono: Boolean },
  template: `
    <div class="flex items-start gap-2">
      <span class="w-20 shrink-0 text-foreground-muted">{{ label }}</span>
      <span :class="['text-foreground truncate', mono ? 'font-mono text-[10px]' : '']">{{ value }}</span>
    </div>
  `,
}

export default { components: { Row } }
</script>
