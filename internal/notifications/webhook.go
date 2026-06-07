package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
)

const (
	defaultHTTPTimeout = 5 * time.Second
	maxDispatchRetries = 2
	retryBackoffBase   = 500 * time.Millisecond
)

// NotifiableActions lists audit actions operators can subscribe to for outbound alerts.
var NotifiableActions = []string{
	db.ActionSysStart,
	db.ActionSysConfigUpdate,
	db.ActionSysUpdateAvailable,
	db.ActionSysUpdateApplied,
	db.ActionAuthLoginSuccess,
	db.ActionAuthLoginFailed,
	db.ActionCertIssueNative,
	db.ActionCertIssueCSR,
	db.ActionCertRevoke,
	db.ActionCertRenew,
	db.ActionEABGenerate,
	db.ActionEABRevoke,
	db.ActionSCEPChallengeRotated,
	db.ActionSSHUserCertIssue,
	db.ActionSSHHostCertIssue,
	db.ActionWebhookCreated,
	db.ActionWebhookDeleted,
	db.ActionWebhookUpdated,
}

// Payload is the standardized JSON body delivered to webhook endpoints and SSE clients.
type Payload struct {
	NotificationID string          `json:"notification_id,omitempty"`
	Timestamp      string          `json:"timestamp"`
	Action         string          `json:"action"`
	Actor          ActorPayload    `json:"actor"`
	IPAddress      string          `json:"ip_address"`
	Resource       ResourcePayload `json:"resource"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
	RequestID      string          `json:"request_id,omitempty"`
	HTTPMethod     string          `json:"http_method,omitempty"`
	Endpoint       string          `json:"endpoint,omitempty"`
	StatusCode     int             `json:"status_code,omitempty"`
}

// ActorPayload identifies who performed the audited action.
type ActorPayload struct {
	Type  string   `json:"type"`
	ID    string   `json:"id"`
	Roles []string `json:"roles,omitempty"`
}

// ResourcePayload carries certificate or provisioner context from the audit entry.
type ResourcePayload struct {
	Provisioner string `json:"provisioner,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// DispatchResult summarizes a single webhook delivery attempt.
type DispatchResult struct {
	StatusCode int
	Latency    time.Duration
	Error      string
}

// Dispatcher asynchronously delivers audit events to subscribed webhook endpoints,
// persists operator notifications, and broadcasts payloads to SSE clients.
type Dispatcher struct {
	webhookStore      *db.WebhookStore
	notificationStore *db.NotificationStore
	client            *http.Client
	sseMu             sync.RWMutex
	sseClients        map[string]*sseClient
}

// NewDispatcher constructs a webhook dispatcher backed by webhookStore and notificationStore.
func NewDispatcher(webhookStore *db.WebhookStore, notificationStore *db.NotificationStore) *Dispatcher {
	return &Dispatcher{
		webhookStore:      webhookStore,
		notificationStore: notificationStore,
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

// NotifyAudit elevates whitelisted audit actions to operator notifications and SSE,
// and schedules webhook delivery for subscribed outbound endpoints.
func (d *Dispatcher) NotifyAudit(entry db.AuditLog) {
	if d == nil || d.webhookStore == nil {
		return
	}
	action := strings.TrimSpace(entry.Action)
	if action == "" {
		return
	}

	payload := PayloadFromAudit(entry)
	if ShouldElevateOperatorNotification(action) {
		if d.notificationStore != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stored, err := d.notificationStore.Insert(ctx, db.Notification{
				Timestamp: entry.Timestamp,
				Action:    entry.Action,
				Level:     NotificationLevel(entry.Action),
				Message:   NotificationMessage(entry),
				Metadata:  notificationMetadata(entry),
			})
			cancel()
			if err != nil {
				logging.Logger().Error("notification: persist audit event",
					slog.Any("error", err),
					slog.String("action", action),
				)
			} else {
				payload.NotificationID = stored.ID
			}
		}

		if body, err := json.Marshal(payload); err == nil {
			d.broadcastSSE(body)
		}
	}

	if ValidNotifiableAction(action) {
		go d.dispatchAction(action, payload)
	}
}

// DispatchTest delivers a synthetic payload to a specific webhook for connectivity checks.
func (d *Dispatcher) DispatchTest(ctx context.Context, webhook db.Webhook) DispatchResult {
	payload := Payload{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Action:    "WEBHOOK_TEST",
		Actor: ActorPayload{
			Type: "System",
			ID:   "arx-ca",
		},
		IPAddress: "127.0.0.1",
		Resource: ResourcePayload{
			Provisioner: "test",
			Fingerprint: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		Metadata: map[string]any{
			"test": true,
		},
	}
	return d.deliver(ctx, webhook, payload)
}

// PayloadFromAudit maps a persisted audit log entry to the outbound webhook payload.
func PayloadFromAudit(entry db.AuditLog) Payload {
	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return Payload{
		Timestamp: ts.UTC().Format(time.RFC3339Nano),
		Action:    entry.Action,
		Actor: ActorPayload{
			Type:  entry.ActorType,
			ID:    entry.ActorID,
			Roles: entry.ActorRoles,
		},
		IPAddress: entry.IPAddress,
		Resource: ResourcePayload{
			Provisioner: entry.Provisioner,
			Fingerprint: entry.Fingerprint,
		},
		Metadata:   entry.Metadata,
		RequestID:  entry.RequestID,
		HTTPMethod: entry.HTTPMethod,
		Endpoint:   entry.Endpoint,
		StatusCode: entry.StatusCode,
	}
}

func (d *Dispatcher) dispatchAction(action string, payload Payload) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout*time.Duration(maxDispatchRetries+1)+retryBackoffBase*2)
	defer cancel()

	webhooks, err := d.webhookStore.ListActiveSubscribed(ctx, action)
	if err != nil {
		logging.Logger().Error("webhook: list subscribed endpoints",
			slog.Any("error", err),
			slog.String("action", action),
		)
		return
	}
	for _, webhook := range webhooks {
		wh := webhook
		go func() {
			result := d.deliver(context.Background(), wh, payload)
			if result.Error != "" {
				logging.Logger().Warn("webhook: delivery failed",
					slog.String("webhook_id", wh.ID),
					slog.String("webhook_name", wh.Name),
					slog.String("action", action),
					slog.String("error", result.Error),
					slog.Int("status_code", result.StatusCode),
				)
				return
			}
			logging.Logger().Debug("webhook: delivered",
				slog.String("webhook_id", wh.ID),
				slog.String("action", action),
				slog.Int("status_code", result.StatusCode),
				slog.Duration("latency", result.Latency),
			)
		}()
	}
}

func (d *Dispatcher) deliver(parent context.Context, webhook db.Webhook, payload Payload) DispatchResult {
	url := strings.TrimSpace(webhook.URL)
	if url == "" {
		return DispatchResult{Error: "webhook url is empty"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return DispatchResult{Error: fmt.Sprintf("marshal payload: %v", err)}
	}

	var lastResult DispatchResult
	for attempt := 0; attempt <= maxDispatchRetries; attempt++ {
		if attempt > 0 {
			backoff := retryBackoffBase * time.Duration(attempt)
			select {
			case <-parent.Done():
				lastResult.Error = parent.Err().Error()
				return lastResult
			case <-time.After(backoff):
			}
		}

		ctx, cancel := context.WithTimeout(parent, defaultHTTPTimeout)
		lastResult = d.deliverOnce(ctx, url, webhook.SecretToken, body)
		cancel()

		if lastResult.Error == "" && lastResult.StatusCode >= 200 && lastResult.StatusCode < 300 {
			return lastResult
		}
		if lastResult.Error == "" {
			lastResult.Error = fmt.Sprintf("unexpected status code %d", lastResult.StatusCode)
		}
	}
	return lastResult
}

func (d *Dispatcher) deliverOnce(ctx context.Context, url, secret string, body []byte) DispatchResult {
	started := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return DispatchResult{Error: fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "arx-ca-webhook/1.0")
	if secret = strings.TrimSpace(secret); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", "sha256="+signature)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return DispatchResult{Error: err.Error(), Latency: time.Since(started)}
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	return DispatchResult{
		StatusCode: resp.StatusCode,
		Latency:    time.Since(started),
	}
}

// ValidNotifiableAction reports whether action can be subscribed to.
func ValidNotifiableAction(action string) bool {
	action = strings.TrimSpace(action)
	for _, candidate := range NotifiableActions {
		if strings.EqualFold(candidate, action) {
			return true
		}
	}
	return false
}

// NormalizeSubscribedEvents uppercases known actions and drops unknown values.
func NormalizeSubscribedEvents(events []string) []string {
	if len(events) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(events))
	out := make([]string, 0, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		if !ValidNotifiableAction(event) {
			continue
		}
		for _, candidate := range NotifiableActions {
			if strings.EqualFold(candidate, event) {
				event = candidate
				break
			}
		}
		if _, exists := seen[event]; exists {
			continue
		}
		seen[event] = struct{}{}
		out = append(out, event)
	}
	return out
}
