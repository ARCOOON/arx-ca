<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { downloadCRL, fetchCRLStatus, type CRLStatus } from '@/api/crl'
import { downloadCABundle, fetchCAInfo, fetchCAProvisioners } from '@/api/ca'
import { fetchHealth } from '@/api/health'
import { listCertificates } from '@/api/certificates'
import type { CAInfoResponse, CAProvisionerDetail, HealthReport } from '@/types/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  formatCertDate,
  formatUsageList,
  parseBackendDetails,
  shortenFingerprint,
} from '@/utils/ca'
import { extractApiError } from '@/utils/errors'
import { formatBytes } from '@/utils/format'

const health = ref<HealthReport | null>(null)
const caInfo = ref<CAInfoResponse | null>(null)
const caProvisioners = ref<CAProvisionerDetail[]>([])
const certificateTotal = ref<number | null>(null)
const isLoading = ref(true)
const errorMessage = ref('')
const chainDownloading = ref(false)
const chainError = ref('')
const crlStatus = ref<CRLStatus | null>(null)
const crlDownloading = ref(false)
const crlError = ref('')

const backendDetails = computed(() => parseBackendDetails(health.value?.ca_backend.message))

const crlStatusLabel = computed(() => {
  if (!crlStatus.value) return 'Unknown'
  if (!crlStatus.value.available) return 'Unavailable'
  if (crlStatus.value.expiresAt) return `Available · next update ${crlStatus.value.expiresAt}`
  return 'Available'
})

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'healthy') return 'default'
  if (status === 'unhealthy' || status === 'degraded') return 'destructive'
  return 'secondary'
}

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const [healthReport, certificateList, caInfoReport, provisionersReport, crlReport] = await Promise.all([
      fetchHealth(),
      listCertificates(),
      fetchCAInfo(),
      fetchCAProvisioners(),
      fetchCRLStatus(),
    ])
    health.value = healthReport
    certificateTotal.value = certificateList.total
    caInfo.value = caInfoReport
    caProvisioners.value = provisionersReport.provisioners
    crlStatus.value = crlReport
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load dashboard metrics')
  } finally {
    isLoading.value = false
  }
})

async function handleDownloadCAChain(): Promise<void> {
  chainDownloading.value = true
  chainError.value = ''
  try {
    await downloadCABundle()
  } catch (error) {
    chainError.value = extractApiError(error, 'Failed to download CA bundle')
  } finally {
    chainDownloading.value = false
  }
}

