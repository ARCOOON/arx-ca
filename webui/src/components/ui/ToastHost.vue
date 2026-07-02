<script setup lang="ts">
import { useToast } from '../../composables/useToast'
import X from 'lucide-vue-next/dist/esm/icons/x.js'

const { toasts, dismissToast } = useToast()
</script>

<template>
  <div
    class="pointer-events-none fixed bottom-3 right-3 z-[60] flex w-full max-w-sm flex-col gap-2 px-3 sm:bottom-4 sm:right-4 sm:px-0"
    aria-live="polite"
    aria-atomic="true"
  >
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="pointer-events-auto flex items-start gap-2 rounded-lg border px-3 py-2.5 text-xs"
      :class="toast.tone === 'success' ? 'ui-alert-success' : 'ui-alert-error'"
      role="status"
    >
      <p class="min-w-0 flex-1">{{ toast.message }}</p>
      <button
        type="button"
        class="ui-btn-secondary inline-flex h-6 w-6 shrink-0 items-center justify-center p-0"
        aria-label="Dismiss notification"
        @click="dismissToast(toast.id)"
      >
        <X class="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>
