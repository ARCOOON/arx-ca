package database

import (
	"database/sql"
	"fmt"
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
	return nil
}
