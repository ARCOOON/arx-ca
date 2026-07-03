<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import {
  fetchSshRoots,
  fetchSshStats,
  generateSshHost,
  generateSshUser,
  inspectSshCertificate,
  listSshCertificates,
} from '@/api/ssh'
import type {
  SshCertificateInspection,
  SshCertificateListItem,
  SshCertificateResponse,
  SshRootKey,
  SshStatsResponse,
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { downloadTextFile } from '@/utils/download'
import { extractApiError } from '@/utils/errors'
import { formatDateTime } from '@/utils/format'

const PAGE_SIZE = 50

const certificates = ref<SshCertificateListItem[]>([])
const total = ref(0)
const offset = ref(0)
const tableLoading = ref(true)
const tableError = ref('')

const stats = ref<SshStatsResponse | null>(null)
const statsLoading = ref(true)

const userRoots = ref<SshRootKey[]>([])
const hostRoots = ref<SshRootKey[]>([])

const userModalOpen = ref(false)
const userPublicKey = ref('')
const userPrincipals = ref('')
const userTtl = ref('4h')
const userLoading = ref(false)
const userError = ref('')
const userResult = ref<SshCertificateResponse | null>(null)

const hostModalOpen = ref(false)
const hostPublicKey = ref('')
const hostPrincipals = ref('')
const hostTtl = ref('8760h')
const hostLoading = ref(false)
const hostError = ref('')
const hostResult = ref<SshCertificateResponse | null>(null)

const inspectModalOpen = ref(false)
const inspectCertificate = ref('')
const inspectLoading = ref(false)
const inspectError = ref('')
const inspectResult = ref<SshCertificateInspection | null>(null)

function parsePrincipals(raw: string): string[] {
  return raw.split(',').map((v) => v.trim()).filter((v) => v.length > 0)
}

function truncateFingerprint(value: string): string {
  if (!value || value.length <= 20) return value || '—'
  return `${value.slice(0, 10)}…${value.slice(-10)}`
}

function isActive(row: SshCertificateListItem): boolean {
  const now = Date.now()
  const validAfter = Date.parse(row.valid_after)
  const validBefore = Date.parse(row.valid_before)
  return !Number.isNaN(validAfter) && !Number.isNaN(validBefore) && validAfter <= now && validBefore >= now
}

async function loadStats(): Promise<void> {
  statsLoading.value = true
  try {
    stats.value = await fetchSshStats()
  } catch {
    stats.value = null
  } finally {
    statsLoading.value = false
  }
}

async function loadCertificates(): Promise<void> {
  tableLoading.value = true
  tableError.value = ''
  try {
    const response = await listSshCertificates(PAGE_SIZE, offset.value)
    certificates.value = response.certificates
    total.value = response.total
  } catch (error) {
    tableError.value = extractApiError(error, 'Failed to load SSH certificates')
    certificates.value = []
    total.value = 0
  } finally {
    tableLoading.value = false
  }
}

async function loadRoots(): Promise<void> {
  try {
    const roots = await fetchSshRoots()
    userRoots.value = roots.user_keys
    hostRoots.value = roots.host_keys
  } catch {
    userRoots.value = []
    hostRoots.value = []
  }
}

onMounted(() => {
  void loadStats()
  void loadCertificates()
  void loadRoots()
})

watch(offset, () => void loadCertificates())

async function submitUserCert(): Promise<void> {
  userLoading.value = true
  userError.value = ''
  userResult.value = null
  try {
    userResult.value = await generateSshUser({
      public_key: userPublicKey.value.trim(),
      principals: parsePrincipals(userPrincipals.value),
      ttl: userTtl.value.trim() || undefined,
    })
    void loadCertificates()
    void loadStats()
  } catch (error) {
    userError.value = extractApiError(error, 'Failed to generate SSH user certificate')
  } finally {
    userLoading.value = false
  }
}

async function submitHostCert(): Promise<void> {
  hostLoading.value = true
  hostError.value = ''
  hostResult.value = null
  try {
    hostResult.value = await generateSshHost({
      public_key: hostPublicKey.value.trim(),
      principals: parsePrincipals(hostPrincipals.value),
      ttl: hostTtl.value.trim() || undefined,
    })
    void loadCertificates()
    void loadStats()
  } catch (error) {
    hostError.value = extractApiError(error, 'Failed to generate SSH host certificate')
  } finally {
    hostLoading.value = false
  }
}

async function submitInspect(): Promise<void> {
  inspectLoading.value = true
  inspectError.value = ''
  inspectResult.value = null
  try {
    inspectResult.value = await inspectSshCertificate({ certificate: inspectCertificate.value.trim() })
  } catch (error) {
    inspectError.value = extractApiError(error, 'Failed to inspect SSH certificate')
  } finally {
    inspectLoading.value = false
  }
}

function downloadCert(result: SshCertificateResponse, filename: string): void {
  downloadTextFile(filename, result.certificate)
}
</script>

<template>
  <div class="space-y-4">
    <section class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">User certs</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ stats?.total_user_certs ?? '—' }}</p>
        </CardContent>
      </Card>
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Host certs</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ stats?.total_host_certs ?? '—' }}</p>
        </CardContent>
      </Card>
      <Card class="rounded-lg border border-border shadow-none">
        <CardHeader class="pb-2">
          <CardTitle class="text-xs font-normal uppercase tracking-wide text-muted-foreground">Active now</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-lg font-semibold">{{ stats?.active_now ?? '—' }}</p>
        </CardContent>
      </Card>
    </section>

    <div class="flex flex-wrap gap-2">
      <Button class="rounded-lg" @click="userModalOpen = true">Generate user cert</Button>
      <Button variant="secondary" class="rounded-lg" @click="hostModalOpen = true">Generate host cert</Button>
      <Button variant="outline" class="rounded-lg" @click="inspectModalOpen = true">Inspect certificate</Button>
    </div>

    <Card class="rounded-lg border border-border shadow-none">
      <CardHeader>
        <CardTitle class="text-sm">SSH CA roots</CardTitle>
      </CardHeader>
      <CardContent class="grid grid-cols-1 gap-4 md:grid-cols-2 text-sm">
        <div>
          <p class="mb-2 font-medium">User keys</p>
          <ul class="space-y-1 text-muted-foreground">
            <li v-for="(key, index) in userRoots" :key="index">{{ key.key_type }} · {{ truncateFingerprint(key.fingerprint) }}</li>
            <li v-if="userRoots.length === 0">No user CA keys</li>
          </ul>
        </div>
        <div>
          <p class="mb-2 font-medium">Host keys</p>
          <ul class="space-y-1 text-muted-foreground">
            <li v-for="(key, index) in hostRoots" :key="index">{{ key.key_type }} · {{ truncateFingerprint(key.fingerprint) }}</li>
            <li v-if="hostRoots.length === 0">No host CA keys</li>
          </ul>
        </div>
      </CardContent>
    </Card>

    <Card class="rounded-lg border border-border shadow-none">
      <CardHeader class="flex-row items-center justify-between">
        <CardTitle class="text-sm">Certificate inventory</CardTitle>
        <span class="text-xs text-muted-foreground">{{ total }} total</span>
      </CardHeader>
      <CardContent>
        <Alert v-if="tableError" variant="destructive" class="mb-3 rounded-lg">
          <AlertDescription>{{ tableError }}</AlertDescription>
        </Alert>

        <div class="overflow-x-auto rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Type</TableHead>
                <TableHead>Principals</TableHead>
                <TableHead>Fingerprint</TableHead>
                <TableHead>Valid from</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="tableLoading">
                <TableCell colspan="6" class="text-center text-muted-foreground">Loading…</TableCell>
              </TableRow>
              <TableRow v-else-if="certificates.length === 0">
                <TableCell colspan="6" class="text-center text-muted-foreground">No SSH certificates found.</TableCell>
              </TableRow>
              <TableRow v-for="row in certificates" :key="row.id">
                <TableCell>
                  <Badge variant="outline" class="rounded-md capitalize">{{ row.cert_type }}</Badge>
                </TableCell>
                <TableCell>{{ row.principals.join(', ') || '—' }}</TableCell>
                <TableCell class="font-mono text-xs">{{ truncateFingerprint(row.fingerprint) }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(row.valid_after) }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(row.valid_before) }}</TableCell>
                <TableCell>
                  <Badge :variant="isActive(row) ? 'default' : 'secondary'" class="rounded-md">
                    {{ isActive(row) ? 'Active' : 'Inactive' }}
                  </Badge>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>

        <div v-if="total > PAGE_SIZE" class="mt-3 flex justify-end gap-2">
          <Button variant="outline" size="sm" class="rounded-lg" :disabled="offset === 0" @click="offset -= PAGE_SIZE">Previous</Button>
          <Button variant="outline" size="sm" class="rounded-lg" :disabled="offset + PAGE_SIZE >= total" @click="offset += PAGE_SIZE">Next</Button>
        </div>
      </CardContent>
    </Card>

    <!-- User cert dialog -->
    <Dialog v-model:open="userModalOpen">
      <DialogContent class="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Generate SSH user certificate</DialogTitle>
        </DialogHeader>
        <div class="space-y-3">
          <div class="space-y-2">
            <Label>Public key</Label>
            <Textarea v-model="userPublicKey" rows="3" class="rounded-lg font-mono text-xs" />
          </div>
          <div class="space-y-2">
            <Label>Principals (comma-separated)</Label>
            <Input v-model="userPrincipals" class="rounded-lg" />
          </div>
          <div class="space-y-2">
            <Label>TTL</Label>
            <Input v-model="userTtl" class="rounded-lg" />
          </div>
          <Alert v-if="userError" variant="destructive" class="rounded-lg">
            <AlertDescription>{{ userError }}</AlertDescription>
          </Alert>
          <div v-if="userResult" class="rounded-lg border border-border bg-muted/30 p-3 text-xs">
            <p class="font-medium">Certificate issued (serial {{ userResult.serial }})</p>
            <Button variant="link" class="h-auto p-0" @click="downloadCert(userResult, 'ssh-user-cert.pub')">Download</Button>
          </div>
        </div>
        <DialogFooter>
          <Button class="rounded-lg" :disabled="userLoading" @click="submitUserCert">
            {{ userLoading ? 'Generating…' : 'Generate' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Host cert dialog -->
    <Dialog v-model:open="hostModalOpen">
      <DialogContent class="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Generate SSH host certificate</DialogTitle>
        </DialogHeader>
        <div class="space-y-3">
          <div class="space-y-2">
            <Label>Public key</Label>
            <Textarea v-model="hostPublicKey" rows="3" class="rounded-lg font-mono text-xs" />
          </div>
          <div class="space-y-2">
            <Label>Principals (comma-separated)</Label>
            <Input v-model="hostPrincipals" class="rounded-lg" />
          </div>
          <div class="space-y-2">
            <Label>TTL</Label>
            <Input v-model="hostTtl" class="rounded-lg" />
          </div>
          <Alert v-if="hostError" variant="destructive" class="rounded-lg">
            <AlertDescription>{{ hostError }}</AlertDescription>
          </Alert>
          <div v-if="hostResult" class="rounded-lg border border-border bg-muted/30 p-3 text-xs">
            <p class="font-medium">Certificate issued (serial {{ hostResult.serial }})</p>
            <Button variant="link" class="h-auto p-0" @click="downloadCert(hostResult, 'ssh-host-cert.pub')">Download</Button>
          </div>
        </div>
        <DialogFooter>
          <Button class="rounded-lg" :disabled="hostLoading" @click="submitHostCert">
            {{ hostLoading ? 'Generating…' : 'Generate' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Inspect dialog -->
    <Dialog v-model:open="inspectModalOpen">
      <DialogContent class="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Inspect SSH certificate</DialogTitle>
        </DialogHeader>
        <div class="space-y-3">
          <Textarea v-model="inspectCertificate" rows="5" class="rounded-lg font-mono text-xs" placeholder="Paste certificate…" />
          <Alert v-if="inspectError" variant="destructive" class="rounded-lg">
            <AlertDescription>{{ inspectError }}</AlertDescription>
          </Alert>
          <div v-if="inspectResult" class="space-y-1 rounded-lg border border-border bg-muted/30 p-3 text-sm">
            <p><span class="text-muted-foreground">Type:</span> {{ inspectResult.certificate_type }}</p>
            <p><span class="text-muted-foreground">Key ID:</span> {{ inspectResult.key_id }}</p>
            <p><span class="text-muted-foreground">Principals:</span> {{ inspectResult.principals.join(', ') }}</p>
            <p><span class="text-muted-foreground">Valid:</span> {{ formatDateTime(inspectResult.valid_after) }} – {{ formatDateTime(inspectResult.valid_before) }}</p>
          </div>
        </div>
        <DialogFooter>
          <Button class="rounded-lg" :disabled="inspectLoading" @click="submitInspect">
            {{ inspectLoading ? 'Inspecting…' : 'Inspect' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
