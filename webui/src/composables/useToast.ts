import { ref } from 'vue'

export type ToastTone = 'success' | 'error'

export interface ToastMessage {
  id: number
  message: string
  tone: ToastTone
}

const toasts = ref<ToastMessage[]>([])
let nextId = 1

function dismissToast(id: number): void {
  toasts.value = toasts.value.filter((toast) => toast.id !== id)
}

export function useToast() {
  function showToast(message: string, tone: ToastTone = 'success', durationMs = 4500): void {
    const id = nextId++
    toasts.value = [...toasts.value, { id, message, tone }]
    window.setTimeout(() => dismissToast(id), durationMs)
  }

  return {
    toasts,
    showToast,
    dismissToast,
  }
}
