package models

import "time"

// CreateCertificateTemplateRequest defines a new issuance template.
type CreateCertificateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Body is a Go text/template that renders JSON describing SANs and extensions.
	Body string `json:"body"`
}

// CertificateTemplate is a stored issuance template.
type CertificateTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListCertificateTemplatesResponse returns all registered templates.
type ListCertificateTemplatesResponse struct {
	Templates []CertificateTemplate `json:"templates"`
	Total     int                   `json:"total"`
}
