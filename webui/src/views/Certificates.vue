<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  ShieldCheck,
  CalendarClock,
  Ban,
  Search,
  RefreshCw,
  Plus,
  Eye,
  Download,
  Copy,
  Loader2,
} from '@lucide/vue'
import { toast } from 'vue-sonner'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Separator } from '@/components/ui/separator'
import StatCard from '@/components/StatCard.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import {
  listCertificates,
  fetchCertificateStats,
  fetchCertificate,
  issueCertificate,
  issueCertificateWithToken,
  autoIssueCertificate,
  revokeCertificate,
  downloadCertificateBundle,
} from '@/api/certificates'
import type {
  CertificateRecordDetail,
  CertificateStatsResponse,
  CertificateSummary,
} from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import { formatDate, formatDateTime, isExpiringSoon } from '@/lib/format'
import { downloadBlob, copyToClipboard } from '@/lib/download'
import { extractApiError } from '@/lib/errors'

const authStore = useAuthStore()
const isSuperAdmin = computed(() => authStore.hasRole('SuperAdmin'))

const stats = ref<CertificateStatsResponse | null>(null)
const certificates = ref<CertificateSummary[]>([])
const isLoading = ref(false)

const filterCommonName = ref('')
const filterSerial = ref('')
const filterStatus = ref<'all' | 'valid' | 'revoked' | 'expired'>('all')

function statusOf(cert: CertificateSummary): 'valid' | 'revoked' | 'expired' {
  if (cert.revoked) return 'revoked'
  if (new Date(cert.not_after).getTime() < Date.now()) return 'expired'
  return 'valid'
}

async function loadStats(): Promise<void> {
  try {
    stats.value = await fetchCertificateStats()
  } catch {
    // Stats are non-critical for the list workflow.
  }
}

async function loadCertificates(): Promise<void> {
  isLoading.value = true
  try {
    const data = await listCertificates({
      common_name: filterCommonName.value.trim() || undefined,
      serial_number: filterSerial.value.trim() || undefined,
      status: filterStatus.value === 'all' ? undefined : filterStatus.value,
    })
    certificates.value = data.certificates
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to load certificates'))
  } finally {
    isLoading.value = false
  }
}

function refreshAll(): void {
  void loadStats()
  void loadCertificates()
}

/* ----------------------------- Issue dialog ----------------------------- */

const issueOpen = ref(false)
const issueTab = ref('csr')
const issueBusy = ref(false)

const csrForm = ref({ csr: '', ttl: '' })
const tokenForm = ref({ token: '', csr: '', ttl: '' })
const autoForm = ref({ common_name: '', dns_sans: '', ip_sans: '', ttl: '' })

const issuedPem = ref('')
const issuedKey = ref('')
const issuedSerial = ref('')

function resetIssueForms(): void {
  csrForm.value = { csr: '', ttl: '' }
  tokenForm.value = { token: '', csr: '', ttl: '' }
  autoForm.value = { common_name: '', dns_sans: '', ip_sans: '', ttl: '' }
  issuedPem.value = ''
  issuedKey.value = ''
  issuedSerial.value = ''
}

function openIssue(): void {
  resetIssueForms()
  issueTab.value = 'csr'
  issueOpen.value = true
}

function splitList(value: string): string[] {
  return value
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean)
}

async function submitCsr(): Promise<void> {
  if (!csrForm.value.csr.trim()) {
    toast.error('A PEM-encoded CSR is required.')
    return
  }
  issueBusy.value = true
  try {
    const result = await issueCertificate({
      csr: csrForm.value.csr.trim(),
      ttl: csrForm.value.ttl.trim() || undefined,
    })
    issuedPem.value = result.certificate_pem
    issuedSerial.value = result.serial
    toast.success('Certificate issued')
    refreshAll()
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to issue certificate'))
  } finally {
    issueBusy.value = false
  }
}

async function submitToken(): Promise<void> {
  if (!tokenForm.value.token.trim() || !tokenForm.value.csr.trim()) {
    toast.error('Token and CSR are required.')
    return
  }
  issueBusy.value = true
  try {
    const result = await issueCertificateWithToken({
      token: tokenForm.value.token.trim(),
      csr: tokenForm.value.csr.trim(),
      ttl: tokenForm.value.ttl.trim() || undefined,
    })
    issuedPem.value = result.certificate_pem
    issuedSerial.value = result.serial
    toast.success('Certificate issued with token')
    refreshAll()
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to issue certificate'))
  } finally {
    issueBusy.value = false
  }
}

