//go:build !windows

package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// Lint handles POST /api/v1/certificates/lint.
func (h *CertificateHandler) Lint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.LintCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.CertificatePEM) == "" {
			api.WriteError(w, http.StatusBadRequest, "certificate_pem is required")
			return
		}

		resp, err := h.engine.LintCertificate(req.CertificatePEM)
		if err != nil {
			if strings.Contains(err.Error(), "certificate_pem") || strings.Contains(err.Error(), "parse certificate") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("certificates: lint: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "certificate lint failed")
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}
