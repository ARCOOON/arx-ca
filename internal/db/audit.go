package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// AuditLog is an immutable forensic audit record persisted in the application database.
type AuditLog struct {
	ID          string
	Timestamp   time.Time
	RequestID   string
	IPAddress   string
	HTTPMethod  string
	Endpoint    string
	StatusCode  int
	ActorType   string
	ActorID     string
	ActorRoles  []string
	Action      string
	Provisioner string
	Fingerprint string
	Metadata    map[string]any
}

// AuditStore provides append-only persistence and read access for audit logs.
type AuditStore struct {
	db *sql.DB
}

// NewAuditStore constructs an AuditStore backed by db.
func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Migrate applies embedded application schema migrations.
func Migrate(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	migrations := []struct {
		path string
		name string
	}{
		{path: "migrations/001_audit_logs.sql", name: "audit_logs"},
		{path: "migrations/002_webhooks.sql", name: "webhooks"},
		{path: "migrations/003_notifications.sql", name: "notifications"},
		{path: "migrations/004_notifications_archive.sql", name: "notifications_archive"},
		{path: "migrations/005_ssh_certificates.sql", name: "ssh_certificates"},
	}
	for _, migration := range migrations {
		ddl, err := migrationFS.ReadFile(migration.path)
		if err != nil {
			return fmt.Errorf("read %s migration: %w", migration.name, err)
		}
		if _, err := db.Exec(string(ddl)); err != nil {
			return fmt.Errorf("migrate %s table: %w", migration.name, err)
		}
	}
	return nil
}

// Insert appends a single audit log entry. Updates and deletes are intentionally unsupported.
func (s *AuditStore) Insert(ctx context.Context, entry AuditLog) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("audit store is not initialized")
	}

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = uuid.NewString()
	}

	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	rolesJSON, err := json.Marshal(entry.ActorRoles)
	if err != nil {
		return fmt.Errorf("marshal actor_roles: %w", err)
	}
	metadata := entry.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var provisioner, fingerprint any
	if p := strings.TrimSpace(entry.Provisioner); p != "" {
		provisioner = p
	}
	if f := strings.TrimSpace(entry.Fingerprint); f != "" {
		fingerprint = f
	}

	query := `
INSERT INTO audit_logs (
	id, timestamp, request_id, ip_address, http_method, endpoint,
	status_code, actor_type, actor_id, actor_roles, action,
	provisioner, fingerprint, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []any{
		id,
		ts.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(entry.RequestID),
		strings.TrimSpace(entry.IPAddress),
		strings.TrimSpace(entry.HTTPMethod),
		strings.TrimSpace(entry.Endpoint),
		entry.StatusCode,
		strings.TrimSpace(entry.ActorType),
		strings.TrimSpace(entry.ActorID),
		string(rolesJSON),
		strings.TrimSpace(entry.Action),
		provisioner,
		fingerprint,
		string(metadataJSON),
	}

	if isPostgreSQL(s.db) {
		query = `
INSERT INTO audit_logs (
	id, timestamp, request_id, ip_address, http_method, endpoint,
	status_code, actor_type, actor_id, actor_roles, action,
	provisioner, fingerprint, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// ListResult contains a page of audit logs and the total matching row count.
type ListResult struct {
	Logs  []AuditLog
	Total int
}

// List returns audit logs ordered by timestamp descending with limit/offset pagination.
func (s *AuditStore) List(ctx context.Context, limit, offset int) (ListResult, error) {
	if s == nil || s.db == nil {
		return ListResult{}, fmt.Errorf("audit store is not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM audit_logs`
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	listQuery := `
SELECT id, timestamp, request_id, ip_address, http_method, endpoint,
	status_code, actor_type, actor_id, actor_roles, action,
	provisioner, fingerprint, metadata
FROM audit_logs
ORDER BY timestamp DESC
LIMIT ? OFFSET ?`
	listArgs := []any{limit, offset}
	if isPostgreSQL(s.db) {
		listQuery = `
SELECT id, timestamp, request_id, ip_address, http_method, endpoint,
	status_code, actor_type, actor_id, actor_roles, action,
	provisioner, fingerprint, metadata
FROM audit_logs
ORDER BY timestamp DESC
LIMIT $1 OFFSET $2`
		listArgs = []any{limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	logs := make([]AuditLog, 0, limit)
	for rows.Next() {
		var (
			log         AuditLog
			tsRaw       string
			rolesRaw    string
			metadataRaw string
			provisioner sql.NullString
			fingerprint sql.NullString
		)
		if err := rows.Scan(
			&log.ID,
			&tsRaw,
			&log.RequestID,
			&log.IPAddress,
			&log.HTTPMethod,
			&log.Endpoint,
			&log.StatusCode,
			&log.ActorType,
			&log.ActorID,
			&rolesRaw,
			&log.Action,
			&provisioner,
			&fingerprint,
			&metadataRaw,
		); err != nil {
			return ListResult{}, fmt.Errorf("scan audit log: %w", err)
		}

		parsedTS, err := time.Parse(time.RFC3339Nano, tsRaw)
		if err != nil {
			parsedTS, err = time.Parse(time.RFC3339, tsRaw)
			if err != nil {
				return ListResult{}, fmt.Errorf("parse audit timestamp: %w", err)
			}
		}
		log.Timestamp = parsedTS.UTC()

		if err := json.Unmarshal([]byte(rolesRaw), &log.ActorRoles); err != nil {
			log.ActorRoles = nil
		}
		if err := json.Unmarshal([]byte(metadataRaw), &log.Metadata); err != nil {
			log.Metadata = map[string]any{}
		}
		if provisioner.Valid {
			log.Provisioner = provisioner.String
		}
		if fingerprint.Valid {
			log.Fingerprint = fingerprint.String
		}

		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return ListResult{Logs: logs, Total: total}, nil
}

func isPostgreSQL(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}
