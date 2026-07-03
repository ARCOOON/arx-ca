import type { ApiEnvelope } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const DEFAULT_API_BASE_URL = 'https://localhost:8443/api/v1'

/**
 * Resolve the REST API base URL for the SPA.
 * - Explicit VITE_API_BASE_URL wins (build-time override).
 * - In the browser, default to same-origin /api/v1 (packaged WebUI + API proxy).
 * - Fall back to DEFAULT_API_BASE_URL for non-browser contexts.
 */
export function resolveApiBaseURL(): string {
  const envBase = import.meta.env.VITE_API_BASE_URL
  if (typeof envBase === 'string' && envBase.trim() !== '') {
    return envBase.trim().replace(/\/$/, '')
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}/api/v1`
  }

  return DEFAULT_API_BASE_URL
}

/** Error carrying the HTTP status and the backend-provided message. */
export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export type QueryParams = Record<string, string | number | boolean | undefined>

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  query?: QueryParams
}

function buildURL(path: string, query?: RequestOptions['query']): string {
  const base = resolveApiBaseURL()
  const url = new URL(`${base}${path}`)
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== '') {
        url.searchParams.set(key, String(value))
      }
    }
  }
  return url.toString()
}

function authHeaders(): Record<string, string> {
  const authStore = useAuthStore()
  const headers: Record<string, string> = {}
  if (authStore.token) {
    headers.Authorization = `Bearer ${authStore.token}`
  }
  return headers
}

async function handleUnauthorized(status: number, path: string): Promise<void> {
  if (status === 401 && !path.includes('/auth/login')) {
    const authStore = useAuthStore()
    await authStore.logout()
  }
}

/** Perform a JSON request against the API and unwrap the response envelope. */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query } = options

  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...authHeaders(),
  }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(buildURL(path, query), {
    method,
    headers,
    credentials: 'include',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  await handleUnauthorized(response.status, path)

  let envelope: ApiEnvelope<T> | null = null
  try {
    envelope = (await response.json()) as ApiEnvelope<T>
  } catch {
    envelope = null
  }

  if (!response.ok) {
    const message = envelope?.error ?? `Request failed with status ${response.status}`
    throw new ApiError(message, response.status)
  }

  if (envelope?.error) {
    throw new ApiError(envelope.error, response.status)
  }

  return (envelope?.data ?? null) as T
}

/** Perform a request that returns raw binary content (ZIP, CRL, key material). */
export async function requestBlob(path: string, options: RequestOptions = {}): Promise<Blob> {
  const { method = 'GET', body, query } = options

  const headers: Record<string, string> = { ...authHeaders() }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(buildURL(path, query), {
    method,
    headers,
    credentials: 'include',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  await handleUnauthorized(response.status, path)

  if (!response.ok) {
    throw new ApiError(`Request failed with status ${response.status}`, response.status)
  }

  return response.blob()
}
