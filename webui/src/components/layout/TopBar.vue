<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme, type ThemeMode } from '@/composables/useTheme'
import { useNotificationsStore } from '@/store/notifications'
import { useAuthStore } from '@/store/auth'
import { markAllRead, archiveAll } from '@/api/notifications'
import { logout as apiLogout } from '@/api/auth'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const { mode, setMode } = useTheme()
const notifStore = useNotificationsStore()
const authStore = useAuthStore()
const toast = useToast()

const notifOpen = ref(false)
const userMenuOpen = ref(false)

const themes: Array<{ value: ThemeMode; label: string }> = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'auto', label: 'Auto' },
]

async function handleMarkAllRead(): Promise<void> {
  try {
    await markAllRead()
    notifStore.setNotifications(
      notifStore.items.map((n) => ({ ...n, is_read: true })),
      0,
      notifStore.total,
    )
  } catch {
    toast.error('Failed to mark notifications as read')
  }
}

async function handleArchiveAll(): Promise<void> {
  try {
    await archiveAll()
    notifStore.clearAll()
    notifOpen.value = false
  } catch {
    toast.error('Failed to archive notifications')
  }
}

async function handleLogout(): Promise<void> {
  try {
    await apiLogout()
  } finally {
    await authStore.logout()
  }
}
</script>

<template>
  <header
    class="flex items-center justify-between border-b border-border bg-surface px-5 h-12 shrink-0 z-10"
  >
    <!-- Breadcrumb -->
    <div class="flex items-center gap-1.5 text-sm min-w-0">
      <span class="text-foreground-subtle">ARX CA</span>
      <svg class="h-3.5 w-3.5 text-foreground-subtle shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="9 18 15 12 9 6"/>
      </svg>
      <span class="font-medium text-foreground truncate">{{ route.meta.title as string ?? '' }}</span>
      <span
        v-if="route.meta.subtitle"
        class="hidden sm:inline text-foreground-subtle truncate before:content-['—'] before:mx-1.5"
      >{{ route.meta.subtitle as string }}</span>
    </div>

    <!-- Right controls -->
    <div class="flex items-center gap-1">
      <!-- Theme selector -->
      <div class="flex items-center gap-0.5 rounded-lg border border-border bg-surface-raised p-0.5">
        <button
          v-for="t in themes"
          :key="t.value"
          type="button"
          :class="[
            'rounded-md px-2.5 py-1 text-xs font-medium transition-all',
            mode === t.value
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'text-foreground-muted hover:text-foreground hover:bg-accent',
          ]"
          @click="setMode(t.value)"
        >
          {{ t.label }}
        </button>
      </div>

      <!-- Notification bell -->
      <div class="relative">
        <button
          type="button"
          class="relative flex h-8 w-8 items-center justify-center rounded-md text-foreground-muted hover:text-foreground hover:bg-accent transition-colors"
          @click="notifOpen = !notifOpen"
        >
          <svg class="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
            <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
          </svg>
          <span
            v-if="notifStore.unreadCount > 0"
            class="absolute top-1.5 right-1.5 flex h-2 w-2 items-center justify-center rounded-full bg-destructive text-[9px] text-white font-bold"
          />
        </button>

        <!-- Notification dropdown -->
        <Transition
          enter-active-class="transition-all duration-150"
          enter-from-class="opacity-0 translate-y-1 scale-95"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition-all duration-100"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0 translate-y-1 scale-95"
        >
          <div
            v-if="notifOpen"
            v-click-outside="() => (notifOpen = false)"
            class="absolute right-0 top-10 z-30 w-80 rounded-lg border border-border bg-popover shadow-xl"
          >
            <div class="flex items-center justify-between px-4 py-3 border-b border-border">
              <span class="text-sm font-semibold text-foreground">Notifications</span>
              <div class="flex gap-1">
                <button
                  type="button"
                  class="text-xs text-foreground-muted hover:text-foreground transition-colors px-1"
                  @click="handleMarkAllRead"
                >
                  Mark read
                </button>
                <button
                  type="button"
                  class="text-xs text-foreground-muted hover:text-foreground transition-colors px-1"
                  @click="handleArchiveAll"
                >
                  Clear
                </button>
              </div>
            </div>
            <div class="max-h-80 overflow-y-auto divide-y divide-border">
              <div
                v-if="notifStore.items.length === 0"
                class="flex items-center justify-center py-10 text-sm text-foreground-muted"
              >
                No notifications
              </div>
              <div
                v-for="n in notifStore.items.slice(0, 20)"
                :key="n.id"
                class="flex items-start gap-3 px-4 py-3 hover:bg-accent transition-colors"
              >
                <span
                  :class="[
                    'mt-1 h-1.5 w-1.5 shrink-0 rounded-full',
                    n.level === 'critical' ? 'bg-destructive' : 'bg-info',
                    n.is_read ? 'opacity-30' : '',
                  ]"
                />
                <div class="flex-1 min-w-0">
                  <p class="text-xs text-foreground leading-snug break-words">{{ n.message }}</p>
                  <p class="mt-0.5 text-[10px] text-foreground-subtle">
                    {{ new Date(n.timestamp).toLocaleString() }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>

      <!-- User menu -->
      <div class="relative">
        <button
          type="button"
          class="flex h-8 items-center gap-1.5 rounded-md px-2 text-foreground-muted hover:text-foreground hover:bg-accent transition-colors"
          @click="userMenuOpen = !userMenuOpen"
        >
          <div class="h-5 w-5 rounded-full bg-primary/20 flex items-center justify-center">
            <svg class="h-3 w-3 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="8" r="4"/><path d="M6 20v-2a6 6 0 0 1 12 0v2"/>
            </svg>
          </div>
          <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>

        <Transition
          enter-active-class="transition-all duration-150"
          enter-from-class="opacity-0 translate-y-1 scale-95"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition-all duration-100"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0 translate-y-1 scale-95"
        >
          <div
            v-if="userMenuOpen"
            class="absolute right-0 top-10 z-30 w-44 rounded-lg border border-border bg-popover shadow-xl py-1"
          >
            <div class="px-3 py-2 border-b border-border">
              <p class="text-xs font-medium text-foreground truncate">
                {{ authStore.roles[0] ?? 'Operator' }}
              </p>
            </div>
            <button
              type="button"
              class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-accent transition-colors"
              @click="handleLogout"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16 17 21 12 16 7"/>
                <line x1="21" y1="12" x2="9" y2="12"/>
              </svg>
              Sign out
            </button>
          </div>
        </Transition>
      </div>
    </div>
  </header>
</template>
