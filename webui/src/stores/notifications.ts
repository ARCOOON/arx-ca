import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { toast } from 'vue-sonner'
import {
  archiveAllNotifications as archiveAllApi,
  deleteNotification as deleteApi,
  listNotifications,
  markAllNotificationsRead as markAllApi,
  markNotificationRead as markReadApi,
} from '@/api/notifications'
import { resolveApiBaseURL } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { NotificationEntry } from '@/types/api'

interface AuditStreamPayload {
  notification_id?: string
  timestamp: string
  action: string
  actor?: { type: string; id: string; roles?: string[] }
  metadata?: Record<string, unknown>
}

const MAX_ITEMS = 200
const CRITICAL_ACTIONS = new Set(['AUTH_LOGIN_FAILED', 'CERT_REVOKE', 'EAB_REVOKE'])

function isCritical(action: string): boolean {
  return CRITICAL_ACTIONS.has(action.trim().toUpperCase())
}

function humanize(action: string): string {
  return action.replaceAll('_', ' ').toLowerCase()
}

export const useNotificationsStore = defineStore('notifications', () => {
  const items = ref<NotificationEntry[]>([])
  let eventSource: EventSource | null = null
  let reconnectTimer: number | null = null

  const unreadCount = computed(() => items.value.filter((item) => !item.is_read).length)

  async function load(): Promise<void> {
    const authStore = useAuthStore()
    if (!authStore.isAuthenticated) {
      return
    }
    try {
      const data = await listNotifications({ limit: 100 })
      items.value = data.notifications
    } catch {
      // Keep existing state when the history endpoint is temporarily unavailable.
    }
  }

  function prepend(entry: NotificationEntry): void {
    if (items.value.some((item) => item.id === entry.id)) {
      return
    }
    items.value = [entry, ...items.value].slice(0, MAX_ITEMS)
  }

  function handlePayload(payload: AuditStreamPayload): void {
    const actor = payload.actor?.id?.trim() || 'system'
    const message = `${actor}: ${humanize(payload.action)}`
    const critical = isCritical(payload.action)

    if (critical) {
      toast.error(message)
    } else {
      toast(message)
    }

    const id = payload.notification_id?.trim()
    if (!id) {
      return
    }
    prepend({
      id,
      action: payload.action,
      level: critical ? 'critical' : 'info',
      message,
      timestamp: payload.timestamp,
      is_read: false,
      metadata: payload.metadata,
    })
  }

  function buildStreamURL(): string {
    const authStore = useAuthStore()
    const url = new URL(`${resolveApiBaseURL()}/notifications/stream`)
    if (authStore.token) {
      url.searchParams.set('access_token', authStore.token)
    }
    return url.toString()
  }

  function connect(): void {
    const authStore = useAuthStore()
    if (eventSource || !authStore.isAuthenticated) {
      return
    }
    void load()

    eventSource = new EventSource(buildStreamURL(), { withCredentials: true })
    eventSource.addEventListener('audit', (event: MessageEvent<string>) => {
      try {
        handlePayload(JSON.parse(event.data) as AuditStreamPayload)
      } catch {
        // Ignore malformed SSE payloads.
      }
    })
    eventSource.onerror = () => {
      disconnect()
      reconnectTimer = window.setTimeout(() => {
        if (useAuthStore().isAuthenticated) {
          connect()
        }
      }, 5000)
    }
  }

  function disconnect(): void {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  async function markRead(id: string): Promise<void> {
    await markReadApi(id)
    items.value = items.value.map((item) =>
      item.id === id ? { ...item, is_read: true } : item,
    )
  }

  async function markAllRead(): Promise<void> {
    await markAllApi()
    items.value = items.value.map((item) => ({ ...item, is_read: true }))
  }

  async function remove(id: string): Promise<void> {
    await deleteApi(id)
    items.value = items.value.filter((item) => item.id !== id)
  }

  async function archiveAll(): Promise<void> {
    await archiveAllApi()
    items.value = []
  }

  return {
    items,
    unreadCount,
    load,
    connect,
    disconnect,
    markRead,
    markAllRead,
    remove,
    archiveAll,
  }
})
