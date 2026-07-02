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
      return 'border-destructive/40 bg-card'
    case 'success':
      return 'border-primary/30 bg-card'
    default:
      return 'border-border bg-card'
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
      class="ui-btn-secondary self-end px-2 py-1 text-[10px]"
      @click="clearAll"
    >
      Clear All
    </button>

    <TransitionGroup name="notification" tag="div" class="flex flex-col gap-2">
      <article
        v-for="item in notifications"
        :key="item.id"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5"
        :class="cardClass(item.tone)"
        role="status"
      >
        <component
          :is="item.tone === 'critical' ? OctagonAlert : item.tone === 'success' ? CircleCheck : Bell"
          class="mt-0.5 h-4 w-4 shrink-0"
          :class="iconClass(item.tone)"
          aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium ui-text-primary">{{ item.message }}</p>
          <p class="mt-0.5 font-mono text-[10px] ui-text-muted">{{ item.action }}</p>
          <p class="mt-1 text-[10px] ui-text-muted">{{ formatDateTime(item.timestamp) }}</p>
        </div>
        <button
          type="button"
          class="ui-btn-secondary inline-flex h-6 w-6 shrink-0 items-center justify-center p-0"
          aria-label="Dismiss notification"
          @click="remove(item.id)"
        >
          <X class="h-3.5 w-3.5" aria-hidden="true" />
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
