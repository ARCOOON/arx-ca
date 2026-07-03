<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Switch from '@/components/ui/Switch.vue'
import Spinner from '@/components/ui/Spinner.vue'
import Dialog from '@/components/ui/Dialog.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import {
  fetchWebhooks,
  fetchWebhookEvents,
  createWebhook,
  updateWebhook,
  deleteWebhook,
  testWebhook,
} from '@/api/webhooks'
import type { WebhookResponse, WebhookEventOption } from '@/types/api'
import { formatDate } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const webhooks = ref<WebhookResponse[]>([])
const events = ref<WebhookEventOption[]>([])

const formOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingId = ref('')
const saving = ref(false)
const testing = ref(false)

const formUrl = ref('')
const formName = ref('')
const formSecret = ref('')
const formActive = ref(true)
const formEvents = ref<string[]>([])

async function load(): Promise<void> {
  loading.value = true
  try {
    const [wh, ev] = await Promise.all([fetchWebhooks(), fetchWebhookEvents().catch(() => ({ events: [] }))])
    webhooks.value = wh.webhooks
    events.value = ev.events
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

function openCreate(): void {
  formMode.value = 'create'
  editingId.value = ''
  formUrl.value = ''
  formName.value = ''
  formSecret.value = ''
  formActive.value = true
  formEvents.value = []
  formOpen.value = true
}

function openEdit(wh: WebhookResponse): void {
  formMode.value = 'edit'
  editingId.value = wh.id
  formUrl.value = wh.url
  formName.value = wh.name
  formSecret.value = ''
  formActive.value = wh.active
  formEvents.value = [...wh.subscribed_events]
  formOpen.value = true
}

function toggleEvent(action: string): void {
  const idx = formEvents.value.indexOf(action)
  if (idx === -1) formEvents.value.push(action)
  else formEvents.value.splice(idx, 1)
}

async function handleSave(): Promise<void> {
  saving.value = true
  try {
    const payload = {
      url: formUrl.value,
      name: formName.value,
      secret_token: formSecret.value || undefined,
      active: formActive.value,
      subscribed_events: formEvents.value,
    }
    if (formMode.value === 'create') {
      await createWebhook(payload)
      toast.success('Webhook created')
    } else {
      await updateWebhook(editingId.value, payload)
      toast.success('Webhook updated')
    }
    formOpen.value = false
    void load()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: string): Promise<void> {
  if (!confirm('Delete this webhook?')) return
  try {
    await deleteWebhook(id)
    toast.success('Webhook deleted')
    void load()
  } catch (err) {
    toast.error(extractErrorMessage(err))
  }
}

async function handleTest(id: string): Promise<void> {
  testing.value = true
  try {
    const res = await testWebhook(id)
    if (res.success) {
      toast.success(`Test succeeded (${res.status_code}) in ${res.latency_ms}ms`)
    } else {
      toast.warning(`Test failed (${res.status_code}): ${res.error ?? ''}`)
    }
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <p class="text-sm text-foreground-muted">{{ webhooks.length }} webhook(s) configured</p>
      <Button size="sm" @click="openCreate">Add Webhook</Button>
    </div>

    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <div v-else-if="webhooks.length === 0" class="flex flex-col items-center justify-center py-16 text-foreground-muted text-sm gap-2">
      <p>No webhooks configured.</p>
      <Button variant="outline" size="sm" @click="openCreate">Create first webhook</Button>
    </div>

    <div v-else class="space-y-3">
      <Card
        v-for="wh in webhooks"
        :key="wh.id"
        class="px-5 py-4"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2 mb-1">
              <p class="text-sm font-semibold text-foreground">{{ wh.name }}</p>
              <StatusBadge :status="wh.active ? 'enabled' : 'disabled'" />
            </div>
            <p class="text-xs font-mono text-foreground-muted truncate">{{ wh.url }}</p>
            <div class="flex flex-wrap gap-1 mt-2">
              <Badge
                v-for="ev in wh.subscribed_events.slice(0, 4)"
                :key="ev"
                variant="outline"
                class="text-[10px]"
              >
                {{ ev }}
              </Badge>
              <Badge v-if="wh.subscribed_events.length > 4" variant="secondary" class="text-[10px]">
                +{{ wh.subscribed_events.length - 4 }} more
              </Badge>
            </div>
            <p class="mt-1.5 text-[10px] text-foreground-subtle">Updated {{ formatDate(wh.updated_at) }}</p>
          </div>
          <div class="flex shrink-0 gap-1">
            <Button variant="ghost" size="sm" :disabled="testing" @click="handleTest(wh.id)">Test</Button>
            <Button variant="ghost" size="sm" @click="openEdit(wh)">Edit</Button>
            <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="handleDelete(wh.id)">Delete</Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Webhook form dialog -->
    <Dialog
      :open="formOpen"
      :title="formMode === 'create' ? 'Add Webhook' : 'Edit Webhook'"
      max-width="max-w-lg"
      @close="formOpen = false"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <Label>Name</Label>
          <Input v-model="formName" placeholder="My Webhook" />
        </div>
        <div class="space-y-1.5">
          <Label>Endpoint URL</Label>
          <Input v-model="formUrl" placeholder="https://hooks.example.com/receiver" type="url" />
        </div>
        <div class="space-y-1.5">
          <Label>Secret Token (optional)</Label>
          <Input v-model="formSecret" placeholder="Leave blank to keep existing" type="password" />
        </div>
        <div class="flex items-center gap-3">
          <Switch v-model="formActive" />
          <Label>Active</Label>
        </div>
        <div v-if="events.length" class="space-y-2">
          <Label>Subscribed Events</Label>
          <div class="grid grid-cols-2 gap-1.5 max-h-48 overflow-y-auto rounded-md border border-border p-3">
            <label
              v-for="ev in events"
              :key="ev.action"
              class="flex items-center gap-2 text-xs cursor-pointer hover:text-foreground transition-colors"
              :class="formEvents.includes(ev.action) ? 'text-foreground' : 'text-foreground-muted'"
            >
              <input
                type="checkbox"
                :checked="formEvents.includes(ev.action)"
                class="h-3 w-3 accent-primary"
                @change="toggleEvent(ev.action)"
              />
              {{ ev.label }}
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="formOpen = false">Cancel</Button>
        <Button :disabled="saving || !formUrl || !formName" @click="handleSave">
          <Spinner v-if="saving" size="sm" />
          <span>{{ saving ? 'Saving…' : formMode === 'create' ? 'Create' : 'Save' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>
