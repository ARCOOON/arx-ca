package middleware

import (
	"net/http"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/auth"
)

// RequirePermission enforces that the authenticated principal holds the given permission.
// It must run after RequireAdmin, RequireServiceAccountOrAdmin, or RequireAdminOrMTLS.
// mTLS-authenticated requests bypass RBAC because identity is bound in the handler.
func RequirePermission(perm auth.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.MTLSAuthenticatedFromContext(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		roles, ok := auth.RolesFromContext(r.Context())
		if !ok {
			api.WriteError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		if !auth.HasPermission(roles, perm) {
			api.WriteError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		next.ServeHTTP(w, r)
	})
}
