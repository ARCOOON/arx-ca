package acmeprotocol

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// VerifyDNS01 looks up TXT records at _acme-challenge.<domain> and verifies that at least one
// record equals the SHA-256 digest of the key authorization, base64url-encoded (RFC 8555 §8.4).
func VerifyDNS01(ctx context.Context, identifier, keyAuth string) error {
	zone := dns01ChallengeName(identifier)
	expected := DNS01Digest(keyAuth)

	resolver := &net.Resolver{}
	txtRecords, err := resolver.LookupTXT(ctx, zone)
	if err != nil {
		return fmt.Errorf("dns-01: lookup TXT for %s: %w", zone, err)
	}

	for _, record := range txtRecords {
		if strings.TrimSpace(record) == expected {
			return nil
		}
	}
	return fmt.Errorf("dns-01: expected TXT digest not found at %s", zone)
}

func dns01ChallengeName(identifier string) string {
	host := strings.TrimPrefix(identifier, "*.")
	return "_acme-challenge." + rootedName(host)
}
