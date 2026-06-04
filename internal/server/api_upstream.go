package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

// BuildAPIUpstreamURL returns the loopback URL used by the WebUI listener to proxy API traffic.
func BuildAPIUpstreamURL(cfg arxconfig.ServerConfig) (*url.URL, error) {
	scheme := "http"
	if cfg.Server.TLS.Enabled {
		scheme = "https"
	}

	hostPort := cfg.ListenAddress()
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, fmt.Errorf("parse API listen address %q: %w", hostPort, err)
	}
	host = strings.TrimSpace(host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}
	return u, nil
}

// APIProxyTransport returns an http.RoundTripper suitable for loopback API proxying.
func APIProxyTransport(upstream *url.URL, apiTLS bool) http.RoundTripper {
	base := http.DefaultTransport.(*http.Transport).Clone()
	if apiTLS && upstream != nil && strings.EqualFold(upstream.Scheme, "https") {
		base.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // loopback self-signed API TLS during local drop-in UI
			MinVersion:         tls.VersionTLS12,
		}
	}
	return base
}
