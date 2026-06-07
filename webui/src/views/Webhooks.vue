<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  createWebhook,
  deleteWebhook,
  listWebhookEvents,
  listWebhooks,
  testWebhook,
  updateWebhook,
} from '../api/webhooks'
import type { WebhookEventOption, WebhookResponse } from '../types/api'
import DataTable from '../components/ui/DataTable.vue'
import Modal from '../components/ui/Modal.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { usePreferences } from '../composables/usePreferences'
import { useToast } from '../composables/useToast'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'

const { showToast } = useToast()
const { showApiHints } = usePreferences()

const webhooks = ref<WebhookResponse[]>([])
const eventOptions = ref<WebhookEventOption[]>([])
const isLoading = ref(true)
const errorMessage = ref('')

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)
const editorName = ref('')
const editorURL = ref('')
const editorSecret = ref('')
const editorActive = ref(true)
const editorEvents = ref<string[]>([])
const editorError = ref('')
const editorSaving = ref(false)
const editorTesting = ref(false)

const deleteConfirmOpen = ref(false)
const deletingWebhook = ref<WebhookResponse | null>(null)
const deleteInProgress = ref(false)

const tableColumns = [
  { key: 'name', label: 'Name' },
  { key: 'url', label: 'Endpoint', cellClass: 'font-mono text-[11px] break-all' },
  { key: 'active', label: 'Status', headerClass: 'w-24' },
  { key: 'events', label: 'Subscriptions' },
  { key: 'updated_at', label: 'Updated' },
  { key: 'actions', label: '', headerClass: 'w-40' },
]

const editorTitle = computed(() => (editorMode.value === 'create' ? 'Add Webhook' : 'Edit Webhook'))
const selectedEventSet = computed(() => new Set(editorEvents.value))

onMounted(() => {
  void loadAll()
})

async function loadAll(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const [hooksResponse, eventsResponse] = await Promise.all([listWebhooks(), listWebhookEvents()])
    webhooks.value = hooksResponse.webhooks
    eventOptions.value = eventsResponse.events
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load webhooks')
  } finally {
    isLoading.value = false
  }
}

function openCreateModal(): void {
  editorMode.value = 'create'
  editingId.value = null
  editorName.value = ''
  editorURL.value = ''
  editorSecret.value = ''
  editorActive.value = true
  editorEvents.value = []
  editorError.value = ''
  editorOpen.value = true
}

function openEditModal(webhook: WebhookResponse): void {
  editorMode.value = 'edit'
  editingId.value = webhook.id
  editorName.value = webhook.name
  editorURL.value = webhook.url
  editorSecret.value = ''
  editorActive.value = webhook.active
  editorEvents.value = [...webhook.subscribed_events]
  editorError.value = ''
  editorOpen.value = true
}

function closeEditor(): void {
  if (editorSaving.value || editorTesting.value) {
    return
  }
  editorOpen.value = false
}

function toggleEvent(action: string): void {
  const next = new Set(editorEvents.value)
  if (next.has(action)) {
    next.delete(action)
  } else {
    next.add(action)
  }
  editorEvents.value = Array.from(next)
}

function openDeleteConfirm(webhook: WebhookResponse): void {
  deletingWebhook.value = webhook
  deleteConfirmOpen.value = true
}

function closeDeleteConfirm(): void {
  if (deleteInProgress.value) {
    return
  }
  deleteConfirmOpen.value = false
  deletingWebhook.value = null
}

async function submitEditor(): Promise<void> {
  editorError.value = ''

  const name = editorName.value.trim()
  const url = editorURL.value.trim()
  if (!name) {
    editorError.value = 'Name is required.'
    return
  }
  if (!url) {
    editorError.value = 'Webhook URL is required.'
    return
  }
  if (editorEvents.value.length === 0) {
    editorError.value = 'Select at least one event subscription.'
    return
  }

  editorSaving.value = true

  try {
    if (editorMode.value === 'create') {
      await createWebhook({
        name,
        url,
        secret_token: editorSecret.value.trim() || undefined,
        active: editorActive.value,
        subscribed_events: editorEvents.value,
      })
      showToast('Webhook created.', 'success')
    } else if (editingId.value) {
      await updateWebhook(editingId.value, {
        name,
        url,
        secret_token: editorSecret.value.trim() || undefined,
        active: editorActive.value,
        subscribed_events: editorEvents.value,
      })
      showToast('Webhook updated.', 'success')
    }
    editorOpen.value = false
    await loadAll()
  } catch (error) {
    editorError.value = extractApiError(error, 'Failed to save webhook')
  } finally {
    editorSaving.value = false
  }
}

