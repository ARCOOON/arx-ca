package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.step.sm/crypto/pemutil"

	"github.com/ARCOOON/arx-ca/internal/database"
	"github.com/ARCOOON/arx-ca/internal/models"
)

func persistIssuedCertificate(
	ctx context.Context,
	store *database.CertificateStore,
	requestorID string,
	certPEM string,
) error {
	if store == nil {
		return nil
	}

	pemData := strings.TrimSpace(certPEM)
	if pemData == "" {
		return fmt.Errorf("certificate_pem is required")
	}

	cert, err := pemutil.ParseCertificate([]byte(pemData))
	if err != nil {
		return fmt.Errorf("parse issued certificate: %w", err)
	}

	rec := database.IssuedCertificate{
		Serial:         cert.SerialNumber.String(),
		CommonName:     cert.Subject.CommonName,
		Subject:        cert.Subject.String(),
		CertificatePEM: pemData,
		NotBefore:      cert.NotBefore.UTC(),
		NotAfter:       cert.NotAfter.UTC(),
		RequestorID:    requestorID,
		CreatedAt:      time.Now().UTC(),
	}

	return store.Save(ctx, rec)
}

func certificateRecordFromStore(rec *database.IssuedCertificate, revoked bool) models.CertificateRecordDetail {
	if rec == nil {
		return models.CertificateRecordDetail{}
	}

	detail := models.CertificateRecordDetail{
		Serial:         rec.Serial,
		CommonName:     rec.CommonName,
		Subject:        rec.Subject,
		NotBefore:      rec.NotBefore.UTC().Format(time.RFC3339),
		NotAfter:       rec.NotAfter.UTC().Format(time.RFC3339),
		RequestorID:    rec.RequestorID,
		CertificatePEM: rec.CertificatePEM,
		Revoked:        revoked,
	}

	if cert, err := pemutil.ParseCertificate([]byte(rec.CertificatePEM)); err == nil && cert != nil {
		detail.DNSNames = append([]string(nil), cert.DNSNames...)
		for _, ip := range cert.IPAddresses {
			detail.IPAddresses = append(detail.IPAddresses, ip.String())
		}
	}

	return detail
}
