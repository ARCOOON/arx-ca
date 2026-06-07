package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/ARCOOON/arx-ca/internal/repository"
)

const maxAuthBodyBytes = 1 << 20 // 1 MiB

// AuthHandler serves authentication and service-account management endpoints.
type AuthHandler struct {
	jwtManager    *auth.JWTManager
	keyStore      *auth.APIKeyStore
	userStore     *repository.UserStore
	sessionCookie auth.SessionCookieConfig
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(
	jwtManager *auth.JWTManager,
	keyStore *auth.APIKeyStore,
	userStore *repository.UserStore,
	sessionCookie auth.SessionCookieConfig,
) *AuthHandler {
	return &AuthHandler{
		jwtManager:    jwtManager,
		keyStore:      keyStore,
		userStore:     userStore,
		sessionCookie: sessionCookie,
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
			logging.Logger().Debug("auth: login failed", slog.String("reason", "invalid json payload"), slog.Any("error", err))
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("email", strings.TrimSpace(req.Email))
		}

		user, reason, err := auth.AuthenticateUser(r.Context(), h.userStore, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				if reason != "" {
					logging.Logger().Debug("auth: login failed",
						slog.String("reason", string(reason)),
						slog.String("email", strings.TrimSpace(req.Email)),
					)
				}
				recordAuditAction(r, db.ActionAuthLoginFailed)
				if ac := auditFromRequest(r); ac != nil {
					ac.PutMetadata("result", "failure")
				}
				api.WriteError(w, http.StatusUnauthorized, "invalid email or password")
				return
			}
			logging.Logger().Error("auth: login credential validation error", slog.Any("error", err))
			api.WriteError(w, http.StatusInternalServerError, "login failed")
			return
		}

		roles := auth.RolesForUser(user)
		if len(roles) == 0 {
			roles = auth.RolesForAdmin(user.Email)
		}
		token, expiresAt, err := h.jwtManager.GenerateToken(user.Email, roles)
		if err != nil {
			logging.Logger().Error("auth: generate jwt", slog.Any("error", err))
			api.WriteError(w, http.StatusInternalServerError, "login failed")
			return
		}

		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = string(role)
		}

		recordAuditAction(r, db.ActionAuthLoginSuccess)
		logging.Logger().Debug("auth: login succeeded", slog.String("email", user.Email))
		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("result", "success")
			roleNames := make([]string, len(roles))
			for i, role := range roles {
				roleNames[i] = string(role)
			}
			ac.SetActor("User", user.Email, roleNames...)
		}

		auth.SetSessionCookie(w, r, h.sessionCookie, token)

		api.WriteSuccess(w, http.StatusOK, models.LoginResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			TokenType: h.jwtManager.TokenType(),
			Roles:     roleNames,
		})
	})
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		recordAuditAction(r, "AUTH_LOGOUT")

		auth.ClearSessionCookie(w, r, h.sessionCookie)
		api.WriteSuccess(w, http.StatusOK, map[string]string{"status": "logged_out"})
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

		recordAuditAction(r, "SERVICE_ACCOUNT_CREATE")
		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("service_account_name", strings.TrimSpace(req.Name))
		}

		account, plaintextKey, err := h.keyStore.CreateServiceAccount(req.Name, roles)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidServiceAccountName):
				api.WriteError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, auth.ErrDuplicateServiceAccount):
				api.WriteError(w, http.StatusConflict, "service account name already exists")
			default:
				logging.Logger().Error("auth: create service account", slog.Any("error", err))
				api.WriteError(w, http.StatusInternalServerError, "failed to create service account")
			}
			return
		}

		roleNames := make([]string, len(account.Roles))
		for i, role := range account.Roles {
			roleNames[i] = string(role)
		}

		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("service_account_id", account.ID)
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
