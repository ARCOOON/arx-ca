package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/your-org/arx-ca/internal/api"
	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/models"
)

const maxAuthBodyBytes = 1 << 20 // 1 MiB

// AuthHandler serves authentication and service-account management endpoints.
type AuthHandler struct {
	jwtManager *auth.JWTManager
	keyStore   *auth.APIKeyStore
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(jwtManager *auth.JWTManager, keyStore *auth.APIKeyStore) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		keyStore:   keyStore,
	}
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.LoginRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := auth.ValidateAdminCredentials(req.Username, req.Password); err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				api.WriteError(w, http.StatusUnauthorized, "invalid username or password")
				return
			}
			log.Printf("auth: login credential validation error: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "login failed")
			return
		}

		roles := auth.RolesForAdmin(req.Username)
		token, expiresAt, err := h.jwtManager.GenerateToken(req.Username, roles)
		if err != nil {
			log.Printf("auth: generate jwt: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "login failed")
			return
		}

		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = string(role)
		}

		api.WriteSuccess(w, http.StatusOK, models.LoginResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			TokenType: h.jwtManager.TokenType(),
			Roles:     roleNames,
		})
	})
}

// CreateServiceAccount handles POST /api/v1/auth/service-accounts (admin only).
func (h *AuthHandler) CreateServiceAccount() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.CreateServiceAccountRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		roles := make([]auth.Role, 0, len(req.Roles))
		for _, name := range req.Roles {
			role := auth.Role(strings.TrimSpace(name))
			if auth.ValidRole(role) {
				roles = append(roles, role)
			}
		}

		account, plaintextKey, err := h.keyStore.CreateServiceAccount(req.Name, roles)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidServiceAccountName):
				api.WriteError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, auth.ErrDuplicateServiceAccount):
				api.WriteError(w, http.StatusConflict, "service account name already exists")
			default:
				log.Printf("auth: create service account: %v", err)
				api.WriteError(w, http.StatusInternalServerError, "failed to create service account")
			}
			return
		}

		roleNames := make([]string, len(account.Roles))
		for i, role := range account.Roles {
			roleNames[i] = string(role)
		}

		api.WriteSuccess(w, http.StatusCreated, models.ServiceAccountResponse{
			ID:        account.ID,
			Name:      account.Name,
			Roles:     roleNames,
			APIKey:    plaintextKey,
			CreatedAt: account.CreatedAt,
		})
	})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return errors.New("malformed JSON body")
		}
		return errors.New("invalid request body")
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
