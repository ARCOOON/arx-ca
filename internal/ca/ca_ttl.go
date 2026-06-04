package ca

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultMaxCertTTL = 8760 * time.Hour
	maxTLSCertKey     = "maxTLSCertDuration"
	defaultTLSCertKey = "defaultTLSCertDuration"
)

// syncCAConfigMaxTTL updates step-ca authority and provisioner claims so signing honors maxTTL.
func syncCAConfigMaxTTL(configPath string, maxTTL time.Duration) error {
	if maxTTL <= 0 {
		return fmt.Errorf("max certificate TTL must be positive")
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read CA config for max TTL sync: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse CA config for max TTL sync: %w", err)
	}

	authorityDoc, ok := doc["authority"].(map[string]any)
	if !ok {
		authorityDoc = map[string]any{}
		doc["authority"] = authorityDoc
	}

	changed := false
	if updateClaimsMap(ensureClaimsMap(authorityDoc), maxTTL) {
		changed = true
	}

	if provisioners, ok := authorityDoc["provisioners"].([]any); ok {
		for _, entry := range provisioners {
			prov, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if updateClaimsMap(ensureClaimsMap(prov), maxTTL) {
				changed = true
			}
		}
	}

	if !changed {
		return nil
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal CA config after max TTL sync: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return fmt.Errorf("write CA config after max TTL sync: %w", err)
	}
	log.Printf("ca: synchronized maxTLSCertDuration to %s in %s", formatCertDuration(maxTTL), configPath)
	return nil
}

func ensureClaimsMap(parent map[string]any) map[string]any {
	claims, ok := parent["claims"].(map[string]any)
	if !ok || claims == nil {
		claims = map[string]any{}
		parent["claims"] = claims
	}
	return claims
}

func updateClaimsMap(claims map[string]any, maxTTL time.Duration) bool {
	changed := false
	maxValue := formatCertDuration(maxTTL)

	if current, ok := parseClaimDuration(claims[maxTLSCertKey]); !ok || current < maxTTL {
		if claims[maxTLSCertKey] != maxValue {
			claims[maxTLSCertKey] = maxValue
			changed = true
		}
	}

	if current, ok := parseClaimDuration(claims[defaultTLSCertKey]); ok && current > maxTTL {
		if claims[defaultTLSCertKey] != maxValue {
			claims[defaultTLSCertKey] = maxValue
			changed = true
		}
	}

	return changed
}

func parseClaimDuration(value any) (time.Duration, bool) {
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, false
		}
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, false
		}
		return d, true
	default:
		return 0, false
	}
}

func formatCertDuration(d time.Duration) string {
	d = d.Truncate(time.Minute)
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return d.String()
}
