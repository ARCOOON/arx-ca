package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	arxconfig "github.com/your-org/arx-ca/internal/config"
)

const (
	envJWTIssuer = "CA_API_JWT_ISSUER"
)

// LoadJWTManagerFromConfig builds a JWTManager from server security configuration.
func LoadJWTManagerFromConfig(sec arxconfig.SecurityConfig) (*JWTManager, error) {
	secret := strings.TrimSpace(sec.JWTSecret)
	if secret == "" {
		return nil, fmt.Errorf("auth: JWT signing secret must be configured")
	}

	issuer := strings.TrimSpace(os.Getenv(envJWTIssuer))
	expiry := sec.TokenExpiration()
	return NewJWTManager(secret, issuer, expiry)
}

// LoadJWTManagerFromEnv builds a JWTManager using environment configuration.
// Deprecated: prefer LoadJWTManagerFromConfig after InitServerConfig.
func LoadJWTManagerFromEnv() (*JWTManager, error) {
	secret := strings.TrimSpace(os.Getenv("CA_API_JWT_SECRET"))
	if secret == "" {
		return nil, fmt.Errorf("auth: %s is not set", "CA_API_JWT_SECRET")
	}
	issuer := strings.TrimSpace(os.Getenv(envJWTIssuer))
	expiry := defaultJWTExpiry
	if raw := os.Getenv("CA_API_JWT_EXPIRY"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("auth: invalid CA_API_JWT_EXPIRY: %w", err)
		}
		expiry = parsed
	}
	return NewJWTManager(secret, issuer, expiry)
}
