package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
	"github.com/google/uuid"
)

// RecordSystemEvent persists a system-originated audit entry and notifies subscribers.
func RecordSystemEvent(store *db.AuditStore, notifier *Dispatcher, action string, metadata map[string]any) {
	if store == nil {
		return
	}

	entry := db.AuditLog{
		ID:         uuid.NewString(),
		Timestamp:  time.Now().UTC(),
		RequestID:  uuid.NewString(),
		IPAddress:  "127.0.0.1",
		HTTPMethod: "SYSTEM",
		Endpoint:   "/system",
		StatusCode: 200,
		ActorType:  "System",
		ActorID:    "arx-ca",
		Action:     action,
		Metadata:   metadata,
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Insert(ctx, entry); err != nil {
		logging.Logger().Error("audit: persist system event",
			slog.Any("error", err),
			slog.String("action", action),
		)
		return
	}
	if notifier != nil {
		notifier.NotifyAudit(entry)
	}
}
