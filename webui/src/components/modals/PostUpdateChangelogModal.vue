<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from '@/components/ui/Dialog.vue'
import Button from '@/components/ui/Button.vue'
import Spinner from '@/components/ui/Spinner.vue'
import { fetchCurrentChangelog } from '@/api/updater'
import type { UpdaterChangelogResponse } from '@/types/api'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const loading = ref(false)
const error = ref<string | null>(null)
const changelog = ref<UpdaterChangelogResponse | null>(null)

watch(
  () => props.open,
  async (opened) => {
    if (!opened) return
    loading.value = true
    error.value = null
    try {
      changelog.value = await fetchCurrentChangelog()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load changelog'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

function renderMarkdown(md: string): string {
  // Minimal markdown rendering: headers, bold, code, bullet lists
  return md
    .replace(/^### (.+)$/gm, '<h3 class="text-sm font-semibold mt-4 mb-1 text-foreground">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="text-base font-semibold mt-5 mb-2 text-foreground">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="text-lg font-bold mt-6 mb-2 text-foreground">$1</h1>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="rounded bg-muted px-1 py-0.5 font-mono text-xs text-foreground">$1</code>')
    .replace(/^- (.+)$/gm, '<li class="ml-4 list-disc text-sm">$1</li>')
    .replace(/\n\n/g, '</p><p class="my-2">')
    .replace(/^(?!<[hlpc])/gm, '<p class="text-sm text-foreground-muted leading-relaxed">')
}
</script>

<template>
  <Dialog
    :open="open"
    title="What's New"
    max-width="max-w-2xl"
    @close="emit('close')"
  >
    <template #header>
      <div>
        <div class="flex items-center gap-2">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10">
            <svg class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div>
            <h2 class="text-base font-semibold text-foreground leading-none">What's New</h2>
            <p v-if="changelog" class="mt-0.5 text-xs text-foreground-muted">
              ARX CA {{ changelog.version }}
            </p>
          </div>
        </div>
      </div>
    </template>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-16">
      <Spinner size="lg" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Changelog content -->
    <div
      v-else-if="changelog"
      class="prose-sm prose-neutral max-w-none"
      v-html="renderMarkdown(changelog.markdown)"
    />

    <template #footer>
      <Button @click="emit('close')">Close</Button>
    </template>
  </Dialog>
</template>
