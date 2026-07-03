<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import PostUpdateChangelogModal from '@/components/modals/PostUpdateChangelogModal.vue'
import SideNav from '@/components/layout/SideNav.vue'
import TopBar from '@/components/layout/TopBar.vue'
import { useNotifications } from '@/composables/useNotifications'
import { useUpdater } from '@/composables/useUpdater'
import { useAuthStore } from '@/store/auth'

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
  <div class="flex h-screen w-screen overflow-hidden bg-background">
    <div
      v-if="mobileNavOpen"
      class="fixed inset-0 z-30 bg-background/60 backdrop-blur-sm md:hidden"
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
      <TopBar @toggle-mobile-nav="toggleMobileNav" @logout="handleLogout" />

      <main class="flex-1 overflow-y-auto p-4 md:p-6">
        <RouterView />
      </main>
    </div>

    <PostUpdateChangelogModal
      :open="triggerChangelogModal"
      @close="dismissChangelogModal"
    />
  </div>
</template>
