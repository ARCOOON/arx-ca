import type { AxiosError } from 'axios'

export function extractErrorMessage(err: unknown, fallback = 'An unexpected error occurred.'): string {
  if (!err) return fallback

  const axiosErr = err as AxiosError<{ error?: string; message?: string }>
  if (axiosErr.response?.data) {
    const data = axiosErr.response.data
    if (data.error) return data.error
    if (data.message) return data.message
  }
  if (axiosErr.message) return axiosErr.message

  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err

  return fallback
}
