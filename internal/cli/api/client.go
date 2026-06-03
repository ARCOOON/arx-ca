package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/models"
)

const (
	maxBodyBytes = 2 << 20
	httpTimeout  = 30 * time.Second
)

// Client talks to the arx-ca HTTP API using a stored admin JWT.
type Client struct {
	BaseURL    string
	BearerAuth string
	HTTPClient *http.Client
}

// NewClient builds an API client with a normalized base URL and Bearer token.
func NewClient(baseURL, bearerAuth string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("server url is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		BaseURL:    baseURL,
		BearerAuth: strings.TrimSpace(bearerAuth),
		HTTPClient: &http.Client{Timeout: httpTimeout},
	}, nil
}

// Login authenticates with admin credentials and returns the login payload.
func Login(ctx context.Context, baseURL string, req models.LoginRequest) (*models.LoginResponse, error) {
	client, err := NewClient(baseURL, "")
	if err != nil {
		return nil, err
	}

	var resp models.LoginResponse
	if err := client.postJSON(ctx, "/api/v1/auth/login", req, &resp, false); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Token) == "" {
		return nil, fmt.Errorf("login response did not include a token")
	}
	return &resp, nil
}

// Health fetches GET /api/v1/health.
func (c *Client) Health(ctx context.Context) (*models.HealthReport, error) {
	var report models.HealthReport
	if err := c.getJSON(ctx, "/api/v1/health", &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// ListCertificates fetches GET /api/v1/certificates.
func (c *Client) ListCertificates(ctx context.Context) (*models.ListCertificatesResponse, error) {
	var list models.ListCertificatesResponse
	if err := c.getJSON(ctx, "/api/v1/certificates", &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// AutoCertificate calls POST /api/v1/certificates/auto.
func (c *Client) AutoCertificate(ctx context.Context, req models.AutoCertificateRequest) (*models.AutoCertificateResponse, error) {
	var resp models.AutoCertificateResponse
	if err := c.postJSON(ctx, "/api/v1/certificates/auto", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeCertificate calls POST /api/v1/certificates/revoke.
func (c *Client) RevokeCertificate(ctx context.Context, serial, reason string) (*models.RevokeCertificateResponse, error) {
	req := models.RevokeCertificateRequest{Serial: serial, Reason: reason}
	var resp models.RevokeCertificateResponse
	if err := c.postJSON(ctx, "/api/v1/certificates/revoke", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ACMEStatus fetches GET /api/v1/acme/status.
func (c *Client) ACMEStatus(ctx context.Context) (*models.ACMEStatusResponse, error) {
	var resp models.ACMEStatusResponse
	if err := c.getJSON(ctx, "/api/v1/acme/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SCEPStatus fetches GET /api/v1/scep/status.
func (c *Client) SCEPStatus(ctx context.Context) (*models.SCEPStatusResponse, error) {
	var resp models.SCEPStatusResponse
	if err := c.getJSON(ctx, "/api/v1/scep/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// K8sStatus fetches GET /api/v1/k8s/status.
func (c *Client) K8sStatus(ctx context.Context) (*models.K8sProvisionerStatusResponse, error) {
	var resp models.K8sProvisionerStatusResponse
	if err := c.getJSON(ctx, "/api/v1/k8s/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NDESStatus fetches GET /api/v1/ndes/status.
func (c *Client) NDESStatus(ctx context.Context) (*models.NDESStatusResponse, error) {
	var resp models.NDESStatusResponse
	if err := c.getJSON(ctx, "/api/v1/ndes/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, out, true)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any, withAuth bool) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req, out, withAuth)
}

func (c *Client) doJSON(req *http.Request, out any, withAuth bool) error {
	if withAuth {
		if c.BearerAuth == "" {
			return fmt.Errorf("not logged in; run arx login first")
		}
		req.Header.Set("Authorization", c.BearerAuth)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxBodyBytes)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var envelope models.APIResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if envelope.Error != nil && *envelope.Error != "" {
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, *envelope.Error)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}

	data, err := json.Marshal(envelope.Data)
	if err != nil {
		return fmt.Errorf("marshal data field: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode data field: %w", err)
	}
	return nil
}
