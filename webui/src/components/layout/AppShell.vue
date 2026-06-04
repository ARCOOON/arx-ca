<script setup lang="ts">
import { ref } from 'vue'
import { RouterView } from 'vue-router'
import { useAuthStore } from '../../store/auth'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const SIDEBAR_COLLAPSED_KEY = 'arx_sidebar_collapsed'

function readCollapsedPreference(): boolean {
  return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
}

const authStore = useAuthStore()
const sidebarCollapsed = ref(readCollapsedPreference())

function onSidebarCollapsedChange(value: boolean): void {
  sidebarCollapsed.value = value
  localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(value))
}

function handleLogout(): void {
  void authStore.logout()
}
</script>

<template>
  <div class="flex min-h-screen bg-zinc-950 text-zinc-100">
    <SideNav :collapsed="sidebarCollapsed" @update:collapsed="onSidebarCollapsedChange" />

    <div class="flex min-w-0 flex-1 flex-col">
      <TopBar @logout="handleLogout" />

      <main class="flex-1 overflow-auto p-4">
        <RouterView />
      </main>
    </div>
  </div>
</template>
