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
	serialNumber := ""
	if cert.SerialNumber != nil {
		serialNumber = cert.SerialNumber.String()
	}
	return models.CACertificateInfo{
		Subject:            subjectInfoFromPKIX(cert.Subject),
		Issuer:             subjectInfoFromPKIX(cert.Issuer),
		NotBefore:          cert.NotBefore.UTC().Format(time.RFC3339),
		NotAfter:           cert.NotAfter.UTC().Format(time.RFC3339),
		SerialNumber:       serialNumber,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		KeyUsages:          keyUsageNames(cert.KeyUsage),
		ExtKeyUsages:       extKeyUsageNames(cert.ExtKeyUsage),
		Fingerprint:        strings.ToLower(hex.EncodeToString(sum[:])),
		PEM:                pem,
	}
}

func keyUsageNames(usage x509.KeyUsage) []string {
	type keyUsageEntry struct {
		flag x509.KeyUsage
		name string
	}
	entries := []keyUsageEntry{
		{x509.KeyUsageDigitalSignature, "digitalSignature"},
		{x509.KeyUsageContentCommitment, "contentCommitment"},
		{x509.KeyUsageKeyEncipherment, "keyEncipherment"},
		{x509.KeyUsageDataEncipherment, "dataEncipherment"},
		{x509.KeyUsageKeyAgreement, "keyAgreement"},
		{x509.KeyUsageCertSign, "certSign"},
		{x509.KeyUsageCRLSign, "crlSign"},
		{x509.KeyUsageEncipherOnly, "encipherOnly"},
		{x509.KeyUsageDecipherOnly, "decipherOnly"},
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if usage&entry.flag != 0 {
			out = append(out, entry.name)
		}
	}
	return out
}

func extKeyUsageNames(usages []x509.ExtKeyUsage) []string {
	if len(usages) == 0 {
		return nil
	}
	names := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageAny:                            "any",
		x509.ExtKeyUsageServerAuth:                     "serverAuth",
		x509.ExtKeyUsageClientAuth:                     "clientAuth",
		x509.ExtKeyUsageCodeSigning:                    "codeSigning",
		x509.ExtKeyUsageEmailProtection:                "emailProtection",
		x509.ExtKeyUsageIPSECEndSystem:                 "ipsecEndSystem",
		x509.ExtKeyUsageIPSECTunnel:                    "ipsecTunnel",
		x509.ExtKeyUsageIPSECUser:                      "ipsecUser",
		x509.ExtKeyUsageTimeStamping:                   "timeStamping",
		x509.ExtKeyUsageOCSPSigning:                    "ocspSigning",
		x509.ExtKeyUsageMicrosoftServerGatedCrypto:     "microsoftServerGatedCrypto",
		x509.ExtKeyUsageNetscapeServerGatedCrypto:      "netscapeServerGatedCrypto",
		x509.ExtKeyUsageMicrosoftCommercialCodeSigning: "microsoftCommercialCodeSigning",
		x509.ExtKeyUsageMicrosoftKernelCodeSigning:     "microsoftKernelCodeSigning",
	}

	out := make([]string, 0, len(usages))
	for _, usage := range usages {
		if name, ok := names[usage]; ok {
			out = append(out, name)
			continue
		}
		out = append(out, fmt.Sprintf("unknown(%d)", usage))
	}
	return out
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
