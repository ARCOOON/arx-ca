package config

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptWorkFactor = 12

// ErrPasswordHashPersist indicates server.yaml could not be updated after hashing a plaintext password.
type ErrPasswordHashPersist struct {
	Path string
	Err  error
}

func (e *ErrPasswordHashPersist) Error() string {
	return fmt.Sprintf("persist hashed initial admin password in %s: %v", e.Path, e.Err)
}

func (e *ErrPasswordHashPersist) Unwrap() error {
	return e.Err
}

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

// HashPlaintextPasswords replaces plaintext admin passwords in cfg with bcrypt hashes and
// writes the updated configuration to configPath when any field was migrated.
// Returns true when security.initial_admin_password was migrated from plaintext.
func HashPlaintextPasswords(configPath string, cfg *ServerConfig) (initialPasswordMigrated bool, err error) {
	if cfg == nil {
		return false, fmt.Errorf("server config is nil")
	}

	fileMigrated := false
	var primaryHash string

	initial := strings.TrimSpace(cfg.Security.InitialAdminPassword)
	if initial != "" && !IsBcryptPasswordHash(initial) {
		hashed, hashErr := bcrypt.GenerateFromPassword([]byte(initial), bcryptWorkFactor)
		if hashErr != nil {
			return false, fmt.Errorf("hash initial admin password: %w", hashErr)
		}
		primaryHash = string(hashed)
		cfg.Security.InitialAdminPassword = primaryHash
		initialPasswordMigrated = true
		fileMigrated = true
	} else if IsBcryptPasswordHash(initial) {
		primaryHash = initial
	}

	bootstrap := strings.TrimSpace(cfg.Bootstrap.AdminPasswordHash)
	switch {
	case bootstrap != "" && !IsBcryptPasswordHash(bootstrap):
		if primaryHash != "" {
			cfg.Bootstrap.AdminPasswordHash = primaryHash
		} else {
			hashed, hashErr := bcrypt.GenerateFromPassword([]byte(bootstrap), bcryptWorkFactor)
			if hashErr != nil {
				return initialPasswordMigrated, fmt.Errorf("hash bootstrap admin password: %w", hashErr)
			}
			primaryHash = string(hashed)
			cfg.Bootstrap.AdminPasswordHash = primaryHash
		}
		fileMigrated = true
	case bootstrap == "" && primaryHash != "":
		cfg.Bootstrap.AdminPasswordHash = primaryHash
		fileMigrated = true
	}

	if !fileMigrated {
		return initialPasswordMigrated, nil
	}

	raw, marshalErr := marshalYAMLConfig(*cfg)
	if marshalErr != nil {
		return initialPasswordMigrated, &ErrPasswordHashPersist{Path: configPath, Err: marshalErr}
	}
	if writeErr := os.WriteFile(configPath, raw, 0o600); writeErr != nil {
		return initialPasswordMigrated, &ErrPasswordHashPersist{Path: configPath, Err: writeErr}
	}

	return initialPasswordMigrated, nil
}

// BootstrapAdminPasswordHash returns the bcrypt hash used to seed the initial admin user.
func (c ServerConfig) BootstrapAdminPasswordHash() string {
	if h := strings.TrimSpace(c.Security.InitialAdminPassword); IsBcryptPasswordHash(h) {
		return h
	}
	return strings.TrimSpace(c.Bootstrap.AdminPasswordHash)
}
