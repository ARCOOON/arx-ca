package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// PublicHandler serves unauthenticated read-only certificate endpoints for local agents.
type PublicHandler struct {
	engine *ca.PKIEngine
}

// NewPublicHandler constructs a public handler bound to the PKI engine.
func NewPublicHandler(engine *ca.PKIEngine) *PublicHandler {
	return &PublicHandler{engine: engine}
}

// IntermediateCert handles GET /api/v1/public/ca/intermediate.
func (h *PublicHandler) IntermediateCert() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		pemBytes := h.engine.IntermediateCertPEM()
		if len(pemBytes) == 0 {
			api.WriteError(w, http.StatusInternalServerError, "intermediate certificate is unavailable")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.IntermediateCertResponse{
			PEM: string(pemBytes),
		})
	})
}

// ListCertificates handles GET /api/v1/public/certificates.
func (h *PublicHandler) ListCertificates() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp, err := h.engine.ListPublicCertificates(r.Context())
		if err != nil {
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("public: list certificates: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// GetCertificate handles GET /api/v1/public/certificates/{serial}.
func (h *PublicHandler) GetCertificate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		serial := strings.TrimSpace(r.PathValue("serial"))
		if serial == "" {
			api.WriteError(w, http.StatusBadRequest, "serial is required")
			return
		}

		pem, err := h.engine.GetCertificatePEM(r.Context(), serial)
		if err != nil {
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("public: get certificate %s: %v", serial, err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.CertificatePEMResponse{
			CertificatePEM: pem,
			Serial:         serial,
		})
	})
}
