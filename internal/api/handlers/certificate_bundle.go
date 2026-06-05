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
	certificatePEM := strings.TrimSpace(input.CertificatePEM)
	if !strings.HasSuffix(certificatePEM, "\n") {
		certificatePEM += "\n"
	}
	privateKeyPEM := strings.TrimSpace(input.PrivateKeyPEM)
	if !strings.HasSuffix(privateKeyPEM, "\n") {
		privateKeyPEM += "\n"
	}

	entries := []struct {
		name    string
		content string
	}{
		{name: "certificate.crt", content: certificatePEM},
		{name: "certificate.pem", content: certificatePEM},
		{name: "private.key", content: privateKeyPEM},
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %q: %w", entry.name, err)
		}
		if _, err := file.Write([]byte(entry.content)); err != nil {
			return nil, fmt.Errorf("write zip entry %q: %w", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}
