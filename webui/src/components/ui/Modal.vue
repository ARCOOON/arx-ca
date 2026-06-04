<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import X from 'lucide-vue-next/dist/esm/icons/x.js'

const props = defineProps<{
  open: boolean
  title: string
  wide?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

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
      class="ui-overlay fixed inset-0 z-50 flex items-start justify-center overflow-y-auto px-4 py-10"
      role="presentation"
      @click="handleBackdropClick"
    >
      <div
        class="ui-elevated w-full"
        :class="wide ? 'max-w-2xl' : 'max-w-lg'"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        @click.stop
      >
        <header class="ui-border-b flex items-center justify-between px-4 py-3">
          <h2 class="text-sm font-semibold ui-text-primary">{{ title }}</h2>
          <button
            type="button"
            class="ui-btn-secondary inline-flex h-7 w-7 items-center justify-center p-0"
            aria-label="Close dialog"
            @click="emit('close')"
          >
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </header>

        <div class="px-4 py-4">
          <slot />
        </div>

        <footer v-if="$slots.footer" class="ui-border-t flex items-center justify-end gap-2 px-4 py-3">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>
