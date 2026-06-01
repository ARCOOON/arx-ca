package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/your-org/arx-ca/internal/repository"
)

// LoginFailureReason describes an internal auth failure point for debug logging only.
type LoginFailureReason string

const (
	LoginFailureMissingEmail    LoginFailureReason = "missing email"
	LoginFailureMissingPassword LoginFailureReason = "missing password"
	LoginFailureUserNotFound    LoginFailureReason = "user not found in db"
	LoginFailureBcryptMismatch  LoginFailureReason = "bcrypt hash mismatch"
	LoginFailureDatabaseError   LoginFailureReason = "database error during credential lookup"
)

// AuthenticateUser verifies email and password against the users table.
// On failure it returns ErrInvalidCredentials and a LoginFailureReason for server-side logging.
func AuthenticateUser(ctx context.Context, store *repository.UserStore, email, password string) (*repository.User, LoginFailureReason, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, LoginFailureMissingEmail, ErrInvalidCredentials
	}
	if password == "" {
		return nil, LoginFailureMissingPassword, ErrInvalidCredentials
	}

	user, err := store.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, LoginFailureUserNotFound, ErrInvalidCredentials
		}
		return nil, LoginFailureDatabaseError, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, LoginFailureBcryptMismatch, ErrInvalidCredentials
		}
		return nil, LoginFailureDatabaseError, err
	}

	return user, "", nil
}

// RolesForUser returns RBAC roles for a database user record.
func RolesForUser(user *repository.User) []Role {
	if user == nil {
		return []Role{RoleSuperAdmin}
	}
	role := Role(strings.TrimSpace(user.Role))
	if ValidRole(role) {
		return []Role{role}
	}
	return nil
}
