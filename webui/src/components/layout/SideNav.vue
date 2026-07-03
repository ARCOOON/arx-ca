<script setup lang="ts">
import { useRoute } from 'vue-router'
import { cn } from '@/utils/cn'

interface NavItem {
  name: string
  route: string
  icon: string
}

const route = useRoute()

const navGroups: Array<{ label?: string; items: NavItem[] }> = [
  {
    items: [
      { name: 'Dashboard', route: 'dashboard', icon: 'dashboard' },
    ],
  },
  {
    label: 'PKI',
    items: [
      { name: 'Certificates', route: 'certificates', icon: 'certificate' },
      { name: 'SSH CA', route: 'ssh', icon: 'key' },
      { name: 'Templates', route: 'templates', icon: 'template' },
    ],
  },
  {
    label: 'Enrollment',
    items: [
      { name: 'ACME', route: 'acme', icon: 'acme' },
      { name: 'SCEP', route: 'scep', icon: 'scep' },
      { name: 'NDES', route: 'ndes', icon: 'ndes' },
      { name: 'Provisioners', route: 'provisioners', icon: 'provisioner' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { name: 'Audit Log', route: 'audit', icon: 'audit' },
      { name: 'Webhooks', route: 'webhooks', icon: 'webhook' },
    ],
  },
  {
    label: 'System',
    items: [
      { name: 'Settings', route: 'settings', icon: 'settings' },
    ],
  },
]

function isActive(routeName: string): boolean {
  return route.name === routeName
}
</script>

<template>
  <nav class="flex flex-col h-full bg-sidebar-bg border-r border-sidebar-border w-56 shrink-0 select-none">
    <!-- Logo / Brand -->
    <div class="flex items-center gap-2.5 px-4 py-4 border-b border-sidebar-border">
      <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-primary">
        <svg class="h-4 w-4 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <span class="text-sm font-semibold text-foreground tracking-tight">ARX CA</span>
    </div>

    <!-- Navigation items -->
    <div class="flex-1 overflow-y-auto py-2 px-2 space-y-4">
      <div v-for="(group, i) in navGroups" :key="i">
        <p
          v-if="group.label"
          class="px-2 pb-1 text-[10px] font-semibold uppercase tracking-widest text-foreground-subtle"
        >
          {{ group.label }}
        </p>
        <ul class="space-y-0.5">
          <li v-for="item in group.items" :key="item.route">
            <RouterLink
              :to="{ name: item.route }"
              :class="
                cn(
                  'group flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-all',
                  isActive(item.route)
                    ? 'bg-sidebar-item-active text-sidebar-item-active-text border-l-2 border-sidebar-item-active-border pl-[calc(0.625rem-2px)]'
                    : 'text-foreground-muted hover:bg-sidebar-item-hover hover:text-foreground border-l-2 border-transparent',
                )
              "
            >
              <!-- Icon -->
              <span class="flex h-4 w-4 items-center justify-center shrink-0">
                <NavIcon :icon="item.icon" :active="isActive(item.route)" />
              </span>
              <span class="truncate">{{ item.name }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>
    </div>

    <!-- Bottom user area / logout -->
    <div class="border-t border-sidebar-border p-2">
      <RouterLink
        :to="{ name: 'settings' }"
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-foreground-subtle hover:text-foreground hover:bg-sidebar-item-hover transition-colors"
      >
        <svg class="h-3.5 w-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
        <span>Settings</span>
      </RouterLink>
    </div>
  </nav>
</template>

<script lang="ts">
// Inline NavIcon to keep component self-contained
const NavIcon = {
  props: { icon: String, active: Boolean },
  template: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" class="h-full w-full">
      <template v-if="icon === 'dashboard'">
        <rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/>
        <rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>
      </template>
      <template v-else-if="icon === 'certificate'">
        <rect x="3" y="5" width="18" height="14" rx="2"/><line x1="3" y1="10" x2="21" y2="10"/>
        <line x1="8" y1="14" x2="12" y2="14"/>
      </template>
      <template v-else-if="icon === 'key'">
        <circle cx="7.5" cy="15.5" r="4.5"/><path d="m21 2-9.6 9.6M15.5 7.5l3 3"/>
      </template>
      <template v-else-if="icon === 'template'">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="13" y2="17"/>
      </template>
      <template v-else-if="icon === 'acme'">
        <path d="M21 2H3v16h5l4 4 4-4h5V2z"/><path d="M12 6v6l3 3"/>
      </template>
      <template v-else-if="icon === 'scep'">
        <path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/>
      </template>
      <template v-else-if="icon === 'ndes'">
        <rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/>
      </template>
      <template v-else-if="icon === 'provisioner'">
        <circle cx="12" cy="12" r="10"/><path d="M12 8v4l3 3"/>
      </template>
      <template v-else-if="icon === 'audit'">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
        <line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/><polyline points="9 9 10 9"/>
      </template>
      <template v-else-if="icon === 'webhook'">
        <path d="M18 16.4a4 4 0 0 1-5.8-5.5l2-2A4 4 0 0 1 20 14.6"/>
        <path d="M6 7.6a4 4 0 0 1 5.8 5.5l-2 2A4 4 0 0 1 4 9.4"/>
      </template>
      <template v-else>
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </template>
    </svg>
  `,
}

export default {
  components: { NavIcon },
}
</script>
