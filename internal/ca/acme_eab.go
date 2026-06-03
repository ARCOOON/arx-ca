package ca

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/ARCOOON/arx-ca/internal/models"
)

// CreateACMEEABKey mints an ACME External Account Binding MAC key for client registration.
func (e *PKIEngine) CreateACMEEABKey(ctx context.Context, req models.CreateACMEEABKeyRequest) (*models.ACMEEABKeyResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if e.acmeDB == nil {
		return nil, errors.New("ACME database is not configured")
	}

	provisionerName := strings.TrimSpace(req.Provisioner)
	if provisionerName == "" {
		provisionerName = acmeProvisionerName
	}

	reference := strings.TrimSpace(req.Reference)
	if len(reference) > 256 {
		return nil, errors.New("reference must not exceed 256 characters")
	}

	prov, err := e.auth.LoadProvisionerByName(provisionerName)
	if err != nil {
		return nil, fmt.Errorf("provisioner %q not found", provisionerName)
	}

	acmeProv, ok := prov.(*provisioner.ACME)
	if !ok {
		return nil, fmt.Errorf("provisioner %q is not an ACME provisioner", provisionerName)
	}

	eak, err := e.acmeDB.CreateExternalAccountKey(ctx, acmeProv.GetID(), reference)
	if err != nil {
		return nil, fmt.Errorf("create external account key: %w", err)
	}
	if eak == nil || len(eak.HmacKey) == 0 {
		return nil, errors.New("external account key was not created")
	}

	return &models.ACMEEABKeyResponse{
		KeyID:       eak.ID,
		Provisioner: provisionerName,
		HmacKey:     base64.RawURLEncoding.EncodeToString(eak.HmacKey),
		Reference:   eak.Reference,
		CreatedAt:   eak.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ACMEEABRequired reports whether the default ACME provisioner requires EAB.
func (e *PKIEngine) ACMEEABRequired() bool {
	if e == nil || e.auth == nil {
		return false
	}
	prov, err := e.auth.LoadProvisionerByName(acmeProvisionerName)
	if err != nil {
		return false
	}
	acmeProv, ok := prov.(*provisioner.ACME)
	if !ok {
		return false
	}
	return acmeProv.RequireEAB
}

// ACMEConfiguredChallenges returns enabled ACME challenge types for the default provisioner.
func (e *PKIEngine) ACMEConfiguredChallenges() []string {
	if e == nil || e.auth == nil {
		return nil
	}
	prov, err := e.auth.LoadProvisionerByName(acmeProvisionerName)
	if err != nil {
		return nil
	}
	acmeProv, ok := prov.(*provisioner.ACME)
	if !ok {
		return nil
	}
	if len(acmeProv.Challenges) == 0 {
		return []string{
			string(provisioner.HTTP_01),
			string(provisioner.DNS_01),
			string(provisioner.TLS_ALPN_01),
		}
	}
	out := make([]string, 0, len(acmeProv.Challenges))
	for _, ch := range acmeProv.Challenges {
		out = append(out, string(ch))
	}
	return out
}

// ACMEDeviceAttestationEnabled reports whether device-attest-01 is enabled.
func (e *PKIEngine) ACMEDeviceAttestationEnabled() bool {
	challenges := e.ACMEConfiguredChallenges()
	for _, ch := range challenges {
		if ch == string(provisioner.DEVICE_ATTEST_01) {
			return true
		}
	}
	return false
}

// ACMEAttestationFormats returns configured device attestation formats.
func (e *PKIEngine) ACMEAttestationFormats() []string {
	if e == nil || e.auth == nil {
		return nil
	}
	prov, err := e.auth.LoadProvisionerByName(acmeProvisionerName)
	if err != nil {
		return nil
	}
	acmeProv, ok := prov.(*provisioner.ACME)
	if !ok {
		return nil
	}
	if len(acmeProv.AttestationFormats) == 0 {
		return []string{
			string(provisioner.APPLE),
			string(provisioner.STEP),
			string(provisioner.TPM),
		}
	}
	out := make([]string, 0, len(acmeProv.AttestationFormats))
	for _, f := range acmeProv.AttestationFormats {
		out = append(out, string(f))
	}
	return out
}

// ACMEProvisioner returns the loaded ACME provisioner for advanced enrollment hooks.
func (e *PKIEngine) ACMEProvisioner(name string) (*provisioner.ACME, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if strings.TrimSpace(name) == "" {
		name = acmeProvisionerName
	}
	prov, err := e.auth.LoadProvisionerByName(name)
	if err != nil {
		return nil, fmt.Errorf("provisioner %q not found", name)
	}
	acmeProv, ok := prov.(*provisioner.ACME)
	if !ok {
		return nil, fmt.Errorf("provisioner %q is not an ACME provisioner", name)
	}
	return acmeProv, nil
}

// ACMEExternalAccountKeyCount is a convenience wrapper for operational checks.
func (e *PKIEngine) ACMEExternalAccountKeyCount(ctx context.Context, provisionerID string) (int, error) {
	if e.acmeDB == nil {
		return 0, errors.New("ACME database is not configured")
	}
	keys, _, err := e.acmeDB.GetExternalAccountKeys(ctx, provisionerID, "", 100)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}
