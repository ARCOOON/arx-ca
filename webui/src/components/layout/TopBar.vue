<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Bell, LogOut, Menu, Moon, Sun, UserRound } from 'lucide-vue-next'
import NotificationDrawer from '../NotificationDrawer.vue'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { useNotifications } from '../../composables/useNotifications'
import { useAuthStore } from '../../store/auth'
import { applyTheme, resolveInitialTheme, type ThemeMode } from '../../composables/useTheme'

const emit = defineEmits<{
  logout: []
  'toggle-mobile-nav': []
}>()

const route = useRoute()
const authStore = useAuthStore()
const { unreadCount } = useNotifications()
const theme = ref<ThemeMode>(resolveInitialTheme())
const drawerOpen = ref(false)

const pageTitle = computed(() => {
  const title = route.meta.title
  return typeof title === 'string' ? title : 'Console'
})

const pageSubtitle = computed(() => {
  const subtitle = route.meta.subtitle
  return typeof subtitle === 'string' ? subtitle : ''
})

const roleLabel = computed(() => {
  if (authStore.roles.length > 0) {
    return authStore.roles.join(', ')
  }
  return 'Administrator'
})

const isDark = computed(() => theme.value === 'dark')

const badgeLabel = computed(() => {
  const count = unreadCount.value
  if (count <= 0) {
    return ''
  }
  return count > 99 ? '99+' : String(count)
})

function onThemeToggle(checked: boolean): void {
  theme.value = checked ? 'dark' : 'light'
  applyTheme(theme.value)
}

function toggleDrawer(): void {
  drawerOpen.value = !drawerOpen.value
}

function closeDrawer(): void {
  drawerOpen.value = false
}
</script>

<template>
  <header class="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-card px-3 py-3 sm:gap-3 sm:px-5">
    <div class="flex min-w-0 flex-1 items-center gap-2 sm:gap-3">
      <Button
        type="button"
        variant="outline"
        size="icon"
        class="md:hidden"
        aria-label="Open navigation menu"
        @click="emit('toggle-mobile-nav')"
      >
        <Menu class="size-4" aria-hidden="true" />
      </Button>

      <div class="min-w-0">
        <h1 class="truncate font-heading text-base font-semibold">{{ pageTitle }}</h1>
        <p v-if="pageSubtitle" class="truncate text-xs text-muted-foreground">{{ pageSubtitle }}</p>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2 sm:gap-3">
      <div class="hidden items-center gap-2 sm:flex">
        <span class="sr-only">Color theme</span>
        <Sun v-if="isDark" class="size-3.5 text-muted-foreground" aria-hidden="true" />
        <Moon v-else class="size-3.5 text-muted-foreground" aria-hidden="true" />
        <Switch
          :checked="isDark"
          :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @update:checked="onThemeToggle"
        />
      </div>

      <Button
        type="button"
        variant="outline"
        size="icon"
        class="relative"
        :aria-label="unreadCount > 0 ? `${unreadCount} unread notifications` : 'Open notifications'"
        @click="toggleDrawer"
      >
        <Bell class="size-4" aria-hidden="true" />
        <span
          v-if="unreadCount > 0"
          class="absolute -top-1 -right-1 inline-flex min-h-[18px] min-w-[18px] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold leading-none text-destructive-foreground"
          aria-hidden="true"
        >
          {{ badgeLabel }}
        </span>
      </Button>

      <div
        class="hidden min-h-10 items-center gap-2 rounded-md border border-input bg-background px-3 py-1.5 sm:flex"
        aria-label="Signed in user"
      >
        <UserRound class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
        <div class="flex min-w-0 flex-col items-end justify-center text-right">
          <span class="whitespace-nowrap text-[10px] font-medium uppercase leading-tight tracking-wide text-muted-foreground">
            Signed in
          </span>
          <span class="mt-0.5 max-w-48 truncate whitespace-nowrap text-xs leading-tight text-foreground/80">
            {{ roleLabel }}
          </span>
        </div>
      </div>

      <Button type="button" variant="outline" size="sm" @click="emit('logout')">
        <LogOut class="size-3.5" aria-hidden="true" />
        Logout
      </Button>
    </div>
  </header>

  <NotificationDrawer :open="drawerOpen" @close="closeDrawer" />
</template>
