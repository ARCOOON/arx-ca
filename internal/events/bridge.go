package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
	"github.com/ARCOOON/arx-ca/internal/notifications"
	"github.com/google/uuid"
)

// BridgeDeps carries dependencies for event bridge subscribers.
type BridgeDeps struct {
	AuditStore *db.AuditStore
	Notifier   *notifications.Dispatcher
}

// RegisterBridges wires audit, notification, and structured logging subscribers.
func RegisterBridges(manager *Manager, deps BridgeDeps) {
	if manager == nil {
		return
	}

	manager.Subscribe(EventAuditRecorded, func(evt Event) {
		handleAuditRecorded(deps, evt.Payload)
	})

	systemEvents := []string{
		EventSystemStarted,
		EventSystemConfigUpdated,
		EventSystemUpdateAvail,
		EventSystemUpdateApplied,
	}
	for _, name := range systemEvents {
		manager.Subscribe(name, func(evt Event) {
			handleSystemEvent(deps, evt.Name, evt.Payload)
		})
	}

	certEvents := []string{
		EventCertIssuedNative,
		EventCertIssuedCSR,
		EventCertIssuedAuto,
		EventCertRevoked,
		EventCertRenewed,
		EventCertRekeyed,
	}
	for _, name := range certEvents {
		manager.Subscribe(name, func(evt Event) {
			logging.Logger().Info("certificate event",
				slog.String("event", name),
				slog.String("serial", stringValue(evt.Payload, KeySerial)),
				slog.String("alias", stringValue(evt.Payload, KeyAlias)),
			)
		})
	}
}

func handleAuditRecorded(deps BridgeDeps, payload map[string]any) {
	if deps.AuditStore == nil {
		return
	}

	entry := auditLogFromPayload(payload)
	if entry.Action == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := deps.AuditStore.Insert(ctx, entry); err != nil {
		logging.Logger().Error("events: persist audit entry",
			slog.Any("error", err),
			slog.String("action", entry.Action),
		)
		return
	}
	if deps.Notifier != nil {
		deps.Notifier.NotifyAudit(entry)
	}
}

func handleSystemEvent(deps BridgeDeps, eventName string, payload map[string]any) {
	if deps.AuditStore == nil {
		return
	}

	action := systemEventToAuditAction(eventName)
	if action == "" {
		return
	}

	metadata := map[string]any{}
	for k, v := range payload {
		metadata[k] = v
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := deps.AuditStore.Insert(ctx, entry); err != nil {
		logging.Logger().Error("events: persist system event",
			slog.Any("error", err),
			slog.String("event", eventName),
		)
		return
	}
	if deps.Notifier != nil {
		deps.Notifier.NotifyAudit(entry)
	}
}

func systemEventToAuditAction(eventName string) string {
	switch eventName {
	case EventSystemStarted:
		return db.ActionSysStart
	case EventSystemConfigUpdated:
		return db.ActionSysConfigUpdate
	case EventSystemUpdateAvail:
		return db.ActionSysUpdateAvailable
	case EventSystemUpdateApplied:
		return db.ActionSysUpdateApplied
	default:
		return ""
	}
}

func auditLogFromPayload(payload map[string]any) db.AuditLog {
	entry := db.AuditLog{
		ID:          uuid.NewString(),
		Timestamp:   time.Now().UTC(),
		RequestID:   stringValue(payload, KeyRequestID),
		IPAddress:   stringValue(payload, KeyIPAddress),
		HTTPMethod:  stringValue(payload, KeyHTTPMethod),
		Endpoint:    stringValue(payload, KeyEndpoint),
		StatusCode:  intValue(payload, KeyStatusCode),
		Action:      stringValue(payload, KeyAction),
		Provisioner: stringValue(payload, KeyProvisioner),
		Fingerprint: stringValue(payload, KeyFingerprint),
		ActorType:   stringValue(payload, KeyActorType),
		ActorID:     stringValue(payload, KeyActorID),
	}
	if entry.RequestID == "" {
		entry.RequestID = uuid.NewString()
	}
	if entry.HTTPMethod == "" {
		entry.HTTPMethod = "SYSTEM"
	}
	if entry.Endpoint == "" {
		entry.Endpoint = "/system"
	}
	if entry.StatusCode == 0 {
		entry.StatusCode = 200
	}
	if roles, ok := payload[KeyActorRoles].([]string); ok && len(roles) > 0 {
		entry.ActorRoles = append([]string(nil), roles...)
	}
	if meta, ok := payload[KeyMetadata].(map[string]any); ok {
		entry.Metadata = meta
	} else {
		entry.Metadata = map[string]any{}
	}
	return entry
}

func stringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func intValue(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// EmitSystemEvent triggers a system-originated event through the manager.
func EmitSystemEvent(manager *Manager, name string, payload map[string]any) {
	if manager == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	manager.Trigger(name, payload)
}
