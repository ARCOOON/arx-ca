package ca

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	zcrypto "github.com/zmap/zcrypto/x509"
	"github.com/zmap/zlint/v3"
	"github.com/zmap/zlint/v3/lint"
	"go.step.sm/crypto/pemutil"

	"github.com/your-org/arx-ca/internal/models"
)

var complianceLintSources = lint.SourceList{
	lint.RFC5280,
	lint.CABFBaselineRequirements,
	lint.CABFEVGuidelines,
}

// LintCertificate parses a PEM certificate and returns RFC 5280 and CA/Browser Forum lint findings.
func (e *PKIEngine) LintCertificate(certificatePEM string) (*models.LintCertificateResponse, error) {
	if e == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	pemBytes := []byte(strings.TrimSpace(certificatePEM))
	if len(pemBytes) == 0 {
		return nil, errors.New("certificate_pem is required")
	}

	cert, err := pemutil.ParseCertificate(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	zcert, err := zcrypto.ParseCertificate(cert.Raw)
	if err != nil {
		return nil, fmt.Errorf("parse certificate for linting: %w", err)
	}

	registry, err := lint.GlobalRegistry().Filter(lint.FilterOptions{
		IncludeSources: complianceLintSources,
	})
	if err != nil {
		return nil, fmt.Errorf("configure lint registry: %w", err)
	}

	resultSet := zlint.LintCertificateEx(zcert, registry)
	if resultSet == nil {
		return nil, errors.New("lint execution failed")
	}

	findings := make([]models.CertificateLintFinding, 0)
	summary := models.LintCertificateSummary{}

	for name, result := range resultSet.Results {
		if result == nil || isPassingLintStatus(result.Status) {
			continue
		}

		finding := models.CertificateLintFinding{
			Lint:     name,
			Source:   string(result.LintMetadata.Source),
			Severity: strings.ToLower(result.Status.String()),
			Message:  strings.TrimSpace(result.Details),
		}
		findings = append(findings, finding)

		switch result.Status {
		case lint.Notice:
			summary.Notices++
		case lint.Warn:
			summary.Warnings++
		case lint.Error:
			summary.Errors++
		case lint.Fatal:
			summary.Fatals++
		}
	}

	return &models.LintCertificateResponse{
		Findings: findings,
		Summary:  summary,
	}, nil
}

func isPassingLintStatus(status lint.LintStatus) bool {
	switch status {
	case lint.Pass, lint.NA, lint.NE, lint.Reserved:
		return true
	default:
		return false
	}
}
