<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import ChevronLeft from 'lucide-vue-next/dist/esm/icons/chevron-left.js'
import ChevronRight from 'lucide-vue-next/dist/esm/icons/chevron-right.js'
import ClipboardList from 'lucide-vue-next/dist/esm/icons/clipboard-list.js'
import FileKey2 from 'lucide-vue-next/dist/esm/icons/file-key.js'
import FileStack from 'lucide-vue-next/dist/esm/icons/file-stack.js'
import GlobeLock from 'lucide-vue-next/dist/esm/icons/globe-lock.js'
import KeyRound from 'lucide-vue-next/dist/esm/icons/key-round.js'
import LayoutDashboard from 'lucide-vue-next/dist/esm/icons/layout-dashboard.js'
import Network from 'lucide-vue-next/dist/esm/icons/network.js'
import Server from 'lucide-vue-next/dist/esm/icons/server.js'
import Settings from 'lucide-vue-next/dist/esm/icons/settings.js'
import ShieldCheck from 'lucide-vue-next/dist/esm/icons/shield-check.js'
import Webhook from 'lucide-vue-next/dist/esm/icons/webhook.js'
import Terminal from 'lucide-vue-next/dist/esm/icons/terminal.js'
import X from 'lucide-vue-next/dist/esm/icons/x.js'
import { Button } from '@/components/ui/button'
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
  return route.path === path || route.path.startsWith(`${path}/`)
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
    :class="cn(
      'fixed inset-y-0 left-0 z-40 flex h-full shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-[transform,width] duration-150 md:relative md:z-auto',
      sidebarWidthClass,
      mobileDrawerClass,
    )"
  >
    <div
      class="flex items-center border-b border-sidebar-border px-3 py-4"
      :class="collapsed ? 'justify-center' : 'gap-2.5'"
    >
      <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-sidebar-border bg-sidebar-accent">
        <ShieldCheck class="h-4 w-4 text-primary" />
      </div>
      <div v-if="!collapsed" class="min-w-0 flex-1">
        <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Arx CA</p>
        <p class="font-heading text-sm font-semibold tracking-tight">Management</p>
      </div>
      <Button
        variant="outline"
        size="icon-sm"
        class="md:hidden"
        aria-label="Close navigation menu"
        @click="emit('close-mobile')"
      >
        <X class="h-4 w-4" />
      </Button>
    </div>

    <nav class="custom-scrollbar flex-1 space-y-1 overflow-y-auto p-3">
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
        <component :is="item.icon" class="h-4 w-4 shrink-0" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="hidden border-t border-sidebar-border p-2 md:block">
      <Button
        variant="outline"
        class="w-full"
        :aria-label="collapsed ? 'Expand navigation' : 'Collapse navigation'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="h-4 w-4" />
        <ChevronLeft v-else class="h-4 w-4" />
      </Button>
    </div>
  </aside>
</template>
