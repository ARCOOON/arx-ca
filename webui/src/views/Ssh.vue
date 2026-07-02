<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import {
  fetchSshRoots,
  fetchSshStats,
  generateSshHost,
  generateSshUser,
  inspectSshCertificate,
  listSshCertificates,
} from '../api/ssh'
import type {
  SshCertificateInspection,
  SshCertificateListItem,
  SshCertificateResponse,
  SshRootKey,
  SshStatsResponse,
} from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import Modal from '../components/ui/Modal.vue'
import Pagination from '../components/Pagination.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { usePreferences } from '../composables/usePreferences'
import { downloadTextFile } from '../utils/download'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'

const PAGE_SIZE = 50

const { showApiHints } = usePreferences()

const certificates = ref<SshCertificateListItem[]>([])
const total = ref(0)
const offset = ref(0)
const tableLoading = ref(true)
const tableError = ref('')

const stats = ref<SshStatsResponse | null>(null)
const statsLoading = ref(true)
const statsError = ref('')

const userRoots = ref<SshRootKey[]>([])
const hostRoots = ref<SshRootKey[]>([])
const rootsLoading = ref(true)
const rootsError = ref('')

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

const tableColumns = [
  { key: 'cert_type', label: 'Type' },
  { key: 'principals', label: 'Principals' },
  { key: 'fingerprint', label: 'Fingerprint', cellClass: 'font-mono text-[11px]' },
  { key: 'valid_after', label: 'Valid From' },
  { key: 'valid_before', label: 'Expires At' },
]

function parsePrincipals(raw: string): string[] {
  return raw
    .split(',')
    .map((value) => value.trim())
    .filter((value) => value.length > 0)
}

function truncateFingerprint(value: string): string {
  if (!value) {
    return '—'
  }
  if (value.length <= 20) {
    return value
  }
  return `${value.slice(0, 10)}…${value.slice(-10)}`
}

function certTypeLabel(certType: string): string {
  return certType === 'host' ? 'Host' : 'User'
}

function certTypeTone(certType: string): 'valid' | 'neutral' {
  return certType === 'host' ? 'neutral' : 'valid'
}

function isActive(row: SshCertificateListItem): boolean {
  const now = Date.now()
  const validAfter = Date.parse(row.valid_after)
  const validBefore = Date.parse(row.valid_before)
  return !Number.isNaN(validAfter) && !Number.isNaN(validBefore) && validAfter <= now && validBefore >= now
}

