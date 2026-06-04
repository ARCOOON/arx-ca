export type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'arx_theme'

export function resolveInitialTheme(): ThemeMode {
  if (typeof window === 'undefined') {
    return 'dark'
  }

  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark') {
    return stored
  }

  if (window.matchMedia('(prefers-color-scheme: light)').matches) {
    return 'light'
  }

  return 'dark'
}

export function applyTheme(theme: ThemeMode): void {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem(STORAGE_KEY, theme)
}

export function initTheme(): ThemeMode {
  const theme = resolveInitialTheme()
  applyTheme(theme)
  return theme
}

export function toggleTheme(current: ThemeMode): ThemeMode {
  const next: ThemeMode = current === 'dark' ? 'light' : 'dark'
  applyTheme(next)
  return next
}
