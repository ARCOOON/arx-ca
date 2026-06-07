package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// SSHHandler serves SSH certificate authority endpoints.
type SSHHandler struct {
	engine     *ca.PKIEngine
	jwtManager *auth.JWTManager
	keyStore   *auth.APIKeyStore
	sshStore   *db.SSHCertificateStore
}

// NewSSHHandler constructs an SSH handler bound to the PKI engine.
func NewSSHHandler(engine *ca.PKIEngine, jwtManager *auth.JWTManager, keyStore *auth.APIKeyStore, sshStore *db.SSHCertificateStore) *SSHHandler {
	return &SSHHandler{
		engine:     engine,
		jwtManager: jwtManager,
		keyStore:   keyStore,
		sshStore:   sshStore,
	}
}

// GenerateUser handles POST /api/v1/ssh/generate/user.
func (h *SSHHandler) GenerateUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.GenerateSSHUserRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.PublicKey) == "" {
			api.WriteError(w, http.StatusBadRequest, "public_key is required")
			return
		}
		if len(req.Principals) == 0 {
			api.WriteError(w, http.StatusBadRequest, "principals is required")
			return
		}

		resp, err := h.engine.GenerateSSHUserCertificate(r.Context(), req)
		if err != nil {
			if isSSHClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("ssh: generate-user: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		recordAuditAction(r, db.ActionSSHUserCertIssue)
		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(strings.TrimSpace(req.Provisioner))
			ac.PutMetadata("principals", strings.Join(resp.Principals, ","))
			ac.PutMetadata("serial", resp.Serial)
		}

		h.persistSSHCertificate(r.Context(), resp)

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// GenerateHost handles POST /api/v1/ssh/generate/host.
func (h *SSHHandler) GenerateHost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.GenerateSSHHostRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.PublicKey) == "" {
			api.WriteError(w, http.StatusBadRequest, "public_key is required")
			return
		}
		if len(req.Principals) == 0 {
			api.WriteError(w, http.StatusBadRequest, "principals is required")
			return
		}

		resp, err := h.engine.GenerateSSHHostCertificate(r.Context(), req)
		if err != nil {
			if isSSHClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("ssh: generate-host: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		recordAuditAction(r, db.ActionSSHHostCertIssue)
		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(strings.TrimSpace(req.Provisioner))
			ac.PutMetadata("principals", strings.Join(resp.Principals, ","))
			ac.PutMetadata("serial", resp.Serial)
		}

		h.persistSSHCertificate(r.Context(), resp)

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// SignUser handles POST /api/v1/ssh/sign-user.
// API credentials mint an internal JWK token; an optional OIDC/provisioner token
// in the request body authorizes SSO users directly.
func (h *SSHHandler) SignUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.SignSSHUserRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.PublicKey) == "" {
			api.WriteError(w, http.StatusBadRequest, "public_key is required")
			return
		}
		if strings.TrimSpace(req.Principal) == "" && len(req.Principals) == 0 {
			api.WriteError(w, http.StatusBadRequest, "principal is required")
			return
		}

		signToken := strings.TrimSpace(req.Token)
		if signToken == "" {
			if !h.isAuthenticated(r) {
				api.WriteError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			token, err := h.engine.MintSSHUserSignToken(req)
			if err != nil {
				if isSSHClientError(err) {
					api.WriteError(w, http.StatusBadRequest, err.Error())
					return
				}
				status, message := ca.MapCAError(err)
				if status >= http.StatusInternalServerError {
					log.Printf("ssh: sign-user mint token: %v", err)
				}
				api.WriteError(w, status, message)
				return
			}
			signToken = token
		}

		resp, err := h.engine.SignSSHUserCertificate(r.Context(), req, signToken)
		if err != nil {
			if isSSHClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("ssh: sign-user: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		recordAuditAction(r, "SSH_SIGN_USER")
		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(strings.TrimSpace(req.Provisioner))
			ac.PutMetadata("principal", strings.TrimSpace(req.Principal))
			ac.PutMetadata("serial", resp.Serial)
		}

		h.persistSSHCertificate(r.Context(), resp)

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// SignHost handles POST /api/v1/ssh/sign-host.
func (h *SSHHandler) SignHost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.SignSSHHostRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.PublicKey) == "" {
			api.WriteError(w, http.StatusBadRequest, "public_key is required")
			return
		}
		if strings.TrimSpace(req.Hostname) == "" && len(req.Principals) == 0 {
			api.WriteError(w, http.StatusBadRequest, "hostname is required")
			return
		}

		resp, err := h.engine.SignSSHHostCertificate(r.Context(), req)
		if err != nil {
			if isSSHClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("ssh: sign-host: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		recordAuditAction(r, "SSH_SIGN_HOST")
		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(strings.TrimSpace(req.Provisioner))
			ac.PutMetadata("hostname", strings.TrimSpace(req.Hostname))
			ac.PutMetadata("serial", resp.Serial)
		}

		h.persistSSHCertificate(r.Context(), resp)

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Inspect handles POST /api/v1/ssh/inspect.
func (h *SSHHandler) Inspect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.InspectSSHCertificateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if strings.TrimSpace(req.Certificate) == "" {
			api.WriteError(w, http.StatusBadRequest, "certificate is required")
			return
		}

		resp, err := h.engine.InspectSSHCertificate(req.Certificate)
		if err != nil {
			if isSSHClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("ssh: inspect: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "ssh certificate inspection failed")
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// ListCertificates handles GET /api/v1/ssh/certificates.
func (h *SSHHandler) ListCertificates() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if h.sshStore == nil {
			api.WriteError(w, http.StatusInternalServerError, "ssh certificate store is unavailable")
			return
		}

		limit := parseSSHQueryInt(r, "limit", 50)
		offset := parseSSHQueryInt(r, "offset", 0)

		result, err := h.sshStore.List(r.Context(), db.SSHCertificateListOptions{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			log.Printf("ssh: list certificates: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to list SSH certificates")
			return
		}

		certificates := make([]models.SSHCertificateListItem, 0, len(result.Certificates))
		for _, entry := range result.Certificates {
			certificates = append(certificates, models.SSHCertificateListItem{
				ID:          entry.ID,
				Serial:      entry.Serial,
				CertType:    entry.CertType,
				Principals:  entry.Principals,
				Fingerprint: entry.Fingerprint,
				ValidAfter:  entry.ValidAfter.UTC().Format(time.RFC3339),
				ValidBefore: entry.ValidBefore.UTC().Format(time.RFC3339),
			})
		}

		api.WriteSuccess(w, http.StatusOK, models.ListSSHCertificatesResponse{
			Certificates: certificates,
			Total:        result.Total,
			Limit:        limit,
			Offset:       offset,
		})
	})
}

// Roots handles GET /api/v1/ssh/roots.
func (h *SSHHandler) Roots() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp, err := h.engine.GetSSHRoots(r.Context())
		if err != nil {
			if strings.Contains(err.Error(), "not configured") || strings.Contains(err.Error(), "no SSH CA") {
				api.WriteError(w, http.StatusNotFound, err.Error())
				return
			}
			log.Printf("ssh: roots: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to retrieve SSH CA roots")
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

func (h *SSHHandler) isAuthenticated(r *http.Request) bool {
	if token, ok := extractOptionalBearerToken(r.Header.Get("Authorization")); ok {
		if _, err := h.jwtManager.ValidateToken(token); err == nil {
			return true
		}
	}

	key := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if key == "" {
		if token, ok := extractOptionalBearerToken(r.Header.Get("Authorization")); ok {
			key = token
		}
	}

	if key == "" {
		return false
	}

	_, err := h.keyStore.ValidateAPIKey(key)
	return err == nil
}

func extractOptionalBearerToken(headerValue string) (string, bool) {
	const bearerPrefix = "Bearer "
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" || !strings.HasPrefix(headerValue, bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(headerValue, bearerPrefix))
	if token == "" {
		return "", false
	}
	return token, true
}

func isSSHClientError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "public_key") ||
		strings.Contains(msg, "principal") ||
		strings.Contains(msg, "hostname") ||
		strings.Contains(msg, "certificate is required") ||
		strings.Contains(msg, "parse ssh") ||
		strings.Contains(msg, "invalid ttl") ||
		strings.Contains(msg, "not an SSH certificate") ||
		strings.Contains(msg, "at least one principal")
}

func (h *SSHHandler) persistSSHCertificate(ctx context.Context, resp *models.SSHCertificateResponse) {
	if h.sshStore == nil || resp == nil {
		return
	}

	fingerprint, err := ca.SSHCertificateFingerprint(resp.Certificate)
	if err != nil {
		log.Printf("ssh: persist certificate fingerprint: %v", err)
		return
	}

	validAfter, err := time.Parse(time.RFC3339, resp.ValidAfter)
	if err != nil {
		log.Printf("ssh: persist certificate valid_after: %v", err)
		return
	}
	validBefore, err := time.Parse(time.RFC3339, resp.ValidBefore)
	if err != nil {
		log.Printf("ssh: persist certificate valid_before: %v", err)
		return
	}

	entry := db.SSHCertificate{
		Serial:      fmt.Sprintf("%d", resp.Serial),
		CertType:    resp.CertificateType,
		Principals:  resp.Principals,
		Fingerprint: fingerprint,
		ValidAfter:  validAfter,
		ValidBefore: validBefore,
	}

	if err := h.sshStore.Insert(ctx, entry); err != nil {
		log.Printf("ssh: persist certificate: %v", err)
	}
}

func parseSSHQueryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
