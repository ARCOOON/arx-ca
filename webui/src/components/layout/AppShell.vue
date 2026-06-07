<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useNotifications } from '../../composables/useNotifications'
import { useAuthStore } from '../../store/auth'
import NotificationToaster from '../NotificationToaster.vue'
import ToastHost from '../ui/ToastHost.vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const SIDEBAR_COLLAPSED_KEY = 'arx_sidebar_collapsed'

function readCollapsedPreference(): boolean {
  return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
}

const authStore = useAuthStore()
const route = useRoute()
const { connect, disconnect } = useNotifications()
const sidebarCollapsed = ref(readCollapsedPreference())
const mobileNavOpen = ref(false)

function onSidebarCollapsedChange(value: boolean): void {
  sidebarCollapsed.value = value
  localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(value))
}

function toggleMobileNav(): void {
  mobileNavOpen.value = !mobileNavOpen.value
}

function closeMobileNav(): void {
  mobileNavOpen.value = false
}

function handleLogout(): void {
  disconnect()
  void authStore.logout()
}

onMounted(() => {
  if (authStore.isAuthenticated) {
    connect()
  }
})

onUnmounted(() => {
  disconnect()
})

watch(
  () => authStore.isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      connect()
      return
    }
    disconnect()
  },
)

watch(
  () => route.path,
  () => {
    mobileNavOpen.value = false
  },
)
</script>

<template>
  <div class="ui-shell flex h-screen w-screen overflow-hidden">
    <div
      v-if="mobileNavOpen"
      class="ui-mobile-nav-backdrop fixed inset-0 z-30 md:hidden"
      role="presentation"
      aria-hidden="true"
      @click="closeMobileNav"
    />

    <SideNav
      :collapsed="sidebarCollapsed"
      :mobile-open="mobileNavOpen"
      @update:collapsed="onSidebarCollapsedChange"
      @close-mobile="closeMobileNav"
    />

    <div class="flex h-full min-w-0 flex-1 flex-col overflow-hidden">
      <div class="shrink-0">
        <TopBar @toggle-mobile-nav="toggleMobileNav" @logout="handleLogout" />
      </div>

      <main class="custom-scrollbar flex-1 overflow-y-auto p-4 md:p-6">
        <RouterView />
      </main>
    </div>

    <NotificationToaster />
    <ToastHost />
  </div>
</template>
