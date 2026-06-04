<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  LayoutDashboard,
  FileKey2,
  GlobeLock,
  Network,
  Settings,
  ChevronLeft,
  ChevronRight,
  ShieldCheck,
} from 'lucide-vue-next'

const props = defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
}>()

const route = useRoute()

interface NavItem {
  label: string
  to: string
  icon: typeof LayoutDashboard
}

const navItems: NavItem[] = [
  { label: 'Dashboard', to: '/dashboard', icon: LayoutDashboard },
  { label: 'Certificates', to: '/certificates', icon: FileKey2 },
  { label: 'ACME', to: '/acme', icon: GlobeLock },
  { label: 'SCEP', to: '/scep', icon: Network },
  { label: 'Settings', to: '/settings', icon: Settings },
]

const sidebarWidthClass = computed(() => (props.collapsed ? 'w-14' : 'w-52'))

function isActive(path: string): boolean {
  return route.path === path
}

function toggleCollapsed(): void {
  emit('update:collapsed', !props.collapsed)
}
</script>

<template>
  <aside
    class="flex shrink-0 flex-col border-r border-zinc-800 bg-zinc-900 transition-[width] duration-150"
    :class="sidebarWidthClass"
  >
    <div
      class="flex items-center border-b border-zinc-800 px-3 py-3"
      :class="collapsed ? 'justify-center' : 'gap-2.5'"
    >
      <div class="flex h-8 w-8 shrink-0 items-center justify-center border border-zinc-700 bg-zinc-950">
        <ShieldCheck class="h-4 w-4 text-emerald-400" aria-hidden="true" />
      </div>
      <div v-if="!collapsed" class="min-w-0">
        <p class="truncate text-sm font-semibold text-zinc-50">Arx CA</p>
        <p class="text-[10px] uppercase tracking-wide text-zinc-500">Management</p>
      </div>
    </div>

    <nav class="flex-1 space-y-0.5 px-2 py-3">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex items-center border border-transparent text-xs transition"
        :class="[
          collapsed ? 'justify-center px-0 py-2' : 'gap-2.5 px-2.5 py-2',
          isActive(item.to)
            ? 'border-zinc-700 bg-zinc-800 text-zinc-50'
            : 'text-zinc-400 hover:border-zinc-800 hover:bg-zinc-800/50 hover:text-zinc-200',
        ]"
        :title="collapsed ? item.label : undefined"
      >
        <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="border-t border-zinc-800 p-2">
      <button
        type="button"
        class="flex w-full items-center justify-center border border-zinc-700 bg-zinc-950 py-2 text-zinc-400 transition hover:border-zinc-600 hover:text-zinc-200"
        :aria-label="collapsed ? 'Expand navigation' : 'Collapse navigation'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="h-4 w-4" aria-hidden="true" />
        <ChevronLeft v-else class="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  </aside>
</template>
