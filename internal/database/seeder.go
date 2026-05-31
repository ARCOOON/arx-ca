package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/config"
)

// SeedInitialAdmin inserts the configured bootstrap admin when the users table is empty.
func SeedInitialAdmin(db *sql.DB, cfg config.Bootstrap) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	def := config.DefaultServerConfig().Bootstrap
	email := strings.TrimSpace(cfg.AdminEmail)
	if email == "" {
		email = def.AdminEmail
	}
	hash := strings.TrimSpace(cfg.AdminPasswordHash)
	if hash == "" {
		hash = def.AdminPasswordHash
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	role := string(auth.RoleSuperAdmin)

	if isPostgreSQL(db) {
		_, err := db.Exec(
			`INSERT INTO users (id, email, password_hash, role, created_at) VALUES ($1, $2, $3, $4, $5)`,
			id, email, hash, role, createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert bootstrap admin user: %w", err)
		}
	} else {
		_, err := db.Exec(
			`INSERT INTO users (id, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, email, hash, role, createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert bootstrap admin user: %w", err)
		}
	}
	return nil
}

func isPostgreSQL(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}
