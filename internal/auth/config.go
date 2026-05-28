package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

const (
	envJWTSecret = "CA_API_JWT_SECRET"
	envJWTIssuer = "CA_API_JWT_ISSUER"
	envJWTExpiry = "CA_API_JWT_EXPIRY"
)

// LoadJWTManagerFromEnv builds a JWTManager using environment configuration.
// If CA_API_JWT_SECRET is unset, a random 32-byte secret is generated (tokens do not survive restarts).
func LoadJWTManagerFromEnv() (*JWTManager, error) {
	secret := os.Getenv(envJWTSecret)
	if secret == "" {
		generated, err := generateRandomSecret(32)
		if err != nil {
			return nil, fmt.Errorf("auth: generate jwt secret: %w", err)
		}
		secret = generated
		fmt.Fprintf(os.Stderr, "auth: warning: %s not set; using ephemeral JWT signing secret\n", envJWTSecret)
	}

	issuer := os.Getenv(envJWTIssuer)
	expiry := defaultJWTExpiry
	if raw := os.Getenv(envJWTExpiry); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("auth: invalid %s: %w", envJWTExpiry, err)
		}
		expiry = parsed
	}

	return NewJWTManager(secret, issuer, expiry)
}

func generateRandomSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
