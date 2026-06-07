package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// AuditHandler serves forensic audit log query endpoints.
type AuditHandler struct {
	store *db.AuditStore
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(store *db.AuditStore) *AuditHandler {
	return &AuditHandler{store: store}
}

// List handles GET /api/v1/audit.
func (h *AuditHandler) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "audit store is unavailable")
			return
		}

		limit := parseAuditQueryInt(r, "limit", 50)
		offset := parseAuditQueryInt(r, "offset", 0)

		result, err := h.store.List(r.Context(), limit, offset)
		if err != nil {
			log.Printf("audit: list: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to list audit logs")
			return
		}

		logs := make([]models.AuditLogEntry, 0, len(result.Logs))
		for _, entry := range result.Logs {
			logs = append(logs, models.AuditLogEntry{
				ID:          entry.ID,
				Timestamp:   entry.Timestamp.UTC().Format("2006-01-02T15:04:05.000000Z"),
				RequestID:   entry.RequestID,
				IPAddress:   entry.IPAddress,
				HTTPMethod:  entry.HTTPMethod,
				Endpoint:    entry.Endpoint,
				StatusCode:  entry.StatusCode,
				ActorType:   entry.ActorType,
				ActorID:     entry.ActorID,
				ActorRoles:  entry.ActorRoles,
				Action:      entry.Action,
				Provisioner: entry.Provisioner,
				Fingerprint: entry.Fingerprint,
				Metadata:    entry.Metadata,
			})
		}

		api.WriteSuccess(w, http.StatusOK, models.ListAuditLogsResponse{
			Logs:   logs,
			Total:  result.Total,
			Limit:  limit,
			Offset: offset,
		})
	})
}

func parseAuditQueryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
