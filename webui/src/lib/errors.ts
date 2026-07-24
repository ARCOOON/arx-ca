import { ApiError } from '@/api/client'

/** Extract a user-facing message from an unknown error thrown by the API layer. */
export function extractApiError(error: unknown, fallback = 'Something went wrong'): string {
  if (error instanceof ApiError) {
    return error.message || fallback
  }
  if (error instanceof Error) {
    return error.message || fallback
  }
  return fallback
}
