import { readonly, ref } from 'vue'

const STORAGE_KEY = 'arx_ui_show_api_hints'

const showApiHints = ref<boolean>(resolveInitialShowApiHints())

function resolveInitialShowApiHints(): boolean {
  if (typeof window === 'undefined') {
    return false
  }

  return localStorage.getItem(STORAGE_KEY) === 'true'
}

export function usePreferences() {
  function setShowApiHints(value: boolean): void {
    showApiHints.value = value
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, String(value))
    }
  }

  return {
    showApiHints: readonly(showApiHints),
    setShowApiHints,
  }
}
