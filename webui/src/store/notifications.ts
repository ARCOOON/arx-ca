import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { NotificationEntry } from '@/types/api'

export const useNotificationsStore = defineStore('notifications', () => {
  const items = ref<NotificationEntry[]>([])
  const unreadCount = ref(0)
  const total = ref(0)

  function setNotifications(list: NotificationEntry[], count: number, totalCount: number): void {
    items.value = list
    unreadCount.value = count
    total.value = totalCount
  }

  function decrementUnread(): void {
    if (unreadCount.value > 0) unreadCount.value--
  }

  function clearAll(): void {
    items.value = []
    unreadCount.value = 0
    total.value = 0
  }

  function addItem(item: NotificationEntry): void {
    items.value.unshift(item)
    if (!item.is_read) unreadCount.value++
    total.value++
  }

  return { items, unreadCount, total, setNotifications, decrementUnread, clearAll, addItem }
})
