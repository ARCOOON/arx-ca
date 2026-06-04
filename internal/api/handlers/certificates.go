package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/models"
)

const maxCertificateBodyBytes = 1 << 20 // 1 MiB

// CertificateHandler serves protected certificate lifecycle endpoints.
type CertificateHandler struct {
	engine *ca.PKIEngine
}

// NewCertificateHandler constructs a certificate handler bound to the PKI engine.
func NewCertificateHandler(engine *ca.PKIEngine) *CertificateHandler {
	return &CertificateHandler{engine: engine}
}

// Issue handles POST /api/v1/certificates/issue.
func (h *CertificateHandler) Issue() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.IssueCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		csrPEM := strings.TrimSpace(req.CSR)
		if csrPEM == "" {
			api.WriteError(w, http.StatusBadRequest, "csr is required")
			return
		}

		resp, err := h.engine.IssueCertificate(r.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "invalid ttl") ||
				strings.Contains(err.Error(), "exceeds configured maximum") ||
				strings.Contains(err.Error(), "csr is required") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: issue: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Auto handles POST /api/v1/certificates/auto.
func (h *CertificateHandler) Auto() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.AutoCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := h.engine.AutoCertificate(r.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "common_name") || strings.Contains(err.Error(), "invalid ip_sans") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: auto: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Revoke handles POST /api/v1/certificates/revoke.
func (h *CertificateHandler) Revoke() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.RevokeCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.Serial) == "" {
			api.WriteError(w, http.StatusBadRequest, "serial is required")
			return
		}

		resp, err := h.engine.RevokeCertificate(r.Context(), req.Serial, req.Reason, req.ReasonCode)
		if err != nil {
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: revoke: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// IssueWithToken handles POST /api/v1/certificates/issue-with-token.
func (h *CertificateHandler) IssueWithToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.IssueCertificateWithTokenRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.Token) == "" {
			api.WriteError(w, http.StatusBadRequest, "token is required")
			return
		}
		if strings.TrimSpace(req.CSR) == "" {
			api.WriteError(w, http.StatusBadRequest, "csr is required")
			return
		}

		resp, err := h.engine.IssueCertificateWithToken(r.Context(), req.Token, req.CSR, req.TTL, req.TemplateID, req.Metadata)
		if err != nil {
			if strings.Contains(err.Error(), "token is required") || strings.Contains(err.Error(), "parse certificate signing request") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: issue-with-token: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Generate handles POST /api/v1/certificates/generate.
func (h *CertificateHandler) Generate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.GenerateCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := h.engine.GenerateCertificate(r.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "common_name") ||
				strings.Contains(err.Error(), "key_algo") ||
				strings.Contains(err.Error(), "invalid sans") ||
				strings.Contains(err.Error(), "unsupported key_algo") ||
				strings.Contains(err.Error(), "invalid ttl") ||
				strings.Contains(err.Error(), "exceeds configured maximum") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: generate: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// List handles GET /api/v1/certificates.
func (h *CertificateHandler) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp, err := h.engine.ListCertificates(r.Context())
		if err != nil {
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("certificates: list: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}
