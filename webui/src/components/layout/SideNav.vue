<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  ChevronLeft,
  ChevronRight,
  ClipboardList,
  FileKey2,
  FileStack,
  GlobeLock,
  KeyRound,
  LayoutDashboard,
  Network,
  Server,
  Settings,
  ShieldCheck,
  Terminal,
  Webhook,
  X,
} from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<{
  collapsed: boolean
  mobileOpen: boolean
}>()

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
  'close-mobile': []
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
  { label: 'NDES', to: '/ndes', icon: Server },
  { label: 'Provisioners', to: '/provisioners', icon: KeyRound },
  { label: 'Templates', to: '/templates', icon: FileStack },
  { label: 'SSH CA', to: '/ssh', icon: Terminal },
  { label: 'Audit Log', to: '/audit', icon: ClipboardList },
  { label: 'Webhooks', to: '/webhooks', icon: Webhook },
  { label: 'Settings', to: '/settings', icon: Settings },
]

const sidebarWidthClass = computed(() => (props.collapsed ? 'w-14' : 'w-56'))

const mobileDrawerClass = computed(() =>
  props.mobileOpen
    ? 'translate-x-0'
    : '-translate-x-full pointer-events-none md:pointer-events-auto md:translate-x-0',
)

function isActive(path: string): boolean {
  return route.path === path
}

function toggleCollapsed(): void {
  emit('update:collapsed', !props.collapsed)
}

function handleNavClick(): void {
  emit('close-mobile')
}
</script>

<template>
  <aside
    class="fixed inset-y-0 left-0 z-40 flex h-full shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-[transform,width] duration-150 md:relative md:z-auto"
    :class="[sidebarWidthClass, mobileDrawerClass]"
  >
    <div
      class="flex items-center border-b border-sidebar-border px-3 py-4"
      :class="collapsed ? 'justify-center' : 'gap-2.5'"
    >
      <div class="flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-background">
        <ShieldCheck class="size-4 text-primary" aria-hidden="true" />
      </div>
      <div v-if="!collapsed" class="min-w-0 flex-1">
        <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">ARX CA</p>
        <p class="font-heading text-lg font-semibold tracking-tight">Management</p>
      </div>
      <button
        type="button"
        class="inline-flex size-7 shrink-0 items-center justify-center rounded-md border border-input bg-background p-0 md:hidden"
        aria-label="Close navigation menu"
        @click="emit('close-mobile')"
      >
        <X class="size-4" aria-hidden="true" />
      </button>
    </div>

    <nav class="custom-scrollbar flex flex-1 flex-col gap-1 overflow-y-auto p-3">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        :class="cn(
          'flex items-center rounded-md border-l-4 px-3 py-2 text-sm transition-colors',
          collapsed ? 'justify-center px-0' : 'gap-3',
          isActive(item.to)
            ? 'border-primary bg-primary/15 font-semibold text-foreground'
            : 'border-transparent font-medium text-sidebar-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground',
        )"
        :title="collapsed ? item.label : undefined"
        @click="handleNavClick"
      >
        <component :is="item.icon" class="size-4 shrink-0" aria-hidden="true" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="hidden border-t border-sidebar-border p-2 md:block">
      <button
        type="button"
        class="flex w-full items-center justify-center rounded-md border border-input bg-background py-2"
        :aria-label="collapsed ? 'Expand navigation' : 'Collapse navigation'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="size-4" aria-hidden="true" />
        <ChevronLeft v-else class="size-4" aria-hidden="true" />
      </button>
    </div>
  </aside>
</template>
