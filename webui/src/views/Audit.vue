<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { listAuditLogs } from '../api/audit'
import type { AuditLogEntry } from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import Pagination from '../components/Pagination.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'
import ChevronDown from 'lucide-vue-next/dist/esm/icons/chevron-down.js'
import ChevronRight from 'lucide-vue-next/dist/esm/icons/chevron-right.js'
import { usePreferences } from '../composables/usePreferences'
import Filter from 'lucide-vue-next/dist/esm/icons/list-filter.js'

const PAGE_SIZE = 50

const AUDIT_ACTIONS = [
  'AUTH_LOGIN_SUCCESS',
  'AUTH_LOGIN_FAILED',
  'AUTH_LOGOUT',
  'CERT_ISSUE_NATIVE',
  'CERT_ISSUE_CSR',
  'CERT_REVOKE',
  'CERT_RENEW',
  'CERT_AUTO',
  'EAB_GENERATE',
  'EAB_REVOKE',
  'SCEP_CHALLENGE_ROTATED',
  'SSH_USER_CERT_ISSUE',
  'SSH_HOST_CERT_ISSUE',
  'SSH_SIGN_USER',
  'SSH_SIGN_HOST',
  'WEBHOOK_CREATED',
  'WEBHOOK_UPDATED',
  'WEBHOOK_DELETED',
  'WEBHOOK_TEST',
  'SERVICE_ACCOUNT_CREATE',
  'PROVISIONER_TOKEN',
  'TEMPLATE_CREATE',
  'HTTP_WRITE',
  'HTTP_UPDATE',
  'HTTP_DELETE',
] as const

const logs = ref<AuditLogEntry[]>([])
const total = ref(0)
const offset = ref(0)
const isLoading = ref(true)
const errorMessage = ref('')
const expandedIds = ref<Set<string>>(new Set())
const { showApiHints } = usePreferences()

const filtersOpen = ref(false)

const draftAction = ref('')
const draftActor = ref('')
const draftIP = ref('')

const appliedAction = ref('')
const appliedActor = ref('')
const appliedIP = ref('')

const tableColumns = [
  { key: 'expand', label: '', headerClass: 'w-10' },
  { key: 'timestamp', label: 'Timestamp' },
  { key: 'action', label: 'Action' },
  { key: 'actor', label: 'Actor' },
  { key: 'ip_address', label: 'IP' },
  { key: 'status_code', label: 'Status' },
  { key: 'fingerprint', label: 'Fingerprint', cellClass: 'font-mono text-[11px]' },
]

const currentPage = computed(() => Math.floor(offset.value / PAGE_SIZE) + 1)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const hasActiveFilters = computed(
  () => Boolean(appliedAction.value || appliedActor.value || appliedIP.value),
)

function actorLabel(row: AuditLogEntry): string {
  if (!row.actor_id) {
    return row.actor_type
  }
  return `${row.actor_type}: ${row.actor_id}`
}

function statusTone(code: number): 'enabled' | 'expired' | 'revoked' | 'neutral' {
  if (code >= 200 && code < 300) {
    return 'enabled'
  }
  if (code >= 400 && code < 500) {
    return 'expired'
  }
  if (code >= 500) {
    return 'revoked'
  }
  return 'neutral'
}

function truncateFingerprint(value: string | undefined): string {
  if (!value) {
    return '—'
  }
  if (value.length <= 16) {
    return value
  }
  return `${value.slice(0, 8)}…${value.slice(-8)}`
}

function userAgent(row: AuditLogEntry): string {
  const ua = row.metadata?.user_agent
  return typeof ua === 'string' && ua.trim() ? ua : '—'
}

function metadataPreview(row: AuditLogEntry): string {
  try {
    return JSON.stringify(row.metadata ?? {}, null, 2)
  } catch {
    return '{}'
  }
}

function isExpanded(id: string): boolean {
  return expandedIds.value.has(id)
}

function toggleExpanded(id: string): void {
  const next = new Set(expandedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expandedIds.value = next
}

async function loadLogs(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const response = await listAuditLogs({
      limit: PAGE_SIZE,
      offset: offset.value,
      action: appliedAction.value || undefined,
      actor: appliedActor.value || undefined,
      ip: appliedIP.value || undefined,
    })
    logs.value = response.logs
    total.value = response.total
    expandedIds.value = new Set()
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load audit logs')
  } finally {
    isLoading.value = false
  }
}

function reloadFromFirstPage(): void {
  if (offset.value === 0) {
    void loadLogs()
    return
  }
  offset.value = 0
}

function applyFilters(): void {
  appliedAction.value = draftAction.value.trim()
  appliedActor.value = draftActor.value.trim()
  appliedIP.value = draftIP.value.trim()
  reloadFromFirstPage()
}

function clearFilters(): void {
  draftAction.value = ''
  draftActor.value = ''
  draftIP.value = ''
  appliedAction.value = ''
  appliedActor.value = ''
  appliedIP.value = ''
  reloadFromFirstPage()
}

function onOffsetChange(nextOffset: number): void {
  offset.value = nextOffset
}

watch(offset, () => {
  void loadLogs()
})

onMounted(() => {
  void loadLogs()
})
</script>

