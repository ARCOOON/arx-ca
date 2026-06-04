package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const caPasswordEnvVar = "ARX_CA_PASSWORD"

const configFileMode = 0o600

// ErrConfigHealPersist indicates server.yaml could not be updated after auto-securing configuration.
type ErrConfigHealPersist struct {
	Path string
	Err  error
}

func (e *ErrConfigHealPersist) Error() string {
	return fmt.Sprintf("persist auto-secured configuration to %s: %v", e.Path, e.Err)
}

func (e *ErrConfigHealPersist) Unwrap() error {
	return e.Err
}

// HealServerConfig inspects cfg immediately after server.yaml is parsed. It generates a
// JWT secret when missing, bcrypt-hashes plaintext admin passwords, and rewrites server.yaml
// when any field was secured. Database connection strings, ca.password (symmetric key
// material), and ARX_CA_PASSWORD overrides are never modified or persisted here.
func HealServerConfig(configPath string, cfg *ServerConfig) error {
	if cfg == nil {
		return fmt.Errorf("server config is nil")
	}

	configNeedsRewrite := false

	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		secret, err := GenerateJWTSecret(32)
		if err != nil {
			return fmt.Errorf("generate JWT secret: %w", err)
		}
		cfg.Security.JWTSecret = secret
		configNeedsRewrite = true
		log.Println("No JWT secret found. Auto-generated a secure 256-bit key.")
	}

	passwordHealed, err := healAdminPasswords(cfg)
	if err != nil {
		return err
	}
	if passwordHealed {
		configNeedsRewrite = true
		log.Println("Detected clear-text admin password in configuration. Automatically secured via bcrypt.")
	}

	if healWebUICORS(cfg) {
		configNeedsRewrite = true
	}

	if !configNeedsRewrite {
		return nil
	}

	raw, err := marshalYAMLConfig(*cfg)
	if err != nil {
		return &ErrConfigHealPersist{Path: configPath, Err: err}
	}
	if err := os.WriteFile(configPath, raw, configFileMode); err != nil {
		return &ErrConfigHealPersist{Path: configPath, Err: err}
	}
	return nil
}

func healAdminPasswords(cfg *ServerConfig) (healed bool, err error) {
	var primaryHash string

	initial := strings.TrimSpace(cfg.Security.InitialAdminPassword)
	if initial != "" && !IsBcryptPasswordHash(initial) {
		hashed, hashErr := bcrypt.GenerateFromPassword([]byte(initial), bcryptWorkFactor)
		if hashErr != nil {
			return false, fmt.Errorf("hash initial admin password: %w", hashErr)
		}
		primaryHash = string(hashed)
		cfg.Security.InitialAdminPassword = primaryHash
		healed = true
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
				return healed, fmt.Errorf("hash bootstrap admin password: %w", hashErr)
			}
			primaryHash = string(hashed)
			cfg.Bootstrap.AdminPasswordHash = primaryHash
		}
		healed = true
	case bootstrap == "" && primaryHash != "":
		cfg.Bootstrap.AdminPasswordHash = primaryHash
		healed = true
	}

	return healed, nil
}

// healWebUICORS normalizes wildcard CORS lists and reports whether the configuration changed.
func healWebUICORS(cfg *ServerConfig) bool {
	if cfg == nil {
		return false
	}
	before := cloneWebUICORSConfig(cfg.WebUI.CORS)
	normalizeWebUICORS(&cfg.WebUI.CORS)
	return !webUICORSEqual(before, cfg.WebUI.CORS)
}

func cloneWebUICORSConfig(c WebUICORSConfig) WebUICORSConfig {
	return WebUICORSConfig{
		AllowedOrigins: append([]string(nil), c.AllowedOrigins...),
		AllowedMethods: append([]string(nil), c.AllowedMethods...),
		AllowedHeaders: append([]string(nil), c.AllowedHeaders...),
	}
}

func webUICORSEqual(a, b WebUICORSConfig) bool {
	return stringSlicesEqual(a.AllowedOrigins, b.AllowedOrigins) &&
		stringSlicesEqual(a.AllowedMethods, b.AllowedMethods) &&
		stringSlicesEqual(a.AllowedHeaders, b.AllowedHeaders)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// normalizeWebUICORS ensures each CORS list is strictly ["*"] when a wildcard entry is present.
func normalizeWebUICORS(cors *WebUICORSConfig) {
	if cors == nil {
		return
	}
	cors.AllowedOrigins = normalizeCORSList(cors.AllowedOrigins)
	cors.AllowedMethods = normalizeCORSList(cors.AllowedMethods)
	cors.AllowedHeaders = normalizeCORSList(cors.AllowedHeaders)
}

func normalizeCORSList(items []string) []string {
	for _, item := range items {
		if strings.TrimSpace(item) == "*" {
			return []string{"*"}
		}
	}
	return items
}

// applyCAPasswordEnvOverride replaces ca.password in memory when ARX_CA_PASSWORD is set.
// The override is never written back to server.yaml during auto-healing.
func applyCAPasswordEnvOverride(cfg *ServerConfig) {
	if cfg == nil {
		return
	}
	if v, ok := os.LookupEnv(caPasswordEnvVar); ok {
		cfg.CA.Password = v
	}
}
