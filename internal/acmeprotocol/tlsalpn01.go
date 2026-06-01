package acmeprotocol

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	stepacme "github.com/smallstep/certificates/acme"
)

var (
	idPeAcmeIdentifier           = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 31}
	idPeAcmeIdentifierV1Obsolete = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 30, 1}
)

// VerifyTLSALPN01 connects to the identifier on port 443 (or CA_API_ACME_TLS_PORT), negotiates
// ALPN acme-tls/1, and verifies the self-signed certificate's id-pe-acmeIdentifier extension
// (RFC 8737).
func VerifyTLSALPN01(ctx context.Context, dialer TLSDialer, identifier string, keyAuth string) error {
	if dialer == nil {
		dialer = defaultTLSDialer{}
	}

	config := &tls.Config{
		NextProtos:         []string{"acme-tls/1"},
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverNameForTLS(identifier),
		InsecureSkipVerify: true, //nolint:gosec // challenge certificates are always self-signed
	}

	hostPort := tlsAlpn01HostPort(identifier)
	conn, err := dialer.TLSDialContext(ctx, "tcp", hostPort, config)
	if err != nil {
		return fmt.Errorf("tls-alpn-01: dial %s: %w", hostPort, err)
	}
	defer conn.Close()

	cs := conn.ConnectionState()
	if cs.NegotiatedProtocol != "acme-tls/1" {
		return fmt.Errorf("tls-alpn-01: ALPN acme-tls/1 not negotiated with %s", hostPort)
	}

	certs := cs.PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("tls-alpn-01: no peer certificate from %s", hostPort)
	}

	leaf := certs[0]
	if err := verifyTLSALPN01Leaf(leaf, identifier); err != nil {
		return err
	}

	hashedKeyAuth := sha256.Sum256([]byte(keyAuth))
	foundObsolete := false

	for _, ext := range leaf.Extensions {
		if idPeAcmeIdentifier.Equal(ext.Id) {
			if !ext.Critical {
				return fmt.Errorf("tls-alpn-01: acmeIdentifier extension must be critical")
			}
			var extValue []byte
			rest, err := asn1.Unmarshal(ext.Value, &extValue)
			if err != nil || len(rest) > 0 || len(hashedKeyAuth) != len(extValue) {
				return fmt.Errorf("tls-alpn-01: malformed acmeIdentifier extension value")
			}
			if subtle.ConstantTimeCompare(hashedKeyAuth[:], extValue) != 1 {
				return fmt.Errorf("tls-alpn-01: acmeIdentifier hash mismatch (expected %s)",
					hex.EncodeToString(hashedKeyAuth[:]))
			}
			return nil
		}
		if idPeAcmeIdentifierV1Obsolete.Equal(ext.Id) {
			foundObsolete = true
		}
	}

	if foundObsolete {
		return fmt.Errorf("tls-alpn-01: obsolete id-pe-acmeIdentifier extension present")
	}
	return fmt.Errorf("tls-alpn-01: missing id-pe-acmeIdentifier extension")
}

func verifyTLSALPN01Leaf(leaf *x509.Certificate, identifier string) error {
	if len(leaf.DNSNames) == 0 {
		if len(leaf.IPAddresses) != 1 || !leaf.IPAddresses[0].Equal(net.ParseIP(identifier)) {
			return fmt.Errorf("tls-alpn-01: leaf certificate must contain a single matching IP address")
		}
		return nil
	}
	if len(leaf.DNSNames) != 1 || !strings.EqualFold(leaf.DNSNames[0], identifier) {
		return fmt.Errorf("tls-alpn-01: leaf certificate must contain a single matching DNS name")
	}
	return nil
}

func tlsAlpn01HostPort(identifier string) string {
	host := tlsAlpn01ChallengeHost(identifier)
	if stepacme.InsecurePortTLSALPN01 == 0 {
		return net.JoinHostPort(host, "443")
	}
	return net.JoinHostPort(host, strconv.Itoa(stepacme.InsecurePortTLSALPN01))
}

func tlsAlpn01ChallengeHost(name string) string {
	if ip := net.ParseIP(name); ip != nil {
		return name
	}
	return rootedName(name)
}

func serverNameForTLS(identifier string) string {
	if ip := net.ParseIP(identifier); ip != nil {
		return ""
	}
	return strings.TrimSuffix(identifier, ".")
}

// TLSDialer dials a TLS endpoint for tls-alpn-01 validation.
type TLSDialer interface {
	TLSDialContext(ctx context.Context, network, addr string, config *tls.Config) (*tls.Conn, error)
}

type defaultTLSDialer struct{}

func (defaultTLSDialer) TLSDialContext(ctx context.Context, network, addr string, config *tls.Config) (*tls.Conn, error) {
	d := &net.Dialer{}
	return tls.DialWithDialer(d, network, addr, config)
}
