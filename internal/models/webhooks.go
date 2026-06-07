package models

// WebhookResponse is the API representation of a configured webhook.
type WebhookResponse struct {
	ID               string   `json:"id"`
	URL              string   `json:"url"`
	Name             string   `json:"name"`
	Active           bool     `json:"active"`
	SubscribedEvents []string `json:"subscribed_events"`
	HasSecretToken   bool     `json:"has_secret_token"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// ListWebhooksResponse wraps webhook list payloads.
type ListWebhooksResponse struct {
	Webhooks []WebhookResponse `json:"webhooks"`
}

// WebhookEventsResponse lists actions available for subscription.
type WebhookEventsResponse struct {
	Events []WebhookEventOption `json:"events"`
}

// WebhookEventOption describes a subscribable audit action for the WebUI.
type WebhookEventOption struct {
	Action      string `json:"action"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// CreateWebhookRequest creates a webhook endpoint.
type CreateWebhookRequest struct {
	URL              string   `json:"url"`
	Name             string   `json:"name"`
	SecretToken      string   `json:"secret_token,omitempty"`
	Active           *bool    `json:"active,omitempty"`
	SubscribedEvents []string `json:"subscribed_events"`
}

// UpdateWebhookRequest replaces webhook configuration.
type UpdateWebhookRequest struct {
	URL              string   `json:"url"`
	Name             string   `json:"name"`
	SecretToken      string   `json:"secret_token,omitempty"`
	Active           bool     `json:"active"`
	SubscribedEvents []string `json:"subscribed_events"`
}

// WebhookTestResponse summarizes a connectivity probe.
type WebhookTestResponse struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}
