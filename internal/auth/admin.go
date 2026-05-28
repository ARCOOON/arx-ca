package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BootstrapAdminUsername is the fixed admin account used for initial JWT login.
	BootstrapAdminUsername = "admin"

	// bootstrapAdminPasswordHash is the bcrypt hash (cost 12) of the bootstrap password
	// "ArxRootCA-Bootstrap-Admin-2026!". Change this hash after first login in production.
	bootstrapAdminPasswordHash = "$2a$12$uXEIayfOof1VT.7r.gxqpeuGvZxhRXxYNFrBDr7yZDOZ3GYVKVyPq"
)

var (
	// ErrInvalidCredentials is returned when admin username or password is wrong.
	ErrInvalidCredentials = errors.New("invalid admin credentials")
)

// ValidateAdminCredentials checks the bootstrap admin username and password.
func ValidateAdminCredentials(username, password string) error {
	if username != BootstrapAdminUsername {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(bootstrapAdminPasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}
