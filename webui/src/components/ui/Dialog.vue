<script setup lang="ts">
import { cn } from '@/utils/cn'

defineProps<{
  open: boolean
  title?: string
  description?: string
  maxWidth?: string
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        aria-modal="true"
        role="dialog"
      >
        <!-- Overlay -->
        <div
          class="absolute inset-0 bg-black/40 backdrop-blur-sm"
          @click="emit('close')"
        />

        <!-- Panel -->
        <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 scale-95 translate-y-1"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition-all duration-150"
          leave-from-class="opacity-100 scale-100 translate-y-0"
          leave-to-class="opacity-0 scale-95 translate-y-1"
        >
          <div
            v-if="open"
            :class="
              cn(
                'relative z-10 flex flex-col rounded-xl border border-border bg-card shadow-xl',
                'w-full',
                maxWidth ?? 'max-w-lg',
              )
            "
          >
            <!-- Header -->
            <div v-if="title || $slots.header" class="flex items-start justify-between gap-4 px-6 pt-6 pb-4">
              <slot name="header">
                <div>
                  <h2 class="text-base font-semibold text-foreground leading-none">{{ title }}</h2>
                  <p v-if="description" class="mt-1.5 text-sm text-foreground-muted">{{ description }}</p>
                </div>
              </slot>
              <button
                type="button"
                class="ml-auto flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-foreground-subtle hover:text-foreground hover:bg-accent transition-colors"
                @click="emit('close')"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 6 6 18M6 6l12 12" stroke-linecap="round" />
                </svg>
              </button>
            </div>

            <!-- Body -->
            <div class="flex-1 overflow-y-auto px-6 pb-6 max-h-[70vh]">
              <slot />
            </div>

            <!-- Footer -->
            <div
              v-if="$slots.footer"
              class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border"
            >
              <slot name="footer" />
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
