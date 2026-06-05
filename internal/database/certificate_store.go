package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// IssuedCertificate is a persisted public certificate record (no private key material).
type IssuedCertificate struct {
	Serial         string
	CommonName     string
	Subject        string
	CertificatePEM string
	NotBefore      time.Time
	NotAfter       time.Time
	RequestorID    string
	CreatedAt      time.Time
}

const issuedCertificatesDDL = `
CREATE TABLE IF NOT EXISTS issued_certificates (
	serial TEXT PRIMARY KEY,
	common_name TEXT NOT NULL,
	subject TEXT NOT NULL,
	certificate_pem TEXT NOT NULL,
	not_before TEXT NOT NULL,
	not_after TEXT NOT NULL,
	requestor_id TEXT NOT NULL,
	created_at TEXT NOT NULL
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

// Save inserts or replaces a public certificate record. Private keys must never be passed here.
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

	query := `INSERT INTO issued_certificates (
		serial, common_name, subject, certificate_pem, not_before, not_after, requestor_id, created_at
	) VALUES (` + s.placeholders(8) + `)
	ON CONFLICT(serial) DO UPDATE SET
		common_name = excluded.common_name,
		subject = excluded.subject,
		certificate_pem = excluded.certificate_pem,
		not_before = excluded.not_before,
		not_after = excluded.not_after,
		requestor_id = excluded.requestor_id,
		created_at = excluded.created_at`

	_, err := s.db.ExecContext(ctx, query,
		serial,
		strings.TrimSpace(rec.CommonName),
		strings.TrimSpace(rec.Subject),
		pem,
		rec.NotBefore.UTC().Format(time.RFC3339),
		rec.NotAfter.UTC().Format(time.RFC3339),
		requestorID,
		createdAt.Format(time.RFC3339),
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

	query := `SELECT serial, common_name, subject, certificate_pem, not_before, not_after, requestor_id, created_at
		FROM issued_certificates WHERE serial = ` + s.placeholder(1)

	var (
		rec          IssuedCertificate
		notBeforeRaw string
		notAfterRaw  string
		createdAtRaw string
	)

	err := s.db.QueryRowContext(ctx, query, serial).Scan(
		&rec.Serial,
		&rec.CommonName,
		&rec.Subject,
		&rec.CertificatePEM,
		&notBeforeRaw,
		&notAfterRaw,
		&rec.RequestorID,
		&createdAtRaw,
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

	return &rec, nil
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
