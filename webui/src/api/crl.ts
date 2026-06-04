import { apiClient } from './client'

export interface CRLStatus {
  available: boolean
  expiresAt: string | null
  format: 'der' | 'pem'
}

export async function fetchCRLStatus(): Promise<CRLStatus> {
  try {
    const response = await apiClient.get<Blob>('/crl', {
      params: { pem: '' },
      responseType: 'blob',
    })

    const expiresHeader = response.headers.expires
    const expiresAt =
      typeof expiresHeader === 'string' && expiresHeader.trim() !== '' ? expiresHeader : null

    return {
      available: true,
      expiresAt,
      format: 'pem',
    }
  } catch {
    return {
      available: false,
      expiresAt: null,
      format: 'pem',
    }
  }
}

export async function downloadCRL(format: 'der' | 'pem'): Promise<void> {
  const response = await apiClient.get<Blob>('/crl', {
    params: format === 'pem' ? { pem: '' } : undefined,
    responseType: 'blob',
  })

  const blob = response.data
  const filename =
    format === 'pem'
      ? 'crl.pem'
      : 'crl.crl'

  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
