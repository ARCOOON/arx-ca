import { onMounted, onUnmounted } from 'vue'
import { useNotificationsStore } from '@/store/notifications'
import { fetchNotifications, createNotificationStream } from '@/api/notifications'
import type { NotificationEntry } from '@/types/api'
import { useToast } from './useToast'

export function useNotifications() {
  const store = useNotificationsStore()
  const toast = useToast()
  let eventSource: EventSource | null = null

  async function loadNotifications(): Promise<void> {
    try {
      const data = await fetchNotifications({ limit: 50, offset: 0 })
      store.setNotifications(data.notifications, data.unread_count, data.total)
    } catch {
      // Best-effort; do not surface to user
    }
  }

  function startStream(): void {
    if (eventSource) return
    try {
      eventSource = createNotificationStream((raw: string) => {
        try {
          const item = JSON.parse(raw) as NotificationEntry
          store.addItem(item)
          if (item.level === 'critical') {
            toast.warning(item.message)
          }
        } catch {
          // Ignore malformed SSE frames
        }
      })
    } catch {
      // SSE may not be available in all environments
    }
  }

  function stopStream(): void {
    eventSource?.close()
    eventSource = null
  }

  onMounted(() => {
    void loadNotifications()
    startStream()
  })

  onUnmounted(() => {
    stopStream()
  })

  return { store, loadNotifications }
}
