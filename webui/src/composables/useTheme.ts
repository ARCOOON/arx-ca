import { onMounted, onUnmounted, ref, watch } from 'vue'

export type ThemePreference = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'arx_theme_preference'

const preference = ref<ThemePreference>('auto')
const resolved = ref<'light' | 'dark'>('light')

let mediaQuery: MediaQueryList | null = null

function readStoredPreference(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return stored
  }
  const legacy = localStorage.getItem('arx_theme')
  if (legacy === 'light' || legacy === 'dark') {
    return legacy
  }
  return 'auto'
}

function resolveTheme(pref: ThemePreference): 'light' | 'dark' {
  if (pref === 'light' || pref === 'dark') {
    return pref
  }
  return mediaQuery?.matches ? 'dark' : 'light'
}

export function applyTheme(theme: 'light' | 'dark') {
  resolved.value = theme
  document.documentElement.setAttribute('data-theme', theme)
}

function syncFromPreference() {
  applyTheme(resolveTheme(preference.value))
}

export function useTheme() {
  function setPreference(next: ThemePreference) {
    preference.value = next
    localStorage.setItem(STORAGE_KEY, next)
    syncFromPreference()
  }

  onMounted(() => {
    preference.value = readStoredPreference()
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    syncFromPreference()
    mediaQuery.addEventListener('change', syncFromPreference)
  })

  onUnmounted(() => {
    mediaQuery?.removeEventListener('change', syncFromPreference)
  })

  watch(preference, syncFromPreference)

  return {
    preference,
    resolved,
    setPreference,
  }
}
