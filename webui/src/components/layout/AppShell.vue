<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { watch } from 'vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import TopBar from '@/components/layout/TopBar.vue'
import PostUpdateChangelogModal from '@/components/PostUpdateChangelogModal.vue'
import { useNotificationsStore } from '@/stores/notifications'
import { useUpdater } from '@/composables/useUpdater'

const COLLAPSE_KEY = 'arx_sidebar_collapsed'

const collapsed = ref(localStorage.getItem(COLLAPSE_KEY) === 'true')
const mobileNavOpen = ref(false)

const route = useRoute()
const notifications = useNotificationsStore()
const { showChangelog, runVersionDriftCheck, dismissChangelog } = useUpdater()

function toggleCollapse(): void {
  collapsed.value = !collapsed.value
  localStorage.setItem(COLLAPSE_KEY, String(collapsed.value))
}

// Close the mobile drawer on navigation.
watch(
  () => route.fullPath,
  () => {
    mobileNavOpen.value = false
  },
)

onMounted(() => {
  notifications.connect()
  runVersionDriftCheck()
})

onBeforeUnmount(() => {
  notifications.disconnect()
})
</script>

<template>
  <div class="flex h-screen w-full overflow-hidden bg-background">
    <!-- Desktop rail -->
    <aside
      class="hidden shrink-0 border-r border-sidebar-border transition-[width] duration-200 md:block"
      :class="collapsed ? 'w-16' : 'w-60'"
    >
      <Sidebar :collapsed="collapsed" @toggle-collapse="toggleCollapse" />
    </aside>

    <!-- Mobile drawer -->
    <Transition name="fade">
      <div
        v-if="mobileNavOpen"
        class="fixed inset-0 z-40 bg-black/40 md:hidden"
        @click="mobileNavOpen = false"
      />
    </Transition>
    <Transition name="slide">
      <aside
        v-if="mobileNavOpen"
        class="fixed inset-y-0 left-0 z-50 w-64 border-r border-sidebar-border md:hidden"
      >
        <Sidebar :collapsed="false" @navigate="mobileNavOpen = false" />
      </aside>
    </Transition>

    <!-- Main column -->
    <div class="flex min-w-0 flex-1 flex-col">
      <TopBar @toggle-mobile-nav="mobileNavOpen = true" />
      <main class="min-h-0 flex-1 overflow-y-auto">
        <div class="mx-auto w-full max-w-7xl p-4 sm:p-6">
          <RouterView />
        </div>
      </main>
    </div>

    <PostUpdateChangelogModal :open="showChangelog" @close="dismissChangelog" />
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.2s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(-100%);
}
</style>
