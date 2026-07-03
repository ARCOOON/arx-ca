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
import { fetchTemplates, createTemplate } from '@/api/templates'
import type { CertificateTemplate } from '@/types/api'
import { formatDate } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const templates = ref<CertificateTemplate[]>([])

const createOpen = ref(false)
const creating = ref(false)
const formName = ref('')
const formDesc = ref('')
const formBody = ref('')

onMounted(async () => {
  try {
    const res = await fetchTemplates()
    templates.value = res.templates
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
})

async function handleCreate(): Promise<void> {
  creating.value = true
  try {
    const t = await createTemplate({ name: formName.value, description: formDesc.value || undefined, body: formBody.value })
    templates.value.unshift(t)
    createOpen.value = false
    toast.success('Template created')
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    creating.value = false
  }
}

function openCreate(): void {
  formName.value = ''
  formDesc.value = ''
  formBody.value = ''
  createOpen.value = true
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <p class="text-sm text-foreground-muted">{{ templates.length }} template(s)</p>
      <Button size="sm" @click="openCreate">New Template</Button>
    </div>

    <div v-if="loading" class="flex justify-center py-16"><Spinner size="lg" /></div>

    <div v-else-if="templates.length === 0" class="flex flex-col items-center py-16 text-foreground-muted text-sm gap-2">
      <p>No certificate templates defined.</p>
      <Button variant="outline" size="sm" @click="openCreate">Create first template</Button>
    </div>

    <div v-else class="space-y-3">
      <Card v-for="t in templates" :key="t.id" class="px-5 py-4 space-y-2">
        <div class="flex items-start justify-between gap-2">
          <div>
            <p class="text-sm font-semibold text-foreground">{{ t.name }}</p>
            <p v-if="t.description" class="text-xs text-foreground-muted mt-0.5">{{ t.description }}</p>
          </div>
          <Badge variant="secondary" class="shrink-0 text-[10px]">{{ t.id.slice(0, 8) }}</Badge>
        </div>
        <pre class="rounded-md bg-muted p-3 text-[10px] font-mono text-foreground-muted overflow-x-auto max-h-24">{{ t.body }}</pre>
        <p class="text-[10px] text-foreground-subtle">Updated {{ formatDate(t.updated_at) }}</p>
      </Card>
    </div>

    <Dialog :open="createOpen" title="New Certificate Template" max-width="max-w-xl" @close="createOpen = false">
      <div class="space-y-3">
        <div class="space-y-1.5">
          <Label>Name</Label>
          <Input v-model="formName" placeholder="webserver-2year" />
        </div>
        <div class="space-y-1.5">
          <Label>Description (optional)</Label>
          <Input v-model="formDesc" placeholder="Standard web server certificate" />
        </div>
        <div class="space-y-1.5">
          <Label>Template Body (JSON/Go template)</Label>
          <Textarea v-model="formBody" placeholder='{"subject": {"commonName": "{{.Subject.CommonName}}"}}' :rows="8" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="createOpen = false">Cancel</Button>
        <Button :disabled="creating || !formName || !formBody" @click="handleCreate">
          <Spinner v-if="creating" size="sm" />
          <span>{{ creating ? 'Creating…' : 'Create' }}</span>
        </Button>
      </template>
    </Dialog>
  </div>
</template>
