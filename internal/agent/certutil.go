package agent

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// GetCertTTL reads a PEM-encoded X.509 certificate from disk and returns the
// remaining validity period until NotAfter.
func GetCertTTL(certPath string) (time.Duration, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
		return 0, fmt.Errorf("read certificate %s: %w", certPath, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return 0, fmt.Errorf("decode PEM block from %s: no PEM data found", certPath)
	}
	if block.Type != "CERTIFICATE" {
		return 0, fmt.Errorf("decode PEM block from %s: expected CERTIFICATE, got %q", certPath, block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return 0, fmt.Errorf("parse certificate %s: %w", certPath, err)
	}

	return time.Until(cert.NotAfter), nil
}
