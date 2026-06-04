export function parseBackendDetails(message?: string): Array<{ label: string; value: string }> {
  if (!message) {
    return []
  }

  const details: Array<{ label: string; value: string }> = []
  for (const segment of message.split(';')) {
    const trimmed = segment.trim()
    if (!trimmed) {
      continue
    }

    const separator = trimmed.indexOf('=')
    if (separator > 0) {
      const key = trimmed.slice(0, separator).trim()
      const value = trimmed.slice(separator + 1).trim()
      details.push({
        label: key.charAt(0).toUpperCase() + key.slice(1),
        value,
      })
      continue
    }

    if (trimmed.toLowerCase().startsWith('ca operational')) {
      details.push({ label: 'Status', value: 'Operational' })
    }
  }

  return details
}

export function downloadCertificate(filename: string, pem: string): void {
  const blob = new Blob([pem], { type: 'application/x-pem-file' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

export function formatCertDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function shortenFingerprint(fingerprint: string): string {
  const normalized = fingerprint.trim().toLowerCase()
  if (normalized.length <= 16) {
    return normalized
  }
  return `${normalized.slice(0, 8)}…${normalized.slice(-8)}`
}

export function formatUsageList(usages?: string[]): string {
  if (!usages || usages.length === 0) {
    return '—'
  }
  return usages.join(', ')
}
