//go:build windows

package ca

import (
	"errors"

	"github.com/your-org/arx-ca/internal/models"
)

// LintCertificate is not available on Windows builds.
func (e *PKIEngine) LintCertificate(certificatePEM string) (*models.LintCertificateResponse, error) {
	if e == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	return nil, errors.New("certificate linting is not supported on Windows builds")
}
