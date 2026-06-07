package models

// NotificationEntry is a stateful operator notification returned by the management API.
type NotificationEntry struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Action    string         `json:"action"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	IsRead    bool           `json:"is_read"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ListNotificationsResponse paginates persistent notification history.
type ListNotificationsResponse struct {
	Notifications []NotificationEntry `json:"notifications"`
	Total         int                 `json:"total"`
	UnreadCount   int                 `json:"unread_count"`
	Limit         int                 `json:"limit"`
	Offset        int                 `json:"offset"`
}

// MarkAllNotificationsReadResponse summarizes a bulk read operation.
type MarkAllNotificationsReadResponse struct {
	Updated int64 `json:"updated"`
}

// ArchiveAllNotificationsResponse summarizes a bulk archive (soft-delete) operation.
type ArchiveAllNotificationsResponse struct {
	Archived int64 `json:"archived"`
}
