package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/your-org/arx-ca/internal/logging"
)

// User is a row from the application users table.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    string
}

// UserStore loads admin users from the application database.
type UserStore struct {
	db *sql.DB
}

// NewUserStore constructs a UserStore backed by db.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// GetByEmail returns the user with the given email address.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(email)
	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE email = ` + s.placeholder(1)
	logging.Logger().Debug("db query", "sql", query, "email", email)

	var u User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	logging.Logger().Debug("db query result", "user_id", u.ID, "email", u.Email, "role", u.Role)
	return &u, nil
}

func (s *UserStore) placeholder(n int) string {
	if isPostgreSQL(s.db) {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func isPostgreSQL(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}
