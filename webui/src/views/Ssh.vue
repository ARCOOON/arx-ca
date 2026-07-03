<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  TerminalSquare,
  UserRound,
  ServerCog,
  Plus,
  ScanSearch,
  Copy,
  Download,
  Loader2,
  RefreshCw,
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import StatCard from '@/components/StatCard.vue'
import {
  fetchSshStats,
  listSshCertificates,
  fetchSshRoots,
  generateSshUserCertificate,
  generateSshHostCertificate,
  inspectSshCertificate,
} from '@/api/ssh'
import type {
  SshCertificateInspection,
  SshCertificateListItem,
  SshCertificateResponse,
  SshRootsResponse,
  SshStatsResponse,
} from '@/types/api'
import { formatDateTime } from '@/lib/format'
import { copyToClipboard, downloadBlob } from '@/lib/download'
import { extractApiError } from '@/lib/errors'

const stats = ref<SshStatsResponse | null>(null)
const certificates = ref<SshCertificateListItem[]>([])
const roots = ref<SshRootsResponse | null>(null)
const isLoading = ref(false)

async function loadAll(): Promise<void> {
  isLoading.value = true
  const [statsResult, listResult, rootsResult] = await Promise.allSettled([
    fetchSshStats(),
    listSshCertificates({ limit: 50 }),
    fetchSshRoots(),
  ])
  if (statsResult.status === 'fulfilled') stats.value = statsResult.value
  if (listResult.status === 'fulfilled') certificates.value = listResult.value.certificates
  if (rootsResult.status === 'fulfilled') roots.value = rootsResult.value
  isLoading.value = false
}

/* ----------------------------- Generate --------------------------------- */

const generateOpen = ref(false)
const generateTab = ref('user')
const generateBusy = ref(false)
const form = ref({ public_key: '', principals: '', ttl: '' })
const generated = ref<SshCertificateResponse | null>(null)

function openGenerate(): void {
  form.value = { public_key: '', principals: '', ttl: '' }
  generated.value = null
  generateTab.value = 'user'
  generateOpen.value = true
}

function principalsList(): string[] {
  return form.value.principals
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean)
}

async function submitGenerate(): Promise<void> {
  if (!form.value.public_key.trim()) {
    toast.error('An SSH public key is required.')
    return
  }
  const principals = principalsList()
  if (principals.length === 0) {
    toast.error('At least one principal is required.')
    return
  }

  generateBusy.value = true
  try {
    const payload = {
      public_key: form.value.public_key.trim(),
      principals,
      ttl: form.value.ttl.trim() || undefined,
    }
    generated.value =
      generateTab.value === 'host'
        ? await generateSshHostCertificate(payload)
        : await generateSshUserCertificate(payload)
    toast.success('SSH certificate generated')
    void loadAll()
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to generate certificate'))
  } finally {
    generateBusy.value = false
  }
}

/* ------------------------------ Inspect --------------------------------- */

const inspectOpen = ref(false)
const inspectBusy = ref(false)
const inspectInput = ref('')
const inspectResult = ref<SshCertificateInspection | null>(null)

function openInspect(): void {
  inspectInput.value = ''
  inspectResult.value = null
  inspectOpen.value = true
}

async function submitInspect(): Promise<void> {
  if (!inspectInput.value.trim()) {
    toast.error('Paste an SSH certificate to inspect.')
    return
  }
  inspectBusy.value = true
  try {
    inspectResult.value = await inspectSshCertificate({ certificate: inspectInput.value.trim() })
  } catch (error) {
    toast.error(extractApiError(error, 'Failed to inspect certificate'))
  } finally {
    inspectBusy.value = false
  }
}

