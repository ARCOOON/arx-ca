package ca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/models"
)

// CAProvisioners reads and sanitizes provisioners from the active ca.json configuration.
func (e *PKIEngine) CAProvisioners() (*models.CAProvisionersResponse, error) {
	if e == nil {
		return nil, fmt.Errorf("CA engine is not initialized")
	}

	configPath := strings.TrimSpace(e.configPath)
	if configPath == "" {
		return nil, fmt.Errorf("CA configuration path is unavailable")
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read CA configuration: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse CA configuration: %w", err)
	}

	authorityDoc, ok := doc["authority"].(map[string]any)
	if !ok {
		return &models.CAProvisionersResponse{
			Provisioners: []models.CAProvisionerDetail{},
			Total:        0,
		}, nil
	}

	entries, ok := authorityDoc["provisioners"].([]any)
	if !ok || len(entries) == 0 {
		return &models.CAProvisionersResponse{
			Provisioners: []models.CAProvisionerDetail{},
			Total:        0,
		}, nil
	}

	provisioners := make([]models.CAProvisionerDetail, 0, len(entries))
	for _, entry := range entries {
		provDoc, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		detail, ok := sanitizeProvisionerDetail(provDoc)
		if !ok {
			continue
		}
		provisioners = append(provisioners, detail)
	}

	return &models.CAProvisionersResponse{
		Provisioners: provisioners,
		Total:        len(provisioners),
	}, nil
}

func sanitizeProvisionerDetail(raw map[string]any) (models.CAProvisionerDetail, bool) {
	name := strings.TrimSpace(stringFromAny(raw["name"]))
	typ := strings.ToUpper(strings.TrimSpace(stringFromAny(raw["type"])))
	if name == "" || typ == "" {
		return models.CAProvisionerDetail{}, false
	}

	detail := models.CAProvisionerDetail{
		Name: name,
		Type: typ,
	}

	switch typ {
	case "ACME":
		if requireEAB, ok := boolFromAny(raw["requireEAB"]); ok {
			detail.RequireEAB = &requireEAB
		}
		detail.Challenges = parseACMEChallengesFromConfig(raw["challenges"])
	case "SCEP":
		if scepChallengeConfigured(raw) {
			detail.Challenge = "configured"
		}
	}

	return detail, true
}

func parseACMEChallengesFromConfig(value any) []string {
	switch entries := value.(type) {
	case []any:
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			if challenge := acmeChallengeLabel(entry); challenge != "" {
				out = append(out, challenge)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			if challenge := acmeChallengeLabel(entry); challenge != "" {
				out = append(out, challenge)
			}
		}
		return out
	default:
		return nil
	}
}

func acmeChallengeLabel(value any) string {
	switch v := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(v))
	case float64:
		return acmeChallengeNameFromNumber(int(v))
	default:
		return ""
	}
}

func acmeChallengeNameFromNumber(value int) string {
	switch value {
	case 0:
		return "http-01"
	case 1:
		return "dns-01"
	case 2:
		return "tls-alpn-01"
	case 3:
		return "device-attest-01"
	default:
		return ""
	}
}

func scepChallengeConfigured(raw map[string]any) bool {
	for _, key := range []string{"challengePassword", "challenge", "ChallengePassword"} {
		if strings.TrimSpace(stringFromAny(raw[key])) != "" {
			return true
		}
	}
	return false
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func boolFromAny(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	default:
		return false, false
	}
}
