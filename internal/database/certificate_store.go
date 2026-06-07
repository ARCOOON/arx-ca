package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	// CertificateStatusActive indicates an issued certificate that has not been revoked.
	CertificateStatusActive = "ACTIVE"
	// CertificateStatusRevoked indicates a certificate revoked in the application archive.
	CertificateStatusRevoked = "REVOKED"
)

// IssuedCertificate is a persisted certificate record. Private key material is stored
// only as AES-256-GCM encrypted bytes when escrow is enabled for native generation.
type IssuedCertificate struct {
	Serial              string
	CommonName          string
	Subject             string
	CertificatePEM      string
	EncryptedPrivateKey []byte
	NotBefore           time.Time
	NotAfter            time.Time
	RequestorID         string
	CreatedAt           time.Time
	Status              string
	RevokedAt           *time.Time
	ReasonCode          *int
	RevocationReason    string
}

const issuedCertificatesDDL = `
CREATE TABLE IF NOT EXISTS issued_certificates (
	serial TEXT PRIMARY KEY,
	common_name TEXT NOT NULL,
	subject TEXT NOT NULL,
	certificate_pem TEXT NOT NULL,
	encrypted_private_key BLOB,
	not_before TEXT NOT NULL,
	not_after TEXT NOT NULL,
	requestor_id TEXT NOT NULL,
	created_at TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'ACTIVE',
	revoked_at TEXT,
	reason_code INTEGER,
	revocation_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_issued_certificates_requestor_id ON issued_certificates(requestor_id);
CREATE INDEX IF NOT EXISTS idx_issued_certificates_not_after ON issued_certificates(not_after);
`

// CertificateStore persists issued public certificates in the application database.
type CertificateStore struct {
	db *sql.DB
}

// NewCertificateStore constructs a CertificateStore backed by db.
func NewCertificateStore(db *sql.DB) *CertificateStore {
	return &CertificateStore{db: db}
}

// Save inserts or replaces a certificate record. EncryptedPrivateKey is optional and
// preserved on upsert when omitted.
func (s *CertificateStore) Save(ctx context.Context, rec IssuedCertificate) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("certificate store is not initialized")
	}

	serial := strings.TrimSpace(rec.Serial)
	if serial == "" {
		return fmt.Errorf("serial is required")
	}
	pem := strings.TrimSpace(rec.CertificatePEM)
	if pem == "" {
		return fmt.Errorf("certificate_pem is required")
	}
	requestorID := strings.TrimSpace(rec.RequestorID)
	if requestorID == "" {
		return fmt.Errorf("requestor_id is required")
	}

	createdAt := rec.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	status := strings.TrimSpace(rec.Status)
	if status == "" {
		status = CertificateStatusActive
	}

	query := `INSERT INTO issued_certificates (
		serial, common_name, subject, certificate_pem, encrypted_private_key, not_before, not_after, requestor_id, created_at, status
	) VALUES (` + s.placeholders(10) + `)
	ON CONFLICT(serial) DO UPDATE SET
		common_name = excluded.common_name,
		subject = excluded.subject,
		certificate_pem = excluded.certificate_pem,
		encrypted_private_key = COALESCE(excluded.encrypted_private_key, issued_certificates.encrypted_private_key),
		not_before = excluded.not_before,
		not_after = excluded.not_after,
		requestor_id = excluded.requestor_id,
		created_at = excluded.created_at,
		status = CASE
			WHEN issued_certificates.status = '` + CertificateStatusRevoked + `' THEN issued_certificates.status
			ELSE excluded.status
		END`

	var encryptedKey any
	if len(rec.EncryptedPrivateKey) > 0 {
		encryptedKey = rec.EncryptedPrivateKey
	}

	_, err := s.db.ExecContext(ctx, query,
		serial,
		strings.TrimSpace(rec.CommonName),
		strings.TrimSpace(rec.Subject),
		pem,
		encryptedKey,
		rec.NotBefore.UTC().Format(time.RFC3339),
		rec.NotAfter.UTC().Format(time.RFC3339),
		requestorID,
		createdAt.Format(time.RFC3339),
		status,
	)
	if err != nil {
		return fmt.Errorf("save issued certificate: %w", err)
	}
	return nil
}

