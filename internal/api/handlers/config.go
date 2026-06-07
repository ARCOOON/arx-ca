package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/ARCOOON/arx-ca/internal/notifications"
)

const maxSettingsConfigBody = 256 * 1024

// ConfigHandler serves server.yaml settings management endpoints.
type ConfigHandler struct {
	manager *config.Manager
	audit   *db.AuditStore
	notify  *notifications.Dispatcher
}

// NewConfigHandler constructs a ConfigHandler.
func NewConfigHandler(manager *config.Manager, audit *db.AuditStore, notify *notifications.Dispatcher) *ConfigHandler {
	return &ConfigHandler{
		manager: manager,
		audit:   audit,
		notify:  notify,
	}
}

// Get handles GET /api/v1/settings/config.
func (h *ConfigHandler) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.manager == nil {
			api.WriteError(w, http.StatusInternalServerError, "configuration manager is unavailable")
			return
		}

		cfg := h.manager.GetMasked()
		api.WriteSuccess(w, http.StatusOK, models.NewSettingsConfigResponse(cfg))
	})
}

// Put handles PUT /api/v1/settings/config.
func (h *ConfigHandler) Put() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.manager == nil {
			api.WriteError(w, http.StatusInternalServerError, "configuration manager is unavailable")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxSettingsConfigBody)
		defer r.Body.Close()

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			api.WriteError(w, http.StatusBadRequest, "request body is required")
			return
		}

		var patch config.ServerConfigPatch
		if err := json.Unmarshal(raw, &patch); err != nil {
			api.WriteError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		if settingsPatchEmpty(patch) {
			api.WriteError(w, http.StatusBadRequest, "at least one configuration section is required")
			return
		}

		updated, err := h.manager.Update(patch)
		if err != nil {
			log.Printf("settings/config: update failed: %v", err)
			api.WriteError(w, http.StatusBadRequest, sanitizeConfigError(err))
			return
		}

		notifications.RecordSystemEvent(h.audit, h.notify, db.ActionSysConfigUpdate, map[string]any{
			"path":    h.manager.Path(),
			"source":  "webui",
			"method":  r.Method,
			"actor":   actorFromRequest(r),
			"updated": changedSections(patch),
		})

		api.WriteSuccess(w, http.StatusOK, models.NewSettingsConfigResponse(updated))
	})
}

func settingsPatchEmpty(patch config.ServerConfigPatch) bool {
	return patch.Server == nil &&
		patch.Database == nil &&
		patch.CA == nil &&
		patch.CABootstrap == nil &&
		patch.Security == nil &&
		patch.Bootstrap == nil &&
		patch.Telemetry == nil &&
		patch.Service == nil &&
		patch.WebUI == nil &&
		patch.Updater == nil
}

func changedSections(patch config.ServerConfigPatch) []string {
	sections := make([]string, 0, 10)
	if patch.Server != nil {
		sections = append(sections, "server")
	}
	if patch.Database != nil {
		sections = append(sections, "database")
	}
	if patch.CA != nil {
		sections = append(sections, "ca")
	}
	if patch.CABootstrap != nil {
		sections = append(sections, "ca_bootstrap")
	}
	if patch.Security != nil {
		sections = append(sections, "security")
	}
	if patch.Bootstrap != nil {
		sections = append(sections, "bootstrap")
	}
	if patch.Telemetry != nil {
		sections = append(sections, "telemetry")
	}
	if patch.Service != nil {
		sections = append(sections, "service")
	}
	if patch.WebUI != nil {
		sections = append(sections, "webui")
	}
	if patch.Updater != nil {
		sections = append(sections, "updater")
	}
	return sections
}

func sanitizeConfigError(err error) string {
	if err == nil {
		return "configuration update failed"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "configuration update failed"
	}
	lower := strings.ToLower(message)
	if strings.Contains(lower, "password") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "jwt") {
		return "configuration update failed"
	}
	return message
}

func actorFromRequest(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if actor := strings.TrimSpace(r.Header.Get("X-Actor-Email")); actor != "" {
		return actor
	}
	return "administrator"
}
