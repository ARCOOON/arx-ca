<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createTemplate, listTemplates } from '../api/templates'
import type { CertificateTemplate } from '../types/api'
import { usePreferences } from '../composables/usePreferences'
import DataTable from '../components/ui/DataTable.vue'
import Modal from '../components/ui/Modal.vue'
import { extractApiError } from '../utils/errors'
import { formatDateTime } from '../utils/format'

const { showApiHints } = usePreferences()

const templates = ref<CertificateTemplate[]>([])
const isLoading = ref(true)
const errorMessage = ref('')

const createModalOpen = ref(false)
const createName = ref('')
const createDescription = ref('')
const createBody = ref('')
const createError = ref('')
const createSuccess = ref('')
const isCreating = ref(false)

const tableColumns = [
  { key: 'name', label: 'Name' },
  { key: 'description', label: 'Description' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: '', headerClass: 'w-24' },
]

const detailModalOpen = ref(false)
const selectedTemplate = ref<CertificateTemplate | null>(null)

async function loadTemplates(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const response = await listTemplates()
    templates.value = response.templates
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load templates')
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  void loadTemplates()
})

function openCreateModal(): void {
  createError.value = ''
  createSuccess.value = ''
  createName.value = ''
  createDescription.value = ''
  createBody.value = ''
  createModalOpen.value = true
}

function closeCreateModal(): void {
  if (isCreating.value) {
    return
  }
  createModalOpen.value = false
}

function openTemplateDetail(template: CertificateTemplate): void {
  selectedTemplate.value = template
  detailModalOpen.value = true
}

function closeTemplateDetail(): void {
  detailModalOpen.value = false
  selectedTemplate.value = null
}

async function submitCreate(): Promise<void> {
  createError.value = ''
  createSuccess.value = ''

  const name = createName.value.trim()
  const body = createBody.value.trim()

  if (!name) {
    createError.value = 'Template name is required.'
    return
  }
  if (!body) {
    createError.value = 'Template body is required.'
    return
  }

  isCreating.value = true

  try {
    const created = await createTemplate({
      name,
      description: createDescription.value.trim() || undefined,
      body,
    })
    createSuccess.value = `Template "${created.name}" created.`
    await loadTemplates()
    createName.value = ''
    createDescription.value = ''
    createBody.value = ''
  } catch (error) {
    createError.value = extractApiError(error, 'Failed to create template')
  } finally {
    isCreating.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p v-if="showApiHints" class="text-xs text-muted-foreground">
        Issuance templates from
        <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">GET /api/v1/templates</code>
      </p>
      <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" @click="openCreateModal">Create Template</button>
    </div>

    <div v-if="errorMessage" class="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
      {{ errorMessage }}
    </div>

    <DataTable
      :columns="tableColumns"
      :rows="templates"
      :row-key="(row) => row.id"
      :loading="isLoading"
      empty-message="No certificate templates registered."
    >
      <template #cell-created_at="{ row }">
        {{ formatDateTime(row.created_at) }}
      </template>
      <template #cell-actions="{ row }">
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 text-[11px]" @click="openTemplateDetail(row)">
          View
        </button>
      </template>
    </DataTable>

    <Modal :open="createModalOpen" title="Create Template" wide @close="closeCreateModal">
      <p v-if="showApiHints" class="mb-3 text-xs text-muted-foreground">
        Registers a Go text/template body via
        <code class="rounded-md border border-border bg-muted px-1 font-mono text-xs text-foreground">POST /api/v1/templates</code>.
      </p>

      <div v-if="createError" class="mb-3 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive text-xs" role="alert">
        {{ createError }}
      </div>
      <div v-if="createSuccess" class="mb-3 rounded-lg border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-foreground" role="status">
        {{ createSuccess }}
      </div>

      <label class="block text-xs font-medium text-foreground/80" for="template-name">Name</label>
      <input id="template-name" v-model="createName" type="text" class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5" autocomplete="off" />

      <label class="mt-3 block text-xs font-medium text-foreground/80" for="template-desc">
        Description (optional)
      </label>
      <input
        id="template-desc"
        v-model="createDescription"
        type="text"
        class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5"
        autocomplete="off"
      />

      <label class="mt-3 block text-xs font-medium text-foreground/80" for="template-body">
        Template body
      </label>
      <textarea
        id="template-body"
        v-model="createBody"
        rows="12"
        class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono shadow-none outline-none focus-visible:ring-1 focus-visible:ring-ring mt-1.5 font-mono text-[11px]"
        placeholder='{"subject": {"commonName": "{{ .CommonName }}"}}'
        spellcheck="false"
      />

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" :disabled="isCreating" @click="closeCreateModal">
          Cancel
        </button>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-none transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50" :disabled="isCreating" @click="submitCreate">
          {{ isCreating ? 'Creating…' : 'Create Template' }}
        </button>
      </template>
    </Modal>

    <Modal :open="detailModalOpen" title="Template Details" wide @close="closeTemplateDetail">
      <template v-if="selectedTemplate">
        <dl class="grid gap-3 text-xs sm:grid-cols-2">
          <div>
            <dt class="font-medium text-muted-foreground">ID</dt>
            <dd class="mt-0.5 font-mono text-foreground">{{ selectedTemplate.id }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Name</dt>
            <dd class="mt-0.5 text-foreground">{{ selectedTemplate.name }}</dd>
          </div>
          <div v-if="selectedTemplate.description" class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Description</dt>
            <dd class="mt-0.5 text-foreground">{{ selectedTemplate.description }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Created</dt>
            <dd class="mt-0.5 text-foreground">{{ formatDateTime(selectedTemplate.created_at) }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Updated</dt>
            <dd class="mt-0.5 text-foreground">{{ formatDateTime(selectedTemplate.updated_at) }}</dd>
          </div>
        </dl>
        <div class="mt-4">
          <p class="text-xs font-medium text-foreground/80">Body</p>
          <pre class="rounded-md border border-input bg-muted/30 mt-1.5 max-h-64 overflow-auto p-3 font-mono text-[10px] text-foreground/80">{{ selectedTemplate.body }}</pre>
        </div>
      </template>

      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50" @click="closeTemplateDetail">Close</button>
      </template>
    </Modal>
  </div>
</template>