async function loadStats(): Promise<void> {
  statsLoading.value = true
  statsError.value = ''

  try {
    stats.value = await fetchSshStats()
  } catch (error) {
    statsError.value = extractApiError(error, 'Failed to load SSH statistics')
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
  rootsLoading.value = true
  rootsError.value = ''

  try {
    const roots = await fetchSshRoots()
    userRoots.value = roots.user_keys
    hostRoots.value = roots.host_keys
  } catch (error) {
    rootsError.value = extractApiError(error, 'Failed to load SSH CA roots')
  } finally {
    rootsLoading.value = false
  }
}

onMounted(() => {
  void loadStats()
  void loadCertificates()
  void loadRoots()
})

watch(offset, () => {
  void loadCertificates()
})

function openUserModal(): void {
  userError.value = ''
  userResult.value = null
  userModalOpen.value = true
}

function closeUserModal(): void {
  if (userLoading.value) {
    return
  }
  userModalOpen.value = false
}

function openHostModal(): void {
  hostError.value = ''
  hostResult.value = null
  hostModalOpen.value = true
}

function closeHostModal(): void {
  if (hostLoading.value) {
    return
  }
  hostModalOpen.value = false
}

function openInspectModal(): void {
  inspectError.value = ''
  inspectResult.value = null
  inspectModalOpen.value = true
}

function closeInspectModal(): void {
  if (inspectLoading.value) {
    return
  }
  inspectModalOpen.value = false
}

async function submitUserCertificate(): Promise<void> {
  userError.value = ''
  userResult.value = null

  const publicKey = userPublicKey.value.trim()
  const principals = parsePrincipals(userPrincipals.value)

  if (!publicKey) {
    userError.value = 'Public key is required.'
    return
  }
  if (principals.length === 0) {
    userError.value = 'At least one principal is required.'
    return
  }

  userLoading.value = true

  try {
    userResult.value = await generateSshUser({
      public_key: publicKey,
      principals,
      ttl: userTtl.value.trim() || undefined,
    })
    await Promise.all([loadCertificates(), loadStats()])
  } catch (error) {
    userError.value = extractApiError(error, 'Failed to generate SSH user certificate')
  } finally {
    userLoading.value = false
  }
}

async function submitHostCertificate(): Promise<void> {
  hostError.value = ''
  hostResult.value = null

  const publicKey = hostPublicKey.value.trim()
  const principals = parsePrincipals(hostPrincipals.value)

  if (!publicKey) {
    hostError.value = 'Public key is required.'
    return
  }
  if (principals.length === 0) {
    hostError.value = 'At least one principal is required.'
    return
  }

  hostLoading.value = true

  try {
    hostResult.value = await generateSshHost({
      public_key: publicKey,
      principals,
      ttl: hostTtl.value.trim() || undefined,
    })
    await Promise.all([loadCertificates(), loadStats()])
  } catch (error) {
    hostError.value = extractApiError(error, 'Failed to generate SSH host certificate')
  } finally {
    hostLoading.value = false
  }
}

async function submitInspect(): Promise<void> {
  inspectError.value = ''
  inspectResult.value = null

  const certificate = inspectCertificate.value.trim()
  if (!certificate) {
    inspectError.value = 'Certificate is required.'
    return
  }

  inspectLoading.value = true

  try {
    inspectResult.value = await inspectSshCertificate({ certificate })
  } catch (error) {
    inspectError.value = extractApiError(error, 'Failed to inspect SSH certificate')
  } finally {
    inspectLoading.value = false
  }
}

function downloadCertificate(result: SshCertificateResponse, filename: string): void {
  downloadTextFile(filename, `${result.certificate.trim()}\n`, 'text/plain')
}

function downloadRootKey(key: SshRootKey, filename: string): void {
  downloadTextFile(filename, `${key.public_key.trim()}\n`)
}
</script>

<template>
  <div class="space-y-4">
    <section class="grid grid-cols-1 gap-4 md:grid-cols-3 mb-6">
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">User Certificates</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (stats?.total_user_certs ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Persisted user certificate records</p>
      </article>
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Host Certificates</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (stats?.total_host_certs ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Persisted host certificate records</p>
      </article>
      <article class="bg-card border-border-muted px-4 py-3">
        <p class="text-[10px] uppercase tracking-wide text-muted-foreground">Currently Active</p>
        <p class="mt-1 text-lg font-semibold text-foreground">
          {{ statsLoading ? '…' : (stats?.active_now ?? '—') }}
        </p>
        <p class="text-xs text-muted-foreground">Valid at the current time</p>
      </article>
    </section>

    <p v-if="statsError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md px-3 py-2 text-xs" role="alert">
      {{ statsError }}
    </p>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold text-foreground">SSH Certificate Inventory</h2>
            <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
              Issued certificates persisted for auditing via
              <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/ssh/certificates</code>.
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="openUserModal">New User Cert</button>
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="openHostModal">New Host Cert</button>
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="openInspectModal">Inspect Key</button>
          </div>
        </div>
      </header>

      <div class="px-4 py-3">
        <div v-if="tableError" class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md text-xs" role="alert">
          {{ tableError }}
        </div>

        <DataTable
          :columns="tableColumns"
          :rows="certificates"
          :row-key="(row) => row.id"
          :loading="tableLoading"
          empty-message="No SSH certificates have been issued yet."
        >
          <template #cell-cert_type="{ row }">
            <StatusBadge :label="certTypeLabel(row.cert_type)" :tone="certTypeTone(row.cert_type)" />
          </template>

          <template #cell-principals="{ row }">
            <span class="text-xs text-foreground/80">{{ row.principals.join(', ') || '—' }}</span>
          </template>

          <template #cell-fingerprint="{ row }">
            <span :title="row.fingerprint">{{ truncateFingerprint(row.fingerprint) }}</span>
          </template>

          <template #cell-valid_after="{ row }">
            {{ formatDateTime(row.valid_after) }}
          </template>

          <template #cell-valid_before="{ row }">
            <span class="inline-flex items-center gap-2">
              {{ formatDateTime(row.valid_before) }}
              <StatusBadge v-if="isActive(row)" label="Active" tone="valid" />
            </span>
          </template>
        </DataTable>

        <Pagination
          v-if="total > PAGE_SIZE"
          class="mt-3"
          :total="total"
          :limit="PAGE_SIZE"
          :offset="offset"
          @update:offset="offset = $event"
        />
      </div>
    </section>

    <section class="bg-card border-border-muted">
      <header class="border-b border-border px-4 py-2.5">
        <h2 class="text-sm font-semibold text-foreground">SSH CA Roots</h2>
        <p class="mt-0.5 text-xs text-muted-foreground">
          Trust anchors for client and server configuration.<template v-if="showApiHints">
            See
            <a
              href="https://github.com/ARCOOON/arx-ca/wiki/SSH-CA-Setup"
              target="_blank"
              rel="noopener noreferrer"
              class="ui-link"
            >Wiki → SSH CA Setup</a>
            for deployment steps.</template>
        </p>
      </header>

      <div v-if="rootsLoading" class="px-4 py-3 text-sm text-muted-foreground">Loading SSH roots…</div>
      <div v-else-if="rootsError" class="px-4 py-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
        {{ rootsError }}
      </div>
      <div v-else class="grid grid-cols-1 gap-px lg:grid-cols-2">
        <article
          v-for="section in [
            { title: 'User CA', keys: userRoots, prefix: 'ssh-user-ca' },
            { title: 'Host CA', keys: hostRoots, prefix: 'ssh-host-ca' },
          ]"
          :key="section.title"
          class="px-4 py-3"
        >
          <p class="text-[10px] uppercase tracking-wide text-muted-foreground">{{ section.title }}</p>
          <div v-if="section.keys.length === 0" class="mt-2 text-xs text-muted-foreground">No keys configured.</div>
          <ul v-else class="mt-2 space-y-2">
            <li
              v-for="(key, index) in section.keys"
              :key="`${section.prefix}-${index}`"
              class="rounded-md border border-border p-2 text-xs"
            >
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div class="min-w-0">
                  <StatusBadge :label="key.key_type" tone="neutral" />
                  <p class="mt-1 truncate font-mono text-foreground/80" :title="key.fingerprint">
                    {{ key.fingerprint }}
                  </p>
                </div>
                <button
                  type="button"
                  class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 shrink-0 text-[11px]"
                  @click="downloadRootKey(key, `${section.prefix}-${index + 1}.pub`)"
                >
                  Download .pub
                </button>
              </div>
            </li>
          </ul>
        </article>
      </div>
    </section>

    <Modal :open="userModalOpen" title="New User Certificate" wide @close="closeUserModal">
      <div class="space-y-3">
        <p v-if="showApiHints" class="text-xs text-muted-foreground">
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/ssh/generate/user</code>
        </p>

        <div v-if="userError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md text-xs" role="alert">
          {{ userError }}
        </div>

        <label class="block text-xs font-medium text-foreground/80">SSH public key</label>
        <textarea
          v-model="userPublicKey"
          rows="4"
          class="ui-textarea font-mono text-[11px]"
          placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI..."
          spellcheck="false"
        />

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium text-foreground/80">Principals</label>
            <input
              v-model="userPrincipals"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              placeholder="root, admin"
              autocomplete="off"
            />
            <p class="mt-1 text-[10px] text-muted-foreground">Comma-separated Unix usernames.</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-foreground/80">TTL / validity</label>
            <input v-model="userTtl" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" placeholder="4h" />
          </div>
        </div>

        <div v-if="userResult" class="space-y-2 rounded-md border border-border p-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-xs text-foreground/80">
              Signed {{ userResult.certificate_type }} certificate (serial {{ userResult.serial }}).
            </p>
            <button
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 text-[11px]"
              @click="downloadCertificate(userResult, 'ssh-user-cert.pub')"
            >
              Download .pub
            </button>
          </div>
          <pre class="rounded-md border border-input bg-muted/30 max-h-48 overflow-auto rounded-md p-3 font-mono text-[10px] text-foreground/80">{{ userResult.certificate }}</pre>
        </div>
      </div>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="userLoading" @click="closeUserModal">Cancel</button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="userLoading" @click="submitUserCertificate">
          {{ userLoading ? 'Generating…' : 'Generate User Certificate' }}
        </button>
      </template>
    </Modal>

    <Modal :open="hostModalOpen" title="New Host Certificate" wide @close="closeHostModal">
      <div class="space-y-3">
        <p v-if="showApiHints" class="text-xs text-muted-foreground">
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/ssh/generate/host</code>
        </p>

        <div v-if="hostError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md text-xs" role="alert">
          {{ hostError }}
        </div>

        <label class="block text-xs font-medium text-foreground/80">SSH public key</label>
        <textarea
          v-model="hostPublicKey"
          rows="4"
          class="ui-textarea font-mono text-[11px]"
          placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
          spellcheck="false"
        />

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="block text-xs font-medium text-foreground/80">Principals</label>
            <input
              v-model="hostPrincipals"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
              placeholder="web-01.example.com, 10.0.0.5"
              autocomplete="off"
            />
            <p class="mt-1 text-[10px] text-muted-foreground">Comma-separated hostnames or IP addresses.</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-foreground/80">TTL / validity</label>
            <input v-model="hostTtl" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" placeholder="8760h" />
          </div>
        </div>

        <div v-if="hostResult" class="space-y-2 rounded-md border border-border p-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-xs text-foreground/80">
              Signed {{ hostResult.certificate_type }} certificate (serial {{ hostResult.serial }}).
            </p>
            <button
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 text-[11px]"
              @click="downloadCertificate(hostResult, 'ssh-host-cert.pub')"
            >
              Download .pub
            </button>
          </div>
          <pre class="rounded-md border border-input bg-muted/30 max-h-48 overflow-auto rounded-md p-3 font-mono text-[10px] text-foreground/80">{{ hostResult.certificate }}</pre>
        </div>
      </div>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="hostLoading" @click="closeHostModal">Cancel</button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="hostLoading" @click="submitHostCertificate">
          {{ hostLoading ? 'Generating…' : 'Generate Host Certificate' }}
        </button>
      </template>
    </Modal>

    <Modal :open="inspectModalOpen" title="Inspect SSH Certificate" wide @close="closeInspectModal">
      <div class="space-y-3">
        <p v-if="showApiHints" class="text-xs text-muted-foreground">
          <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/ssh/inspect</code>
        </p>

        <div v-if="inspectError" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive rounded-md text-xs" role="alert">
          {{ inspectError }}
        </div>

        <label class="block text-xs font-medium text-foreground/80">SSH certificate</label>
        <textarea
          v-model="inspectCertificate"
          rows="6"
          class="ui-textarea font-mono text-[11px]"
          placeholder="ssh-ed25519-cert-v01@openssh.com AAAA..."
          spellcheck="false"
        />

        <div v-if="inspectResult" class="space-y-2 rounded-md border border-border p-3 text-xs">
          <dl class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <div>
              <dt class="text-muted-foreground">Type</dt>
              <dd class="text-foreground/80">{{ inspectResult.certificate_type }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Serial</dt>
              <dd class="font-mono text-foreground/80">{{ inspectResult.serial }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Key ID</dt>
              <dd class="text-foreground/80">{{ inspectResult.key_id }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Public key type</dt>
              <dd class="text-foreground/80">{{ inspectResult.public_key_type }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Valid after</dt>
              <dd class="text-foreground/80">{{ formatDateTime(inspectResult.valid_after) }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Valid before</dt>
              <dd class="text-foreground/80">{{ formatDateTime(inspectResult.valid_before) }}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="text-muted-foreground">Principals</dt>
              <dd class="text-foreground/80">{{ inspectResult.principals.join(', ') || '—' }}</dd>
            </div>
          </dl>
        </div>
      </div>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="inspectLoading" @click="closeInspectModal">
          Close
        </button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="inspectLoading" @click="submitInspect">
          {{ inspectLoading ? 'Inspecting…' : 'Inspect Certificate' }}
        </button>
      </template>
    </Modal>
  </div>
</template>
