package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultJWTIssuer = "arx-ca"
	defaultJWTExpiry = 24 * time.Hour
	jwtSigningMethod = "HS256"
	adminTokenType   = "Bearer"
)

var (
	// ErrInvalidToken is returned when a JWT cannot be parsed or validated.
	ErrInvalidToken = errors.New("invalid or expired token")
	// ErrEmptyJWTSecret is returned when no signing secret is configured.
	ErrEmptyJWTSecret = errors.New("JWT signing secret must not be empty")
)

// JWTManager issues and validates HS256 JWTs for admin users.
type JWTManager struct {
	secret []byte
	issuer string
	expiry time.Duration
}

// AdminClaims are embedded in admin JWT access tokens.
type AdminClaims struct {
	Username string `json:"username"`
	Roles    []Role `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTManager constructs a JWTManager. secret must be non-empty.
func NewJWTManager(secret string, issuer string, expiry time.Duration) (*JWTManager, error) {
	if secret == "" {
		return nil, ErrEmptyJWTSecret
	}
	if issuer == "" {
		issuer = defaultJWTIssuer
	}
	if expiry <= 0 {
		expiry = defaultJWTExpiry
	}
	return &JWTManager{
		secret: []byte(secret),
		issuer: issuer,
		expiry: expiry,
	}, nil
}

// GenerateToken creates a signed JWT for the given admin username and roles.
func (m *JWTManager) GenerateToken(username string, roles []Role) (token string, expiresAt time.Time, err error) {
	now := time.Now().UTC()
	expiresAt = now.Add(m.expiry)
	roles = NormalizeRoles(roles)
	if len(roles) == 0 {
		roles = RolesForAdmin(username)
	}

	claims := AdminClaims{
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign jwt: %w", err)
	}
	return signed, expiresAt, nil
}

// ValidateToken parses and verifies an admin JWT, returning its claims.
func (m *JWTManager) ValidateToken(tokenString string) (*AdminClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwtSigningMethod}))

	parsed, err := parser.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwtSigningMethod {
			return nil, fmt.Errorf("auth: unexpected signing method %q", t.Method.Alg())
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*AdminClaims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != m.issuer {
		return nil, ErrInvalidToken
	}
	if claims.Username == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// TokenType returns the authorization scheme for admin tokens.
func (m *JWTManager) TokenType() string {
	return adminTokenType
}
