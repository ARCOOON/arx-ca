package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/ca"
)

const (
	headerAuthorization = "Authorization"
	headerAPIKey        = "X-API-Key"
	bearerPrefix        = "Bearer "
)

// RequireAdmin validates a Bearer JWT and injects the admin username into the request context.
func RequireAdmin(jwtManager *auth.JWTManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r.Header.Get(headerAuthorization))
		if err != nil {
			api.WriteError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			api.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := auth.WithAdminUsername(r.Context(), claims.Username)
		roles := claims.Roles
		if len(roles) == 0 {
			roles = auth.RolesForAdmin(claims.Username)
		}
		ctx = auth.WithRoles(ctx, roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireServiceAccountOrAdmin accepts either a valid service-account API key or an admin JWT.
func RequireServiceAccountOrAdmin(jwtManager *auth.JWTManager, store *auth.APIKeyStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, err := extractBearerToken(r.Header.Get(headerAuthorization)); err == nil {
			if claims, jwtErr := jwtManager.ValidateToken(token); jwtErr == nil {
				ctx := auth.WithAdminUsername(r.Context(), claims.Username)
				roles := claims.Roles
				if len(roles) == 0 {
					roles = auth.RolesForAdmin(claims.Username)
				}
				ctx = auth.WithRoles(ctx, roles)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		key := strings.TrimSpace(r.Header.Get(headerAPIKey))
		if key == "" {
			if bearer, err := extractBearerToken(r.Header.Get(headerAuthorization)); err == nil {
				key = bearer
			}
		}

		if key == "" {
			api.WriteError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		account, err := store.ValidateAPIKey(key)
		if err != nil {
			if errors.Is(err, auth.ErrAPIKeyNotFound) {
				api.WriteError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			api.WriteError(w, http.StatusInternalServerError, "authentication failed")
			return
		}

		ctx := auth.WithServiceAccount(r.Context(), account)
		ctx = auth.WithRoles(ctx, account.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireServiceAccount validates an API key from X-API-Key or Authorization: Bearer headers.
func RequireServiceAccount(store *auth.APIKeyStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get(headerAPIKey))
		if key == "" {
			var err error
			key, err = extractBearerToken(r.Header.Get(headerAuthorization))
			if err != nil {
				api.WriteError(w, http.StatusUnauthorized, "api key required")
				return
			}
		}

		account, err := store.ValidateAPIKey(key)
		if err != nil {
			if errors.Is(err, auth.ErrAPIKeyNotFound) {
				api.WriteError(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			api.WriteError(w, http.StatusInternalServerError, "authentication failed")
			return
		}

		ctx := auth.WithServiceAccount(r.Context(), account)
		ctx = auth.WithRoles(ctx, account.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdminOrMTLS accepts either a valid admin JWT or a verified mTLS client certificate.
func RequireAdminOrMTLS(jwtManager *auth.JWTManager, certValidator *ca.ClientCertValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, err := extractBearerToken(r.Header.Get(headerAuthorization)); err == nil {
			if claims, jwtErr := jwtManager.ValidateToken(token); jwtErr == nil {
				ctx := auth.WithAdminUsername(r.Context(), claims.Username)
				roles := claims.Roles
				if len(roles) == 0 {
					roles = auth.RolesForAdmin(claims.Username)
				}
				ctx = auth.WithRoles(ctx, roles)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if certValidator != nil {
			cert, err := certValidator.ClientCertificateFromRequest(r)
			if err == nil {
				ctx := auth.WithMTLSAuthentication(r.Context(), cert)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		api.WriteError(w, http.StatusUnauthorized, "authentication required")
	})
}

func extractBearerToken(headerValue string) (string, error) {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return "", errors.New("missing authorization header")
	}
	if !strings.HasPrefix(headerValue, bearerPrefix) {
		return "", errors.New("authorization header must use Bearer scheme")
	}
	token := strings.TrimSpace(strings.TrimPrefix(headerValue, bearerPrefix))
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

// AdminUsername is a convenience accessor for handlers behind RequireAdmin.
func AdminUsername(ctx context.Context) (string, bool) {
	return auth.AdminUsernameFromContext(ctx)
}
