<script setup lang="ts">
import { computed } from 'vue'
import { Moon, Sun, MonitorSmartphone, Check } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useThemeStore, type ThemeMode } from '@/stores/theme'

const theme = useThemeStore()

const options: { value: ThemeMode; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'auto', label: 'Auto (system)', icon: MonitorSmartphone },
]

const activeIcon = computed(() => {
  if (theme.mode === 'auto') {
    return MonitorSmartphone
  }
  return theme.mode === 'dark' ? Moon : Sun
})
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" aria-label="Change theme">
        <component :is="activeIcon" class="size-[1.15rem]" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-44">
      <DropdownMenuLabel>Appearance</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        v-for="option in options"
        :key="option.value"
        class="justify-between"
        @select="theme.setMode(option.value)"
      >
        <span class="flex items-center gap-2">
          <component :is="option.icon" class="size-4" />
          {{ option.label }}
        </span>
        <Check v-if="theme.mode === option.value" class="size-4 text-primary" />
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
