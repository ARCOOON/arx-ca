import { ref } from 'vue'

export type ToastVariant = 'default' | 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  message: string
  variant: ToastVariant
  duration: number
}

const toasts = ref<Toast[]>([])

function genId(): string {
  return Math.random().toString(36).slice(2, 9)
}

export function useToast() {
  function push(message: string, variant: ToastVariant = 'default', duration = 3500): void {
    const id = genId()
    toasts.value.push({ id, message, variant, duration })
    setTimeout(() => dismiss(id), duration)
  }

  function dismiss(id: string): void {
    const idx = toasts.value.findIndex((t) => t.id === id)
    if (idx !== -1) toasts.value.splice(idx, 1)
  }

  return {
    toasts,
    push,
    dismiss,
    success: (msg: string) => push(msg, 'success'),
    error: (msg: string) => push(msg, 'error'),
    warning: (msg: string) => push(msg, 'warning'),
    info: (msg: string) => push(msg, 'info'),
  }
}
