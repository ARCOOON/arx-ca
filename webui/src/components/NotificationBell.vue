<script setup lang="ts">
import { ref } from 'vue'
import { Bell, CheckCheck, Trash2 } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { useNotifications } from '@/composables/useNotifications'
import { formatDateTime } from '@/utils/format'

const {
  persistentNotifications,
  unreadCount,
  markAsRead,
  markAllAsRead,
  archiveAllPersistent,
} = useNotifications()

const open = ref(false)

async function handleMarkAllRead(): Promise<void> {
  await markAllAsRead()
}

async function handleArchiveAll(): Promise<void> {
  await archiveAllPersistent()
}

async function handleMarkRead(id: string): Promise<void> {
  await markAsRead(id)
}
</script>

<template>
  <DropdownMenu v-model:open="open">
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" class="relative size-8 rounded-lg" aria-label="Notifications">
        <Bell class="size-4" />
        <Badge
          v-if="unreadCount > 0"
          class="absolute -right-0.5 -top-0.5 flex size-4 items-center justify-center rounded-full p-0 text-[10px]"
        >
          {{ unreadCount > 9 ? '9+' : unreadCount }}
        </Badge>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-80 rounded-lg p-0">
      <div class="flex items-center justify-between px-3 py-2">
        <p class="text-sm font-medium">Notifications</p>
        <div class="flex gap-1">
          <Button variant="ghost" size="icon" class="size-7 rounded-md" title="Mark all read" @click="handleMarkAllRead">
            <CheckCheck class="size-3.5" />
          </Button>
          <Button variant="ghost" size="icon" class="size-7 rounded-md" title="Archive all" @click="handleArchiveAll">
            <Trash2 class="size-3.5" />
          </Button>
        </div>
      </div>
      <Separator />
      <ScrollArea class="h-72">
        <div v-if="persistentNotifications.length === 0" class="px-3 py-6 text-center text-sm text-muted-foreground">
          No notifications yet.
        </div>
        <button
          v-for="item in persistentNotifications"
          :key="item.id"
          type="button"
          class="flex w-full flex-col gap-0.5 border-b border-border px-3 py-2.5 text-left transition-colors hover:bg-accent"
          :class="{ 'bg-accent/50': !item.is_read }"
          @click="handleMarkRead(item.id)"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="truncate text-xs font-medium">{{ item.action.replaceAll('_', ' ') }}</span>
            <Badge
              v-if="item.level === 'critical'"
              variant="destructive"
              class="shrink-0 rounded-md text-[10px]"
            >
              Critical
            </Badge>
          </div>
          <span class="text-xs text-muted-foreground">{{ item.message }}</span>
          <span class="text-[10px] text-muted-foreground">{{ formatDateTime(item.timestamp) }}</span>
        </button>
      </ScrollArea>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
