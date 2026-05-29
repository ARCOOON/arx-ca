package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/ca"
	"github.com/your-org/arx-ca/internal/models"
)

// SSHHandler serves SSH certificate authority endpoints.
type SSHHandler struct {
	engine     *ca.PKIEngine
	jwtManager *auth.JWTManager
	keyStore   *auth.APIKeyStore
}

// NewSSHHandler constructs an SSH handler bound to the PKI engine and API authentication.
func NewSSHHandler(engine *ca.PKIEngine, jwtManager *auth.JWTManager, keyStore *auth.APIKeyStore) *SSHHandler {
	return &SSHHandler{
		engine:     engine,
		jwtManager: jwtManager,
		keyStore:   keyStore,
	}
}

// SignUser handles POST /api/v1/ssh/sign-user.
// Accepts either API authentication (admin JWT or service-account API key) or an OIDC token
// from an SSO/OIDC provisioner configured in the CA.
func (h *SSHHandler) SignUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if !h.engine.SSHEnabled() {
			api.WriteError(w, http.StatusServiceUnavailable, "SSH certificate authority is not configured")
			return
		}

		var req models.SignSSHUserRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		oidcToken := strings.TrimSpace(req.OIDCToken)
		if oidcToken == "" {
			if err := h.requireAPIAuthentication(w, r); err != nil {
				return
			}
		}

		resp, err := h.engine.SignSSHUser(r.Context(), req, oidcToken)
		if err != nil {
			h.writeSSHError(w, err)
			return
		}

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

		if !h.engine.SSHEnabled() {
			api.WriteError(w, http.StatusServiceUnavailable, "SSH certificate authority is not configured")
			return
		}

		var req models.SignSSHHostRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := h.engine.SignSSHHost(r.Context(), req)
		if err != nil {
			h.writeSSHError(w, err)
			return
		}

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

		resp, err := h.engine.InspectSSHCertificate(r.Context(), req.Certificate)
		if err != nil {
			h.writeSSHError(w, err)
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// Roots handles GET /api/v1/ssh/roots.
func (h *SSHHandler) Roots() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if !h.engine.SSHEnabled() {
			api.WriteError(w, http.StatusServiceUnavailable, "SSH certificate authority is not configured")
			return
		}

		resp, err := h.engine.GetSSHRoots(r.Context())
		if err != nil {
			h.writeSSHError(w, err)
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

func (h *SSHHandler) requireAPIAuthentication(w http.ResponseWriter, r *http.Request) error {
	if h.authenticateAPIRequest(r) {
		return nil
	}
	api.WriteError(w, http.StatusUnauthorized, "authentication required: provide admin JWT, service-account API key, or oidc_token")
	return errors.New("unauthenticated")
}

func (h *SSHHandler) authenticateAPIRequest(r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token != "" {
			if _, err := h.jwtManager.ValidateToken(token); err == nil {
				return true
			}
			if _, err := h.keyStore.ValidateAPIKey(token); err == nil {
				return true
			}
		}
	}

	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		if _, err := h.keyStore.ValidateAPIKey(key); err == nil {
			return true
		}
	}

	return false
}

func (h *SSHHandler) writeSSHError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not configured"),
		strings.Contains(msg, "no SSH CA"):
		api.WriteError(w, http.StatusServiceUnavailable, msg)
	case strings.Contains(msg, "required"),
		strings.Contains(msg, "invalid"),
		strings.Contains(msg, "parse"),
		strings.Contains(msg, "malformed"):
		api.WriteError(w, http.StatusBadRequest, msg)
	case strings.Contains(msg, "no OIDC provisioner"),
		strings.Contains(msg, "unauthorized"),
		strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "not allowed"):
		status, message := ca.MapCAError(err)
		api.WriteError(w, status, message)
	default:
		status, message := ca.MapCAError(err)
		if status >= http.StatusInternalServerError {
			log.Printf("ssh: %v", err)
		}
		api.WriteError(w, status, message)
	}
}
