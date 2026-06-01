package models

import "time"

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned after a successful admin login.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	TokenType string    `json:"token_type"`
	Roles     []string  `json:"roles,omitempty"`
}

// CreateServiceAccountRequest is the JSON body for POST /api/v1/auth/service-accounts.
type CreateServiceAccountRequest struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles,omitempty"`
}

// ServiceAccountResponse is returned when a service account is created.
type ServiceAccountResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Roles     []string  `json:"roles"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}
