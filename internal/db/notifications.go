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

const (
	NotificationLevelInfo     = "info"
	NotificationLevelCritical = "critical"
)

// Notification is a stateful operator notification persisted separately from immutable audit logs.
type Notification struct {
	ID        string
	Timestamp time.Time
	Action    string
	Level     string
	Message   string
	IsRead    bool
	Metadata  map[string]any
}

// NotificationListOptions controls pagination and unread filtering for List.
type NotificationListOptions struct {
	Limit        int
	Offset       int
	UnreadOnly   bool
	IncludeTotal bool
}

// NotificationListResult contains a page of notifications and aggregate counts.
type NotificationListResult struct {
	Notifications []Notification
	Total         int
	UnreadCount   int
}

// NotificationStore provides CRUD access to persistent operator notifications.
type NotificationStore struct {
	db *sql.DB
}

// NewNotificationStore constructs a NotificationStore backed by db.
func NewNotificationStore(db *sql.DB) *NotificationStore {
	return &NotificationStore{db: db}
}

// Insert persists a new notification row and returns the stored record.
func (s *NotificationStore) Insert(ctx context.Context, entry Notification) (Notification, error) {
	if s == nil || s.db == nil {
		return Notification{}, fmt.Errorf("notification store is not initialized")
	}

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = uuid.NewString()
	}

	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	level := strings.TrimSpace(entry.Level)
	if level == "" {
		level = NotificationLevelInfo
	}
	if level != NotificationLevelInfo && level != NotificationLevelCritical {
		return Notification{}, fmt.Errorf("invalid notification level %q", level)
	}

	message := strings.TrimSpace(entry.Message)
	if message == "" {
		return Notification{}, fmt.Errorf("notification message is required")
	}

	action := strings.TrimSpace(entry.Action)
	if action == "" {
		return Notification{}, fmt.Errorf("notification action is required")
	}

	metadata := entry.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return Notification{}, fmt.Errorf("marshal metadata: %w", err)
	}

	isRead := 0
	if entry.IsRead {
		isRead = 1
	}

	query := `
INSERT INTO notifications (id, timestamp, action, level, message, is_read, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		id,
		ts.UTC().Format(time.RFC3339Nano),
		action,
		level,
		message,
		isRead,
		string(metadataJSON),
	}
	if isPostgreSQL(s.db) {
		query = `
INSERT INTO notifications (id, timestamp, action, level, message, is_read, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return Notification{}, fmt.Errorf("insert notification: %w", err)
	}

	entry.ID = id
	entry.Timestamp = ts.UTC()
	entry.Action = action
	entry.Level = level
	entry.Message = message
	entry.Metadata = metadata
	return entry, nil
}

// List returns notifications ordered by timestamp descending with optional unread filtering.
func (s *NotificationStore) List(ctx context.Context, opts NotificationListOptions) (NotificationListResult, error) {
	if s == nil || s.db == nil {
		return NotificationListResult{}, fmt.Errorf("notification store is not initialized")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	whereClause := " WHERE is_archived = 0"
	countArgs := []any{}
	if opts.UnreadOnly {
		whereClause += " AND is_read = 0"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM notifications" + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return NotificationListResult{}, fmt.Errorf("count notifications: %w", err)
	}

	var unreadCount int
	unreadQuery := "SELECT COUNT(*) FROM notifications WHERE is_archived = 0 AND is_read = 0"
	if err := s.db.QueryRowContext(ctx, unreadQuery).Scan(&unreadCount); err != nil {
		return NotificationListResult{}, fmt.Errorf("count unread notifications: %w", err)
	}

	listQuery := `
SELECT id, timestamp, action, level, message, is_read, metadata
FROM notifications` + whereClause + `
ORDER BY timestamp DESC
LIMIT ? OFFSET ?`
	listArgs := []any{limit, offset}
	if isPostgreSQL(s.db) {
		listQuery = `
SELECT id, timestamp, action, level, message, is_read, metadata
FROM notifications` + whereClause + `
ORDER BY timestamp DESC
LIMIT $1 OFFSET $2`
		listArgs = []any{limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return NotificationListResult{}, fmt.Errorf("list notifications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	notifications := make([]Notification, 0, limit)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return NotificationListResult{}, err
		}
		notifications = append(notifications, item)
	}
	if err := rows.Err(); err != nil {
		return NotificationListResult{}, fmt.Errorf("iterate notifications: %w", err)
	}

	return NotificationListResult{
		Notifications: notifications,
		Total:         total,
		UnreadCount:   unreadCount,
	}, nil
}

// MarkAsRead sets is_read to true for a single notification.
func (s *NotificationStore) MarkAsRead(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("notification store is not initialized")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("notification id is required")
	}

	query := `UPDATE notifications SET is_read = 1 WHERE id = ?`
	args := []any{id}
	if isPostgreSQL(s.db) {
		query = `UPDATE notifications SET is_read = 1 WHERE id = $1`
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark notification read rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

// MarkAllAsRead sets is_read to true for all unread notifications.
func (s *NotificationStore) MarkAllAsRead(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("notification store is not initialized")
	}

	query := `UPDATE notifications SET is_read = 1 WHERE is_archived = 0 AND is_read = 0`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read rows affected: %w", err)
	}
	return rows, nil
}

// ArchiveAll soft-deletes every visible notification by setting is_archived to true.
func (s *NotificationStore) ArchiveAll(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("notification store is not initialized")
	}

	result, err := s.db.ExecContext(ctx, `UPDATE notifications SET is_archived = 1 WHERE is_archived = 0`)
	if err != nil {
		return 0, fmt.Errorf("archive all notifications: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("archive all notifications rows affected: %w", err)
	}
	return rows, nil
}

// Delete removes a notification by identifier.
func (s *NotificationStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("notification store is not initialized")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("notification id is required")
	}

	query := `DELETE FROM notifications WHERE id = ?`
	args := []any{id}
	if isPostgreSQL(s.db) {
		query = `DELETE FROM notifications WHERE id = $1`
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete notification rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row notificationScanner) (Notification, error) {
	var (
		item        Notification
		tsRaw       string
		isRead      int
		metadataRaw string
	)
	if err := row.Scan(
		&item.ID,
		&tsRaw,
		&item.Action,
		&item.Level,
		&item.Message,
		&isRead,
		&metadataRaw,
	); err != nil {
		return Notification{}, fmt.Errorf("scan notification: %w", err)
	}

	parsedTS, err := parseTimestamp(tsRaw)
	if err != nil {
		return Notification{}, err
	}
	item.Timestamp = parsedTS
	item.IsRead = isRead != 0

	if err := json.Unmarshal([]byte(metadataRaw), &item.Metadata); err != nil {
		item.Metadata = map[string]any{}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}

	return item, nil
}
