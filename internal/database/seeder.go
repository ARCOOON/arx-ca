package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/logging"
)

// EnsureBootstrapAdmin inserts the configured bootstrap admin when no user exists with admin_email.
func EnsureBootstrapAdmin(db *sql.DB, cfg config.Bootstrap) error {
	def := config.DefaultServerConfig().Bootstrap
	email := strings.TrimSpace(cfg.AdminEmail)
	if email == "" {
		email = def.AdminEmail
	}
	hash := strings.TrimSpace(cfg.AdminPasswordHash)
	if hash == "" {
		hash = def.AdminPasswordHash
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = ` + userEmailPlaceholder(db)
	logging.Logger().Debug("db query", "sql", query, "email", email)
	if err := db.QueryRow(query, email).Scan(&count); err != nil {
		return fmt.Errorf("lookup bootstrap admin user: %w", err)
	}
	if count > 0 {
		logging.Logger().Debug("bootstrap admin already present", "email", email)
		return nil
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	role := string(auth.RoleSuperAdmin)

	logging.Logger().Info("creating bootstrap admin user", "email", email)
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

func userEmailPlaceholder(db *sql.DB) string {
	if isPostgreSQL(db) {
		return "$1"
	}
	return "?"
}

// SeedInitialAdmin is deprecated; use EnsureBootstrapAdmin.
func SeedInitialAdmin(db *sql.DB, cfg config.Bootstrap) error {
	return EnsureBootstrapAdmin(db, cfg)
}

func isPostgreSQL(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}
