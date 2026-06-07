package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
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

		if err := h.validateRenewalIdentity(r, req.CertificatePEM, req.RenewToken); err != nil {
			if isRenewalIdentityError(err) {
				api.WriteError(w, http.StatusForbidden, err.Error())
				return
			}
			if isRenewalClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			api.WriteError(w, status, message)
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

		recordCertAudit(r, db.ActionCertRenew, "", resp.CertificatePEM)
		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("serial", resp.Serial)
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

		if err := h.validateRenewalIdentity(r, req.CertificatePEM, req.RenewToken); err != nil {
			if isRenewalIdentityError(err) {
				api.WriteError(w, http.StatusForbidden, err.Error())
				return
			}
			if isRenewalClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			api.WriteError(w, status, message)
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

		recordCertAudit(r, "CERT_REKEY", "", resp.CertificatePEM)
		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("serial", resp.Serial)
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

func (h *RenewalHandler) validateRenewalIdentity(r *http.Request, certificatePEM, renewToken string) error {
	mtlsCN, mtlsAuth := auth.MTLSCommonNameFromContext(r.Context())
	if !mtlsAuth {
		return nil
	}

	target, err := h.engine.ResolveRenewTarget(certificatePEM, renewToken)
	if err != nil {
		return err
	}

	targetCN := ca.CertificateCommonName(target)
	if targetCN == "" || !strings.EqualFold(targetCN, mtlsCN) {
		return errRenewalCNMismatch
	}
	return nil
}

var errRenewalCNMismatch = &renewalIdentityError{message: "renewal is not permitted for this identity"}

type renewalIdentityError struct {
	message string
}

func (e *renewalIdentityError) Error() string {
	return e.message
}

func isRenewalIdentityError(err error) bool {
	_, ok := err.(*renewalIdentityError)
	return ok
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
