package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

type certificateBundleInput struct {
	CertificatePEM  string
	PrivateKeyPEM   string
	IntermediatePEM string
	RootPEM         string
}

func buildCertificateBundleZip(input certificateBundleInput) ([]byte, error) {
	certificatePEM := strings.TrimSpace(input.CertificatePEM)
	if certificatePEM == "" {
		return nil, fmt.Errorf("certificate_pem is required")
	}
	privateKeyPEM := strings.TrimSpace(input.PrivateKeyPEM)
	if privateKeyPEM == "" {
		return nil, fmt.Errorf("private_key_pem is required")
	}
	intermediatePEM := strings.TrimSpace(input.IntermediatePEM)
	if intermediatePEM == "" {
		return nil, fmt.Errorf("intermediate certificate is unavailable")
	}
	rootPEM := strings.TrimSpace(input.RootPEM)
	if rootPEM == "" {
		return nil, fmt.Errorf("root certificate is unavailable")
	}

	fullChain := certificatePEM
	if !strings.HasSuffix(fullChain, "\n") {
		fullChain += "\n"
	}
	fullChain += intermediatePEM

	files := map[string]string{
		"certificate.crt": certificatePEM,
		"certificate.pem": certificatePEM,
		"private.key":     privateKeyPEM,
		"fullchain.pem":   fullChain,
		"ca.crt":          rootPEM,
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
