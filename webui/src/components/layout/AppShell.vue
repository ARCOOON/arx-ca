<script setup lang="ts">
import { onMounted } from 'vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'
import ToastHost from '@/components/ui/ToastHost.vue'
import PostUpdateChangelogModal from '@/components/modals/PostUpdateChangelogModal.vue'
import { useNotifications } from '@/composables/useNotifications'
import { useUpdater } from '@/composables/useUpdater'

const { triggerChangelogModal, runVersionDriftCheck, dismissChangelogModal } = useUpdater()
useNotifications()

onMounted(() => {
  runVersionDriftCheck()
})
</script>

<template>
  <div class="flex h-screen w-full overflow-hidden bg-background">
    <SideNav />

    <div class="flex flex-1 flex-col overflow-hidden min-w-0">
      <TopBar />

      <main class="flex-1 overflow-y-auto">
        <div class="max-w-screen-xl mx-auto px-6 py-6">
          <RouterView />
        </div>
      </main>
    </div>

    <ToastHost />

    <PostUpdateChangelogModal
      :open="triggerChangelogModal"
      @close="dismissChangelogModal"
    />
  </div>
</template>
