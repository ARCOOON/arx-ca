<script setup lang="ts">
import { RouterLink } from 'vue-router'
import {
  LayoutDashboard,
  ShieldCheck,
  TerminalSquare,
  Settings as SettingsIcon,
  PanelLeftClose,
  PanelLeftOpen,
} from '@lucide/vue'

defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  'toggle-collapse': []
  navigate: []
}>()

const navItems = [
  { to: { name: 'dashboard' }, label: 'Dashboard', icon: LayoutDashboard },
  { to: { name: 'certificates' }, label: 'Certificates', icon: ShieldCheck },
  { to: { name: 'ssh' }, label: 'SSH CA', icon: TerminalSquare },
  { to: { name: 'settings' }, label: 'Settings', icon: SettingsIcon },
]
</script>

<template>
  <div class="flex h-full flex-col bg-sidebar text-sidebar-foreground">
    <!-- Brand -->
    <div
      class="flex h-14 shrink-0 items-center gap-2.5 border-b border-sidebar-border px-3"
      :class="collapsed ? 'justify-center' : ''"
    >
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground"
      >
        <ShieldCheck class="size-5" />
      </div>
      <span v-if="!collapsed" class="truncate text-sm font-semibold tracking-tight">
        Arx CA Console
      </span>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 space-y-1 overflow-y-auto p-2">
      <RouterLink
        v-for="item in navItems"
        :key="item.label"
        :to="item.to"
        v-slot="{ isActive, navigate }"
        custom
      >
        <button
          type="button"
          class="group relative flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-sidebar-ring"
          :class="[
            isActive
              ? 'bg-sidebar-accent text-sidebar-accent-foreground'
              : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground',
            collapsed ? 'justify-center' : '',
          ]"
          :title="collapsed ? item.label : undefined"
          @click="() => { navigate(); emit('navigate') }"
        >
          <!-- WinUI-style vertical active indicator -->
          <span
            class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full bg-primary transition-opacity"
            :class="isActive ? 'opacity-100' : 'opacity-0'"
            aria-hidden="true"
          />
          <component :is="item.icon" class="size-[1.15rem] shrink-0" />
          <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
        </button>
      </RouterLink>
    </nav>

    <!-- Collapse toggle (desktop only) -->
    <div class="hidden shrink-0 border-t border-sidebar-border p-2 md:block">
      <button
        type="button"
        class="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground outline-none transition-colors hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring"
        :class="collapsed ? 'justify-center' : ''"
        :title="collapsed ? 'Expand' : 'Collapse'"
        @click="emit('toggle-collapse')"
      >
        <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="size-[1.15rem] shrink-0" />
        <span v-if="!collapsed">Collapse</span>
      </button>
    </div>
  </div>
</template>
