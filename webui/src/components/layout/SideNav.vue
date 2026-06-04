<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import ChevronLeft from 'lucide-vue-next/dist/esm/icons/chevron-left.js'
import ChevronRight from 'lucide-vue-next/dist/esm/icons/chevron-right.js'
import FileKey2 from 'lucide-vue-next/dist/esm/icons/file-key.js'
import GlobeLock from 'lucide-vue-next/dist/esm/icons/globe-lock.js'
import LayoutDashboard from 'lucide-vue-next/dist/esm/icons/layout-dashboard.js'
import Network from 'lucide-vue-next/dist/esm/icons/network.js'
import Settings from 'lucide-vue-next/dist/esm/icons/settings.js'
import ShieldCheck from 'lucide-vue-next/dist/esm/icons/shield-check.js'

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
    class="ui-border-r ui-surface flex shrink-0 flex-col transition-[width] duration-150"
    :class="sidebarWidthClass"
  >
    <div
      class="ui-border-b flex items-center px-3 py-3"
      :class="collapsed ? 'justify-center' : 'gap-2.5'"
    >
      <div class="ui-brand-icon flex h-8 w-8 shrink-0 items-center justify-center">
        <ShieldCheck class="h-4 w-4" style="color: var(--accent-color)" aria-hidden="true" />
      </div>
      <div v-if="!collapsed" class="min-w-0">
        <p class="truncate text-sm font-semibold ui-text-primary">Arx CA</p>
        <p class="text-[10px] uppercase tracking-wide ui-text-muted">Management</p>
      </div>
    </div>

    <nav class="flex-1 space-y-0.5 px-2 py-3">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="ui-nav-link flex items-center text-xs"
        :class="[
          collapsed ? 'justify-center px-0 py-2' : 'gap-2.5 px-2.5 py-2',
          isActive(item.to) ? 'ui-nav-link-active' : '',
        ]"
        :title="collapsed ? item.label : undefined"
      >
        <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="ui-border-t p-2">
      <button
        type="button"
        class="ui-btn-secondary flex w-full items-center justify-center py-2"
        :aria-label="collapsed ? 'Expand navigation' : 'Collapse navigation'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="h-4 w-4" aria-hidden="true" />
        <ChevronLeft v-else class="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  </aside>
</template>
