package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SSHCertTypeUser = "user"
	SSHCertTypeHost = "host"
)

// SSHCertificate is a persisted SSH certificate record for auditing.
type SSHCertificate struct {
	ID          string
	Serial      string
	CertType    string
	Principals  []string
	Fingerprint string
	ValidAfter  time.Time
	ValidBefore time.Time
}

// SSHCertificateListOptions controls pagination for List.
type SSHCertificateListOptions struct {
	Limit  int
	Offset int
}

// SSHCertificateListResult contains a page of SSH certificates and the total count.
type SSHCertificateListResult struct {
	Certificates []SSHCertificate
	Total        int
}

// SSHCertificateStats aggregates SSH certificate counts for dashboard metrics.
type SSHCertificateStats struct {
	TotalUserCerts int
	TotalHostCerts int
	ActiveNow      int
}

// SSHCertificateStore provides persistence and read access for issued SSH certificates.
type SSHCertificateStore struct {
	db *sql.DB
}

// NewSSHCertificateStore constructs an SSHCertificateStore backed by db.
func NewSSHCertificateStore(db *sql.DB) *SSHCertificateStore {
	return &SSHCertificateStore{db: db}
}

// Insert persists a newly issued SSH certificate record.
func (s *SSHCertificateStore) Insert(ctx context.Context, entry SSHCertificate) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("ssh certificate store is not initialized")
	}

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = uuid.NewString()
	}

	serial := strings.TrimSpace(entry.Serial)
	if serial == "" {
		return fmt.Errorf("serial is required")
	}

	certType := normalizeSSHCertType(entry.CertType)
	if certType != SSHCertTypeUser && certType != SSHCertTypeHost {
		return fmt.Errorf("invalid cert_type %q", entry.CertType)
	}

	fingerprint := strings.TrimSpace(entry.Fingerprint)
	if fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	if entry.ValidAfter.IsZero() || entry.ValidBefore.IsZero() {
		return fmt.Errorf("valid_after and valid_before are required")
	}

	principals := entry.Principals
	if principals == nil {
		principals = []string{}
	}
	principalsJSON, err := json.Marshal(principals)
	if err != nil {
		return fmt.Errorf("marshal principals: %w", err)
	}

	query := `
INSERT INTO ssh_certificates (id, serial, cert_type, principals, fingerprint, valid_after, valid_before)
VALUES (?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		id,
		serial,
		certType,
		string(principalsJSON),
		fingerprint,
		entry.ValidAfter.UTC().Format(time.RFC3339),
		entry.ValidBefore.UTC().Format(time.RFC3339),
	}
	if isPostgreSQL(s.db) {
		query = `
INSERT INTO ssh_certificates (id, serial, cert_type, principals, fingerprint, valid_after, valid_before)
VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert ssh certificate: %w", err)
	}
	return nil
}

// List returns SSH certificates ordered by valid_before descending with pagination.
func (s *SSHCertificateStore) List(ctx context.Context, opts SSHCertificateListOptions) (SSHCertificateListResult, error) {
	if s == nil || s.db == nil {
		return SSHCertificateListResult{}, fmt.Errorf("ssh certificate store is not initialized")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ssh_certificates").Scan(&total); err != nil {
		return SSHCertificateListResult{}, fmt.Errorf("count ssh certificates: %w", err)
	}

	listQuery := `
SELECT id, serial, cert_type, principals, fingerprint, valid_after, valid_before
FROM ssh_certificates
ORDER BY valid_before DESC
LIMIT ? OFFSET ?`
	listArgs := []any{limit, offset}
	if isPostgreSQL(s.db) {
		listQuery = `
SELECT id, serial, cert_type, principals, fingerprint, valid_after, valid_before
FROM ssh_certificates
ORDER BY valid_before DESC
LIMIT $1 OFFSET $2`
	}

	rows, err := s.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return SSHCertificateListResult{}, fmt.Errorf("list ssh certificates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	certificates := make([]SSHCertificate, 0, limit)
	for rows.Next() {
		item, err := scanSSHCertificate(rows)
		if err != nil {
			return SSHCertificateListResult{}, err
		}
		certificates = append(certificates, item)
	}
	if err := rows.Err(); err != nil {
		return SSHCertificateListResult{}, fmt.Errorf("iterate ssh certificates: %w", err)
	}

	return SSHCertificateListResult{
		Certificates: certificates,
		Total:        total,
	}, nil
}

// Stats returns aggregate SSH certificate metrics.
func (s *SSHCertificateStore) Stats(ctx context.Context) (SSHCertificateStats, error) {
	if s == nil || s.db == nil {
		return SSHCertificateStats{}, fmt.Errorf("ssh certificate store is not initialized")
	}

	var stats SSHCertificateStats
	now := time.Now().UTC().Format(time.RFC3339)

	userQuery := "SELECT COUNT(*) FROM ssh_certificates WHERE cert_type = ?"
	hostQuery := "SELECT COUNT(*) FROM ssh_certificates WHERE cert_type = ?"
	activeQuery := `
SELECT COUNT(*) FROM ssh_certificates
WHERE valid_after <= ? AND valid_before >= ?`
	userArgs := []any{SSHCertTypeUser}
	hostArgs := []any{SSHCertTypeHost}
	activeArgs := []any{now, now}

	if isPostgreSQL(s.db) {
		userQuery = "SELECT COUNT(*) FROM ssh_certificates WHERE cert_type = $1"
		hostQuery = "SELECT COUNT(*) FROM ssh_certificates WHERE cert_type = $1"
		activeQuery = `
SELECT COUNT(*) FROM ssh_certificates
WHERE valid_after <= $1 AND valid_before >= $2`
	}

	if err := s.db.QueryRowContext(ctx, userQuery, userArgs...).Scan(&stats.TotalUserCerts); err != nil {
		return SSHCertificateStats{}, fmt.Errorf("count user ssh certificates: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, hostQuery, hostArgs...).Scan(&stats.TotalHostCerts); err != nil {
		return SSHCertificateStats{}, fmt.Errorf("count host ssh certificates: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, activeQuery, activeArgs...).Scan(&stats.ActiveNow); err != nil {
		return SSHCertificateStats{}, fmt.Errorf("count active ssh certificates: %w", err)
	}

	return stats, nil
}

func scanSSHCertificate(scanner interface {
	Scan(dest ...any) error
}) (SSHCertificate, error) {
	var (
		item           SSHCertificate
		principalsJSON string
		validAfterRaw  string
		validBeforeRaw string
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Serial,
		&item.CertType,
		&principalsJSON,
		&item.Fingerprint,
		&validAfterRaw,
		&validBeforeRaw,
	); err != nil {
		return SSHCertificate{}, fmt.Errorf("scan ssh certificate: %w", err)
	}

	if err := json.Unmarshal([]byte(principalsJSON), &item.Principals); err != nil {
		return SSHCertificate{}, fmt.Errorf("parse principals: %w", err)
	}
	if item.Principals == nil {
		item.Principals = []string{}
	}

	var err error
	item.ValidAfter, err = time.Parse(time.RFC3339, validAfterRaw)
	if err != nil {
		return SSHCertificate{}, fmt.Errorf("parse valid_after: %w", err)
	}
	item.ValidBefore, err = time.Parse(time.RFC3339, validBeforeRaw)
	if err != nil {
		return SSHCertificate{}, fmt.Errorf("parse valid_before: %w", err)
	}

	return item, nil
}

func normalizeSSHCertType(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == SSHCertTypeHost || strings.Contains(raw, "host") {
		return SSHCertTypeHost
	}
	return SSHCertTypeUser
}
