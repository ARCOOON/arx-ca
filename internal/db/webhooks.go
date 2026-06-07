package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Webhook represents an outbound notification endpoint subscribed to audit actions.
type Webhook struct {
	ID               string
	URL              string
	Name             string
	SecretToken      string
	Active           bool
	SubscribedEvents []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// WebhookStore provides CRUD access to configured webhooks.
type WebhookStore struct {
	db *sql.DB
}

// NewWebhookStore constructs a WebhookStore backed by db.
func NewWebhookStore(db *sql.DB) *WebhookStore {
	return &WebhookStore{db: db}
}

// List returns all webhooks ordered by name.
func (s *WebhookStore) List(ctx context.Context) ([]Webhook, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("webhook store is not initialized")
	}

	query := `
SELECT id, url, name, secret_token, active, subscribed_events, created_at, updated_at
FROM webhooks
ORDER BY name ASC, created_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]Webhook, 0)
	for rows.Next() {
		wh, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wh)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhooks: %w", err)
	}
	return out, nil
}

// GetByID returns a single webhook by identifier.
func (s *WebhookStore) GetByID(ctx context.Context, id string) (Webhook, error) {
	if s == nil || s.db == nil {
		return Webhook{}, fmt.Errorf("webhook store is not initialized")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return Webhook{}, fmt.Errorf("webhook id is required")
	}

	query := `
SELECT id, url, name, secret_token, active, subscribed_events, created_at, updated_at
FROM webhooks
WHERE id = ?`
	args := []any{id}
	if isPostgreSQL(s.db) {
		query = `
SELECT id, url, name, secret_token, active, subscribed_events, created_at, updated_at
FROM webhooks
WHERE id = $1`
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	wh, err := scanWebhook(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Webhook{}, fmt.Errorf("webhook not found")
		}
		return Webhook{}, err
	}
	return wh, nil
}

// Create inserts a new webhook row.
func (s *WebhookStore) Create(ctx context.Context, wh Webhook) (Webhook, error) {
	if s == nil || s.db == nil {
		return Webhook{}, fmt.Errorf("webhook store is not initialized")
	}

	now := time.Now().UTC()
	id := strings.TrimSpace(wh.ID)
	if id == "" {
		id = uuid.NewString()
	}

	wh.ID = id
	wh.URL = strings.TrimSpace(wh.URL)
	wh.Name = strings.TrimSpace(wh.Name)
	wh.SecretToken = strings.TrimSpace(wh.SecretToken)
	if wh.SubscribedEvents == nil {
		wh.SubscribedEvents = []string{}
	}
	if wh.CreatedAt.IsZero() {
		wh.CreatedAt = now
	}
	if wh.UpdatedAt.IsZero() {
		wh.UpdatedAt = now
	}

	eventsJSON, err := json.Marshal(wh.SubscribedEvents)
	if err != nil {
		return Webhook{}, fmt.Errorf("marshal subscribed_events: %w", err)
	}

	active := 0
	if wh.Active {
		active = 1
	}

	query := `
INSERT INTO webhooks (id, url, name, secret_token, active, subscribed_events, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		wh.ID,
		wh.URL,
		wh.Name,
		nullIfEmpty(wh.SecretToken),
		active,
		string(eventsJSON),
		wh.CreatedAt.UTC().Format(time.RFC3339Nano),
		wh.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if isPostgreSQL(s.db) {
		query = `
INSERT INTO webhooks (id, url, name, secret_token, active, subscribed_events, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return Webhook{}, fmt.Errorf("insert webhook: %w", err)
	}
	return wh, nil
}

// Update replaces an existing webhook row.
func (s *WebhookStore) Update(ctx context.Context, wh Webhook) (Webhook, error) {
	if s == nil || s.db == nil {
		return Webhook{}, fmt.Errorf("webhook store is not initialized")
	}

	wh.ID = strings.TrimSpace(wh.ID)
	if wh.ID == "" {
		return Webhook{}, fmt.Errorf("webhook id is required")
	}

	existing, err := s.GetByID(ctx, wh.ID)
	if err != nil {
		return Webhook{}, err
	}

	wh.URL = strings.TrimSpace(wh.URL)
	wh.Name = strings.TrimSpace(wh.Name)
	if wh.SubscribedEvents == nil {
		wh.SubscribedEvents = []string{}
	}

	secret := strings.TrimSpace(wh.SecretToken)
	if secret == "" {
		secret = existing.SecretToken
	}
	wh.SecretToken = secret
	wh.CreatedAt = existing.CreatedAt
	wh.UpdatedAt = time.Now().UTC()

	eventsJSON, err := json.Marshal(wh.SubscribedEvents)
	if err != nil {
		return Webhook{}, fmt.Errorf("marshal subscribed_events: %w", err)
	}

	active := 0
	if wh.Active {
		active = 1
	}

	query := `
UPDATE webhooks
SET url = ?, name = ?, secret_token = ?, active = ?, subscribed_events = ?, updated_at = ?
WHERE id = ?`
	args := []any{
		wh.URL,
		wh.Name,
		nullIfEmpty(wh.SecretToken),
		active,
		string(eventsJSON),
		wh.UpdatedAt.UTC().Format(time.RFC3339Nano),
		wh.ID,
	}
	if isPostgreSQL(s.db) {
		query = `
UPDATE webhooks
SET url = $1, name = $2, secret_token = $3, active = $4, subscribed_events = $5, updated_at = $6
WHERE id = $7`
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return Webhook{}, fmt.Errorf("update webhook: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Webhook{}, fmt.Errorf("update webhook rows affected: %w", err)
	}
	if rows == 0 {
		return Webhook{}, fmt.Errorf("webhook not found")
	}
	return wh, nil
}

// Delete removes a webhook by identifier.
func (s *WebhookStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("webhook store is not initialized")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("webhook id is required")
	}

	query := `DELETE FROM webhooks WHERE id = ?`
	args := []any{id}
	if isPostgreSQL(s.db) {
		query = `DELETE FROM webhooks WHERE id = $1`
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete webhook rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// ListActiveSubscribed returns active webhooks subscribed to action.
func (s *WebhookStore) ListActiveSubscribed(ctx context.Context, action string) ([]Webhook, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	action = strings.TrimSpace(action)
	if action == "" {
		return nil, nil
	}

	out := make([]Webhook, 0)
	for _, wh := range all {
		if !wh.Active {
			continue
		}
		if webhookSubscribed(wh.SubscribedEvents, action) {
			out = append(out, wh)
		}
	}
	return out, nil
}

func webhookSubscribed(events []string, action string) bool {
	for _, event := range events {
		if strings.EqualFold(strings.TrimSpace(event), action) {
			return true
		}
	}
	return false
}

type webhookScanner interface {
	Scan(dest ...any) error
}

func scanWebhook(row webhookScanner) (Webhook, error) {
	var (
		wh         Webhook
		secret     sql.NullString
		active     int
		eventsRaw  string
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(
		&wh.ID,
		&wh.URL,
		&wh.Name,
		&secret,
		&active,
		&eventsRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return Webhook{}, fmt.Errorf("scan webhook: %w", err)
	}

	wh.Active = active != 0
	if secret.Valid {
		wh.SecretToken = secret.String
	}

	if err := json.Unmarshal([]byte(eventsRaw), &wh.SubscribedEvents); err != nil {
		wh.SubscribedEvents = []string{}
	}
	if wh.SubscribedEvents == nil {
		wh.SubscribedEvents = []string{}
	}

	createdAt, err := parseTimestamp(createdRaw)
	if err != nil {
		return Webhook{}, err
	}
	updatedAt, err := parseTimestamp(updatedRaw)
	if err != nil {
		return Webhook{}, err
	}
	wh.CreatedAt = createdAt
	wh.UpdatedAt = updatedAt
	return wh, nil
}

func parseTimestamp(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse timestamp: %w", err)
		}
	}
	return parsed.UTC(), nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
