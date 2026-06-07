<script setup lang="ts">
import Bell from 'lucide-vue-next/dist/esm/icons/bell.js'
import Check from 'lucide-vue-next/dist/esm/icons/check.js'
import OctagonAlert from 'lucide-vue-next/dist/esm/icons/octagon-alert.js'
import Trash2 from 'lucide-vue-next/dist/esm/icons/trash-2.js'
import X from 'lucide-vue-next/dist/esm/icons/x.js'
import { computed, ref, watch } from 'vue'
import { useNotificationLayout } from '../composables/useNotificationLayout'
import { useNotifications } from '../composables/useNotifications'
import { formatDateTime } from '../utils/format'
import type { NotificationEntry } from '../types/api'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const { layoutStyle } = useNotificationLayout()

const {
  persistentNotifications,
  loadPersistentNotifications,
  markAsRead,
  markAllAsRead,
  deletePersistent,
  archiveAllPersistent,
} = useNotifications()

const busyId = ref<string | null>(null)
const markingAll = ref(false)
const archivingAll = ref(false)

const isDrawer = computed(() => layoutStyle.value === 'drawer')
const isOverlay = computed(() => layoutStyle.value === 'overlay')

const panelTransitionName = computed(() => (isDrawer.value ? 'drawer-panel' : 'overlay-panel'))

const backdropClass = computed(() =>
  isOverlay.value ? 'bg-transparent' : 'ui-overlay',
)

const panelClass = computed(() => {
  const panelBase =
    'ui-surface-muted flex flex-col text-[color:var(--text-primary)] shadow-2xl transition-all duration-300'

  if (isDrawer.value) {
    return `${panelBase} fixed top-0 right-0 z-50 h-full w-96 rounded-none border-y-0 border-r-0`
  }

  return `${panelBase} fixed top-20 right-4 z-50 w-96 max-h-[70vh]`
})

const sortedNotifications = computed(() =>
  [...persistentNotifications.value].sort(
    (left, right) => new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime(),
  ),
)

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      void loadPersistentNotifications()
    }
  },
)

function itemClass(item: NotificationEntry): string {
  const base = 'ui-inset transition-all duration-300 hover:brightness-105'

  if (!item.is_read) {
    return item.level === 'critical'
      ? `border-red-500 ${base} dark:border-red-500`
      : `border-sky-500 ${base} dark:border-sky-500`
  }
  return base
}

function levelIconClass(level: NotificationEntry['level']): string {
  return level === 'critical'
    ? 'text-red-500 dark:text-red-400'
    : 'text-sky-500 dark:text-sky-400'
}

async function onMarkAsRead(id: string): Promise<void> {
  if (busyId.value) {
    return
  }
  busyId.value = id
  try {
    await markAsRead(id)
  } finally {
    busyId.value = null
  }
}

async function onMarkAllAsRead(): Promise<void> {
  if (markingAll.value) {
    return
  }
  markingAll.value = true
  try {
    await markAllAsRead()
  } finally {
    markingAll.value = false
  }
}

async function onDelete(id: string): Promise<void> {
  if (busyId.value) {
    return
  }
  busyId.value = id
  try {
    await deletePersistent(id)
  } finally {
    busyId.value = null
  }
}

