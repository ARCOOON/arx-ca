import type { CertificateSummary } from '../types/api'

export type CertificateLifecycleStatus = 'valid' | 'revoked' | 'expired'

export function extractCommonName(subject: string): string {
  const match = subject.match(/(?:^|,\s*)CN=([^,]+)/i)
  if (match?.[1]) {
    return match[1].trim()
  }
  return subject
}

export function resolveCertificateStatus(certificate: CertificateSummary): CertificateLifecycleStatus {
  if (certificate.revoked) {
    return 'revoked'
  }

  const expiry = new Date(certificate.not_after)
  if (!Number.isNaN(expiry.getTime()) && expiry.getTime() < Date.now()) {
    return 'expired'
  }

  return 'valid'
}
