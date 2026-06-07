package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/ARCOOON/arx-ca/internal/notifications"
)

// NotificationHandler serves real-time and persistent notification endpoints for the WebUI.
type NotificationHandler struct {
	dispatcher *notifications.Dispatcher
	store      *db.NotificationStore
}

// NewNotificationHandler constructs a NotificationHandler.
func NewNotificationHandler(dispatcher *notifications.Dispatcher, store *db.NotificationStore) *NotificationHandler {
	return &NotificationHandler{
		dispatcher: dispatcher,
		store:      store,
	}
}

// Stream handles GET /api/v1/notifications/stream (Server-Sent Events).
func (h *NotificationHandler) Stream() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.dispatcher == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification dispatcher is unavailable")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			api.WriteError(w, http.StatusInternalServerError, "streaming is not supported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		_, events, unregister := h.dispatcher.RegisterSSEClient()
		defer unregister()

		if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
			return
		}
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case payload, open := <-events:
				if !open {
					return
				}
				if _, err := fmt.Fprintf(w, "event: audit\ndata: %s\n\n", payload); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})
}

// List handles GET /api/v1/notifications.
func (h *NotificationHandler) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification store is unavailable")
			return
		}

		limit := parseNotificationQueryInt(r, "limit", 50)
		offset := parseNotificationQueryInt(r, "offset", 0)
		unreadOnly := parseNotificationQueryBool(r, "unread")

		result, err := h.store.List(r.Context(), db.NotificationListOptions{
			Limit:      limit,
			Offset:     offset,
			UnreadOnly: unreadOnly,
		})
		if err != nil {
			log.Printf("notifications: list: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to list notifications")
			return
		}

		items := make([]models.NotificationEntry, 0, len(result.Notifications))
		for _, entry := range result.Notifications {
			items = append(items, models.NotificationEntry{
				ID:        entry.ID,
				Timestamp: entry.Timestamp.UTC().Format("2006-01-02T15:04:05.000000Z"),
				Action:    entry.Action,
				Level:     entry.Level,
				Message:   entry.Message,
				IsRead:    entry.IsRead,
				Metadata:  entry.Metadata,
			})
		}

		api.WriteSuccess(w, http.StatusOK, models.ListNotificationsResponse{
			Notifications: items,
			Total:         result.Total,
			UnreadCount:   result.UnreadCount,
			Limit:         limit,
			Offset:        offset,
		})
	})
}

// MarkRead handles POST /api/v1/notifications/{id}/read.
func (h *NotificationHandler) MarkRead() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification store is unavailable")
			return
		}

		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			api.WriteError(w, http.StatusBadRequest, "notification id is required")
			return
		}

		if err := h.store.MarkAsRead(r.Context(), id); err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "notification not found")
				return
			}
			log.Printf("notifications: mark read: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to mark notification as read")
			return
		}

		api.WriteSuccess(w, http.StatusOK, map[string]string{"id": id, "status": "read"})
	})
}

// MarkAllRead handles POST /api/v1/notifications/read-all.
func (h *NotificationHandler) MarkAllRead() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification store is unavailable")
			return
		}

		updated, err := h.store.MarkAllAsRead(r.Context())
		if err != nil {
			log.Printf("notifications: mark all read: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to mark all notifications as read")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.MarkAllNotificationsReadResponse{
			Updated: updated,
		})
	})
}

// ArchiveAll handles POST /api/v1/notifications/archive-all.
func (h *NotificationHandler) ArchiveAll() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification store is unavailable")
			return
		}

		archived, err := h.store.ArchiveAll(r.Context())
		if err != nil {
			log.Printf("notifications: archive all: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to archive all notifications")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.ArchiveAllNotificationsResponse{
			Archived: archived,
		})
	})
}

// Delete handles DELETE /api/v1/notifications/{id}.
func (h *NotificationHandler) Delete() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h.store == nil {
			api.WriteError(w, http.StatusInternalServerError, "notification store is unavailable")
			return
		}

		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			api.WriteError(w, http.StatusBadRequest, "notification id is required")
			return
		}

		if err := h.store.Delete(r.Context(), id); err != nil {
			if strings.Contains(err.Error(), "not found") {
				api.WriteError(w, http.StatusNotFound, "notification not found")
				return
			}
			log.Printf("notifications: delete: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to delete notification")
			return
		}

		api.WriteSuccess(w, http.StatusOK, map[string]string{"id": id, "status": "deleted"})
	})
}

func parseNotificationQueryInt(r *http.Request, key string, fallback int) int {
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

func parseNotificationQueryBool(r *http.Request, key string) bool {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	switch raw {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
