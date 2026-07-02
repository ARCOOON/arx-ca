<script setup lang="ts">
import Bell from 'lucide-vue-next/dist/esm/icons/bell.js'
import CircleCheck from 'lucide-vue-next/dist/esm/icons/circle-check.js'
import OctagonAlert from 'lucide-vue-next/dist/esm/icons/octagon-alert.js'
import X from 'lucide-vue-next/dist/esm/icons/x.js'
import { useNotifications } from '../composables/useNotifications'
import { formatDateTime } from '../utils/format'

const { notifications, remove, clearAll } = useNotifications()

function cardClass(tone: 'critical' | 'info' | 'success'): string {
  switch (tone) {
    case 'critical':
      return 'border-red-500/60 bg-card'
    case 'success':
      return 'border-emerald-500/50 bg-card'
    default:
      return 'border-sky-500/50 bg-card'
  }
}

function iconClass(tone: 'critical' | 'info' | 'success'): string {
  switch (tone) {
    case 'critical':
      return 'text-red-400'
    case 'success':
      return 'text-emerald-400'
    default:
      return 'text-sky-400'
  }
}
</script>

<template>
  <div
    class="custom-scrollbar pointer-events-auto fixed top-4 right-4 z-50 flex max-h-[50vh] w-80 flex-col gap-2 overflow-y-auto"
    aria-live="polite"
    aria-atomic="false"
  >
    <button
      v-if="notifications.length > 1"
      type="button"
      class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 self-end px-2 py-1 text-[10px]"
      @click="clearAll"
    >
      Clear All
    </button>

    <TransitionGroup name="notification" tag="div" class="flex flex-col gap-2">
      <article
        v-for="item in notifications"
        :key="item.id"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5 shadow-lg"
        :class="cardClass(item.tone)"
        role="status"
      >
        <component
          :is="item.tone === 'critical' ? OctagonAlert : item.tone === 'success' ? CircleCheck : Bell"
          class="mt-0.5 h-4 w-4 shrink-0"
          :class="iconClass(item.tone)"
         
        />
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-foreground">{{ item.message }}</p>
          <p class="mt-0.5 font-mono text-[10px] text-muted-foreground">{{ item.action }}</p>
          <p class="mt-1 text-[10px] text-muted-foreground">{{ formatDateTime(item.timestamp) }}</p>
        </div>
        <button
          type="button"
          class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 inline-flex h-6 w-6 shrink-0 items-center justify-center p-0"
          aria-label="Dismiss notification"
          @click="remove(item.id)"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </article>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.notification-enter-active,
.notification-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.notification-enter-from,
.notification-leave-to {
  opacity: 0;
  transform: translateX(12px);
}

.notification-move {
  transition: transform 0.2s ease;
}
</style>
