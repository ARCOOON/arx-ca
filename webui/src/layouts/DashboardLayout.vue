<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  LayoutDashboard,
  FileKey2,
  GlobeLock,
  Settings,
  LogOut,
  ShieldCheck,
} from 'lucide-vue-next'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const authStore = useAuthStore()

interface NavItem {
  label: string
  to: string
  icon: typeof LayoutDashboard
}

const navItems: NavItem[] = [
  { label: 'Dashboard', to: '/dashboard', icon: LayoutDashboard },
  { label: 'Certificates', to: '/dashboard', icon: FileKey2 },
  { label: 'ACME', to: '/dashboard', icon: GlobeLock },
  { label: 'Settings', to: '/dashboard', icon: Settings },
]

const activePath = computed(() => route.path)

function isActive(path: string): boolean {
  return activePath.value === path
}

function handleLogout(): void {
  void authStore.logout()
}
</script>

<template>
  <div class="flex min-h-screen bg-zinc-950 text-zinc-100">
    <aside class="flex w-64 shrink-0 flex-col border-r border-zinc-800 bg-zinc-900/50">
      <div class="flex items-center gap-3 border-b border-zinc-800 px-5 py-5">
        <div class="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-900">
          <ShieldCheck class="h-5 w-5 text-emerald-400" aria-hidden="true" />
        </div>
        <div>
          <p class="text-sm font-semibold text-zinc-50">Arx CA</p>
          <p class="text-xs text-zinc-500">Admin Console</p>
        </div>
      </div>

      <nav class="flex-1 space-y-1 px-3 py-4">
        <RouterLink
          v-for="item in navItems"
          :key="item.label"
          :to="item.to"
          class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition"
          :class="
            isActive(item.to) && item.label === 'Dashboard'
              ? 'bg-zinc-800 text-zinc-50'
              : 'text-zinc-400 hover:bg-zinc-800/60 hover:text-zinc-200'
          "
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="border-t border-zinc-800 px-4 py-4">
        <p class="truncate text-xs text-zinc-500">Roles</p>
        <p class="mt-1 truncate text-sm text-zinc-300">
          {{ authStore.roles.length > 0 ? authStore.roles.join(', ') : 'Administrator' }}
        </p>
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/30 px-6 py-4">
        <div>
          <h1 class="text-lg font-semibold text-zinc-50">Dashboard</h1>
          <p class="text-sm text-zinc-500">Certificate Authority management console</p>
        </div>

        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-300 transition hover:border-zinc-600 hover:bg-zinc-800 hover:text-zinc-100"
          @click="handleLogout"
        >
          <LogOut class="h-4 w-4" aria-hidden="true" />
          Logout
        </button>
      </header>

      <main class="flex-1 overflow-auto p-6">
        <slot />
      </main>
    </div>
  </div>
</template>
