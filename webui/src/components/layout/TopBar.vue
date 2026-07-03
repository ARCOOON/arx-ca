<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Menu, ChevronRight, LogOut, UserRound } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import NotificationBell from '@/components/NotificationBell.vue'
import { useAuthStore } from '@/stores/auth'

defineEmits<{
  'toggle-mobile-nav': []
}>()

const route = useRoute()
const authStore = useAuthStore()

const breadcrumb = computed(() => (route.meta.breadcrumb as string) ?? (route.meta.title as string) ?? '')
const roleLabel = computed(() => (authStore.roles.length ? authStore.roles.join(', ') : 'Operator'))
</script>

<template>
  <header
    class="flex h-14 shrink-0 items-center gap-3 border-b border-border bg-card px-3 sm:px-4"
  >
    <Button
      variant="ghost"
      size="icon"
      class="md:hidden"
      aria-label="Open navigation"
      @click="$emit('toggle-mobile-nav')"
    >
      <Menu class="size-5" />
    </Button>

    <!-- Breadcrumbs -->
    <nav class="flex min-w-0 items-center gap-1.5 text-sm" aria-label="Breadcrumb">
      <span class="text-muted-foreground">Console</span>
      <ChevronRight class="size-4 shrink-0 text-muted-foreground/60" />
      <span class="truncate font-medium text-foreground">{{ breadcrumb }}</span>
    </nav>

    <div class="ml-auto flex items-center gap-1">
      <NotificationBell />
      <ThemeSwitcher />
      <Separator orientation="vertical" class="mx-1 h-6" />

      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" size="icon" aria-label="Account menu">
            <UserRound class="size-[1.15rem]" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-52">
          <DropdownMenuLabel>
            <p class="text-sm font-medium">Signed in</p>
            <p class="truncate text-xs font-normal text-muted-foreground">{{ roleLabel }}</p>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem class="text-destructive focus:text-destructive" @select="authStore.logout()">
            <LogOut class="size-4" />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </header>
</template>
