package ca

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	// HeaderForwardedClientCert is the standard proxy header carrying URL-encoded client certificate PEM.
	HeaderForwardedClientCert = "X-Forwarded-Client-Cert"
)

// FormatForwardedClientCert builds an XFCC element with Cert=<url-encoded PEM> per Envoy conventions.
func FormatForwardedClientCert(cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", errors.New("certificate is required")
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		return "", errors.New("encode certificate PEM")
	}
	return "Cert=" + url.QueryEscape(string(pemBytes)), nil
}

// ParseForwardedClientCert extracts the leaf certificate from an X-Forwarded-Client-Cert header value.
func ParseForwardedClientCert(header string) (*x509.Certificate, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil, errors.New("empty forwarded client cert header")
	}

	elements := splitXFCC(header)
	for i := len(elements) - 1; i >= 0; i-- {
		pemStr, err := xfccCertValue(elements[i])
		if err != nil {
			continue
		}
		block, _ := pem.Decode([]byte(pemStr))
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, errors.New("invalid forwarded certificate PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse forwarded certificate: %w", err)
		}
		return cert, nil
	}
	return nil, errors.New("no Cert= element in forwarded client cert header")
}

func splitXFCC(header string) []string {
	var elements []string
	var current strings.Builder
	inQuotes := false
	for _, r := range header {
		switch r {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case ',':
			if inQuotes {
				current.WriteRune(r)
				continue
			}
			if s := strings.TrimSpace(current.String()); s != "" {
				elements = append(elements, s)
			}
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if s := strings.TrimSpace(current.String()); s != "" {
		elements = append(elements, s)
	}
	return elements
}

func xfccCertValue(element string) (string, error) {
	element = strings.TrimSpace(element)
	for _, part := range strings.Split(element, ";") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		if !strings.EqualFold(key, "Cert") {
			continue
		}
		value := strings.TrimSpace(part[idx+1:])
		value = strings.Trim(value, `"`)
		decoded, err := url.QueryUnescape(value)
		if err != nil {
			return "", fmt.Errorf("decode Cert value: %w", err)
		}
		return decoded, nil
	}
	return "", errors.New("Cert key not found in XFCC element")
}

// forwardedClientCertFromRequest returns a client certificate from XFCC when the request
// originates from the trusted loopback WebUI reverse proxy.
func forwardedClientCertFromRequest(r *http.Request) (*x509.Certificate, error) {
	if r == nil || !isLoopbackProxyRequest(r) {
		return nil, ErrNoClientCertificate
	}
	header := strings.TrimSpace(r.Header.Get(HeaderForwardedClientCert))
	if header == "" {
		return nil, ErrNoClientCertificate
	}
	return ParseForwardedClientCert(header)
}

func isLoopbackProxyRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
