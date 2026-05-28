package handlers

import (
	"net/http"

	"github.com/your-org/ca-api/internal/api"
	"github.com/your-org/ca-api/internal/ca"
	"github.com/your-org/ca-api/internal/models"
)

// CAHandler serves public CA certificate endpoints.
type CAHandler struct {
	engine *ca.PKIEngine
}

// NewCAHandler constructs a CA handler bound to the PKI engine.
func NewCAHandler(engine *ca.PKIEngine) *CAHandler {
	return &CAHandler{engine: engine}
}

// RootCert handles GET /api/v1/ca/root and returns the Root CA certificate in PEM format.
func (h *CAHandler) RootCert() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		pemBytes := h.engine.RootCertPEM()
		if len(pemBytes) == 0 {
			api.WriteError(w, http.StatusInternalServerError, "root certificate is unavailable")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.RootCertResponse{
			PEM: string(pemBytes),
		})
	})
}
