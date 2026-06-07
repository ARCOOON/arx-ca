package models

// AuditLogEntry is a forensic audit record returned by the management API.
type AuditLogEntry struct {
	ID          string         `json:"id"`
	Timestamp   string         `json:"timestamp"`
	RequestID   string         `json:"request_id"`
	IPAddress   string         `json:"ip_address"`
	HTTPMethod  string         `json:"http_method"`
	Endpoint    string         `json:"endpoint"`
	StatusCode  int            `json:"status_code"`
	ActorType   string         `json:"actor_type"`
	ActorID     string         `json:"actor_id"`
	ActorRoles  []string       `json:"actor_roles,omitempty"`
	Action      string         `json:"action"`
	Provisioner string         `json:"provisioner,omitempty"`
	Fingerprint string         `json:"fingerprint,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ListAuditLogsResponse paginates immutable audit log entries.
type ListAuditLogsResponse struct {
	Logs   []AuditLogEntry `json:"logs"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}
