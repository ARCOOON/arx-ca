<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { LogOut, Menu } from '@lucide/vue'
import NotificationBell from '@/components/NotificationBell.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

const emit = defineEmits<{
  toggleMobileNav: []
  logout: []
}>()

const route = useRoute()

const pageTitle = computed(() => (route.meta.title as string) ?? 'ARX CA')
const pageSubtitle = computed(() => (route.meta.subtitle as string) ?? '')

function handleLogout(): void {
  emit('logout')
}
</script>

<template>
  <header class="flex h-12 items-center gap-3 border-b border-border bg-card px-3 md:px-4">
    <Button
      variant="ghost"
      size="icon"
      class="size-8 rounded-lg md:hidden"
      aria-label="Open navigation"
      @click="emit('toggleMobileNav')"
    >
      <Menu class="size-4" />
    </Button>

    <div class="min-w-0 flex-1">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <span class="text-xs text-muted-foreground">ARX CA</span>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage class="text-sm font-medium">{{ pageTitle }}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <p v-if="pageSubtitle" class="truncate text-[11px] text-muted-foreground">{{ pageSubtitle }}</p>
    </div>

    <div class="flex items-center gap-1">
      <ThemeSwitcher />
      <NotificationBell />
      <Separator orientation="vertical" class="mx-1 h-5" />
      <Button variant="ghost" size="icon" class="size-8 rounded-lg" aria-label="Logout" @click="handleLogout">
        <LogOut class="size-4" />
      </Button>
    </div>
  </header>
</template>
