<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  ShieldCheck,
  CalendarClock,
  Ban,
  Server,
  Download,
  Activity,
  KeyRound,
} from '@lucide/vue'
import { toast } from 'vue-sonner'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import StatCard from '@/components/StatCard.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { fetchHealth } from '@/api/health'
import { fetchCertificateStats } from '@/api/certificates'
import { fetchCAInfo, fetchCAProvisioners, downloadCAChain } from '@/api/ca'
import type {
  CAInfoResponse,
  CAProvisionerDetail,
  CertificateStatsResponse,
  HealthReport,
} from '@/types/api'
import { formatDate, formatBytes } from '@/lib/format'
import { downloadBlob } from '@/lib/download'
import { extractApiError } from '@/lib/errors'

const isLoading = ref(true)
const health = ref<HealthReport | null>(null)
const stats = ref<CertificateStatsResponse | null>(null)
const caInfo = ref<CAInfoResponse | null>(null)
const provisioners = ref<CAProvisionerDetail[]>([])
const downloadingChain = ref(false)

async function loadDashboard(): Promise<void> {
  isLoading.value = true
  const [healthResult, statsResult, caResult, provResult] = await Promise.allSettled([
    fetchHealth(),
    fetchCertificateStats(),
    fetchCAInfo(),
    fetchCAProvisioners(),
  ])

  if (healthResult.status === 'fulfilled') health.value = healthResult.value
  if (statsResult.status === 'fulfilled') stats.value = statsResult.value
  if (caResult.status === 'fulfilled') caInfo.value = caResult.value
  if (provResult.status === 'fulfilled') provisioners.value = provResult.value.provisioners

  isLoading.value = false
}

async function onDownloadChain(): Promise<void> {
  downloadingChain.value = true
  try {
    const blob = await downloadCAChain()
    downloadBlob(blob, 'arx-ca-chain.zip', 'application/zip')
    toast.success('CA bundle downloaded')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to download CA bundle'))
  } finally {
    downloadingChain.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
      <p class="text-sm text-muted-foreground">Overview of your certificate authority.</p>
    </div>

    <!-- Stats -->
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <template v-if="isLoading">
        <Skeleton v-for="n in 4" :key="n" class="h-24 rounded-xl" />
      </template>
      <template v-else>
        <StatCard label="Certificates" :value="stats?.total_issued ?? 0" :icon="ShieldCheck" />
        <StatCard
          label="Expiring (30d)"
          :value="stats?.expiring_30d ?? 0"
          :icon="CalendarClock"
          accent="warning"
        />
        <StatCard
          label="Revoked"
          :value="stats?.total_revoked ?? 0"
          :icon="Ban"
          accent="destructive"
        />
        <StatCard
          label="Goroutines"
          :value="health?.memory.goroutines ?? 0"
          :icon="Activity"
          :hint="health ? formatBytes(health.memory.alloc_bytes) + ' allocated' : ''"
        />
      </template>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <!-- Server health -->
      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Server class="size-4 text-primary" /> Server health
          </CardTitle>
          <CardDescription>Runtime status of the API and CA backend.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <template v-if="isLoading">
            <Skeleton class="h-4 w-full" />
            <Skeleton class="h-4 w-3/4" />
          </template>
          <template v-else-if="health">
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">API status</span>
              <StatusBadge
                :status="health.api.status === 'ok' ? 'enabled' : 'disabled'"
                :label="health.api.status"
              />
            </div>
            <Separator />
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">CA backend</span>
              <StatusBadge
                :status="health.ca_backend.initialized ? 'enabled' : 'disabled'"
                :label="health.ca_backend.engine || health.ca_backend.status"
              />
            </div>
            <Separator />
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">Uptime</span>
              <span class="font-medium">{{ health.uptime.human }}</span>
            </div>
            <Separator />
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">Version</span>
              <span class="font-mono text-xs">{{ health.api.binary_version || health.api.version }}</span>
            </div>
          </template>
          <p v-else class="text-muted-foreground">Health data unavailable.</p>
        </CardContent>
      </Card>

      <!-- CA chain -->
      <Card>
        <CardHeader class="flex-row items-start justify-between gap-2 space-y-0">
          <div>
            <CardTitle class="flex items-center gap-2 text-base">
              <KeyRound class="size-4 text-primary" /> Certificate authority
            </CardTitle>
            <CardDescription>Root and intermediate certificates.</CardDescription>
          </div>
          <Button variant="outline" size="sm" :disabled="downloadingChain" @click="onDownloadChain">
            <Download class="size-4" /> Bundle
          </Button>
        </CardHeader>
        <CardContent class="space-y-4 text-sm">
          <template v-if="isLoading">
            <Skeleton class="h-4 w-full" />
            <Skeleton class="h-4 w-2/3" />
          </template>
          <template v-else-if="caInfo">
            <div>
              <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Root</p>
              <p class="mt-0.5 truncate font-medium">{{ caInfo.root.subject.common_name }}</p>
              <p class="text-xs text-muted-foreground">
                Expires {{ formatDate(caInfo.root.not_after) }}
              </p>
            </div>
            <Separator />
            <div>
              <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Intermediate
              </p>
              <p class="mt-0.5 truncate font-medium">
                {{ caInfo.intermediate.subject.common_name }}
              </p>
              <p class="text-xs text-muted-foreground">
                Expires {{ formatDate(caInfo.intermediate.not_after) }}
              </p>
            </div>
          </template>
          <p v-else class="text-muted-foreground">CA information unavailable.</p>
        </CardContent>
      </Card>
    </div>

    <!-- Provisioners -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">Provisioners</CardTitle>
        <CardDescription>Configured enrollment methods.</CardDescription>
      </CardHeader>
      <CardContent>
        <div v-if="isLoading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Skeleton v-for="n in 3" :key="n" class="h-16 rounded-lg" />
        </div>
        <p v-else-if="provisioners.length === 0" class="text-sm text-muted-foreground">
          No provisioners configured.
        </p>
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="provisioner in provisioners"
            :key="provisioner.name"
            class="rounded-lg border border-border bg-accent/30 px-3 py-2.5"
          >
            <p class="truncate text-sm font-medium">{{ provisioner.name }}</p>
            <p class="text-xs uppercase tracking-wide text-muted-foreground">
              {{ provisioner.type }}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
