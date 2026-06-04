const DNS_NAME_PATTERN =
  /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*$/i

const IPV4_PATTERN =
  /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d?\d)$/

function isIPv6(value: string): boolean {
  if (!value.includes(':')) {
    return false
  }
  try {
    const url = new URL(`http://[${value}]`)
    return url.hostname.includes(':')
  } catch {
    return false
  }
}

export function isValidIPAddress(value: string): boolean {
  const trimmed = value.trim()
  if (IPV4_PATTERN.test(trimmed)) {
    return true
  }
  return isIPv6(trimmed)
}

export function isValidDNSName(value: string): boolean {
  const name = value.trim()
  if (!name || name.length > 253 || name.startsWith('.') || name.endsWith('.') || name.includes('..')) {
    return false
  }
  return DNS_NAME_PATTERN.test(name)
}

export function parseSansInput(raw: string): { sans: string[]; error?: string } {
  const entries = raw
    .split(/[\n,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean)

  const sans: string[] = []
  const seen = new Set<string>()

  for (const entry of entries) {
    const key = entry.toLowerCase()
    if (seen.has(key)) {
      continue
    }

    if (isValidIPAddress(entry)) {
      seen.add(key)
      sans.push(entry)
      continue
    }

    if (isValidDNSName(entry)) {
      seen.add(key)
      sans.push(entry)
      continue
    }

    return {
      sans: [],
      error: `"${entry}" is not a valid DNS name or IP address`,
    }
  }

  return { sans }
}
