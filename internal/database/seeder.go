package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/logging"
)

// EnsureBootstrapAdmin seeds the initial administrator when the users table is empty.
func EnsureBootstrapAdmin(db *sql.DB, cfg config.ServerConfig) error {
	def := config.DefaultServerConfig().Bootstrap
	email := strings.TrimSpace(cfg.Bootstrap.AdminEmail)
	if email == "" {
		email = def.AdminEmail
	}
	hash := cfg.BootstrapAdminPasswordHash()
	if hash == "" {
		hash = def.AdminPasswordHash
	}
	if !config.IsBcryptPasswordHash(hash) {
		return fmt.Errorf("bootstrap admin password hash is not a valid bcrypt hash")
	}

	var userCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if userCount > 0 {
		logging.Logger().Debug("users table not empty; skipping bootstrap admin seed", "user_count", userCount)
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

// SeedInitialAdmin is deprecated; use EnsureBootstrapAdmin.
func SeedInitialAdmin(db *sql.DB, cfg config.ServerConfig) error {
	return EnsureBootstrapAdmin(db, cfg)
}

func isPostgreSQL(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}