<template>
  <div class="space-y-4">
    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <section class="bg-card border-border">
      <header class="border-b border-border flex flex-wrap items-center justify-between gap-2 px-4 py-2.5">
        <div>
          <h2 class="text-sm font-semibold text-foreground">Immutable Audit Log</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs text-muted-foreground">
            Forensic request trail from
            <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/audit</code>
          </p>
        </div>
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{{ total }} entries</span>
          <span>·</span>
          <span>Page {{ currentPage }} / {{ totalPages }}</span>
        </div>
      </header>

      <div class="border-b border-border px-4 py-3">
        <button
          type="button"
          class="flex w-full items-center gap-2 text-left text-xs font-medium text-foreground/80"
          :aria-expanded="filtersOpen"
          @click="filtersOpen = !filtersOpen"
        >
          <Filter class="h-3.5 w-3.5" />
          <span>Search</span>
          <span v-if="hasActiveFilters" class="text-muted-foreground">(active)</span>
          <ChevronDown
            class="ml-auto h-4 w-4 transition-transform"
            :class="{ 'rotate-180': filtersOpen }"
           
          />
        </button>

        <div
          v-show="filtersOpen"
          class="mt-3 grid gap-3 rounded-md border border-border p-3 sm:grid-cols-3"
        >
          <div>
            <label class="block text-xs font-medium text-foreground/80" for="audit-filter-action">
              Action
            </label>
            <select id="audit-filter-action" v-model="draftAction" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full">
              <option value="">All actions</option>
              <option v-for="action in AUDIT_ACTIONS" :key="action" :value="action">
                {{ action }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-xs font-medium text-foreground/80" for="audit-filter-actor">
              Actor
            </label>
            <input
              id="audit-filter-actor"
              v-model="draftActor"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full"
              placeholder="User ID or type"
              autocomplete="off"
              @keydown.enter.prevent="applyFilters"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-foreground/80" for="audit-filter-ip">
              IP
            </label>
            <input
              id="audit-filter-ip"
              v-model="draftIP"
              type="text"
              class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 w-full"
              placeholder="203.0.113.10"
              autocomplete="off"
              @keydown.enter.prevent="applyFilters"
            />
          </div>

          <div class="flex flex-wrap items-end gap-2 sm:col-span-3">
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="isLoading" @click="applyFilters">
              Apply Search
            </button>
            <button
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
              :disabled="isLoading || !hasActiveFilters"
              @click="clearFilters"
            >
              Clear
            </button>
          </div>
        </div>
      </div>

      <DataTable
        :columns="tableColumns"
        :rows="logs"
        :row-key="(row) => row.id"
        :loading="isLoading"
        empty-message="No audit events match the current filters."
      >
        <template #cell-expand="{ row }">
          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 inline-flex h-7 w-7 items-center justify-center p-0"
            :aria-expanded="isExpanded(row.id)"
            :aria-label="isExpanded(row.id) ? 'Collapse details' : 'Expand details'"
            @click="toggleExpanded(row.id)"
          >
            <ChevronDown v-if="isExpanded(row.id)" class="h-4 w-4" />
            <ChevronRight v-else class="h-4 w-4" />
          </button>
        </template>

        <template #cell-timestamp="{ row }">
          {{ formatDateTime(row.timestamp) }}
        </template>

        <template #cell-actor="{ row }">
          <span class="text-foreground">{{ actorLabel(row) }}</span>
        </template>

        <template #cell-status_code="{ row }">
          <StatusBadge :label="String(row.status_code)" :tone="statusTone(row.status_code)" />
        </template>

        <template #cell-fingerprint="{ row }">
          <span :title="row.fingerprint || undefined">{{ truncateFingerprint(row.fingerprint) }}</span>
        </template>

        <template #row-expanded="{ row, columns }">
          <tr v-if="isExpanded(row.id)" class="hover:bg-muted/50">
            <td :colspan="columns.length" class="px-4 py-3">
              <div class="grid gap-3 md:grid-cols-2">
                <div class="space-y-1 text-xs">
                  <p class="font-medium text-foreground">Request ID</p>
                  <p class="break-all font-mono text-foreground/80">{{ row.request_id }}</p>
                </div>
                <div class="space-y-1 text-xs">
                  <p class="font-medium text-foreground">User-Agent</p>
                  <p class="break-all text-foreground/80">{{ userAgent(row) }}</p>
                </div>
              </div>
              <div class="mt-3 space-y-1 text-xs">
                <p class="font-medium text-foreground">Metadata</p>
                <pre class="rounded-md border border-input bg-muted/30 max-h-64 overflow-auto p-3 font-mono text-[10px] text-foreground/80">{{ metadataPreview(row) }}</pre>
              </div>
            </td>
          </tr>
        </template>
      </DataTable>

      <footer
        v-if="total > 0"
        class="border-t border-border flex flex-wrap items-center justify-between gap-3 px-4 py-3"
      >
        <p class="text-xs text-muted-foreground">
          Showing {{ offset + 1 }}–{{ Math.min(offset + PAGE_SIZE, total) }} of {{ total }}
        </p>
        <Pagination
          :total="total"
          :limit="PAGE_SIZE"
          :offset="offset"
          :disabled="isLoading"
          @update:offset="onOffsetChange"
        />
      </footer>
    </section>
  </div>
</template>
