<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchCurrentChangelog } from '../../api/updater'
import { extractApiError } from '../../utils/errors'

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
  breaking: 'text-[var(--status-revoked)]',
  features: 'text-[var(--accent-text)]',
  fixes: 'text-[var(--status-valid)]',
  other: 'ui-text-secondary',
}))

function classifySection(title: string): ChangelogSection['kind'] {
  const normalized = title.trim().toLowerCase()
  if (normalized.includes('breaking')) {
    return 'breaking'
  }
  if (normalized.includes('feature')) {
    return 'features'
  }
  if (normalized.includes('bug') || normalized.includes('fix')) {
    return 'fixes'
  }
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
      if (current && current.items.length > 0) {
        parsed.push(current)
      }
      const title = headingMatch[1].trim()
      current = {
        title,
        kind: classifySection(title),
        items: [],
      }
      continue
    }

    const itemMatch = line.match(/^\*\s+(.+)$/)
    if (itemMatch && current) {
      current.items.push(itemMatch[1].trim())
    }
  }

  if (current && current.items.length > 0) {
    parsed.push(current)
  }

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
    if (open) {
      void loadChangelog()
    }
  },
  { immediate: true },
)

function handleBackdropClick(event: MouseEvent): void {
  if (event.target === event.currentTarget) {
    emit('close')
  }
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && props.open) {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="ui-overlay fixed inset-0 z-50 flex items-center justify-center overflow-y-auto p-2 sm:p-4"
      role="presentation"
      @click="handleBackdropClick"
    >
      <div
        class="ui-elevated ui-dialog flex max-h-[90vh] w-full max-w-[95vw] flex-col sm:max-w-2xl"
        role="dialog"
        aria-modal="true"
        aria-label="Release notes"
        @click.stop
      >
        <header class="ui-border-b flex shrink-0 items-start justify-between gap-3 px-4 py-3">
          <div class="min-w-0">
            <p class="text-[10px] uppercase tracking-wide ui-text-muted">Updated to</p>
            <h2 class="text-sm font-semibold ui-text-primary">
              {{ version || 'New version' }}
            </h2>
          </div>
          <button
            type="button"
            class="ui-btn-secondary inline-flex shrink-0 items-center justify-center rounded-[var(--radius-control)] px-3 py-1.5 text-xs"
            @click="emit('close')"
          >
            Close
          </button>
        </header>

        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          <p v-if="isLoading" class="text-sm ui-text-muted">Loading release notes…</p>

          <p v-else-if="errorMessage" class="ui-alert-error text-sm" role="alert">
            {{ errorMessage }}
          </p>

          <div v-else class="space-y-4">
            <section
              v-for="section in sections"
              :key="section.title"
              class="space-y-2"
            >
              <h3
                class="text-xs font-semibold uppercase tracking-wide"
                :class="sectionToneClass[section.kind]"
              >
                {{ section.title }}
              </h3>
              <ul class="space-y-1.5 text-sm ui-text-secondary">
                <li
                  v-for="(item, index) in section.items"
                  :key="`${section.title}-${index}`"
                  class="flex gap-2"
                >
                  <span
                    class="mt-2 h-1 w-1 shrink-0 rounded-full"
                    :class="{
                      'bg-[var(--status-revoked)]': section.kind === 'breaking',
                      'bg-[var(--accent-border)]': section.kind === 'features',
                      'bg-[var(--status-valid)]': section.kind === 'fixes',
                      'bg-[var(--border-subtle)]': section.kind === 'other',
                    }"
                    aria-hidden="true"
                  />
                  <span>{{ item }}</span>
                </li>
              </ul>
            </section>

            <p v-if="sections.length === 0" class="text-sm ui-text-muted">
              No categorized release notes were found for this version.
            </p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
