<script setup lang="ts">
import { ref, watch } from 'vue'
import { Sparkles, AlertTriangle, Wrench } from '@lucide/vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchCurrentChangelog } from '@/api/updater'
import { extractApiError } from '@/lib/errors'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

type SectionKind = 'breaking' | 'features' | 'fixes' | 'other'

interface ChangelogSection {
  title: string
  kind: SectionKind
  items: string[]
}

const isLoading = ref(false)
const errorMessage = ref('')
const version = ref('')
const sections = ref<ChangelogSection[]>([])

const kindMeta: Record<SectionKind, { icon: typeof Sparkles; class: string }> = {
  breaking: { icon: AlertTriangle, class: 'text-destructive' },
  features: { icon: Sparkles, class: 'text-primary' },
  fixes: { icon: Wrench, class: 'text-success' },
  other: { icon: Sparkles, class: 'text-muted-foreground' },
}

function classify(title: string): SectionKind {
  const normalized = title.trim().toLowerCase()
  if (normalized.includes('breaking')) return 'breaking'
  if (normalized.includes('feature')) return 'features'
  if (normalized.includes('bug') || normalized.includes('fix')) return 'fixes'
  return 'other'
}

function parseChangelog(markdown: string): ChangelogSection[] {
  const parsed: ChangelogSection[] = []
  let current: ChangelogSection | null = null

  for (const rawLine of markdown.split('\n')) {
    const line = rawLine.trimEnd()
    const heading = line.match(/^###\s+(.+)$/)
    if (heading) {
      if (current && current.items.length > 0) {
        parsed.push(current)
      }
      const title = heading[1].trim()
      current = { title, kind: classify(title), items: [] }
      continue
    }
    const item = line.match(/^\*\s+(.+)$/)
    if (item && current) {
      current.items.push(item[1].trim())
    }
  }
  if (current && current.items.length > 0) {
    parsed.push(current)
  }
  return parsed
}

async function load(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  sections.value = []
  version.value = ''
  try {
    const response = await fetchCurrentChangelog()
    version.value = response.version
    sections.value = parseChangelog(response.markdown)
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Failed to load release notes')
  } finally {
    isLoading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      void load()
    }
  },
  { immediate: true },
)

function onOpenChange(next: boolean): void {
  if (!next) {
    emit('close')
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Sparkles class="size-5 text-primary" />
          What's new
        </DialogTitle>
        <DialogDescription>
          Arx CA was updated to
          <span class="font-medium text-foreground">{{ version || 'a new version' }}</span
          >.
        </DialogDescription>
      </DialogHeader>

      <div class="max-h-[55vh] space-y-5 overflow-y-auto pr-1">
        <template v-if="isLoading">
          <Skeleton class="h-4 w-32" />
          <Skeleton class="h-3 w-full" />
          <Skeleton class="h-3 w-5/6" />
          <Skeleton class="h-3 w-2/3" />
        </template>

        <p v-else-if="errorMessage" class="text-sm text-destructive" role="alert">
          {{ errorMessage }}
        </p>

        <template v-else>
          <section v-for="section in sections" :key="section.title" class="space-y-2">
            <h3
              class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide"
              :class="kindMeta[section.kind].class"
            >
              <component :is="kindMeta[section.kind].icon" class="size-3.5" />
              {{ section.title }}
            </h3>
            <ul class="space-y-1.5 text-sm text-muted-foreground">
              <li
                v-for="(entry, index) in section.items"
                :key="`${section.title}-${index}`"
                class="flex gap-2"
              >
                <span
                  class="mt-1.5 size-1.5 shrink-0 rounded-full"
                  :class="{
                    'bg-destructive': section.kind === 'breaking',
                    'bg-primary': section.kind === 'features',
                    'bg-success': section.kind === 'fixes',
                    'bg-border': section.kind === 'other',
                  }"
                  aria-hidden="true"
                />
                <span>{{ entry }}</span>
              </li>
            </ul>
          </section>

          <p v-if="sections.length === 0" class="text-sm text-muted-foreground">
            No categorized release notes were found for this version.
          </p>
        </template>
      </div>

      <DialogFooter>
        <Button @click="emit('close')">Got it</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
