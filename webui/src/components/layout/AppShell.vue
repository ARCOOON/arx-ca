<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useNotifications } from '../../composables/useNotifications'
import { useUpdater } from '../../composables/useUpdater'
import { useAuthStore } from '../../store/auth'
import PostUpdateChangelogModal from '../modals/PostUpdateChangelogModal.vue'
import NotificationToaster from '../NotificationToaster.vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const SIDEBAR_COLLAPSED_KEY = 'arx_sidebar_collapsed'

function readCollapsedPreference(): boolean {
  return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
}

const authStore = useAuthStore()
const route = useRoute()
const { connect, disconnect } = useNotifications()
const { triggerChangelogModal, runVersionDriftCheck, dismissChangelogModal } = useUpdater()
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
    runVersionDriftCheck()
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
      runVersionDriftCheck()
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
  <div class="bg-background flex h-screen w-screen overflow-hidden">
    <div
      v-if="mobileNavOpen"
      class="bg-background/80 fixed inset-0 z-30 md:hidden"
      role="presentation"
     
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
    <PostUpdateChangelogModal
      :open="triggerChangelogModal"
      @close="dismissChangelogModal"
    />
  </div>
</template>
