<script setup lang="ts">
import { useToast } from '@/composables/useToast'
import { cn } from '@/utils/cn'

const { toasts, dismiss } = useToast()

const variantStyles: Record<string, string> = {
  default:     'border-border bg-card text-card-foreground',
  success:     'border-success/30 bg-success/10 text-success',
  error:       'border-destructive/30 bg-destructive/10 text-destructive',
  warning:     'border-warning/30 bg-warning/10 text-warning-foreground',
  info:        'border-info/30 bg-info/10 text-info',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[9999] flex flex-col gap-2 max-w-sm w-full pointer-events-none">
      <TransitionGroup
        enter-active-class="transition-all duration-200"
        enter-from-class="opacity-0 translate-y-2 scale-95"
        enter-to-class="opacity-100 translate-y-0 scale-100"
        leave-active-class="transition-all duration-150"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0 translate-y-2 scale-95"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="
            cn(
              'pointer-events-auto flex items-start gap-3 rounded-lg border px-4 py-3 shadow-md text-sm',
              variantStyles[toast.variant] ?? variantStyles.default,
            )
          "
        >
          <span class="flex-1 break-words">{{ toast.message }}</span>
          <button
            type="button"
            class="ml-auto shrink-0 opacity-60 hover:opacity-100 transition-opacity"
            @click="dismiss(toast.id)"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M18 6 6 18M6 6l12 12" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