async function copyText(value: string, label: string): Promise<void> {
  if (await copyToClipboard(value)) {
    toast.success(`${label} copied`)
  } else {
    toast.error('Clipboard unavailable')
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">SSH CA</h1>
        <p class="text-sm text-muted-foreground">Issue and inspect SSH user and host certificates.</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="openInspect">
          <ScanSearch class="size-4" /> Inspect
        </Button>
        <Button @click="openGenerate">
          <Plus class="size-4" /> Generate
        </Button>
      </div>
    </div>

    <div class="grid gap-4 sm:grid-cols-3">
      <StatCard label="User certs" :value="stats?.total_user_certs ?? 0" :icon="UserRound" />
      <StatCard label="Host certs" :value="stats?.total_host_certs ?? 0" :icon="ServerCog" />
      <StatCard label="Active now" :value="stats?.active_now ?? 0" :icon="TerminalSquare" />
    </div>

    <Card>
      <CardHeader class="flex-row items-center justify-between space-y-0">
        <div>
          <CardTitle class="text-base">Issued certificates</CardTitle>
          <CardDescription>Recently issued SSH certificates.</CardDescription>
        </div>
        <Button variant="ghost" size="icon" title="Refresh" @click="loadAll">
          <RefreshCw class="size-4" :class="{ 'animate-spin': isLoading }" />
        </Button>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Serial</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Principals</TableHead>
              <TableHead>Valid until</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="isLoading">
              <TableCell colspan="4" class="py-10 text-center text-muted-foreground">
                <Loader2 class="mx-auto size-5 animate-spin" />
              </TableCell>
            </TableRow>
            <TableRow v-else-if="certificates.length === 0">
              <TableCell colspan="4" class="py-10 text-center text-muted-foreground">
                No SSH certificates issued yet.
              </TableCell>
            </TableRow>
            <TableRow v-for="cert in certificates" :key="cert.id">
              <TableCell class="max-w-[140px] truncate font-mono text-xs">{{ cert.serial }}</TableCell>
              <TableCell class="capitalize">{{ cert.cert_type }}</TableCell>
              <TableCell class="max-w-[220px] truncate">{{ cert.principals.join(', ') }}</TableCell>
              <TableCell>{{ formatDateTime(cert.valid_before) }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle class="text-base">CA public keys</CardTitle>
        <CardDescription>Add these to <code class="text-xs">known_hosts</code> / <code class="text-xs">TrustedUserCAKeys</code>.</CardDescription>
      </CardHeader>
      <CardContent class="grid gap-4 md:grid-cols-2">
        <div>
          <p class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">User keys</p>
          <p v-if="!roots?.user_keys.length" class="text-sm text-muted-foreground">None.</p>
          <div
            v-for="key in roots?.user_keys ?? []"
            :key="key.fingerprint"
            class="mb-2 flex items-start gap-2 rounded-md border border-border bg-accent/30 p-2"
          >
            <span class="min-w-0 flex-1 break-all font-mono text-xs">{{ key.public_key }}</span>
            <Button variant="ghost" size="icon-sm" @click="copyText(key.public_key, 'Key')">
              <Copy class="size-3.5" />
            </Button>
          </div>
        </div>
        <div>
          <p class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Host keys</p>
          <p v-if="!roots?.host_keys.length" class="text-sm text-muted-foreground">None.</p>
          <div
            v-for="key in roots?.host_keys ?? []"
            :key="key.fingerprint"
            class="mb-2 flex items-start gap-2 rounded-md border border-border bg-accent/30 p-2"
          >
            <span class="min-w-0 flex-1 break-all font-mono text-xs">{{ key.public_key }}</span>
            <Button variant="ghost" size="icon-sm" @click="copyText(key.public_key, 'Key')">
              <Copy class="size-3.5" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Generate dialog -->
    <Dialog v-model:open="generateOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Generate SSH certificate</DialogTitle>
          <DialogDescription>Sign a public key for a user or host.</DialogDescription>
        </DialogHeader>

        <div v-if="generated" class="space-y-3">
          <div class="rounded-md border border-success/30 bg-success/10 px-3 py-2 text-sm">
            {{ generated.certificate_type }} certificate issued for
            {{ generated.principals.join(', ') }}.
          </div>
          <div class="space-y-1.5">
            <Label>Certificate</Label>
            <Textarea :model-value="generated.certificate" readonly rows="5" class="font-mono text-xs" />
            <div class="flex gap-2">
              <Button variant="secondary" size="sm" @click="copyText(generated.certificate, 'Certificate')">
                <Copy class="size-4" /> Copy
              </Button>
              <Button
                variant="secondary"
                size="sm"
                @click="downloadBlob(generated.certificate, `${generated.key_id || 'ssh'}-cert.pub`, 'text/plain')"
              >
                <Download class="size-4" /> Download
              </Button>
            </div>
          </div>
        </div>

        <Tabs v-else v-model="generateTab">
          <TabsList class="w-full">
            <TabsTrigger value="user" class="flex-1">User</TabsTrigger>
            <TabsTrigger value="host" class="flex-1">Host</TabsTrigger>
          </TabsList>
          <TabsContent :value="generateTab" class="space-y-3">
            <div class="space-y-1.5">
              <Label for="pub">SSH public key</Label>
              <Textarea id="pub" v-model="form.public_key" rows="4" placeholder="ssh-ed25519 AAAA..." class="font-mono text-xs" />
            </div>
            <div class="space-y-1.5">
              <Label for="principals">Principals</Label>
              <Input id="principals" v-model="form.principals" :placeholder="generateTab === 'host' ? 'host.example.com' : 'alice, admins'" />
            </div>
            <div class="space-y-1.5">
              <Label for="ssh-ttl">TTL (optional)</Label>
              <Input id="ssh-ttl" v-model="form.ttl" placeholder="e.g. 12h" />
            </div>
            <Button class="w-full" :disabled="generateBusy" @click="submitGenerate">
              <Loader2 v-if="generateBusy" class="size-4 animate-spin" />
              Generate {{ generateTab }} certificate
            </Button>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>

    <!-- Inspect dialog -->
    <Dialog v-model:open="inspectOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Inspect SSH certificate</DialogTitle>
          <DialogDescription>Decode an SSH certificate to view its claims.</DialogDescription>
        </DialogHeader>
        <div class="space-y-3">
          <Textarea v-model="inspectInput" rows="4" placeholder="ssh-ed25519-cert-v01@openssh.com AAAA..." class="font-mono text-xs" />
          <Button class="w-full" :disabled="inspectBusy" @click="submitInspect">
            <Loader2 v-if="inspectBusy" class="size-4 animate-spin" /> Inspect
          </Button>
          <div v-if="inspectResult" class="grid grid-cols-[120px_1fr] gap-y-2 rounded-md border border-border bg-accent/30 p-3 text-sm">
            <span class="text-muted-foreground">Type</span>
            <span class="capitalize">{{ inspectResult.certificate_type }}</span>
            <span class="text-muted-foreground">Key ID</span>
            <span class="break-all">{{ inspectResult.key_id }}</span>
            <span class="text-muted-foreground">Principals</span>
            <span>{{ inspectResult.principals.join(', ') || '—' }}</span>
            <span class="text-muted-foreground">Valid after</span>
            <span>{{ formatDateTime(inspectResult.valid_after) }}</span>
            <span class="text-muted-foreground">Valid before</span>
            <span>{{ formatDateTime(inspectResult.valid_before) }}</span>
            <span class="text-muted-foreground">Key type</span>
            <span>{{ inspectResult.public_key_type }}</span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
