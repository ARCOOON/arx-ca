<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { listAuditLogs } from '../api/audit'
import type { AuditLogEntry } from '../types/api'
import Button from '@/components/ui/Button.vue'
import DataTable from '../components/ui/DataTable.vue'
import Input from '@/components/ui/Input.vue'
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
    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <section class="ui-surface-muted">
      <header class="ui-border-b flex flex-wrap items-center justify-between gap-2 px-4 py-2.5">
        <div>
          <h2 class="text-sm font-semibold ui-text-primary">Immutable Audit Log</h2>
          <p v-if="showApiHints" class="mt-0.5 text-xs ui-text-muted">
            Forensic request trail from
            <code class="ui-code">GET /api/v1/audit</code>
          </p>
        </div>
        <div class="flex items-center gap-2 text-xs ui-text-muted">
          <span>{{ total }} entries</span>
          <span>·</span>
          <span>Page {{ currentPage }} / {{ totalPages }}</span>
        </div>
      </header>

      <div class="ui-border-b px-4 py-3">
        <Button
          variant="ghost"
          class="flex h-auto w-full items-center gap-2 p-0 text-left text-xs font-medium ui-text-secondary hover:bg-transparent"
          :aria-expanded="filtersOpen"
          @click="filtersOpen = !filtersOpen"
        >
          <Filter class="h-3.5 w-3.5" aria-hidden="true" />
          <span>Search</span>
          <span v-if="hasActiveFilters" class="ui-text-muted">(active)</span>
          <ChevronDown
            class="ml-auto h-4 w-4 transition-transform"
            :class="{ 'rotate-180': filtersOpen }"
            aria-hidden="true"
          />
        </Button>

        <div
          v-show="filtersOpen"
          class="mt-3 grid gap-3 rounded-[var(--radius-control)] border border-[var(--border-subtle)] p-3 sm:grid-cols-3"
        >
          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="audit-filter-action">
              Action
            </label>
            <select id="audit-filter-action" v-model="draftAction" class="ui-input mt-1.5 w-full">
              <option value="">All actions</option>
              <option v-for="action in AUDIT_ACTIONS" :key="action" :value="action">
                {{ action }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="audit-filter-actor">
              Actor
            </label>
            <Input
              id="audit-filter-actor"
              v-model="draftActor"
              class="mt-1.5 w-full"
              placeholder="User ID or type"
              autocomplete="off"
              @keydown.enter.prevent="applyFilters"
            />
          </div>

          <div>
            <label class="block text-xs font-medium ui-text-secondary" for="audit-filter-ip">
              IP
            </label>
            <Input
              id="audit-filter-ip"
              v-model="draftIP"
              class="mt-1.5 w-full"
              placeholder="203.0.113.10"
              autocomplete="off"
              @keydown.enter.prevent="applyFilters"
            />
          </div>

          <div class="flex flex-wrap items-end gap-2 sm:col-span-3">
            <Button :disabled="isLoading" @click="applyFilters">
              Apply Search
            </Button>
            <Button
              variant="secondary"
              :disabled="isLoading || !hasActiveFilters"
              @click="clearFilters"
            >
              Clear
            </Button>
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
          <Button
            variant="secondary"
            size="icon"
            class="h-7 w-7"
            :aria-expanded="isExpanded(row.id)"
            :aria-label="isExpanded(row.id) ? 'Collapse details' : 'Expand details'"
            @click="toggleExpanded(row.id)"
          >
            <ChevronDown v-if="isExpanded(row.id)" class="h-4 w-4" aria-hidden="true" />
            <ChevronRight v-else class="h-4 w-4" aria-hidden="true" />
          </Button>
        </template>

        <template #cell-timestamp="{ row }">
          {{ formatDateTime(row.timestamp) }}
        </template>

        <template #cell-actor="{ row }">
          <span class="ui-text-primary">{{ actorLabel(row) }}</span>
        </template>

        <template #cell-status_code="{ row }">
          <StatusBadge :label="String(row.status_code)" :tone="statusTone(row.status_code)" />
        </template>

        <template #cell-fingerprint="{ row }">
          <span :title="row.fingerprint || undefined">{{ truncateFingerprint(row.fingerprint) }}</span>
        </template>

        <template #row-expanded="{ row, columns }">
          <tr v-if="isExpanded(row.id)" class="ui-table-row">
            <td :colspan="columns.length" class="px-4 py-3">
              <div class="grid gap-3 md:grid-cols-2">
                <div class="space-y-1 text-xs">
                  <p class="font-medium ui-text-primary">Request ID</p>
                  <p class="break-all font-mono ui-text-secondary">{{ row.request_id }}</p>
                </div>
                <div class="space-y-1 text-xs">
                  <p class="font-medium ui-text-primary">User-Agent</p>
                  <p class="break-all ui-text-secondary">{{ userAgent(row) }}</p>
                </div>
              </div>
              <div class="mt-3 space-y-1 text-xs">
                <p class="font-medium ui-text-primary">Metadata</p>
                <pre class="ui-inset max-h-64 overflow-auto p-3 font-mono text-[10px] ui-text-secondary">{{ metadataPreview(row) }}</pre>
              </div>
            </td>
          </tr>
        </template>
      </DataTable>

      <footer
        v-if="total > 0"
        class="ui-border-t flex flex-wrap items-center justify-between gap-3 px-4 py-3"
      >
        <p class="text-xs ui-text-muted">
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