async function onArchiveAll(): Promise<void> {
  if (archivingAll.value || sortedNotifications.value.length === 0) {
    return
  }

  const confirmed = window.confirm('Clear all notifications from view?')
  if (!confirmed) {
    return
  }

  archivingAll.value = true
  try {
    await archiveAllPersistent()
  } finally {
    archivingAll.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer-backdrop">
      <div
        v-if="open"
        class="fixed inset-0 z-40 transition-all duration-300"
        :class="backdropClass"
        role="presentation"
        aria-hidden="true"
        @click="emit('close')"
      />
    </Transition>

    <Transition :name="panelTransitionName">
      <aside
        v-if="open"
        :class="panelClass"
        role="dialog"
        aria-label="Notification history"
        aria-modal="true"
      >
        <header class="ui-border-b ui-chrome-bar flex shrink-0 items-center justify-between gap-3 px-4 py-3">
          <div class="min-w-0">
            <h2 class="text-sm font-semibold ui-text-primary">Notifications</h2>
            <p class="text-[11px] ui-text-muted">Persistent audit event history</p>
          </div>
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="ui-btn-secondary px-2 py-1 text-[11px]"
              :disabled="markingAll || sortedNotifications.every((item) => item.is_read)"
              @click="onMarkAllAsRead"
            >
              Mark all as read
            </button>
            <button
              type="button"
              class="ui-btn-secondary inline-flex h-8 w-8 items-center justify-center p-0"
              aria-label="Clear all notifications from view"
              :disabled="archivingAll || sortedNotifications.length === 0"
              @click="onArchiveAll"
            >
              <Trash2 class="h-4 w-4 ui-text-secondary" aria-hidden="true" />
            </button>
            <button
              type="button"
              class="ui-btn-secondary inline-flex h-8 w-8 items-center justify-center p-0"
              aria-label="Close notification drawer"
              @click="emit('close')"
            >
              <X class="h-4 w-4 ui-text-secondary" aria-hidden="true" />
            </button>
          </div>
        </header>

        <div
          class="custom-scrollbar flex-1 overflow-y-auto px-3 py-3 ui-text-primary"
          :class="isOverlay ? 'max-h-[calc(70vh-4rem)]' : ''"
        >
          <p
            v-if="sortedNotifications.length === 0"
            class="ui-inset border-dashed px-4 py-8 text-center text-xs ui-text-muted"
          >
            No notifications yet. Live audit events will appear here.
          </p>

          <ul v-else class="flex flex-col gap-2">
            <li
              v-for="item in sortedNotifications"
              :key="item.id"
              class="rounded-[var(--radius-control)] border px-3 py-2.5"
              :class="itemClass(item)"
            >
              <div class="flex items-start gap-2">
                <component
                  :is="item.level === 'critical' ? OctagonAlert : Bell"
                  class="mt-0.5 h-4 w-4 shrink-0"
                  :class="levelIconClass(item.level)"
                  aria-hidden="true"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-start justify-between gap-2">
                    <p
                      class="text-xs font-medium ui-text-primary"
                      :class="{ 'font-semibold': !item.is_read }"
                    >
                      {{ item.message }}
                    </p>
                    <span
                      v-if="!item.is_read"
                      class="mt-0.5 h-2 w-2 shrink-0 rounded-full bg-sky-400 dark:bg-sky-500"
                      aria-label="Unread"
                    />
                  </div>
                  <p class="mt-0.5 font-mono text-[10px] ui-text-muted">{{ item.action }}</p>
                  <p class="mt-1 text-[10px] ui-text-muted">{{ formatDateTime(item.timestamp) }}</p>
                </div>
              </div>

              <div class="mt-2 flex items-center justify-end gap-2">
                <button
                  v-if="!item.is_read"
                  type="button"
                  class="ui-btn-secondary inline-flex items-center gap-1 px-2 py-1 text-[10px]"
                  :disabled="busyId === item.id"
                  @click="onMarkAsRead(item.id)"
                >
                  <Check class="h-3 w-3" aria-hidden="true" />
                  Mark as read
                </button>
                <button
                  type="button"
                  class="ui-btn-secondary inline-flex h-7 w-7 items-center justify-center p-0"
                  :aria-label="`Delete notification ${item.action}`"
                  :disabled="busyId === item.id"
                  @click="onDelete(item.id)"
                >
                  <Trash2 class="h-3.5 w-3.5 ui-text-secondary" aria-hidden="true" />
                </button>
              </div>
            </li>
          </ul>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-backdrop-enter-active,
.drawer-backdrop-leave-active {
  transition: opacity 0.3s ease;
}

.drawer-backdrop-enter-from,
.drawer-backdrop-leave-to {
  opacity: 0;
}

.drawer-panel-enter-active,
.drawer-panel-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.drawer-panel-enter-from,
.drawer-panel-leave-to {
  transform: translateX(100%);
  opacity: 0.85;
}

.overlay-panel-enter-active,
.overlay-panel-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.overlay-panel-enter-from,
.overlay-panel-leave-to {
  transform: translateY(-8px) scale(0.97);
  opacity: 0;
}
</style>
