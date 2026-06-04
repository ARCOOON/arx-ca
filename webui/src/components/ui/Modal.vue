<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { X } from 'lucide-vue-next'

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
      class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-zinc-950/80 px-4 py-10"
      role="presentation"
      @click="handleBackdropClick"
    >
      <div
        class="w-full border border-zinc-700 bg-zinc-900"
        :class="wide ? 'max-w-2xl' : 'max-w-lg'"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        @click.stop
      >
        <header class="flex items-center justify-between border-b border-zinc-800 px-4 py-3">
          <h2 class="text-sm font-semibold text-zinc-50">{{ title }}</h2>
          <button
            type="button"
            class="inline-flex h-7 w-7 items-center justify-center border border-zinc-700 text-zinc-400 transition hover:border-zinc-600 hover:text-zinc-100"
            aria-label="Close dialog"
            @click="emit('close')"
          >
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </header>

        <div class="px-4 py-4">
          <slot />
        </div>

        <footer v-if="$slots.footer" class="flex items-center justify-end gap-2 border-t border-zinc-800 px-4 py-3">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>