async function submitAuto(): Promise<void> {
  if (!autoForm.value.common_name.trim()) {
    toast.error('A common name is required.')
    return
  }
  issueBusy.value = true
  try {
    const result = await autoIssueCertificate({
      common_name: autoForm.value.common_name.trim(),
      dns_sans: splitList(autoForm.value.dns_sans),
      ip_sans: splitList(autoForm.value.ip_sans),
      ttl: autoForm.value.ttl.trim() || undefined,
    })
    issuedPem.value = result.certificate_pem
    issuedKey.value = result.private_key_pem
    issuedSerial.value = result.serial
    toast.success('Certificate generated')
    refreshAll()
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to generate certificate'))
  } finally {
    issueBusy.value = false
  }
}

async function copyText(value: string, label: string): Promise<void> {
  if (await copyToClipboard(value)) {
    toast.success(`${label} copied`)
  } else {
    toast.error('Clipboard unavailable')
  }
}

/* ----------------------------- Details dialog ---------------------------- */

const detailOpen = ref(false)
const detailLoading = ref(false)
const detail = ref<CertificateRecordDetail | null>(null)

async function openDetail(serial: string): Promise<void> {
  detailOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await fetchCertificate(serial)
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to load certificate'))
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

async function onDownloadBundle(serial: string): Promise<void> {
  try {
    const blob = await downloadCertificateBundle(serial)
    downloadBlob(blob, `certificate-${serial}.zip`, 'application/zip')
    toast.success('Bundle downloaded')
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to download bundle'))
  }
}

/* ------------------------------- Revoke ---------------------------------- */

const revokeOpen = ref(false)
const revokeBusy = ref(false)
const revokeSerial = ref('')
const revokeReasonCode = ref('0')

const reasonCodes = [
  { value: '0', label: 'Unspecified' },
  { value: '1', label: 'Key compromise' },
  { value: '3', label: 'Affiliation changed' },
  { value: '4', label: 'Superseded' },
  { value: '5', label: 'Cessation of operation' },
]

function openRevoke(serial: string): void {
  revokeSerial.value = serial
  revokeReasonCode.value = '0'
  revokeOpen.value = true
}

async function submitRevoke(): Promise<void> {
  revokeBusy.value = true
  try {
    await revokeCertificate({
      serial_number: revokeSerial.value,
      reason_code: Number(revokeReasonCode.value),
    })
    toast.success('Certificate revoked')
    revokeOpen.value = false
    detailOpen.value = false
    refreshAll()
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to revoke certificate'))
  } finally {
    revokeBusy.value = false
  }
}

onMounted(refreshAll)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Certificates</h1>
        <p class="text-sm text-muted-foreground">Issue, inspect, and revoke X.509 certificates.</p>
      </div>
      <Button @click="openIssue">
        <Plus class="size-4" /> Issue certificate
      </Button>
    </div>

    <div class="grid gap-4 sm:grid-cols-3">
      <StatCard label="Issued" :value="stats?.total_issued ?? 0" :icon="ShieldCheck" />
      <StatCard
        label="Expiring (30d)"
        :value="stats?.expiring_30d ?? 0"
        :icon="CalendarClock"
        accent="warning"
      />
      <StatCard label="Revoked" :value="stats?.total_revoked ?? 0" :icon="Ban" accent="destructive" />
    </div>

    <Card>
      <CardHeader class="gap-3">
        <div class="flex flex-wrap items-end gap-3">
          <div class="min-w-[180px] flex-1 space-y-1.5">
            <Label for="cn">Common name</Label>
            <Input id="cn" v-model="filterCommonName" placeholder="example.com" @keyup.enter="loadCertificates" />
          </div>
          <div class="min-w-[180px] flex-1 space-y-1.5">
            <Label for="serial">Serial</Label>
            <Input id="serial" v-model="filterSerial" placeholder="Serial number" @keyup.enter="loadCertificates" />
          </div>
          <div class="w-40 space-y-1.5">
            <Label>Status</Label>
            <Select v-model="filterStatus">
              <SelectTrigger class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="valid">Valid</SelectItem>
                <SelectItem value="revoked">Revoked</SelectItem>
                <SelectItem value="expired">Expired</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="secondary" @click="loadCertificates">
            <Search class="size-4" /> Search
          </Button>
          <Button variant="ghost" size="icon" title="Refresh" @click="refreshAll">
            <RefreshCw class="size-4" :class="{ 'animate-spin': isLoading }" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Serial</TableHead>
              <TableHead>Subject</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Expires</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="isLoading">
              <TableCell colspan="5" class="py-10 text-center text-muted-foreground">
                <Loader2 class="mx-auto size-5 animate-spin" />
              </TableCell>
            </TableRow>
            <TableRow v-else-if="certificates.length === 0">
              <TableCell colspan="5" class="py-10 text-center text-muted-foreground">
                No certificates found.
              </TableCell>
            </TableRow>
            <TableRow v-for="cert in certificates" :key="cert.serial">
              <TableCell class="max-w-[160px] truncate font-mono text-xs">{{ cert.serial }}</TableCell>
              <TableCell class="max-w-[240px] truncate">{{ cert.subject }}</TableCell>
              <TableCell><StatusBadge :status="statusOf(cert)" /></TableCell>
              <TableCell>
                <span :class="{ 'text-warning': isExpiringSoon(cert.not_after) && !cert.revoked }">
                  {{ formatDate(cert.not_after) }}
                </span>
              </TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="icon-sm" title="View details" @click="openDetail(cert.serial)">
                  <Eye class="size-4" />
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- Issue dialog -->
    <Dialog v-model:open="issueOpen">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Issue certificate</DialogTitle>
          <DialogDescription>Sign a CSR, redeem a token, or auto-generate a key pair.</DialogDescription>
        </DialogHeader>

        <div v-if="issuedPem" class="space-y-3">
          <div class="rounded-md border border-success/30 bg-success/10 px-3 py-2 text-sm">
            Certificate <span class="font-mono">{{ issuedSerial }}</span> issued successfully.
          </div>
          <div class="space-y-1.5">
            <Label>Certificate (PEM)</Label>
            <Textarea :model-value="issuedPem" readonly rows="6" class="font-mono text-xs" />
            <div class="flex gap-2">
              <Button variant="secondary" size="sm" @click="copyText(issuedPem, 'Certificate')">
                <Copy class="size-4" /> Copy
              </Button>
              <Button variant="secondary" size="sm" @click="downloadBlob(issuedPem, `certificate-${issuedSerial}.pem`, 'application/x-pem-file')">
                <Download class="size-4" /> Download
              </Button>
            </div>
          </div>
          <div v-if="issuedKey" class="space-y-1.5">
            <Label>Private key (PEM)</Label>
            <Textarea :model-value="issuedKey" readonly rows="6" class="font-mono text-xs" />
            <p class="text-xs text-muted-foreground">
              Store this key securely — it is not retained after you close this dialog.
            </p>
            <div class="flex gap-2">
              <Button variant="secondary" size="sm" @click="copyText(issuedKey, 'Private key')">
                <Copy class="size-4" /> Copy
              </Button>
              <Button variant="secondary" size="sm" @click="downloadBlob(issuedKey, `private-${issuedSerial}.key`, 'application/x-pem-file')">
                <Download class="size-4" /> Download
              </Button>
            </div>
          </div>
        </div>

        <Tabs v-else v-model="issueTab">
          <TabsList class="w-full">
            <TabsTrigger value="csr" class="flex-1">CSR</TabsTrigger>
            <TabsTrigger value="token" class="flex-1">Token</TabsTrigger>
            <TabsTrigger v-if="isSuperAdmin" value="auto" class="flex-1">Auto</TabsTrigger>
          </TabsList>

          <TabsContent value="csr" class="space-y-3">
            <div class="space-y-1.5">
              <Label for="csr-body">Certificate signing request</Label>
              <Textarea id="csr-body" v-model="csrForm.csr" rows="7" placeholder="-----BEGIN CERTIFICATE REQUEST-----" class="font-mono text-xs" />
            </div>
            <div class="space-y-1.5">
              <Label for="csr-ttl">TTL (optional)</Label>
              <Input id="csr-ttl" v-model="csrForm.ttl" placeholder="e.g. 720h" />
            </div>
            <Button class="w-full" :disabled="issueBusy" @click="submitCsr">
              <Loader2 v-if="issueBusy" class="size-4 animate-spin" /> Sign CSR
            </Button>
          </TabsContent>

          <TabsContent value="token" class="space-y-3">
            <div class="space-y-1.5">
              <Label for="tok">Provisioner token</Label>
              <Input id="tok" v-model="tokenForm.token" placeholder="JWT" class="font-mono text-xs" />
            </div>
            <div class="space-y-1.5">
              <Label for="tok-csr">CSR</Label>
              <Textarea id="tok-csr" v-model="tokenForm.csr" rows="6" placeholder="-----BEGIN CERTIFICATE REQUEST-----" class="font-mono text-xs" />
            </div>
            <div class="space-y-1.5">
              <Label for="tok-ttl">TTL (optional)</Label>
              <Input id="tok-ttl" v-model="tokenForm.ttl" placeholder="e.g. 720h" />
            </div>
            <Button class="w-full" :disabled="issueBusy" @click="submitToken">
              <Loader2 v-if="issueBusy" class="size-4 animate-spin" /> Issue with token
            </Button>
          </TabsContent>

          <TabsContent v-if="isSuperAdmin" value="auto" class="space-y-3">
            <div class="space-y-1.5">
              <Label for="auto-cn">Common name</Label>
              <Input id="auto-cn" v-model="autoForm.common_name" placeholder="service.internal" />
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="space-y-1.5">
                <Label for="auto-dns">DNS SANs</Label>
                <Input id="auto-dns" v-model="autoForm.dns_sans" placeholder="a.com, b.com" />
              </div>
              <div class="space-y-1.5">
                <Label for="auto-ip">IP SANs</Label>
                <Input id="auto-ip" v-model="autoForm.ip_sans" placeholder="10.0.0.1" />
              </div>
            </div>
            <div class="space-y-1.5">
              <Label for="auto-ttl">TTL (optional)</Label>
              <Input id="auto-ttl" v-model="autoForm.ttl" placeholder="e.g. 720h" />
            </div>
            <Button class="w-full" :disabled="issueBusy" @click="submitAuto">
              <Loader2 v-if="issueBusy" class="size-4 animate-spin" /> Generate certificate + key
            </Button>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>

    <!-- Detail dialog -->
    <Dialog v-model:open="detailOpen">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Certificate details</DialogTitle>
        </DialogHeader>
        <div v-if="detailLoading" class="py-8 text-center">
          <Loader2 class="mx-auto size-5 animate-spin text-muted-foreground" />
        </div>
        <div v-else-if="detail" class="space-y-4 text-sm">
          <div class="grid grid-cols-[110px_1fr] gap-y-2">
            <span class="text-muted-foreground">Serial</span>
            <span class="break-all font-mono text-xs">{{ detail.serial }}</span>
            <span class="text-muted-foreground">Subject</span>
            <span class="break-all">{{ detail.subject }}</span>
            <span class="text-muted-foreground">Status</span>
            <span>
              <StatusBadge :status="detail.revoked ? 'revoked' : 'valid'" />
            </span>
            <span class="text-muted-foreground">Not before</span>
            <span>{{ formatDateTime(detail.not_before) }}</span>
            <span class="text-muted-foreground">Not after</span>
            <span>{{ formatDateTime(detail.not_after) }}</span>
          </div>
          <div v-if="detail.dns_names?.length" class="flex flex-wrap gap-1.5">
            <span
              v-for="name in detail.dns_names"
              :key="name"
              class="rounded bg-accent px-2 py-0.5 text-xs"
            >{{ name }}</span>
          </div>
          <Separator />
          <div class="space-y-1.5">
            <Label>Certificate (PEM)</Label>
            <Textarea :model-value="detail.certificate_pem" readonly rows="5" class="font-mono text-xs" />
          </div>
        </div>
        <DialogFooter class="gap-2 sm:justify-between">
          <div class="flex gap-2">
            <Button
              v-if="detail && !detail.revoked"
              variant="destructive"
              @click="openRevoke(detail.serial)"
            >
              <Ban class="size-4" /> Revoke
            </Button>
          </div>
          <div class="flex gap-2">
            <Button
              v-if="detail && isSuperAdmin"
              variant="outline"
              @click="onDownloadBundle(detail.serial)"
            >
              <Download class="size-4" /> Bundle
            </Button>
            <Button
              v-if="detail"
              variant="secondary"
              @click="copyText(detail.certificate_pem, 'Certificate')"
            >
              <Copy class="size-4" /> Copy PEM
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Revoke dialog -->
    <Dialog v-model:open="revokeOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Revoke certificate</DialogTitle>
          <DialogDescription>
            This action is irreversible and adds the certificate to the CRL.
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-3">
          <p class="break-all rounded-md bg-accent px-3 py-2 font-mono text-xs">{{ revokeSerial }}</p>
          <div class="space-y-1.5">
            <Label>Reason</Label>
            <Select v-model="revokeReasonCode">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="reason in reasonCodes" :key="reason.value" :value="reason.value">
                  {{ reason.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" @click="revokeOpen = false">Cancel</Button>
          <Button variant="destructive" :disabled="revokeBusy" @click="submitRevoke">
            <Loader2 v-if="revokeBusy" class="size-4 animate-spin" /> Revoke
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
