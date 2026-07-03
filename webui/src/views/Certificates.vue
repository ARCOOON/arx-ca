<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { CertificateLifecycleStatus } from '@/utils/certificate'
import {
  fetchCertificateBySerial,
  fetchCertificateStats,
  generateCertificate,
  issueCertificate,
  listCertificates,
  revokeCertificate,
} from '@/api/certificates'
import type {
  CertificateRecordDetail,
  CertificateStatsResponse,
  CertificateSummary,
  KeyAlgorithm,
} from '@/types/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { extractCommonName, resolveCertificateStatus } from '@/utils/certificate'
import { downloadTextFile } from '@/utils/download'
import { extractApiError } from '@/utils/errors'
import { formatDateTime } from '@/utils/format'

const REVOKE_REASONS = [
  { label: 'Unspecified', code: 0 },
  { label: 'Key compromise', code: 1 },
  { label: 'Superseded', code: 4 },
  { label: 'Cessation of operation', code: 5 },
] as const

const certificates = ref<CertificateSummary[]>([])
const isLoading = ref(true)
const errorMessage = ref('')

const filterCommonName = ref('')
const filterSerial = ref('')
const filterStatus = ref<'all' | CertificateLifecycleStatus>('all')

const certStats = ref<CertificateStatsResponse | null>(null)

const issueModalOpen = ref(false)
const issueMode = ref<'csr' | 'generate'>('csr')
const csrInput = ref('')
const ttlInput = ref('720h')
const isIssuing = ref(false)
const issueError = ref('')

const nativeCommonName = ref('')
const nativeSans = ref('')
const nativeKeyAlgo = ref<KeyAlgorithm>('ECDSA256')
const nativeTtl = ref('720h')

const detailsModalOpen = ref(false)
const detailsLoading = ref(false)
const detailsError = ref('')
const certificateDetail = ref<CertificateRecordDetail | null>(null)

const revokeModalOpen = ref(false)
const revokeTargetSerial = ref('')
const revokeReasonCode = ref(0)
const revokeLoading = ref(false)
const revokeError = ref('')

const statusVariant = (status: CertificateLifecycleStatus): 'default' | 'destructive' | 'secondary' => {
  if (status === 'valid') return 'default'
  if (status === 'revoked') return 'destructive'
  return 'secondary'
}

async function loadCertificates(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await listCertificates({
      common_name: filterCommonName.value || undefined,
      serial_number: filterSerial.value || undefined,
      status: filterStatus.value === 'all' ? undefined : filterStatus.value,
    })
    certificates.value = response.certificates
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load certificates')
    certificates.value = []
  } finally {
    isLoading.value = false
  }
}

async function loadStats(): Promise<void> {
  try {
    certStats.value = await fetchCertificateStats()
  } catch {
    certStats.value = null
  }
}

onMounted(() => {
  void loadCertificates()
  void loadStats()
})

function applyFilters(): void {
  void loadCertificates()
}

async function submitIssue(): Promise<void> {
  isIssuing.value = true
  issueError.value = ''
  try {
    if (issueMode.value === 'csr') {
      const result = await issueCertificate({ csr: csrInput.value.trim(), ttl: ttlInput.value.trim() || undefined })
      downloadTextFile(`${result.serial}.pem`, result.certificate_pem)
    } else {
      const sans = nativeSans.value.split(',').map((s) => s.trim()).filter(Boolean)
      const result = await generateCertificate({
        common_name: nativeCommonName.value.trim(),
        sans: sans.length > 0 ? sans : undefined,
        ttl: nativeTtl.value.trim() || undefined,
        key_algo: nativeKeyAlgo.value,
      })
      downloadTextFile(`${nativeCommonName.value || 'certificate'}.pem`, result.certificate_pem)
      if (result.private_key_pem) {
        downloadTextFile(`${nativeCommonName.value || 'certificate'}-key.pem`, result.private_key_pem)
      }
    }
    issueModalOpen.value = false
    void loadCertificates()
    void loadStats()
  } catch (error) {
    issueError.value = extractApiError(error, 'Failed to issue certificate')
  } finally {
    isIssuing.value = false
  }
}

