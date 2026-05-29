package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/ca"
	"github.com/your-org/arx-ca/internal/models"
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

		resp, err := h.engine.IssueCertificate(r.Context(), csrPEM, req.TTL, req.TemplateID, req.Metadata)
		if err != nil {
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
