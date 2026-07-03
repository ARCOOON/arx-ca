<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  ChevronLeft,
  ChevronRight,
  LayoutDashboard,
  Settings,
  Shield,
  Terminal,
} from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

const props = defineProps<{
  collapsed: boolean
  mobileOpen: boolean
}>()

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
  closeMobile: []
}>()

const route = useRoute()

const navItems = [
  { name: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, to: '/dashboard' },
  { name: 'certificates', label: 'Certificates', icon: Shield, to: '/certificates' },
  { name: 'ssh', label: 'SSH CA', icon: Terminal, to: '/ssh' },
  { name: 'settings', label: 'Settings', icon: Settings, to: '/settings' },
]

const sidebarClass = computed(() =>
  cn(
    'flex h-full flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-all duration-200',
    props.collapsed ? 'w-14' : 'w-56',
    'fixed inset-y-0 left-0 z-40 md:relative md:translate-x-0',
    props.mobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0',
  ),
)

function toggleCollapsed(): void {
  emit('update:collapsed', !props.collapsed)
}

function isActive(name: string): boolean {
  return route.name === name
}
</script>

<template>
  <aside :class="sidebarClass">
    <div class="flex h-12 items-center gap-2 px-3">
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground text-xs font-bold"
      >
        ARX
      </div>
      <span v-if="!collapsed" class="truncate text-sm font-semibold">Certificate Authority</span>
    </div>

    <Separator class="bg-sidebar-border" />

    <nav class="flex flex-1 flex-col gap-1 p-2">
      <RouterLink
        v-for="item in navItems"
        :key="item.name"
        :to="item.to"
        class="flex items-center gap-3 rounded-lg px-2.5 py-2 text-sm transition-colors"
        :class="
          isActive(item.name)
            ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
            : 'text-sidebar-foreground/80 hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground'
        "
        @click="emit('closeMobile')"
      >
        <component :is="item.icon" class="size-4 shrink-0" />
        <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="hidden border-t border-sidebar-border p-2 md:block">
      <Button
        variant="ghost"
        size="icon"
        class="size-8 w-full rounded-lg"
        :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="size-4" />
        <ChevronLeft v-else class="size-4" />
      </Button>
    </div>
  </aside>
</template>
