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
import Button from '../ui/Button.vue'

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

const sidebarWidthClass = computed(() => (props.collapsed ? 'w-14' : 'w-52'))

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
    class="ui-border-r ui-surface fixed inset-y-0 left-0 z-40 flex h-full shrink-0 flex-col transition-[transform,width] duration-150 md:relative md:z-auto"
    :class="[sidebarWidthClass, mobileDrawerClass]"
  >
    <div
      class="ui-border-b flex items-center px-3 py-3"
      :class="collapsed ? 'justify-center' : 'gap-2.5'"
    >
      <div class="ui-brand-icon flex h-8 w-8 shrink-0 items-center justify-center">
        <ShieldCheck class="h-4 w-4" style="color: var(--accent-color)" aria-hidden="true" />
      </div>
      <div v-if="!collapsed" class="min-w-0 flex-1">
        <p class="truncate text-sm font-semibold ui-text-primary">Arx CA</p>
        <p class="text-[10px] uppercase tracking-wide ui-text-muted">Management</p>
      </div>
      <Button
        variant="secondary"
        size="icon"
        class="md:hidden"
        aria-label="Close navigation menu"
        @click="emit('close-mobile')"
      >
        <X class="h-4 w-4" aria-hidden="true" />
      </Button>
    </div>

    <nav class="custom-scrollbar flex-1 space-y-0.5 overflow-y-auto px-2 py-3">
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
        @click="handleNavClick"
      >
        <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="ui-border-t hidden p-2 md:block">
      <Button
        variant="secondary"
        class="w-full"
        :aria-label="collapsed ? 'Expand navigation' : 'Collapse navigation'"
        @click="toggleCollapsed"
      >
        <ChevronRight v-if="collapsed" class="h-4 w-4" aria-hidden="true" />
        <ChevronLeft v-else class="h-4 w-4" aria-hidden="true" />
      </Button>
    </div>
  </aside>
</template>
