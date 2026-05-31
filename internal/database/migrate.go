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

// Migrate applies schema migrations for the application user store.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(usersTableDDL); err != nil {
		return fmt.Errorf("migrate users table: %w", err)
	}
	return nil
}