// GetBySerial returns a persisted certificate record by serial number.
func (s *CertificateStore) GetBySerial(ctx context.Context, serial string) (*IssuedCertificate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("certificate store is not initialized")
	}

	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, fmt.Errorf("serial is required")
	}

	query := `SELECT serial, common_name, subject, certificate_pem, encrypted_private_key, not_before, not_after, requestor_id, created_at, status, revoked_at, reason_code, revocation_reason
		FROM issued_certificates WHERE serial = ` + s.placeholder(1)

	var (
		rec              IssuedCertificate
		notBeforeRaw     string
		notAfterRaw      string
		createdAtRaw     string
		encryptedKeyBlob []byte
		revokedAtRaw     sql.NullString
		reasonCodeRaw    sql.NullInt64
		revocationReason sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, serial).Scan(
		&rec.Serial,
		&rec.CommonName,
		&rec.Subject,
		&rec.CertificatePEM,
		&encryptedKeyBlob,
		&notBeforeRaw,
		&notAfterRaw,
		&rec.RequestorID,
		&createdAtRaw,
		&rec.Status,
		&revokedAtRaw,
		&reasonCodeRaw,
		&revocationReason,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("certificate record not found")
		}
		return nil, fmt.Errorf("load issued certificate: %w", err)
	}

	rec.NotBefore, err = time.Parse(time.RFC3339, notBeforeRaw)
	if err != nil {
		return nil, fmt.Errorf("parse not_before: %w", err)
	}
	rec.NotAfter, err = time.Parse(time.RFC3339, notAfterRaw)
	if err != nil {
		return nil, fmt.Errorf("parse not_after: %w", err)
	}
	rec.CreatedAt, err = time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if len(encryptedKeyBlob) > 0 {
		rec.EncryptedPrivateKey = append([]byte(nil), encryptedKeyBlob...)
	}
	if strings.TrimSpace(rec.Status) == "" {
		rec.Status = CertificateStatusActive
	}
	if revokedAtRaw.Valid && strings.TrimSpace(revokedAtRaw.String) != "" {
		revokedAt, parseErr := time.Parse(time.RFC3339, revokedAtRaw.String)
		if parseErr == nil {
			rec.RevokedAt = &revokedAt
		}
	}
	if reasonCodeRaw.Valid {
		code := int(reasonCodeRaw.Int64)
		rec.ReasonCode = &code
	}
	if revocationReason.Valid {
		rec.RevocationReason = revocationReason.String
	}

	return &rec, nil
}

// MarkRevoked updates revocation metadata for a persisted certificate record.
func (s *CertificateStore) MarkRevoked(ctx context.Context, serial string, reasonCode int, reason string, revokedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("certificate store is not initialized")
	}

	serial = strings.TrimSpace(serial)
	if serial == "" {
		return fmt.Errorf("serial is required")
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}

	query := `UPDATE issued_certificates
		SET status = ` + s.placeholder(1) + `,
			revoked_at = ` + s.placeholder(2) + `,
			reason_code = ` + s.placeholder(3) + `,
			revocation_reason = ` + s.placeholder(4) + `
		WHERE serial = ` + s.placeholder(5)

	result, err := s.db.ExecContext(ctx, query,
		CertificateStatusRevoked,
		revokedAt.UTC().Format(time.RFC3339),
		reasonCode,
		strings.TrimSpace(reason),
		serial,
	)
	if err != nil {
		return fmt.Errorf("mark certificate revoked: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark certificate revoked rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("certificate record not found")
	}
	return nil
}

func (s *CertificateStore) placeholder(n int) string {
	if isPostgreSQL(s.db) {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s *CertificateStore) placeholders(count int) string {
	parts := make([]string, count)
	for i := 1; i <= count; i++ {
		parts[i-1] = s.placeholder(i)
	}
	return strings.Join(parts, ", ")
}