async function runTestFromEditor(): Promise<void> {
  if (editorMode.value !== 'edit' || !editingId.value) {
    editorError.value = 'Save the webhook before testing connectivity.'
    return
  }

  editorTesting.value = true
  editorError.value = ''

  try {
    const result = await testWebhook(editingId.value)
    if (result.success) {
      showToast(`Test delivered (${result.status_code}, ${result.latency_ms} ms).`, 'success')
      return
    }
    const detail = result.error || `HTTP ${result.status_code}`
    showToast(`Test failed: ${detail}`, 'error')
  } catch (error) {
    showToast(extractApiError(error, 'Webhook test failed'), 'error')
  } finally {
    editorTesting.value = false
  }
}

async function runTestFromRow(webhook: WebhookResponse): Promise<void> {
  try {
    const result = await testWebhook(webhook.id)
    if (result.success) {
      showToast(`"${webhook.name}" responded ${result.status_code} in ${result.latency_ms} ms.`, 'success')
      return
    }
    const detail = result.error || `HTTP ${result.status_code}`
    showToast(`"${webhook.name}" test failed: ${detail}`, 'error')
  } catch (error) {
    showToast(extractApiError(error, 'Webhook test failed'), 'error')
  }
}

async function confirmDelete(): Promise<void> {
  if (!deletingWebhook.value) {
    return
  }

  deleteInProgress.value = true

  try {
    await deleteWebhook(deletingWebhook.value.id)
    showToast('Webhook deleted.', 'success')
    deleteConfirmOpen.value = false
    deletingWebhook.value = null
    await loadAll()
  } catch (error) {
    showToast(extractApiError(error, 'Failed to delete webhook'), 'error')
  } finally {
    deleteInProgress.value = false
  }
}

