import { toast } from 'vue-sonner'

export type ToastTone = 'success' | 'error'

export function useToast() {
  function showToast(message: string, tone: ToastTone = 'success', durationMs = 4500): void {
    if (tone === 'error') {
      toast.error(message, { duration: durationMs })
      return
    }
    toast.success(message, { duration: durationMs })
  }

  return {
    showToast,
  }
}
