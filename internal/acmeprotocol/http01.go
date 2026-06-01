package acmeprotocol

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	stepacme "github.com/smallstep/certificates/acme"
)

const defaultHTTP01Timeout = 30 * time.Second

// VerifyHTTP01 performs an outbound HTTP GET to the identifier's ACME challenge URL
// and checks that the response body matches the expected key authorization (RFC 8555 §8.3).
func VerifyHTTP01(ctx context.Context, client *http.Client, identifier, token, expectedKeyAuth string) error {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTP01Timeout}
	}

	challengeURL, displayURL := http01ChallengeURLs(identifier, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, challengeURL, nil)
	if err != nil {
		return fmt.Errorf("http-01: build request for %s: %w", displayURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http-01: GET %s: %w", displayURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http-01: GET %s returned status %d", displayURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("http-01: read response from %s: %w", displayURL, err)
	}

	got := strings.TrimSpace(string(body))
	if got != expectedKeyAuth {
		return fmt.Errorf("http-01: key authorization mismatch for %s", displayURL)
	}
	return nil
}

func http01ChallengeURLs(identifier, token string) (requestURL, displayURL string) {
	host := http01ChallengeHost(identifier)
	path := fmt.Sprintf("/.well-known/acme-challenge/%s", token)

	u := &url.URL{Scheme: "http", Host: host, Path: path}
	display := u.String()

	if port := stepacme.InsecurePortHTTP01; port != 0 {
		u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	}

	return u.String(), display
}

func http01ChallengeHost(value string) string {
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() == nil {
			value = "[" + value + "]"
		}
		return value
	}
	return rootedName(value)
}
