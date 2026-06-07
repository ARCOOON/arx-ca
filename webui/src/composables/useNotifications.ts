import { computed, ref } from 'vue'
import {
  archiveAllNotifications as archiveAllNotificationsApi,
  deleteNotification as deleteNotificationApi,
  listNotifications,
  markAllNotificationsRead as markAllNotificationsReadApi,
  markNotificationRead,
} from '../api/notifications'
import { resolveApiBaseURL } from '../api/client'
import { useAuthStore } from '../store/auth'
import type { NotificationEntry } from '../types/api'

export interface AuditNotificationPayload {
  notification_id?: string
  timestamp: string
  action: string
  actor: {
    type: string
    id: string
    roles?: string[]
  }
  ip_address: string
  resource: {
    provisioner?: string
    fingerprint?: string
  }
  metadata?: Record<string, unknown>
  request_id?: string
  http_method?: string
  endpoint?: string
  status_code?: number
}

export interface UINotification {
  id: string
  action: string
  message: string
  timestamp: string
  sticky: boolean
  tone: 'critical' | 'info' | 'success'
}

const MAX_TOAST_NOTIFICATIONS = 50
const AUTO_DISMISS_MS = 7000
const MAX_PERSISTENT_NOTIFICATIONS = 200

const CRITICAL_ACTIONS = new Set(['AUTH_LOGIN_FAILED', 'CERT_REVOKE', 'EAB_REVOKE'])

const notifications = ref<UINotification[]>([])
const persistentNotifications = ref<NotificationEntry[]>([])
const unreadCount = computed(
  () => persistentNotifications.value.filter((item) => !item.is_read).length,
)

let eventSource: EventSource | null = null
let nextToastId = 1

function isCriticalAction(action: string): boolean {
  return CRITICAL_ACTIONS.has(action.trim().toUpperCase())
}

function levelForAction(action: string): NotificationEntry['level'] {
  return isCriticalAction(action) ? 'critical' : 'info'
}

function toneForAction(action: string): UINotification['tone'] {
  if (isCriticalAction(action)) {
    return 'critical'
  }
  if (action.includes('SUCCESS') || action.includes('CREATED') || action.includes('ISSUE')) {
    return 'success'
  }
  return 'info'
}

function formatMessage(payload: AuditNotificationPayload): string {
  const actor = payload.actor?.id?.trim() || 'system'
  const action = payload.action.replaceAll('_', ' ').toLowerCase()
  return `${actor}: ${action}`
}

function scheduleAutoRemove(id: string): void {
  window.setTimeout(() => {
    removeToast(id)
  }, AUTO_DISMISS_MS)
}

function pushToast(payload: AuditNotificationPayload): void {
  const id = String(nextToastId++)
  const sticky = isCriticalAction(payload.action)
  const entry: UINotification = {
    id,
    action: payload.action,
    message: formatMessage(payload),
    timestamp: payload.timestamp,
    sticky,
    tone: toneForAction(payload.action),
  }

  notifications.value = [entry, ...notifications.value].slice(0, MAX_TOAST_NOTIFICATIONS)

  if (!sticky) {
    scheduleAutoRemove(id)
  }
}

function prependPersistent(entry: NotificationEntry): void {
  const exists = persistentNotifications.value.some((item) => item.id === entry.id)
  if (exists) {
    return
  }
  persistentNotifications.value = [entry, ...persistentNotifications.value].slice(
    0,
    MAX_PERSISTENT_NOTIFICATIONS,
  )
}

function handleAuditPayload(payload: AuditNotificationPayload): void {
  pushToast(payload)

  const notificationId = payload.notification_id?.trim()
  if (!notificationId) {
    return
  }

  prependPersistent({
    id: notificationId,
    action: payload.action,
    level: levelForAction(payload.action),
    message: formatMessage(payload),
    timestamp: payload.timestamp,
    is_read: false,
    metadata: payload.metadata,
  })
}

async function loadPersistentNotifications(): Promise<void> {
  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    return
  }

  try {
    const data = await listNotifications({ limit: 100 })
    persistentNotifications.value = data.notifications
  } catch {
    // Keep existing drawer state when the history API is temporarily unavailable.
  }
}

function buildStreamURL(): string {
  const base = resolveApiBaseURL().replace(/\/$/, '')
  const authStore = useAuthStore()
  const url = new URL(`${base}/notifications/stream`)
  if (authStore.token) {
    url.searchParams.set('access_token', authStore.token)
  }
  return url.toString()
}

function connect(): void {
  if (eventSource) {
    return
  }

  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    return
  }

  void loadPersistentNotifications()

  eventSource = new EventSource(buildStreamURL(), { withCredentials: true })

  eventSource.addEventListener('audit', (event: MessageEvent<string>) => {
    try {
      const payload = JSON.parse(event.data) as AuditNotificationPayload
      handleAuditPayload(payload)
    } catch {
      // Ignore malformed SSE payloads.
    }
  })

  eventSource.onerror = () => {
    disconnect()
    window.setTimeout(() => {
      const authStoreRetry = useAuthStore()
      if (authStoreRetry.isAuthenticated) {
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
}

function removeToast(id: string): void {
  notifications.value = notifications.value.filter((item) => item.id !== id)
}

function clearAllToasts(): void {
  notifications.value = []
}

async function markAsRead(id: string): Promise<void> {
  await markNotificationRead(id)
  persistentNotifications.value = persistentNotifications.value.map((item) =>
    item.id === id ? { ...item, is_read: true } : item,
  )
}

async function markAllAsRead(): Promise<void> {
  await markAllNotificationsReadApi()
  persistentNotifications.value = persistentNotifications.value.map((item) => ({
    ...item,
    is_read: true,
  }))
}

async function deletePersistent(id: string): Promise<void> {
  await deleteNotificationApi(id)
  persistentNotifications.value = persistentNotifications.value.filter((item) => item.id !== id)
}

async function archiveAllPersistent(): Promise<void> {
  await archiveAllNotificationsApi()
  persistentNotifications.value = []
}

export function useNotifications() {
  return {
    notifications,
    persistentNotifications,
    unreadCount,
    connect,
    disconnect,
    loadPersistentNotifications,
    remove: removeToast,
    clearAll: clearAllToasts,
    markAsRead,
    markAllAsRead,
    deletePersistent,
    archiveAllPersistent,
  }
}