function eventSummary(events: string[]): string {
  if (events.length === 0) {
    return '—'
  }
  if (events.length <= 2) {
    return events.join(', ')
  }
  return `${events.slice(0, 2).join(', ')} +${events.length - 2}`
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-xs ui-text-muted">
        Outbound alerts<template v-if="showApiHints">
          via
          <code class="ui-code">GET /api/v1/webhooks</code></template>
        — Discord, Slack, Gotify, and custom HTTP receivers.
      </p>
      <button type="button" class="ui-btn-primary" @click="openCreateModal">Add Webhook</button>
    </div>

    <div v-if="errorMessage" class="ui-alert-error" role="alert">
      {{ errorMessage }}
    </div>

    <DataTable
      :columns="tableColumns"
      :rows="webhooks"
      :row-key="(row) => row.id"
      :loading="isLoading"
      empty-message="No webhooks configured. Add an endpoint to receive audit alerts."
    >
      <template #cell-active="{ row }">
        <StatusBadge :label="row.active ? 'Active' : 'Paused'" :tone="row.active ? 'enabled' : 'disabled'" />
      </template>
      <template #cell-events="{ row }">
        <span class="ui-text-secondary">{{ eventSummary(row.subscribed_events) }}</span>
      </template>
      <template #cell-updated_at="{ row }">
        {{ formatDateTime(row.updated_at) }}
      </template>
      <template #cell-actions="{ row }">
        <div class="flex flex-wrap gap-1.5">
          <button type="button" class="ui-btn-secondary text-[11px]" @click="openEditModal(row)">Edit</button>
          <button type="button" class="ui-btn-secondary text-[11px]" @click="runTestFromRow(row)">Test</button>
          <button type="button" class="ui-btn-secondary text-[11px]" @click="openDeleteConfirm(row)">Delete</button>
        </div>
      </template>
    </DataTable>

    <Modal :open="editorOpen" :title="editorTitle" wide @close="closeEditor">
      <p class="mb-3 text-xs ui-text-muted">
        Deliver JSON audit payloads when subscribed events occur.<template v-if="showApiHints">
          Optional HMAC signatures use
          <code class="ui-code">X-Webhook-Signature: sha256=&lt;hex&gt;</code>.</template>
      </p>

      <div v-if="editorError" class="mb-3 ui-alert-error text-xs" role="alert">
        {{ editorError }}
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <div>
          <label class="block text-xs font-medium ui-text-secondary" for="webhook-name">Name</label>
          <input id="webhook-name" v-model="editorName" type="text" class="ui-input mt-1.5" autocomplete="off" />
        </div>
        <div>
          <label class="block text-xs font-medium ui-text-secondary" for="webhook-url">URL endpoint</label>
          <input
            id="webhook-url"
            v-model="editorURL"
            type="url"
            class="ui-input mt-1.5 font-mono text-[11px]"
            placeholder="https://discord.com/api/webhooks/..."
            autocomplete="off"
          />
        </div>
      </div>

      <div class="mt-3">
        <label class="block text-xs font-medium ui-text-secondary" for="webhook-secret">
          Secret token (optional)
        </label>
        <input
          id="webhook-secret"
          v-model="editorSecret"
          type="password"
          class="ui-input mt-1.5 font-mono text-[11px]"
          :placeholder="editorMode === 'edit' ? 'Leave blank to keep existing secret' : 'HMAC signing key'"
          autocomplete="new-password"
        />
      </div>

      <div class="mt-3 flex items-center justify-between gap-3 rounded-[var(--radius-control)] border border-[var(--border-color)] px-3 py-2.5">
        <div>
          <p class="text-xs font-medium ui-text-secondary">Active</p>
          <p class="mt-0.5 text-[11px] ui-text-muted">Paused webhooks are not dispatched.</p>
        </div>
        <button
          type="button"
          class="ui-theme-toggle shrink-0"
          :data-active="editorActive"
          :aria-pressed="editorActive"
          aria-label="Webhook active"
          @click="editorActive = !editorActive"
        >
          <span class="ui-theme-toggle-thumb" />
        </button>
      </div>

      <div class="mt-4">
        <p class="text-xs font-medium ui-text-secondary">Subscribed events</p>
        <p class="mt-0.5 text-[11px] ui-text-muted">Select audit actions that should trigger this webhook.</p>
        <div class="mt-2 grid gap-2 sm:grid-cols-2">
          <label
            v-for="option in eventOptions"
            :key="option.action"
            class="flex cursor-pointer items-start gap-2 rounded-[var(--radius-control)] border border-[var(--border-color)] px-3 py-2.5"
          >
            <input
              type="checkbox"
              class="mt-0.5"
              :checked="selectedEventSet.has(option.action)"
              @change="toggleEvent(option.action)"
            />
            <span class="min-w-0">
              <span class="block text-xs font-medium ui-text-primary">{{ option.label }}</span>
              <span class="mt-0.5 block text-[11px] ui-text-muted">{{ option.description }}</span>
              <span class="mt-1 block font-mono text-[10px] ui-text-muted">{{ option.action }}</span>
            </span>
          </label>
        </div>
      </div>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <button
            v-if="editorMode === 'edit'"
            type="button"
            class="ui-btn-secondary"
            :disabled="editorSaving || editorTesting"
            @click="runTestFromEditor"
          >
            {{ editorTesting ? 'Testing…' : 'Test Connection' }}
          </button>
          <div v-else />
          <div class="flex flex-wrap gap-2">
            <button type="button" class="ui-btn-secondary" :disabled="editorSaving" @click="closeEditor">
              Cancel
            </button>
            <button type="button" class="ui-btn-primary" :disabled="editorSaving" @click="submitEditor">
              {{ editorSaving ? 'Saving…' : editorMode === 'create' ? 'Create Webhook' : 'Save Changes' }}
            </button>
          </div>
        </div>
      </template>
    </Modal>

    <Modal
      :open="deleteConfirmOpen"
      title="Delete Webhook"
      @close="closeDeleteConfirm"
    >
      <p class="text-sm ui-text-secondary">
        Delete webhook
        <strong class="ui-text-primary">{{ deletingWebhook?.name }}</strong>?
        This cannot be undone.
      </p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-secondary" :disabled="deleteInProgress" @click="closeDeleteConfirm">
            Cancel
          </button>
          <button type="button" class="ui-btn-primary" :disabled="deleteInProgress" @click="confirmDelete">
            {{ deleteInProgress ? 'Deleting…' : 'Delete' }}
          </button>
        </div>
      </template>
    </Modal>
  </div>
</template>
