package acmeprotocol

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// ChallengeClient implements stepacme.Client for ACME challenge validation with bounded timeouts.
type ChallengeClient struct {
	http   *http.Client
	dialer *net.Dialer
}

// NewChallengeClient returns an ACME challenge validation client.
func NewChallengeClient() *ChallengeClient {
	return &ChallengeClient{
		http: &http.Client{
			Timeout: defaultHTTP01Timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					//nolint:gosec // tls-alpn-01 uses self-signed challenge certificates
					InsecureSkipVerify: true,
				},
			},
		},
		dialer: &net.Dialer{Timeout: defaultHTTP01Timeout},
	}
}

// Get issues an HTTP GET for http-01 validation.
func (c *ChallengeClient) Get(url string) (*http.Response, error) {
	return c.http.Get(url)
}

// LookupTxt returns DNS TXT records for dns-01 validation.
func (c *ChallengeClient) LookupTxt(name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTP01Timeout)
	defer cancel()
	var resolver net.Resolver
	return resolver.LookupTXT(ctx, name)
}

// TLSDial connects with TLS for tls-alpn-01 validation.
func (c *ChallengeClient) TLSDial(network, addr string, config *tls.Config) (*tls.Conn, error) {
	return tls.DialWithDialer(c.dialer, network, addr, config)
}
