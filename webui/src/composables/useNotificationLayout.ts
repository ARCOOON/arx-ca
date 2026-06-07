import { readonly, ref } from 'vue'

export type NotificationLayoutStyle = 'drawer' | 'overlay'

const STORAGE_KEY = 'arx_ui_notification_style'

const layoutStyle = ref<NotificationLayoutStyle>(resolveInitialLayout())

function resolveInitialLayout(): NotificationLayoutStyle {
  if (typeof window === 'undefined') {
    return 'drawer'
  }

  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'drawer' || stored === 'overlay') {
    return stored
  }

  return 'drawer'
}

export function useNotificationLayout() {
  function setLayoutStyle(style: NotificationLayoutStyle): void {
    layoutStyle.value = style
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, style)
    }
  }

  return {
    layoutStyle: readonly(layoutStyle),
    setLayoutStyle,
  }
}
