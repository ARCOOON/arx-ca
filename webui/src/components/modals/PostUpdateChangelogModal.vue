<script setup lang="ts">
import { computed, watch } from 'vue'
import { fetchCurrentChangelog } from '@/api/updater'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { extractApiError } from '@/utils/errors'
import { ref } from 'vue'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

interface ChangelogSection {
  title: string
  kind: 'breaking' | 'features' | 'fixes' | 'other'
  items: string[]
}

const isLoading = ref(false)
const errorMessage = ref('')
const version = ref('')
const sections = ref<ChangelogSection[]>([])

const sectionToneClass = computed(() => ({
  breaking: 'text-destructive',
  features: 'text-primary',
  fixes: 'text-success',
  other: 'text-muted-foreground',
}))

function classifySection(title: string): ChangelogSection['kind'] {
  const normalized = title.trim().toLowerCase()
  if (normalized.includes('breaking')) return 'breaking'
  if (normalized.includes('feature')) return 'features'
  if (normalized.includes('bug') || normalized.includes('fix')) return 'fixes'
  return 'other'
}

function parseChangelogMarkdown(markdown: string): ChangelogSection[] {
  const lines = markdown.split('\n')
  const parsed: ChangelogSection[] = []
  let current: ChangelogSection | null = null

  for (const rawLine of lines) {
    const line = rawLine.trimEnd()
    const headingMatch = line.match(/^###\s+(.+)$/)
    if (headingMatch) {
      if (current && current.items.length > 0) parsed.push(current)
      const title = headingMatch[1].trim()
      current = { title, kind: classifySection(title), items: [] }
      continue
    }

    const itemMatch = line.match(/^\*\s+(.+)$/)
    if (itemMatch && current) {
      current.items.push(itemMatch[1].trim())
    }
  }

  if (current && current.items.length > 0) parsed.push(current)
  return parsed
}

async function loadChangelog(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  sections.value = []
  version.value = ''

  try {
    const response = await fetchCurrentChangelog()
    version.value = response.version
    sections.value = parseChangelogMarkdown(response.markdown)
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load release notes')
  } finally {
    isLoading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) void loadChangelog()
  },
  { immediate: true },
)
</script>

<template>
  <Dialog :open="open" @update:open="(value: boolean) => !value && emit('close')">
    <DialogContent class="max-h-[90vh] max-w-lg rounded-lg sm:max-w-2xl">
      <DialogHeader>
        <DialogDescription class="text-xs uppercase tracking-wide">Updated to</DialogDescription>
        <DialogTitle>{{ version || 'New version' }}</DialogTitle>
      </DialogHeader>

      <ScrollArea class="max-h-[50vh] pr-3">
        <p v-if="isLoading" class="text-sm text-muted-foreground">Loading release notes…</p>

        <p v-else-if="errorMessage" class="text-sm text-destructive" role="alert">
          {{ errorMessage }}
        </p>

        <div v-else class="space-y-4">
          <section v-for="section in sections" :key="section.title" class="space-y-2">
            <h3
              class="text-xs font-semibold uppercase tracking-wide"
              :class="sectionToneClass[section.kind]"
            >
              {{ section.title }}
            </h3>
            <ul class="space-y-1.5 text-sm text-muted-foreground">
              <li
                v-for="(item, index) in section.items"
                :key="`${section.title}-${index}`"
                class="flex gap-2"
              >
                <span
                  class="mt-2 size-1 shrink-0 rounded-full bg-border"
                  :class="{
                    'bg-destructive': section.kind === 'breaking',
                    'bg-primary': section.kind === 'features',
                    'bg-success': section.kind === 'fixes',
                  }"
                  aria-hidden="true"
                />
                <span>{{ item }}</span>
              </li>
            </ul>
          </section>

          <p v-if="sections.length === 0" class="text-sm text-muted-foreground">
            No categorized release notes were found for this version.
          </p>
        </div>
      </ScrollArea>

      <DialogFooter>
        <Button variant="secondary" class="rounded-lg" @click="emit('close')">Close</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
