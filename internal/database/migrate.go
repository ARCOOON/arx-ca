package database

import (
	"database/sql"
	"fmt"

	auditdb "github.com/ARCOOON/arx-ca/internal/db"
)

const usersTableDDL = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL,
	created_at TEXT NOT NULL
);`

const acmeSchemaDDL = `
CREATE TABLE IF NOT EXISTS acme_accounts (
	id TEXT PRIMARY KEY,
	key_id TEXT NOT NULL UNIQUE,
	data TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS acme_nonces (
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS acme_orders (
	id TEXT PRIMARY KEY,
	account_id TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_acme_orders_account_id ON acme_orders(account_id);
CREATE TABLE IF NOT EXISTS acme_account_orders (
	account_id TEXT PRIMARY KEY,
	order_ids TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS acme_authorizations (
	id TEXT PRIMARY KEY,
	account_id TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_acme_authorizations_account_id ON acme_authorizations(account_id);
CREATE TABLE IF NOT EXISTS acme_challenges (
	id TEXT PRIMARY KEY,
	authz_id TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS acme_certificates (
	id TEXT PRIMARY KEY,
	serial TEXT,
	data TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_acme_certificates_serial ON acme_certificates(serial);
CREATE TABLE IF NOT EXISTS acme_eab_keys (
	id TEXT PRIMARY KEY,
	provisioner_id TEXT NOT NULL,
	reference_key TEXT,
	data TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_acme_eab_provisioner ON acme_eab_keys(provisioner_id);
CREATE TABLE IF NOT EXISTS acme_eab_provisioner_index (
	provisioner_id TEXT PRIMARY KEY,
	key_ids TEXT NOT NULL
);
`

// Migrate applies schema migrations for the application user store.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(usersTableDDL); err != nil {
		return fmt.Errorf("migrate users table: %w", err)
	}
	if _, err := db.Exec(acmeSchemaDDL); err != nil {
		return fmt.Errorf("migrate acme tables: %w", err)
	}
	if _, err := db.Exec(issuedCertificatesDDL); err != nil {
		return fmt.Errorf("migrate issued_certificates table: %w", err)
	}
	if err := migrateIssuedCertificatesEscrow(db); err != nil {
		return fmt.Errorf("migrate issued_certificates escrow column: %w", err)
	}
	if err := migrateIssuedCertificatesRevocation(db); err != nil {
		return fmt.Errorf("migrate issued_certificates revocation columns: %w", err)
	}
	if err := auditdb.Migrate(db); err != nil {
		return fmt.Errorf("migrate audit_logs table: %w", err)
	}
	return nil
}

func migrateIssuedCertificatesRevocation(db *sql.DB) error {
	columns := []struct {
		name       string
		columnType string
	}{
		{name: "status", columnType: "TEXT NOT NULL DEFAULT 'ACTIVE'"},
		{name: "revoked_at", columnType: "TEXT"},
		{name: "reason_code", columnType: "INTEGER"},
		{name: "revocation_reason", columnType: "TEXT"},
	}

	for _, column := range columns {
		if tableColumnExists(db, "issued_certificates", column.name) {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			`ALTER TABLE issued_certificates ADD COLUMN %s %s`,
			column.name,
			column.columnType,
		)); err != nil {
			return fmt.Errorf("add %s column: %w", column.name, err)
		}
	}

	if _, err := db.Exec(`UPDATE issued_certificates SET status = 'ACTIVE' WHERE status IS NULL OR TRIM(status) = ''`); err != nil {
		return fmt.Errorf("backfill issued_certificates status: %w", err)
	}
	return nil
}

func migrateIssuedCertificatesEscrow(db *sql.DB) error {
	if tableColumnExists(db, "issued_certificates", "encrypted_private_key") {
		return nil
	}

	columnType := "BLOB"
	if isPostgreSQL(db) {
		columnType = "BYTEA"
	}

	_, err := db.Exec(fmt.Sprintf(
		`ALTER TABLE issued_certificates ADD COLUMN encrypted_private_key %s`,
		columnType,
	))
	if err != nil {
		return fmt.Errorf("add encrypted_private_key column: %w", err)
	}
	return nil
}

func tableColumnExists(db *sql.DB, tableName, columnName string) bool {
	if isPostgreSQL(db) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = CURRENT_SCHEMA()
					AND table_name = $1
					AND column_name = $2
			)`,
			tableName,
			columnName,
		).Scan(&exists)
		return err == nil && exists
	}

	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnTyp    string
			notNull      int
			defaultValue any
			pk           int
		)
		if err := rows.Scan(&cid, &name, &columnTyp, &notNull, &defaultValue, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}
