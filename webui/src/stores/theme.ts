import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'arx_theme'

function readStoredMode(): ThemeMode {
  if (typeof localStorage === 'undefined') {
    return 'auto'
  }
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return stored
  }
  return 'auto'
}

function systemPrefersDark(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) {
    return false
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolve(mode: ThemeMode): ResolvedTheme {
  if (mode === 'auto') {
    return systemPrefersDark() ? 'dark' : 'light'
  }
  return mode
}

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(readStoredMode())
  const resolved = ref<ResolvedTheme>(resolve(mode.value))

  let mediaQuery: MediaQueryList | null = null

  function apply(): void {
    resolved.value = resolve(mode.value)
    if (typeof document !== 'undefined') {
      document.documentElement.classList.toggle('dark', resolved.value === 'dark')
    }
  }

  function setMode(next: ThemeMode): void {
    mode.value = next
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, next)
    }
    apply()
  }

  /** Initialize the theme and keep 'auto' in sync with the OS preference. */
  function init(): void {
    apply()
    if (typeof window !== 'undefined' && window.matchMedia && !mediaQuery) {
      mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', () => {
        if (mode.value === 'auto') {
          apply()
        }
      })
    }
  }

  const isDark = computed(() => resolved.value === 'dark')

  return { mode, resolved, isDark, setMode, init }
})
