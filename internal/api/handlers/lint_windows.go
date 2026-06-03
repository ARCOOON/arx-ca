//go:build windows

package handlers

import (
	"net/http"

	"github.com/ARCOOON/arx-ca/internal/api"
)

// Lint handles POST /api/v1/certificates/lint.
func (h *CertificateHandler) Lint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		api.WriteError(w, http.StatusNotImplemented, "Certificate linting is not supported on Windows builds due to dependency limitations.")
	})
}
