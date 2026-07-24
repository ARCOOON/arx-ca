<script setup lang="ts">
import { Bell, CheckCheck, Trash2, CircleDot } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useNotificationsStore } from '@/stores/notifications'
import { formatDateTime } from '@/lib/format'

const notifications = useNotificationsStore()
</script>

<template>
  <Popover>
    <PopoverTrigger as-child>
      <Button variant="ghost" size="icon" class="relative" aria-label="Notifications">
        <Bell class="size-[1.15rem]" />
        <span
          v-if="notifications.unreadCount > 0"
          class="absolute -right-0.5 -top-0.5 flex min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-4 text-primary-foreground"
        >
          {{ notifications.unreadCount > 99 ? '99+' : notifications.unreadCount }}
        </span>
      </Button>
    </PopoverTrigger>
    <PopoverContent align="end" class="w-80 p-0">
      <div class="flex items-center justify-between border-b px-3 py-2.5">
        <p class="text-sm font-semibold">Notifications</p>
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-sm"
            title="Mark all as read"
            :disabled="notifications.unreadCount === 0"
            @click="notifications.markAllRead()"
          >
            <CheckCheck class="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            title="Clear all"
            :disabled="notifications.items.length === 0"
            @click="notifications.archiveAll()"
          >
            <Trash2 class="size-4" />
          </Button>
        </div>
      </div>

      <ScrollArea class="max-h-80">
        <div
          v-if="notifications.items.length === 0"
          class="px-3 py-10 text-center text-sm text-muted-foreground"
        >
          You are all caught up.
        </div>

        <ul v-else class="divide-y">
          <li
            v-for="item in notifications.items"
            :key="item.id"
            class="flex items-start gap-2.5 px-3 py-2.5 transition-colors hover:bg-accent"
            :class="{ 'bg-accent/40': !item.is_read }"
          >
            <CircleDot
              class="mt-0.5 size-3.5 shrink-0"
              :class="
                item.level === 'critical'
                  ? 'text-destructive'
                  : item.is_read
                    ? 'text-muted-foreground/40'
                    : 'text-primary'
              "
            />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm" :class="{ 'font-medium': !item.is_read }">
                {{ item.message }}
              </p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                {{ formatDateTime(item.timestamp) }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-0.5">
              <Button
                v-if="!item.is_read"
                variant="ghost"
                size="icon-sm"
                title="Mark as read"
                @click="notifications.markRead(item.id)"
              >
                <CheckCheck class="size-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                title="Delete"
                @click="notifications.remove(item.id)"
              >
                <Trash2 class="size-3.5" />
              </Button>
            </div>
          </li>
        </ul>
      </ScrollArea>
    </PopoverContent>
  </Popover>
</template>
