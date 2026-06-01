package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/your-org/arx-ca/internal/models"
)

const (
	maxBodyBytes = 2 << 20
	httpTimeout  = 30 * time.Second
)

// Client talks to read-only public endpoints on arx-ca-server.
// It never sends credentials and never requests private keys or signing operations.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient builds an unauthenticated API client with a normalized base URL.
func NewClient(baseURL string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("api url is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: httpTimeout,
		},
	}, nil
}

// FetchRootPEM returns the Root CA certificate PEM from GET /api/v1/ca/root.
func (c *Client) FetchRootPEM(ctx context.Context) (string, error) {
	var payload models.RootCertResponse
	if err := c.getJSON(ctx, "/api/v1/ca/root", &payload); err != nil {
		return "", err
	}
	pem := strings.TrimSpace(payload.PEM)
	if pem == "" {
		return "", fmt.Errorf("root certificate PEM is empty")
	}
	return pem, nil
}

// FetchIntermediatePEM returns the Intermediate CA certificate PEM from GET /api/v1/public/ca/intermediate.
func (c *Client) FetchIntermediatePEM(ctx context.Context) (string, error) {
	var payload models.IntermediateCertResponse
	if err := c.getJSON(ctx, "/api/v1/public/ca/intermediate", &payload); err != nil {
		return "", err
	}
	pem := strings.TrimSpace(payload.PEM)
	if pem == "" {
		return "", fmt.Errorf("intermediate certificate PEM is empty")
	}
	return pem, nil
}

// ListPublicCertificates returns read-only certificate metadata from GET /api/v1/public/certificates.
func (c *Client) ListPublicCertificates(ctx context.Context) (*models.PublicListCertificatesResponse, error) {
	var payload models.PublicListCertificatesResponse
	if err := c.getJSON(ctx, "/api/v1/public/certificates", &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DownloadCertificatePEM returns the public certificate PEM for a serial from GET /api/v1/public/certificates/{serial}.
func (c *Client) DownloadCertificatePEM(ctx context.Context, serial string) (string, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return "", fmt.Errorf("serial is required")
	}
	path := "/api/v1/public/certificates/" + serial
	var payload models.CertificatePEMResponse
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return "", err
	}
	pem := strings.TrimSpace(payload.CertificatePEM)
	if pem == "" {
		return "", fmt.Errorf("certificate PEM is empty")
	}
	return pem, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, out)
}

func (c *Client) doJSON(req *http.Request, out any) error {
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
