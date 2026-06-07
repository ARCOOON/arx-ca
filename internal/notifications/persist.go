package notifications

import (
	"fmt"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/db"
)

// operatorNotificationActions lists audit actions elevated to persistent operator
// notifications and SSE broadcasts. All other audit events remain in the immutable
// audit log only.
var operatorNotificationActions = map[string]struct{}{
	db.ActionAuthLoginFailed:    {},
	db.ActionCertIssueNative:    {},
	db.ActionCertIssueCSR:       {},
	db.ActionCertRevoke:         {},
	db.ActionCertRenew:          {},
	db.ActionEABGenerate:        {},
	db.ActionEABRevoke:          {},
	db.ActionSysUpdateAvailable: {},
}

// suppressedOperatorActions are explicit audit action identifiers that must never
// produce operator notifications even when handlers attach them in the future.
var suppressedOperatorActions = map[string]struct{}{
	"NOTIFICATION_READ":        {},
	"NOTIFICATION_READ_ALL":    {},
	"NOTIFICATION_DELETE":      {},
	"NOTIFICATION_DELETE_ALL":  {},
	"NOTIFICATION_ARCHIVE_ALL": {},
	"AUDIT_LIST":               {},
	"HTTP_GET":                 {},
	"HTTP_READ":                {},
}

// ShouldElevateOperatorNotification reports whether an audit action should be
// persisted as an operator notification and broadcast over SSE.
func ShouldElevateOperatorNotification(action string) bool {
	action = strings.ToUpper(strings.TrimSpace(action))
	if action == "" {
		return false
	}
	if _, suppressed := suppressedOperatorActions[action]; suppressed {
		return false
	}
	if strings.HasPrefix(action, "HTTP_") {
		return false
	}
	_, allowed := operatorNotificationActions[action]
	return allowed
}

var criticalNotificationActions = map[string]struct{}{
	db.ActionAuthLoginFailed: {},
	db.ActionCertRevoke:      {},
	db.ActionEABRevoke:       {},
}

// NotificationLevel maps an audit action to a persisted notification severity.
func NotificationLevel(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return db.NotificationLevelInfo
	}
	if _, critical := criticalNotificationActions[strings.ToUpper(action)]; critical {
		return db.NotificationLevelCritical
	}
	return db.NotificationLevelInfo
}

// NotificationMessage formats a human-readable summary for operator notifications.
func NotificationMessage(entry db.AuditLog) string {
	if msg, ok := entry.Metadata["message"].(string); ok {
		if trimmed := strings.TrimSpace(msg); trimmed != "" {
			return trimmed
		}
	}
	actor := strings.TrimSpace(entry.ActorID)
	if actor == "" {
		actor = "system"
	}
	actionLabel := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(entry.Action)), "_", " ")
	return fmt.Sprintf("%s: %s", actor, actionLabel)
}

func notificationMetadata(entry db.AuditLog) map[string]any {
	metadata := map[string]any{
		"request_id":  entry.RequestID,
		"ip_address":  entry.IPAddress,
		"http_method": entry.HTTPMethod,
		"endpoint":    entry.Endpoint,
		"status_code": entry.StatusCode,
		"actor_type":  entry.ActorType,
		"actor_id":    entry.ActorID,
	}
	if len(entry.ActorRoles) > 0 {
		metadata["actor_roles"] = entry.ActorRoles
	}
	if provisioner := strings.TrimSpace(entry.Provisioner); provisioner != "" {
		metadata["provisioner"] = provisioner
	}
	if fingerprint := strings.TrimSpace(entry.Fingerprint); fingerprint != "" {
		metadata["fingerprint"] = fingerprint
	}
	for key, value := range entry.Metadata {
		metadata[key] = value
	}
	return metadata
}
