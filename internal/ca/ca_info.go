package ca

import (
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.step.sm/crypto/pemutil"

	"github.com/ARCOOON/arx-ca/internal/models"
)

// CAInfo returns parsed metadata for the active Root and Intermediate CA certificates.
func (e *PKIEngine) CAInfo() (models.CAInfoResponse, error) {
	if e == nil || e.auth == nil {
		return models.CAInfoResponse{}, fmt.Errorf("CA engine is not initialized")
	}

	rootPEM := e.RootCertPEM()
	if len(rootPEM) == 0 {
		return models.CAInfoResponse{}, fmt.Errorf("root certificate is unavailable")
	}

	rootCert, err := pemutil.ParseCertificate(rootPEM)
	if err != nil {
		return models.CAInfoResponse{}, fmt.Errorf("parse root certificate: %w", err)
	}

	intermediatePEM := e.IntermediateCertPEM()
	if len(intermediatePEM) == 0 {
		return models.CAInfoResponse{}, fmt.Errorf("intermediate certificate is unavailable")
	}

	intermediateCert, err := pemutil.ParseCertificate(intermediatePEM)
	if err != nil {
		return models.CAInfoResponse{}, fmt.Errorf("parse intermediate certificate: %w", err)
	}

	return models.CAInfoResponse{
		Root:         certificateInfoFromX509(rootCert, string(rootPEM)),
		Intermediate: certificateInfoFromX509(intermediateCert, string(intermediatePEM)),
	}, nil
}

func certificateInfoFromX509(cert *x509.Certificate, pem string) models.CACertificateInfo {
	if cert == nil {
		return models.CACertificateInfo{PEM: pem}
	}
	sum := sha256.Sum256(cert.Raw)
	return models.CACertificateInfo{
		Subject:     subjectInfoFromPKIX(cert.Subject),
		Issuer:      subjectInfoFromPKIX(cert.Issuer),
		NotBefore:   cert.NotBefore.UTC().Format(time.RFC3339),
		NotAfter:    cert.NotAfter.UTC().Format(time.RFC3339),
		Fingerprint: strings.ToLower(hex.EncodeToString(sum[:])),
		PEM:         pem,
	}
}

func subjectInfoFromPKIX(name pkix.Name) models.CASubjectInfo {
	return models.CASubjectInfo{
		CommonName:         name.CommonName,
		Organization:       append([]string(nil), name.Organization...),
		OrganizationalUnit: append([]string(nil), name.OrganizationalUnit...),
		Country:            append([]string(nil), name.Country...),
		Province:           append([]string(nil), name.Province...),
		Locality:           append([]string(nil), name.Locality...),
		StreetAddress:      append([]string(nil), name.StreetAddress...),
		PostalCode:         append([]string(nil), name.PostalCode...),
		SerialNumber:       name.SerialNumber,
	}
}
