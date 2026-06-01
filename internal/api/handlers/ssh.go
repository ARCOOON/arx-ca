package handlers

import (
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

// NewSSHHandler constructs an SSH handler bound to the PKI engine.
func NewSSHHandler(engine *ca.PKIEngine, jwtManager *auth.JWTManager, keyStore *auth.APIKeyStore) *SSHHandler {
	return &SSHHandler{
		engine:     engine,
		jwtManager: jwtManager,
		keyStore:   keyStore,
	}
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
