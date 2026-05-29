package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/ca"
	"github.com/your-org/arx-ca/internal/models"
)

// RenewalHandler serves certificate auto-renewal endpoints for non-ACME clients.
type RenewalHandler struct {
	engine     *ca.PKIEngine
	listenHost string
}

// NewRenewalHandler constructs a renewal handler bound to the PKI engine.
func NewRenewalHandler(engine *ca.PKIEngine, listenHost string) *RenewalHandler {
	return &RenewalHandler{
		engine:     engine,
		listenHost: listenHost,
	}
}

// Renew handles POST /api/v1/certificates/renew.
func (h *RenewalHandler) Renew() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.RenewCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := h.engine.RenewCertificate(r.Context(), req.CertificatePEM, req.RenewToken)
		if err != nil {
			if isRenewalClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: renew: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Rekey handles POST /api/v1/certificates/rekey.
func (h *RenewalHandler) Rekey() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.RekeyCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.CSR) == "" {
			api.WriteError(w, http.StatusBadRequest, "csr is required")
			return
		}

		resp, err := h.engine.RekeyCertificate(r.Context(), req.CertificatePEM, req.CSR, req.RenewToken)
		if err != nil {
			if isRenewalClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: rekey: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// ACMEStatus handles GET /api/v1/acme/status.
func (h *RenewalHandler) ACMEStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp := models.ACMEStatusResponse{
			Enabled:     h.engine.ACMEEnabled(),
			Provisioner: "acme",
			Challenges:  []string{"http-01", "dns-01", "tls-alpn-01"},
		}
		if resp.Enabled {
			resp.DirectoryURL = h.engine.ACMEDirectoryURL(h.listenHost)
			resp.DNSName = h.engine.ACMEDNS()
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

func isRenewalClientError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "certificate_pem") ||
		strings.Contains(msg, "renew_token") ||
		strings.Contains(msg, "parse certificate") ||
		strings.Contains(msg, "parse certificate signing request") ||
		strings.Contains(msg, "csr is required")
}
