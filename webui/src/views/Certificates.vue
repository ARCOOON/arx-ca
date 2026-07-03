<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Spinner from '@/components/ui/Spinner.vue'
import Dialog from '@/components/ui/Dialog.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import Select from '@/components/ui/Select.vue'
import Switch from '@/components/ui/Switch.vue'
import {
  fetchCertificates,
  fetchCertificate,
  issueCertificate,
  generateCertificate,
  revokeCertificate,
  lintCertificate,
} from '@/api/certificates'
import { fetchTemplates } from '@/api/templates'
import type {
  CertificateSummary,
  CertificateRecordDetail,
  CertificateTemplate,
  LintCertificateResponse,
  KeyAlgorithm,
} from '@/types/api'
import { formatDate, daysUntil } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const certificates = ref<CertificateSummary[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 25
const search = ref('')

const detailOpen = ref(false)
const detailLoading = ref(false)
const selectedCert = ref<CertificateRecordDetail | null>(null)

const issueOpen = ref(false)
const issueMode = ref<'csr' | 'generate'>('generate')
const issueCsr = ref('')
const issueTtl = ref('8760h')
const issueTemplateId = ref('')
const generateCN = ref('')
const generateSans = ref('')
const generateAlgo = ref<KeyAlgorithm>('ECDSA256')
const generateServerAuth = ref(true)
const generateClientAuth = ref(false)
const issueLoading = ref(false)
const issuedCert = ref('')
const issuedKey = ref('')

const revokeOpen = ref(false)
const revokeSerial = ref('')
const revokeReason = ref('0')
const revokeLoading = ref(false)

const lintOpen = ref(false)
const lintPem = ref('')
const lintLoading = ref(false)
const lintResult = ref<LintCertificateResponse | null>(null)

const templates = ref<CertificateTemplate[]>([])

async function load(reset = false): Promise<void> {
  if (reset) page.value = 0
  loading.value = true
  try {
    const res = await fetchCertificates({ limit: pageSize, offset: page.value * pageSize })
    certificates.value = res.certificates
    total.value = res.total
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function loadTemplates(): Promise<void> {
  try {
    const res = await fetchTemplates()
    templates.value = res.templates
  } catch {
    // non-critical
  }
}

onMounted(() => {
  void load()
  void loadTemplates()
})

const filtered = computed(() => {
  if (!search.value) return certificates.value
  const q = search.value.toLowerCase()
  return certificates.value.filter(
    (c) =>
      c.serial.toLowerCase().includes(q) ||
      c.subject.toLowerCase().includes(q) ||
      c.dns_names?.some((d) => d.toLowerCase().includes(q)),
  )
})

async function openDetail(serial: string): Promise<void> {
  detailOpen.value = true
  detailLoading.value = true
  selectedCert.value = null
  try {
    selectedCert.value = await fetchCertificate(serial)
  } catch (err) {
    toast.error(extractErrorMessage(err))
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

async function handleIssue(): Promise<void> {
  issueLoading.value = true
  issuedCert.value = ''
  issuedKey.value = ''
  try {
    if (issueMode.value === 'csr') {
      const res = await issueCertificate({
        csr: issueCsr.value,
        ttl: issueTtl.value || undefined,
        template_id: issueTemplateId.value || undefined,
      })
      issuedCert.value = res.certificate_pem
    } else {
      const res = await generateCertificate({
        common_name: generateCN.value,
        sans: generateSans.value ? generateSans.value.split(',').map((s) => s.trim()) : [],
        ttl: issueTtl.value || undefined,
        key_algo: generateAlgo.value,
        is_server_auth: generateServerAuth.value,
        is_client_auth: generateClientAuth.value,
      })
      issuedCert.value = res.certificate_pem
      issuedKey.value = res.private_key_pem
    }
    toast.success('Certificate issued')
    void load()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    issueLoading.value = false
  }
}

async function handleRevoke(): Promise<void> {
  revokeLoading.value = true
  try {
    await revokeCertificate({
      serial_number: revokeSerial.value,
      reason_code: parseInt(revokeSerial.value, 10) || 0,
    })
    toast.success(`Certificate ${revokeSerial.value} revoked`)
    revokeOpen.value = false
    void load()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    revokeLoading.value = false
  }
}

async function handleLint(): Promise<void> {
  lintLoading.value = true
  lintResult.value = null
  try {
    lintResult.value = await lintCertificate({ certificate_pem: lintPem.value })
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    lintLoading.value = false
  }
}

function copyToClipboard(text: string, label: string): void {
  navigator.clipboard.writeText(text).then(() => toast.success(`${label} copied`)).catch(() => {})
}

function certExpiryVariant(notAfter: string): 'success' | 'warning' | 'destructive' {
  const days = daysUntil(notAfter)
  if (days <= 0) return 'destructive'
  if (days <= 30) return 'warning'
  return 'success'
}
</script>

<template>
  <div class="space-y-4">
    <!-- Actions bar -->
    <div class="flex flex-wrap items-center gap-2">
      <Input v-model="search" placeholder="Search serial, subject, DNS…" class="w-64" />
      <div class="flex-1" />
      <Button variant="outline" size="sm" @click="lintOpen = true">Lint PEM</Button>
      <Button variant="outline" size="sm" @click="() => { revokeSerial = ''; revokeOpen = true }">Revoke</Button>
      <Button size="sm" @click="() => { issuedCert = ''; issuedKey = ''; issueOpen = true }">Issue Certificate</Button>
    </div>

    <!-- Table -->
    <Card class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-border bg-muted/50">
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Serial</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Subject</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide hidden md:table-cell">DNS Names</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide hidden lg:table-cell">Expires</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="text-center py-12">
                <Spinner class="mx-auto" />
              </td>
            </tr>
            <tr v-else-if="filtered.length === 0">
              <td colspan="5" class="text-center py-12 text-sm text-foreground-muted">No certificates found</td>
            </tr>
            <tr
              v-for="cert in filtered"
              :key="cert.serial"
              class="border-b border-border last:border-0 hover:bg-muted/30 cursor-pointer transition-colors"
              @click="openDetail(cert.serial)"
            >
              <td class="px-4 py-2.5 font-mono text-xs text-foreground-muted">{{ cert.serial.slice(0, 16) }}…</td>
              <td class="px-4 py-2.5 text-foreground max-w-[200px] truncate">{{ cert.subject }}</td>
              <td class="px-4 py-2.5 text-foreground-muted hidden md:table-cell">
                <span v-if="cert.dns_names?.length" class="text-xs">{{ cert.dns_names.slice(0, 2).join(', ') }}</span>
                <span v-else class="text-xs">—</span>
              </td>
              <td class="px-4 py-2.5 hidden lg:table-cell">
                <Badge v-if="!cert.revoked" :variant="certExpiryVariant(cert.not_after)" class="text-[10px]">
                  {{ daysUntil(cert.not_after) }}d
                </Badge>
              </td>
              <td class="px-4 py-2.5">
                <StatusBadge :status="cert.revoked ? 'revoked' : 'active'" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="flex items-center justify-between px-4 py-2.5 border-t border-border bg-muted/30 text-xs text-foreground-muted">
        <span>{{ total }} total</span>
        <div class="flex gap-1">
          <Button variant="ghost" size="sm" :disabled="page === 0" @click="() => { page--; load() }">Previous</Button>
          <Button variant="ghost" size="sm" :disabled="(page + 1) * pageSize >= total" @click="() => { page++; load() }">Next</Button>
        </div>
      </div>
    </Card>

    <!-- Certificate Detail Dialog -->
    <Dialog :open="detailOpen" title="Certificate Detail" max-width="max-w-2xl" @close="detailOpen = false">
      <div v-if="detailLoading" class="flex justify-center py-12"><Spinner /></div>
      <div v-else-if="selectedCert" class="space-y-4">
        <div class="grid grid-cols-2 gap-3 text-xs">
          <InfoCell label="Serial" :value="selectedCert.serial" mono />
          <InfoCell label="Common Name" :value="selectedCert.common_name" />
          <InfoCell label="Not Before" :value="formatDate(selectedCert.not_before)" />
          <InfoCell label="Not After" :value="formatDate(selectedCert.not_after)" />
          <InfoCell label="Status" :value="selectedCert.revoked ? 'Revoked' : 'Active'" />
          <InfoCell v-if="selectedCert.revocation_reason" label="Revocation Reason" :value="selectedCert.revocation_reason" />
        </div>
        <div v-if="selectedCert.dns_names?.length" class="text-xs">
          <p class="text-foreground-muted mb-1">DNS Names</p>
          <div class="flex flex-wrap gap-1">
            <Badge v-for="d in selectedCert.dns_names" :key="d" variant="outline">{{ d }}</Badge>
          </div>
        </div>
        <div>
          <div class="flex items-center justify-between mb-1">
            <p class="text-xs text-foreground-muted">Certificate PEM</p>
            <Button variant="ghost" size="sm" @click="copyToClipboard(selectedCert.certificate_pem, 'Certificate PEM')">Copy</Button>
          </div>
          <pre class="rounded-md bg-muted p-3 text-[10px] font-mono text-foreground-muted overflow-x-auto max-h-40">{{ selectedCert.certificate_pem }}</pre>
        </div>
      </div>
    </Dialog>

    <!-- Issue Certificate Dialog -->
    <Dialog :open="issueOpen" title="Issue Certificate" max-width="max-w-xl" @close="issueOpen = false">
      <div class="space-y-4">
        <!-- Mode tabs -->
        <div class="flex rounded-lg border border-border bg-muted p-0.5 gap-0.5">
          <button
            v-for="m in [{ v: 'generate', l: 'Generate Key' }, { v: 'csr', l: 'Sign CSR' }]"
            :key="m.v"
            type="button"
            :class="[
              'flex-1 rounded-md py-1.5 text-xs font-medium transition-all',
              issueMode === m.v ? 'bg-card text-foreground shadow-sm' : 'text-foreground-muted hover:text-foreground',
            ]"
            @click="issueMode = m.v as 'csr' | 'generate'"
          >
            {{ m.l }}
          </button>
        </div>

        <div v-if="issueMode === 'generate'" class="space-y-3">
          <div class="space-y-1.5">
            <Label>Common Name</Label>
            <Input v-model="generateCN" placeholder="service.example.com" />
          </div>
          <div class="space-y-1.5">
            <Label>SANs (comma-separated)</Label>
            <Input v-model="generateSans" placeholder="www.example.com, 10.0.0.1" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label>Key Algorithm</Label>
              <Select v-model="generateAlgo">
                <option value="ECDSA256">ECDSA P-256</option>
                <option value="RSA2048">RSA 2048</option>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label>TTL</Label>
              <Input v-model="issueTtl" placeholder="8760h" />
            </div>
          </div>
          <div class="flex items-center gap-6 text-sm">
            <label class="flex items-center gap-2 cursor-pointer">
              <Switch v-model="generateServerAuth" />
              <span class="text-foreground">Server Auth</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <Switch v-model="generateClientAuth" />
              <span class="text-foreground">Client Auth</span>
            </label>
          </div>
        </div>

        <div v-else class="space-y-3">
          <div class="space-y-1.5">
            <Label>PEM-encoded CSR</Label>
            <Textarea v-model="issueCsr" placeholder="-----BEGIN CERTIFICATE REQUEST-----" :rows="6" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label>TTL</Label>
              <Input v-model="issueTtl" placeholder="8760h" />
            </div>
            <div v-if="templates.length" class="space-y-1.5">
              <Label>Template</Label>
              <Select v-model="issueTemplateId">
                <option value="">None</option>
                <option v-for="t in templates" :key="t.id" :value="t.id">{{ t.name }}</option>
              </Select>
            </div>
          </div>
        </div>

        <!-- Results -->
        <div v-if="issuedCert" class="space-y-2">
          <div>
            <div class="flex items-center justify-between mb-1">
              <p class="text-xs text-foreground-muted">Certificate PEM</p>
              <Button variant="ghost" size="sm" @click="copyToClipboard(issuedCert, 'Certificate')">Copy</Button>
            </div>
            <pre class="rounded-md bg-muted p-3 text-[10px] font-mono overflow-x-auto max-h-32">{{ issuedCert }}</pre>
          </div>
          <div v-if="issuedKey">
            <div class="flex items-center justify-between mb-1">
              <p class="text-xs text-foreground-muted">Private Key PEM</p>
              <Button variant="ghost" size="sm" @click="copyToClipboard(issuedKey, 'Private key')">Copy</Button>
            </div>
            <pre class="rounded-md bg-muted p-3 text-[10px] font-mono overflow-x-auto max-h-32">{{ issuedKey }}</pre>
          </div>
        </div>
      </div>

      <template #footer>
        <Button variant="outline" @click="issueOpen = false">Cancel</Button>
        <Button :disabled="issueLoading" @click="handleIssue">
          <Spinner v-if="issueLoading" size="sm" />
          <span>{{ issueLoading ? 'Issuing…' : 'Issue' }}</span>
        </Button>
      </template>
    </Dialog>

    <!-- Revoke Dialog -->
    <Dialog :open="revokeOpen" title="Revoke Certificate" @close="revokeOpen = false">
      <div class="space-y-3">
        <div class="space-y-1.5">
          <Label>Serial Number</Label>
          <Input v-model="revokeSerial" placeholder="hex serial number" />
        </div>
        <div class="space-y-1.5">
          <Label>Reason Code</Label>
          <Select v-model="revokeReason">
            <option value="0">0 — Unspecified</option>
            <option value="1">1 — Key Compromise</option>
            <option value="2">2 — CA Compromise</option>
            <option value="3">3 — Affiliation Changed</option>
            <option value="4">4 — Superseded</option>
            <option value="5">5 — Cessation of Operation</option>
          </Select>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="revokeOpen = false">Cancel</Button>
        <Button variant="destructive" :disabled="revokeLoading || !revokeSerial" @click="handleRevoke">
          <Spinner v-if="revokeLoading" size="sm" />
          <span>{{ revokeLoading ? 'Revoking…' : 'Revoke' }}</span>
        </Button>
      </template>
    </Dialog>

    <!-- Lint Dialog -->
    <Dialog :open="lintOpen" title="Lint Certificate" max-width="max-w-xl" @close="lintOpen = false">
      <div class="space-y-3">
        <div class="space-y-1.5">
          <Label>PEM Certificate</Label>
          <Textarea v-model="lintPem" placeholder="-----BEGIN CERTIFICATE-----" :rows="6" />
        </div>
        <div v-if="lintResult">
          <div class="flex gap-3 mb-2 text-xs">
            <span class="text-destructive">{{ lintResult.summary.errors }} errors</span>
            <span class="text-warning-foreground">{{ lintResult.summary.warnings }} warnings</span>
            <span class="text-foreground-muted">{{ lintResult.summary.notices }} notices</span>
          </div>
          <div class="space-y-1 max-h-48 overflow-y-auto">
            <div
              v-for="(f, i) in lintResult.findings"
              :key="i"
              class="flex gap-2 text-xs rounded-md px-2 py-1.5 bg-muted"
            >
              <Badge
                :variant="f.severity === 'error' ? 'destructive' : f.severity === 'warn' ? 'warning' : 'secondary'"
                class="shrink-0"
              >
                {{ f.severity }}
              </Badge>
              <span class="text-foreground-muted">{{ f.lint }}</span>
              <span v-if="f.message" class="text-foreground">{{ f.message }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="lintOpen = false">Close</Button>
        <Button :disabled="lintLoading || !lintPem" @click="handleLint">
          <Spinner v-if="lintLoading" size="sm" />
          <span>{{ lintLoading ? 'Linting…' : 'Lint' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts">
const InfoCell = {
  props: { label: String, value: String, mono: Boolean },
  template: `
    <div>
      <p class="text-foreground-muted mb-0.5">{{ label }}</p>
      <p :class="['text-foreground break-all', mono ? 'font-mono text-[10px]' : 'font-medium']">{{ value ?? '—' }}</p>
    </div>
  `,
}
export default { components: { InfoCell } }
</script>
