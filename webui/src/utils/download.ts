function triggerBlobDownload(filename: string, blob: Blob): void {
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

export function downloadTextFile(filename: string, content: string, mimeType = 'application/x-pem-file'): void {
  triggerBlobDownload(filename, new Blob([content], { type: mimeType }))
}

export interface CertificateBundleInput {
  certificatePem: string
  privateKeyPem: string
}

/**
 * Builds a minimal ZIP archive (store only, no compression) containing only the leaf certificate and private key.
 */
export function downloadCertificateBundleZip(archiveName: string, input: CertificateBundleInput): void {
  const certificatePem = input.certificatePem.trim()
  const privateKeyPem = input.privateKeyPem.trim()

  const files: Array<{ name: string; data: Uint8Array }> = [
    { name: 'certificate.crt', data: new TextEncoder().encode(certificatePem) },
    { name: 'private.key', data: new TextEncoder().encode(privateKeyPem) },
  ]

  const chunks: Uint8Array[] = []
  let offset = 0
  const centralDirectory: Uint8Array[] = []

  for (const file of files) {
    const nameBytes = new TextEncoder().encode(file.name)
    const localHeader = new Uint8Array(30 + nameBytes.length)
    const view = new DataView(localHeader.buffer)

    view.setUint32(0, 0x04034b50, true)
    view.setUint16(8, 0, true)
    view.setUint16(26, nameBytes.length, true)
    view.setUint32(18, file.data.length, true)
    view.setUint32(22, file.data.length, true)
    view.setUint32(14, crc32(file.data), true)

    localHeader.set(nameBytes, 30)
    chunks.push(localHeader, file.data)

    const cdEntry = new Uint8Array(46 + nameBytes.length)
    const cdView = new DataView(cdEntry.buffer)
    cdView.setUint32(0, 0x02014b50, true)
    cdView.setUint16(10, 0, true)
    cdView.setUint16(28, nameBytes.length, true)
    cdView.setUint32(16, file.data.length, true)
    cdView.setUint32(20, file.data.length, true)
    cdView.setUint32(24, crc32(file.data), true)
    cdView.setUint32(42, offset, true)
    cdEntry.set(nameBytes, 46)
    centralDirectory.push(cdEntry)

    offset += localHeader.length + file.data.length
  }

  const centralSize = centralDirectory.reduce((sum, part) => sum + part.length, 0)
  const endRecord = new Uint8Array(22)
  const endView = new DataView(endRecord.buffer)
  endView.setUint32(0, 0x06054b50, true)
  endView.setUint16(8, files.length, true)
  endView.setUint16(10, files.length, true)
  endView.setUint32(12, centralSize, true)
  endView.setUint32(16, offset, true)

  const zipParts = [...chunks, ...centralDirectory, endRecord]
  const zipSize = zipParts.reduce((sum, part) => sum + part.length, 0)
  const zipBytes = new Uint8Array(zipSize)
  let position = 0
  for (const part of zipParts) {
    zipBytes.set(part, position)
    position += part.length
  }

  triggerBlobDownload(archiveName, new Blob([zipBytes], { type: 'application/zip' }))
}

function crc32(data: Uint8Array): number {
  let crc = 0xffffffff
  for (let index = 0; index < data.length; index += 1) {
    crc ^= data[index] ?? 0
    for (let bit = 0; bit < 8; bit += 1) {
      const mask = -(crc & 1)
      crc = (crc >>> 1) ^ (0xedb88320 & mask)
    }
  }
  return (crc ^ 0xffffffff) >>> 0
}
