import axios from 'axios'

export function extractApiError(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    const apiError = error.response?.data as { error?: string } | undefined
    return apiError?.error ?? error.message
  }

  if (error instanceof Error) {
    return error.message
  }

  return fallback
}