async function handleDownloadCRL(format: 'der' | 'pem'): Promise<void> {
  crlDownloading.value = true
  crlError.value = ''
  try {
    await downloadCRL(format)
  } catch (error) {
    crlError.value = extractApiError(error, 'Failed to download CRL')
  } finally {
    crlDownloading.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <Alert v-if="errorMessage" variant="destructive" class="rounded-lg">
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>

    <div v-if="isLoading" class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
      <Skeleton v-for="index in 4" :key="index" class="h-24 rounded-lg" />
    </div>

    <template v-else-if="health">
      <section class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader class="pb-2">
            <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Uptime</CardTitle>
          </CardHeader>
          <CardContent>
            <p class="text-lg font-semibold">{{ health.uptime.human }}</p>
            <p class="text-xs text-muted-foreground">{{ health.uptime.seconds }} seconds</p>
          </CardContent>
        </Card>

        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader class="pb-2">
            <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">API</CardTitle>
          </CardHeader>
          <CardContent class="flex items-center gap-2">
            <Badge :variant="statusVariant(health.api.status)" class="rounded-md">{{ health.api.status }}</Badge>
            <span class="text-xs text-muted-foreground">v{{ health.api.version }}</span>
          </CardContent>
        </Card>

        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader class="pb-2">
            <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">CA Backend</CardTitle>
          </CardHeader>
          <CardContent>
            <div class="flex flex-wrap items-center gap-2">
              <Badge :variant="statusVariant(health.ca_backend.status)" class="rounded-md">
                {{ health.ca_backend.status }}
              </Badge>
              <span class="text-xs text-muted-foreground">{{ health.ca_backend.engine }}</span>
            </div>
            <dl v-if="backendDetails.length > 0" class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs">
              <template v-for="detail in backendDetails" :key="detail.label">
                <dt class="text-muted-foreground">{{ detail.label }}</dt>
                <dd>{{ detail.value }}</dd>
              </template>
            </dl>
          </CardContent>
        </Card>

        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader class="pb-2">
            <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Certificates</CardTitle>
          </CardHeader>
          <CardContent>
            <p class="text-lg font-semibold">{{ certificateTotal ?? '—' }}</p>
            <p class="text-xs text-muted-foreground">Total issued</p>
          </CardContent>
        </Card>
      </section>

      <section class="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader>
            <CardTitle class="text-sm">Memory</CardTitle>
          </CardHeader>
          <CardContent class="grid grid-cols-2 gap-3 text-sm">
            <div>
              <p class="text-muted-foreground">Heap alloc</p>
              <p class="font-medium">{{ formatBytes(health.memory.heap_alloc_bytes) }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Goroutines</p>
              <p class="font-medium">{{ health.memory.goroutines }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">GC cycles</p>
              <p class="font-medium">{{ health.memory.num_gc }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Binary</p>
              <p class="font-medium">{{ health.api.binary_version }}</p>
            </div>
          </CardContent>
        </Card>

        <Card class="rounded-lg border border-border shadow-none">
          <CardHeader>
            <CardTitle class="text-sm">CRL &amp; Chain</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <p class="text-sm text-muted-foreground">{{ crlStatusLabel }}</p>
            <div class="flex flex-wrap gap-2">
              <Button variant="secondary" size="sm" class="rounded-lg" :disabled="crlDownloading" @click="handleDownloadCRL('pem')">
                Download CRL (PEM)
              </Button>
              <Button variant="secondary" size="sm" class="rounded-lg" :disabled="crlDownloading" @click="handleDownloadCRL('der')">
                Download CRL (DER)
              </Button>
              <Button variant="secondary" size="sm" class="rounded-lg" :disabled="chainDownloading" @click="handleDownloadCAChain">
                Download CA Chain
              </Button>
            </div>
            <p v-if="crlError" class="text-xs text-destructive">{{ crlError }}</p>
            <p v-if="chainError" class="text-xs text-destructive">{{ chainError }}</p>
          </CardContent>
        </Card>
      </section>

      <section v-if="caInfo" class="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <Card
          v-for="cert in [caInfo.root, caInfo.intermediate]"
          :key="cert.serial_number"
          class="rounded-lg border border-border shadow-none"
        >
          <CardHeader>
            <CardTitle class="text-sm">{{ cert.subject.common_name }}</CardTitle>
          </CardHeader>
          <CardContent class="space-y-2 text-sm">
            <p><span class="text-muted-foreground">Serial:</span> {{ cert.serial_number }}</p>
            <p><span class="text-muted-foreground">Valid:</span> {{ formatCertDate(cert.not_before) }} – {{ formatCertDate(cert.not_after) }}</p>
            <p><span class="text-muted-foreground">Fingerprint:</span> {{ shortenFingerprint(cert.fingerprint) }}</p>
            <p v-if="cert.key_usages?.length"><span class="text-muted-foreground">Key usage:</span> {{ formatUsageList(cert.key_usages) }}</p>
          </CardContent>
        </Card>
      </section>

      <Card v-if="caProvisioners.length > 0" class="rounded-lg border border-border shadow-none">
        <CardHeader>
          <CardTitle class="text-sm">Provisioners</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="flex flex-wrap gap-2">
            <Badge
              v-for="provisioner in caProvisioners"
              :key="provisioner.name"
              variant="outline"
              class="rounded-md"
            >
              {{ provisioner.name }} ({{ provisioner.type }})
            </Badge>
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
