package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/ca"
	arxcrypto "github.com/ARCOOON/arx-ca/internal/crypto"
	"github.com/ARCOOON/arx-ca/internal/database"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/ARCOOON/arx-ca/internal/repository"
)

const maxCertificateBodyBytes = 1 << 20 // 1 MiB

// CertificateHandler serves protected certificate lifecycle endpoints.
type CertificateHandler struct {
	engine    *ca.PKIEngine
	certStore *database.CertificateStore
	userStore *repository.UserStore
}

// NewCertificateHandler constructs a certificate handler bound to the PKI engine and application stores.
func NewCertificateHandler(engine *ca.PKIEngine, certStore *database.CertificateStore, userStore *repository.UserStore) *CertificateHandler {
	return &CertificateHandler{
		engine:    engine,
		certStore: certStore,
		userStore: userStore,
	}
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

		if err := h.persistCertificate(r.Context(), w, resp.CertificatePEM, "", ""); err != nil {
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

		if err := h.persistCertificate(r.Context(), w, resp.CertificatePEM, resp.PrivateKeyPEM, h.engine.CAPassword()); err != nil {
			return
		}

		if wantsCertificateBundleZip(r) {
			safeName := strings.TrimSpace(req.CommonName)
			if safeName == "" {
				safeName = resp.Serial
			}
			if err := h.writeCertificateBundleResponse(w, resp.CertificatePEM, resp.PrivateKeyPEM, safeName); err != nil {
				log.Printf("certificates: generate bundle: %v", err)
				api.WriteError(w, http.StatusInternalServerError, "failed to build certificate bundle")
			}
			return
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// GetPrivateKey handles GET /api/v1/certificates/{serial}/key.
func (h *CertificateHandler) GetPrivateKey() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if _, ok := auth.AdminUsernameFromContext(r.Context()); !ok {
			api.WriteError(w, http.StatusForbidden, "super admin JWT required")
			return
		}
		roles, ok := auth.RolesFromContext(r.Context())
		if !ok || !auth.HasRole(roles, auth.RoleSuperAdmin) {
			api.WriteError(w, http.StatusForbidden, "super admin role required")
			return
		}

		serial := strings.TrimSpace(r.PathValue("serial"))
		if serial == "" {
			api.WriteError(w, http.StatusBadRequest, "serial is required")
			return
		}

		privateKeyPEM, err := h.decryptEscrowedPrivateKey(r.Context(), serial)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "escrowed private key not found")
				return
			}
			log.Printf("certificates: decrypt escrowed key: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to retrieve private key")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.CertificatePrivateKeyResponse{
			Serial:        serial,
			PrivateKeyPEM: privateKeyPEM,
		})
	})
}

// DownloadBundle handles GET /api/v1/certificates/{serial}/bundle.
func (h *CertificateHandler) DownloadBundle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if _, ok := auth.AdminUsernameFromContext(r.Context()); !ok {
			api.WriteError(w, http.StatusForbidden, "super admin JWT required")
			return
		}
		roles, ok := auth.RolesFromContext(r.Context())
		if !ok || !auth.HasRole(roles, auth.RoleSuperAdmin) {
			api.WriteError(w, http.StatusForbidden, "super admin role required")
			return
		}

		serial := strings.TrimSpace(r.PathValue("serial"))
		if serial == "" {
			api.WriteError(w, http.StatusBadRequest, "serial is required")
			return
		}

		if h.certStore == nil {
			api.WriteError(w, http.StatusInternalServerError, "certificate archive is unavailable")
			return
		}

		rec, err := h.certStore.GetBySerial(r.Context(), serial)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "certificate record not found")
				return
			}
			log.Printf("certificates: load bundle record: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to load certificate record")
			return
		}

		privateKeyPEM, err := h.decryptEscrowedPrivateKey(r.Context(), serial)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "escrowed private key not found")
				return
			}
			log.Printf("certificates: bundle private key: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to retrieve private key")
			return
		}

		bundle, err := h.buildBundleFromMaterial(rec.CertificatePEM, privateKeyPEM)
		if err != nil {
			log.Printf("certificates: build bundle: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to build certificate bundle")
			return
		}

		safeName := strings.TrimSpace(rec.CommonName)
		if safeName == "" {
			safeName = serial
		}
		writeCertificateBundleBytes(w, bundle, safeName)
	})
}

