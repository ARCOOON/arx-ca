/** Format an ISO timestamp into a locale date-time string. */
export function formatDateTime(value: string | number | undefined | null): string {
  if (value === undefined || value === null || value === '') {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Format an ISO timestamp into a locale date string (no time). */
export function formatDate(value: string | number | undefined | null): string {
  if (value === undefined || value === null || value === '') {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
  })
}

/** Human-readable byte size. */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.floor(Math.log(bytes) / Math.log(1024))
  const value = bytes / 1024 ** index
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

/** Whether a certificate is expiring within `days` days from now. */
export function isExpiringSoon(notAfter: string, days = 30): boolean {
  const expiry = new Date(notAfter).getTime()
  if (Number.isNaN(expiry)) {
    return false
  }
  const threshold = Date.now() + days * 24 * 60 * 60 * 1000
  return expiry > Date.now() && expiry <= threshold
}
