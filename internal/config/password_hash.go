package config

import (
	"strings"
)

const bcryptWorkFactor = 12

// IsBcryptPasswordHash reports whether s looks like a bcrypt password hash.
func IsBcryptPasswordHash(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 7 {
		return false
	}
	switch {
	case strings.HasPrefix(s, "$2a$"), strings.HasPrefix(s, "$2b$"), strings.HasPrefix(s, "$2y$"):
		return true
	default:
		return false
	}
}

// BootstrapAdminPasswordHash returns the bcrypt hash used to seed the initial admin user.
func (c ServerConfig) BootstrapAdminPasswordHash() string {
	if h := strings.TrimSpace(c.Security.InitialAdminPassword); IsBcryptPasswordHash(h) {
		return h
	}
	return strings.TrimSpace(c.Bootstrap.AdminPasswordHash)
}