// GetBySerial handles GET /api/v1/certificates/{serial}.
func (h *CertificateHandler) GetBySerial() http.Handler {
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

		if h.certStore == nil {
			api.WriteError(w, http.StatusInternalServerError, "certificate archive is unavailable")
			return
		}

		rec, err := h.certStore.GetBySerial(r.Context(), serial)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "certificate record not found")
				return
			}
			log.Printf("certificates: get record: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to load certificate record")
			return
		}

		revoked := false
		if h.engine != nil && h.engine.Authority() != nil {
			revoked, _ = h.engine.Authority().IsRevoked(rec.Serial)
		}

		api.WriteSuccess(w, http.StatusOK, certificateRecordFromStore(rec, revoked))
	})
}

func (h *CertificateHandler) persistCertificate(ctx context.Context, w http.ResponseWriter, certPEM, privateKeyPEM, caPassword string) error {
	if h.certStore == nil {
		return nil
	}

	requestorID, err := ResolveRequestorID(ctx, h.userStore)
	if err != nil {
		log.Printf("certificates: resolve requestor: %v", err)
		api.WriteError(w, http.StatusInternalServerError, "failed to record certificate metadata")
		return err
	}

	if err := persistIssuedCertificate(ctx, h.certStore, requestorID, certPEM, privateKeyPEM, caPassword); err != nil {
		log.Printf("certificates: persist record: %v", err)
		api.WriteError(w, http.StatusInternalServerError, "failed to archive issued certificate")
		return err
	}
	return nil
}

func (h *CertificateHandler) decryptEscrowedPrivateKey(ctx context.Context, serial string) (string, error) {
	if h.certStore == nil {
		return "", fmt.Errorf("certificate archive is unavailable")
	}

	rec, err := h.certStore.GetBySerial(ctx, serial)
	if err != nil {
		return "", err
	}
	if len(rec.EncryptedPrivateKey) == 0 {
		return "", fmt.Errorf("escrowed private key not found")
	}

	caPassword := strings.TrimSpace(h.engine.CAPassword())
	if caPassword == "" {
		return "", fmt.Errorf("ca password is unavailable")
	}

	plaintext, err := arxcrypto.DecryptKey(rec.EncryptedPrivateKey, caPassword)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (h *CertificateHandler) buildBundleFromMaterial(certificatePEM, privateKeyPEM string) ([]byte, error) {
	return buildCertificateBundleZip(certificateBundleInput{
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  privateKeyPEM,
	})
}

func wantsCertificateBundleZip(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "zip") {
		return true
	}
	for _, value := range r.Header.Values("Accept") {
		if strings.Contains(strings.ToLower(value), "application/zip") {
			return true
		}
	}
	return false
}

func (h *CertificateHandler) writeCertificateBundleResponse(w http.ResponseWriter, certificatePEM, privateKeyPEM, archiveBaseName string) error {
	bundle, err := h.buildBundleFromMaterial(certificatePEM, privateKeyPEM)
	if err != nil {
		return err
	}
	writeCertificateBundleBytes(w, bundle, archiveBaseName)
	return nil
}

func writeCertificateBundleBytes(w http.ResponseWriter, bundle []byte, archiveBaseName string) {
	filename := sanitizeDownloadFilename(archiveBaseName) + "-bundle.zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(bundle); err != nil {
		log.Printf("certificates: write bundle: %v", err)
	}
}

func sanitizeDownloadFilename(value string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(strings.TrimSpace(value))
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
