<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import LogOut from 'lucide-vue-next/dist/esm/icons/log-out.js'
import Moon from 'lucide-vue-next/dist/esm/icons/moon.js'
import Sun from 'lucide-vue-next/dist/esm/icons/sun.js'
import UserRound from 'lucide-vue-next/dist/esm/icons/user-round.js'
import { useAuthStore } from '../../store/auth'
import { applyTheme, resolveInitialTheme, type ThemeMode, toggleTheme } from '../../composables/useTheme'

const emit = defineEmits<{
  logout: []
}>()

const route = useRoute()
const authStore = useAuthStore()
const theme = ref<ThemeMode>(resolveInitialTheme())

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

function onThemeToggle(): void {
  theme.value = toggleTheme(theme.value)
}

function setLightTheme(): void {
  theme.value = 'light'
  applyTheme('light')
}

function setDarkTheme(): void {
  theme.value = 'dark'
  applyTheme('dark')
}
</script>

<template>
  <header class="ui-border-b ui-elevated flex items-center justify-between px-5 py-3">
    <div class="min-w-0">
      <h1 class="truncate text-base font-semibold ui-text-primary">{{ pageTitle }}</h1>
      <p v-if="pageSubtitle" class="truncate text-xs ui-text-muted">{{ pageSubtitle }}</p>
    </div>

    <div class="flex items-center gap-3">
      <div class="hidden items-center gap-2 sm:flex">
        <span class="sr-only">Color theme</span>
        <Sun v-if="isDark" class="h-3.5 w-3.5 ui-text-muted" aria-hidden="true" />
        <Moon v-else class="h-3.5 w-3.5 ui-text-muted" aria-hidden="true" />
        <button
          type="button"
          class="ui-theme-toggle"
          :data-active="isDark"
          :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="onThemeToggle"
          @keydown.home.prevent="setLightTheme"
          @keydown.end.prevent="setDarkTheme"
        >
          <span class="ui-theme-toggle-thumb" />
        </button>
      </div>

      <div class="ui-topbar-control hidden sm:flex">
        <UserRound class="h-3.5 w-3.5 shrink-0 ui-text-muted" aria-hidden="true" />
        <div class="min-w-0 text-right">
          <p class="text-[10px] uppercase leading-none tracking-wide ui-text-muted">Signed in</p>
          <p class="mt-0.5 max-w-[12rem] truncate text-xs leading-none ui-text-secondary">{{ roleLabel }}</p>
        </div>
      </div>

      <button type="button" class="ui-topbar-control ui-btn-secondary inline-flex items-center gap-1.5" @click="emit('logout')">
        <LogOut class="h-3.5 w-3.5" aria-hidden="true" />
        Logout
      </button>
    </div>
  </header>
</template>
