package acmeprotocol

import (
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"go.step.sm/crypto/jose"
)

// KeyAuthorization builds the ACME key authorization string (token || "." || JWK thumbprint).
func KeyAuthorization(token string, jwk *jose.JSONWebKey) (string, error) {
	thumbprint, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("jwk thumbprint: %w", err)
	}
	encPrint := base64.RawURLEncoding.EncodeToString(thumbprint)
	return fmt.Sprintf("%s.%s", token, encPrint), nil
}

// DNS01Digest returns the SHA-256 digest of keyAuthorization, base64url-encoded (RFC 8555 §8.4).
func DNS01Digest(keyAuthorization string) string {
	sum := sha256.Sum256([]byte(keyAuthorization))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
