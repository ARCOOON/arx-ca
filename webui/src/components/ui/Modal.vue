<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import X from 'lucide-vue-next/dist/esm/icons/x.js'
import Button from './Button.vue'

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
      class="ui-overlay fixed inset-0 z-50 flex items-center justify-center overflow-y-auto p-2 sm:p-4"
      role="presentation"
      @click="handleBackdropClick"
    >
      <div
        class="ui-elevated ui-dialog flex max-h-[90vh] w-full max-w-[95vw] flex-col"
        :class="wide ? 'sm:max-w-2xl' : 'sm:max-w-lg'"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        @click.stop
      >
        <header class="ui-border-b flex shrink-0 items-center justify-between gap-3 px-4 py-3">
          <h2 class="min-w-0 text-sm font-semibold ui-text-primary">{{ title }}</h2>
          <Button variant="secondary" size="icon" aria-label="Close dialog" @click="emit('close')">
            <X class="h-4 w-4" aria-hidden="true" />
          </Button>
        </header>

        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          <slot />
        </div>

        <footer v-if="$slots.footer" class="ui-border-t ui-modal-footer shrink-0 px-4 py-3">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>