async function openDetails(serial: string): Promise<void> {
  detailsModalOpen.value = true
  detailsLoading.value = true
  detailsError.value = ''
  certificateDetail.value = null
  try {
    certificateDetail.value = await fetchCertificateBySerial(serial)
  } catch (error) {
    detailsError.value = extractApiError(error, 'Failed to load certificate details')
  } finally {
    detailsLoading.value = false
  }
}

function openRevoke(serial: string): void {
  revokeTargetSerial.value = serial
  revokeReasonCode.value = 0
  revokeError.value = ''
  revokeModalOpen.value = true
}

async function submitRevoke(): Promise<void> {
  revokeLoading.value = true
  revokeError.value = ''
  try {
    await revokeCertificate({
      serial_number: revokeTargetSerial.value,
      reason_code: revokeReasonCode.value,
    })
    revokeModalOpen.value = false
    void loadCertificates()
    void loadStats()
  } catch (error) {
    revokeError.value = extractApiError(error, 'Failed to revoke certificate')
  } finally {
    revokeLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <section class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Issued</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ certStats?.total_issued ?? '—' }}</p>
        </CardContent>
      </Card>
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Expiring (30d)</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ certStats?.expiring_30d ?? '—' }}</p>
        </CardContent>
      </Card>
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Revoked</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ certStats?.total_revoked ?? '—' }}</p>
        </CardContent>
      </Card>
    </section>

    <Card class="rounded-lg border border-border shadow-none">
      <CardHeader class="flex-row items-center justify-between gap-3">
        <CardTitle class="text-sm">Certificate inventory</CardTitle>
        <Button class="rounded-lg" @click="issueModalOpen = true">Issue certificate</Button>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid grid-cols-1 gap-2 md:grid-cols-4">
          <Input v-model="filterCommonName" placeholder="Common name" class="rounded-lg" />
          <Input v-model="filterSerial" placeholder="Serial number" class="rounded-lg font-mono text-xs" />
          <Select v-model="filterStatus">
            <SelectTrigger class="rounded-lg">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent class="rounded-lg">
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="valid">Valid</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
              <SelectItem value="revoked">Revoked</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="secondary" class="rounded-lg" @click="applyFilters">Apply filters</Button>
        </div>

        <Alert v-if="errorMessage" variant="destructive" class="rounded-lg">
          <AlertDescription>{{ errorMessage }}</AlertDescription>
        </Alert>

        <div class="overflow-x-auto rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Serial</TableHead>
                <TableHead>Common name</TableHead>
                <TableHead>Issued</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead>Status</TableHead>
                <TableHead class="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="isLoading">
                <TableCell colspan="6" class="text-center text-muted-foreground">Loading…</TableCell>
              </TableRow>
              <TableRow v-else-if="certificates.length === 0">
                <TableCell colspan="6" class="text-center text-muted-foreground">No certificates found.</TableCell>
              </TableRow>
              <TableRow v-for="cert in certificates" :key="cert.serial">
                <TableCell class="font-mono text-xs">{{ cert.serial }}</TableCell>
                <TableCell>{{ extractCommonName(cert.subject) }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(cert.not_before) }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(cert.not_after) }}</TableCell>
                <TableCell>
                  <Badge :variant="statusVariant(resolveCertificateStatus(cert))" class="rounded-md capitalize">
                    {{ resolveCertificateStatus(cert) }}
                  </Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" class="rounded-md" @click="openDetails(cert.serial)">Details</Button>
                    <Button
                      v-if="!cert.revoked"
                      variant="ghost"
                      size="sm"
                      class="rounded-md text-destructive"
                      @click="openRevoke(cert.serial)"
                    >
                      Revoke
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>

    <!-- Issue dialog -->
    <Dialog v-model:open="issueModalOpen">
      <DialogContent class="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Issue certificate</DialogTitle>
        </DialogHeader>
        <Tabs v-model="issueMode">
          <TabsList class="rounded-lg">
            <TabsTrigger value="csr" class="rounded-md">CSR signing</TabsTrigger>
            <TabsTrigger value="generate" class="rounded-md">Native generate</TabsTrigger>
          </TabsList>
          <TabsContent value="csr" class="space-y-3 pt-3">
            <div class="space-y-2">
              <Label>CSR (PEM)</Label>
              <Textarea v-model="csrInput" rows="6" class="rounded-lg font-mono text-xs" />
            </div>
            <div class="space-y-2">
              <Label>TTL</Label>
              <Input v-model="ttlInput" class="rounded-lg" />
            </div>
          </TabsContent>
          <TabsContent value="generate" class="space-y-3 pt-3">
            <div class="space-y-2">
              <Label>Common name</Label>
              <Input v-model="nativeCommonName" class="rounded-lg" />
            </div>
            <div class="space-y-2">
              <Label>SANs (comma-separated)</Label>
              <Input v-model="nativeSans" class="rounded-lg" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-2">
                <Label>Key algorithm</Label>
                <Select v-model="nativeKeyAlgo">
                  <SelectTrigger class="rounded-lg"><SelectValue /></SelectTrigger>
                  <SelectContent class="rounded-lg">
                    <SelectItem value="ECDSA256">ECDSA256</SelectItem>
                    <SelectItem value="RSA2048">RSA2048</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="space-y-2">
                <Label>TTL</Label>
                <Input v-model="nativeTtl" class="rounded-lg" />
              </div>
            </div>
          </TabsContent>
        </Tabs>
        <Alert v-if="issueError" variant="destructive" class="rounded-lg">
          <AlertDescription>{{ issueError }}</AlertDescription>
        </Alert>
        <DialogFooter>
          <Button class="rounded-lg" :disabled="isIssuing" @click="submitIssue">
            {{ isIssuing ? 'Issuing…' : 'Issue' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Details dialog -->
    <Dialog v-model:open="detailsModalOpen">
      <DialogContent class="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Certificate details</DialogTitle>
        </DialogHeader>
        <p v-if="detailsLoading" class="text-sm text-muted-foreground">Loading…</p>
        <Alert v-else-if="detailsError" variant="destructive" class="rounded-lg">
          <AlertDescription>{{ detailsError }}</AlertDescription>
        </Alert>
        <div v-else-if="certificateDetail" class="space-y-2 text-sm">
          <p><span class="text-muted-foreground">Serial:</span> {{ certificateDetail.serial }}</p>
          <p><span class="text-muted-foreground">Subject:</span> {{ certificateDetail.subject }}</p>
          <p><span class="text-muted-foreground">Valid:</span> {{ formatDateTime(certificateDetail.not_before) }} – {{ formatDateTime(certificateDetail.not_after) }}</p>
          <p><span class="text-muted-foreground">Requestor:</span> {{ certificateDetail.requestor_id }}</p>
          <p v-if="certificateDetail.revoked" class="text-destructive">Revoked</p>
          <Textarea :model-value="certificateDetail.certificate_pem" readonly rows="6" class="rounded-lg font-mono text-xs" />
        </div>
        <DialogFooter>
          <Button variant="secondary" class="rounded-lg" @click="detailsModalOpen = false">Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Revoke dialog -->
    <Dialog v-model:open="revokeModalOpen">
      <DialogContent class="max-w-md rounded-lg">
        <DialogHeader>
          <DialogTitle>Revoke certificate</DialogTitle>
        </DialogHeader>
        <p class="text-sm text-muted-foreground">Serial: <span class="font-mono">{{ revokeTargetSerial }}</span></p>
        <div class="space-y-2">
          <Label>Reason</Label>
          <Select v-model="revokeReasonCode">
            <SelectTrigger class="rounded-lg"><SelectValue /></SelectTrigger>
            <SelectContent class="rounded-lg">
              <SelectItem v-for="reason in REVOKE_REASONS" :key="reason.code" :value="reason.code">
                {{ reason.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Alert v-if="revokeError" variant="destructive" class="rounded-lg">
          <AlertDescription>{{ revokeError }}</AlertDescription>
        </Alert>
        <DialogFooter>
          <Button variant="destructive" class="rounded-lg" :disabled="revokeLoading" @click="submitRevoke">
            {{ revokeLoading ? 'Revoking…' : 'Revoke' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
