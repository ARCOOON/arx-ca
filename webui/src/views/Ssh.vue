<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Spinner from '@/components/ui/Spinner.vue'
import Dialog from '@/components/ui/Dialog.vue'
import {
  fetchSshStats,
  fetchSshCertificates,
  fetchSshRoots,
  generateSshUser,
  generateSshHost,
  signSshUser,
  signSshHost,
  inspectSshCertificate,
} from '@/api/ssh'
import type { SshStatsResponse, SshCertificateListItem, SshRootsResponse, SshCertificateResponse } from '@/types/api'
import { formatDate } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const stats = ref<SshStatsResponse | null>(null)
const certs = ref<SshCertificateListItem[]>([])
const roots = ref<SshRootsResponse | null>(null)
const total = ref(0)
const page = ref(0)
const pageSize = 25

// Operation dialog
type OpMode = 'gen-user' | 'gen-host' | 'sign-user' | 'sign-host' | 'inspect'
const opOpen = ref(false)
const opMode = ref<OpMode>('gen-user')
const opLoading = ref(false)
const opResult = ref<SshCertificateResponse | null>(null)
const opInspect = ref<Record<string, unknown> | null>(null)

const pubKey = ref('')
const principals = ref('')
const ttl = ref('')
const hostname = ref('')
const inspectCert = ref('')

