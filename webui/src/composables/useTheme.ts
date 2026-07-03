import { onMounted, readonly, ref } from 'vue'

export type ThemePreference = 'light' | 'dark' | 'auto'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'arx_theme_preference'

const preference = ref<ThemePreference>('auto')
const resolvedTheme = ref<ResolvedTheme>('light')

let mediaQuery: MediaQueryList | null = null

function readStoredPreference(): ThemePreference {
  if (typeof window === 'undefined') return 'auto'
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'auto') return stored
  return 'auto'
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function resolveTheme(pref: ThemePreference): ResolvedTheme {
  return pref === 'auto' ? getSystemTheme() : pref
}

function applyResolvedTheme(theme: ResolvedTheme): void {
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

function syncTheme(): void {
  const next = resolveTheme(preference.value)
  resolvedTheme.value = next
  applyResolvedTheme(next)
}

function setPreference(next: ThemePreference): void {
  preference.value = next
  localStorage.setItem(STORAGE_KEY, next)
  syncTheme()
}

function onSystemThemeChange(): void {
  if (preference.value === 'auto') syncTheme()
}

export function initTheme(): void {
  preference.value = readStoredPreference()
  syncTheme()
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', onSystemThemeChange)
}

export function useTheme() {
  onMounted(() => syncTheme())
  return {
    preference: readonly(preference),
    resolvedTheme: readonly(resolvedTheme),
    setPreference,
  }
}
