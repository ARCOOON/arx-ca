import { ref, watchEffect, onMounted, onUnmounted } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'arx_theme'

function getStoredMode(): ThemeMode {
  if (typeof window === 'undefined') return 'auto'
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'auto') return stored
  return 'auto'
}

function resolveEffective(mode: ThemeMode): 'light' | 'dark' {
  if (mode !== 'auto') return mode
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyEffective(effective: 'light' | 'dark'): void {
  document.documentElement.setAttribute('data-theme', effective)
}

const mode = ref<ThemeMode>(getStoredMode())

export function useTheme() {
  let mediaQuery: MediaQueryList | null = null

  function onSystemChange(): void {
    if (mode.value === 'auto') {
      applyEffective(resolveEffective('auto'))
    }
  }

  onMounted(() => {
    applyEffective(resolveEffective(mode.value))
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', onSystemChange)
  })

  onUnmounted(() => {
    mediaQuery?.removeEventListener('change', onSystemChange)
  })

  watchEffect(() => {
    if (typeof window === 'undefined') return
    localStorage.setItem(STORAGE_KEY, mode.value)
    applyEffective(resolveEffective(mode.value))
  })

  function setMode(m: ThemeMode): void {
    mode.value = m
  }

  return { mode, setMode }
}

export function initTheme(): void {
  const m = getStoredMode()
  if (typeof document !== 'undefined') {
    applyEffective(resolveEffective(m))
  }
}