async function load(): Promise<void> {
  loading.value = true
  try {
    const [s, c, r] = await Promise.all([
      fetchSshStats().catch(() => null),
      fetchSshCertificates({ limit: pageSize, offset: page.value * pageSize }),
      fetchSshRoots().catch(() => null),
    ])
    stats.value = s
    certs.value = c.certificates
    total.value = c.total
    roots.value = r
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

function openOp(mode: OpMode): void {
  opMode.value = mode
  opResult.value = null
  opInspect.value = null
  pubKey.value = ''
  principals.value = ''
  ttl.value = ''
  hostname.value = ''
  inspectCert.value = ''
  opOpen.value = true
}

async function handleOp(): Promise<void> {
  opLoading.value = true
  try {
    const principalList = principals.value.split(',').map((p) => p.trim()).filter(Boolean)
    switch (opMode.value) {
      case 'gen-user':
        opResult.value = await generateSshUser({ public_key: pubKey.value, principals: principalList, ttl: ttl.value || undefined })
        break
      case 'gen-host':
        opResult.value = await generateSshHost({ public_key: pubKey.value, principals: principalList, ttl: ttl.value || undefined })
        break
      case 'sign-user':
        opResult.value = await signSshUser({ public_key: pubKey.value, principals: principalList, ttl: ttl.value || undefined })
        break
      case 'sign-host':
        opResult.value = await signSshHost({ public_key: pubKey.value, hostname: hostname.value, ttl: ttl.value || undefined })
        break
      case 'inspect':
        opInspect.value = await inspectSshCertificate({ certificate: inspectCert.value }) as unknown as Record<string, unknown>
        break
    }
    toast.success('Operation completed')
    void load()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    opLoading.value = false
  }
}

function copyToClipboard(text: string, label: string): void {
  navigator.clipboard.writeText(text).then(() => toast.success(`${label} copied`)).catch(() => {})
}

const opLabels: Record<OpMode, string> = {
  'gen-user': 'Generate User Certificate',
  'gen-host': 'Generate Host Certificate',
  'sign-user': 'Sign User Public Key',
  'sign-host': 'Sign Host Public Key',
  'inspect': 'Inspect SSH Certificate',
}
</script>

<template>
  <div class="space-y-6">
    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <template v-else>
      <!-- Stats -->
      <div v-if="stats" class="grid grid-cols-3 gap-4">
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">User Certificates</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ stats.total_user_certs }}</p>
        </Card>
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">Host Certificates</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ stats.total_host_certs }}</p>
        </Card>
        <Card class="px-5 py-5">
          <p class="text-xs text-foreground-muted mb-2">Active Now</p>
          <p class="text-2xl font-bold text-foreground tabular-nums">{{ stats.active_now }}</p>
        </Card>
      </div>

      <!-- Operations -->
      <div class="flex flex-wrap gap-2">
        <Button size="sm" @click="openOp('gen-user')">Generate User Cert</Button>
        <Button size="sm" variant="secondary" @click="openOp('gen-host')">Generate Host Cert</Button>
        <Button size="sm" variant="outline" @click="openOp('sign-user')">Sign User Key</Button>
        <Button size="sm" variant="outline" @click="openOp('sign-host')">Sign Host Key</Button>
        <Button size="sm" variant="ghost" @click="openOp('inspect')">Inspect Cert</Button>
      </div>

      <!-- Certificate list -->
      <Card class="overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h3 class="text-sm font-semibold text-foreground">SSH Certificate History</h3>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border bg-muted/50">
                <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Type</th>
                <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Principals</th>
                <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide hidden md:table-cell">Fingerprint</th>
                <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide hidden lg:table-cell">Valid After</th>
                <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Expires</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="certs.length === 0">
                <td colspan="5" class="text-center py-10 text-sm text-foreground-muted">No SSH certificates</td>
              </tr>
              <tr
                v-for="c in certs"
                :key="c.id"
                class="border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
              >
                <td class="px-4 py-2.5">
                  <Badge :variant="c.cert_type === 'user' ? 'default' : 'secondary'">{{ c.cert_type }}</Badge>
                </td>
                <td class="px-4 py-2.5 text-foreground text-xs">{{ c.principals.join(', ') }}</td>
                <td class="px-4 py-2.5 font-mono text-xs text-foreground-muted hidden md:table-cell">{{ c.fingerprint.slice(0, 24) }}…</td>
                <td class="px-4 py-2.5 text-xs text-foreground-muted hidden lg:table-cell">{{ formatDate(c.valid_after) }}</td>
                <td class="px-4 py-2.5 text-xs text-foreground-muted">{{ formatDate(c.valid_before) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center justify-between px-4 py-2.5 border-t border-border bg-muted/30 text-xs text-foreground-muted">
          <span>{{ total }} total</span>
          <div class="flex gap-1">
            <Button variant="ghost" size="sm" :disabled="page === 0" @click="() => { page--; load() }">Previous</Button>
            <Button variant="ghost" size="sm" :disabled="(page + 1) * pageSize >= total" @click="() => { page++; load() }">Next</Button>
          </div>
        </div>
      </Card>

      <!-- CA Root Keys -->
      <div v-if="roots && (roots.user_keys.length || roots.host_keys.length)" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Card class="px-5 py-4 space-y-3">
          <p class="text-sm font-semibold text-foreground">User CA Keys</p>
          <div v-for="k in roots.user_keys" :key="k.fingerprint" class="space-y-1 text-xs">
            <div class="flex items-center gap-2">
              <Badge variant="outline">{{ k.key_type }}</Badge>
              <span class="font-mono text-foreground-muted">{{ k.fingerprint.slice(0, 32) }}…</span>
            </div>
          </div>
        </Card>
        <Card class="px-5 py-4 space-y-3">
          <p class="text-sm font-semibold text-foreground">Host CA Keys</p>
          <div v-for="k in roots.host_keys" :key="k.fingerprint" class="space-y-1 text-xs">
            <div class="flex items-center gap-2">
              <Badge variant="outline">{{ k.key_type }}</Badge>
              <span class="font-mono text-foreground-muted">{{ k.fingerprint.slice(0, 32) }}…</span>
            </div>
          </div>
        </Card>
      </div>
    </template>

    <!-- Operation Dialog -->
    <Dialog
      :open="opOpen"
      :title="opLabels[opMode]"
      max-width="max-w-xl"
      @close="opOpen = false"
    >
      <div class="space-y-3">
        <template v-if="opMode !== 'inspect'">
          <div class="space-y-1.5">
            <Label>Public Key (OpenSSH format)</Label>
            <Textarea v-model="pubKey" placeholder="ssh-ed25519 AAAA..." :rows="3" />
          </div>
          <div v-if="opMode === 'sign-host' || opMode === 'gen-host'" class="space-y-1.5">
            <Label>Hostname / Principal</Label>
            <Input v-model="hostname" placeholder="server.example.com" />
          </div>
          <div v-else class="space-y-1.5">
            <Label>Principals (comma-separated)</Label>
            <Input v-model="principals" placeholder="alice, root" />
          </div>
          <div class="space-y-1.5">
            <Label>TTL (optional)</Label>
            <Input v-model="ttl" placeholder="24h" />
          </div>
        </template>

        <template v-else>
          <div class="space-y-1.5">
            <Label>SSH Certificate</Label>
            <Textarea v-model="inspectCert" placeholder="ssh-rsa-cert-v01@openssh.com AAAA..." :rows="4" />
          </div>
        </template>

        <!-- Result -->
        <div v-if="opResult" class="space-y-2 pt-2">
          <div>
            <div class="flex items-center justify-between mb-1">
              <p class="text-xs text-foreground-muted">SSH Certificate</p>
              <Button variant="ghost" size="sm" @click="copyToClipboard(opResult.certificate, 'Certificate')">Copy</Button>
            </div>
            <pre class="rounded-md bg-muted p-3 text-[10px] font-mono overflow-x-auto max-h-32">{{ opResult.certificate }}</pre>
          </div>
          <div class="grid grid-cols-2 gap-2 text-xs">
            <div><p class="text-foreground-muted">Type</p><p class="font-medium">{{ opResult.certificate_type }}</p></div>
            <div><p class="text-foreground-muted">Valid After</p><p class="font-medium">{{ formatDate(opResult.valid_after) }}</p></div>
            <div><p class="text-foreground-muted">Principals</p><p class="font-medium">{{ opResult.principals.join(', ') }}</p></div>
            <div><p class="text-foreground-muted">Expires</p><p class="font-medium">{{ formatDate(opResult.valid_before) }}</p></div>
          </div>
        </div>

        <div v-if="opInspect" class="pt-2">
          <pre class="rounded-md bg-muted p-3 text-[10px] font-mono overflow-x-auto max-h-48">{{ JSON.stringify(opInspect, null, 2) }}</pre>
        </div>
      </div>

      <template #footer>
        <Button variant="outline" @click="opOpen = false">Close</Button>
        <Button :disabled="opLoading" @click="handleOp">
          <Spinner v-if="opLoading" size="sm" />
          <span>{{ opLoading ? 'Processing…' : 'Execute' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>
