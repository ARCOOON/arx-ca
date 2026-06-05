package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

type certificateBundleInput struct {
	CertificatePEM string
	PrivateKeyPEM  string
}

func buildCertificateBundleZip(input certificateBundleInput) ([]byte, error) {
	if strings.TrimSpace(input.CertificatePEM) == "" {
		return nil, fmt.Errorf("certificate_pem is required")
	}
	if strings.TrimSpace(input.PrivateKeyPEM) == "" {
		return nil, fmt.Errorf("private_key_pem is required")
	}
	certificatePEM := input.CertificatePEM
	privateKeyPEM := input.PrivateKeyPEM

	files := map[string]string{
		"certificate.crt": certificatePEM,
		"private.key":     privateKeyPEM,
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %q: %w", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			return nil, fmt.Errorf("write zip entry %q: %w", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}
