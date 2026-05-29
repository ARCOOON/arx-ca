package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/ca"
	"github.com/your-org/arx-ca/internal/models"
)

// ProvisionerHandler serves provisioner token endpoints.
type ProvisionerHandler struct {
	engine *ca.PKIEngine
}

// NewProvisionerHandler constructs a provisioner handler bound to the PKI engine.
func NewProvisionerHandler(engine *ca.PKIEngine) *ProvisionerHandler {
	return &ProvisionerHandler{engine: engine}
}

// Token handles POST /api/v1/provisioners/token.
func (h *ProvisionerHandler) Token() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.ProvisionerTokenRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.CommonName) == "" {
			api.WriteError(w, http.StatusBadRequest, "common_name is required")
			return
		}

		resp, err := h.engine.GenerateProvisionerToken(r.Context(), req)
		if err != nil {
			if isClientProvisionerError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("provisioners: token: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

func isClientProvisionerError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "common_name") ||
		strings.Contains(msg, "invalid ip_sans") ||
		strings.Contains(msg, "token_ttl") ||
		strings.Contains(msg, "cannot mint signing tokens") ||
		strings.Contains(msg, "not found")
}
