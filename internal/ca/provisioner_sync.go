package ca

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/ARCOOON/arx-ca/internal/config"
)

// syncCAProvisioners updates ca.json provisioners to match server.yaml enrollment settings.
func syncCAProvisioners(configPath, basePath string, password []byte, prov config.CAProvisionersConfig) error {
	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for provisioner sync: %w", err)
	}
	if cfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	original := cfg.AuthorityConfig.Provisioners
	updated := filterProvisionersByName(original, acmeProvisionerName, scepProvisionerName)

	var changed bool

	if prov.ACMEEnabled() {
		acmeProv, err := buildACMEProvisioner(prov.ACME)
		if err != nil {
			return err
		}
		var acmeChanged bool
		updated, acmeChanged = upsertProvisioner(updated, acmeProv, acmeProvisionerName, provisioner.TypeACME)
		changed = changed || acmeChanged
	} else if provisionerPresent(original, acmeProvisionerName, provisioner.TypeACME) {
		changed = true
	}

	if prov.SCEPEnabled() {
		scepProv, err := buildSCEPProvisioner(basePath, password, prov.SCEP)
		if err != nil {
			return err
		}
		var scepChanged bool
		updated, scepChanged = upsertProvisioner(updated, scepProv, scepProvisionerName, provisioner.TypeSCEP)
		changed = changed || scepChanged
	} else if provisionerPresent(original, scepProvisionerName, provisioner.TypeSCEP) {
		changed = true
	}

	if !changed {
		return nil
	}

	cfg.AuthorityConfig.Provisioners = updated
	return writeCAConfig(configPath, cfg)
}

func filterProvisionersByName(items []provisioner.Interface, names ...string) []provisioner.Interface {
	skip := make(map[string]struct{}, len(names))
	for _, name := range names {
		skip[name] = struct{}{}
	}
	out := make([]provisioner.Interface, 0, len(items))
	for _, p := range items {
		if _, remove := skip[p.GetName()]; remove {
			continue
		}
		out = append(out, p)
	}
	return out
}

func provisionerPresent(items []provisioner.Interface, name string, typ provisioner.Type) bool {
	for _, p := range items {
		if p.GetName() == name && p.GetType() == typ {
			return true
		}
	}
	return false
}

func upsertProvisioner(items []provisioner.Interface, next provisioner.Interface, name string, typ provisioner.Type) ([]provisioner.Interface, bool) {
	for i, p := range items {
		if p.GetName() == name && p.GetType() == typ {
			if provisionerJSONEqual(p, next) {
				return items, false
			}
			items[i] = next
			return items, true
		}
	}
	return append(items, next), true
}

func provisionerJSONEqual(a, b provisioner.Interface) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func buildACMEProvisioner(acmeCfg config.ACMEProvisionerConfig) (*provisioner.ACME, error) {
	deviceAttest := acmeCfg.DeviceAttestation || deviceAttestEnabled()
	challenges, err := parseACMEChallenges(acmeCfg.Challenges, deviceAttest)
	if err != nil {
		return nil, err
	}

	requireEAB := acmeCfg.RequireEAB || strings.EqualFold(os.Getenv("CA_API_ACME_REQUIRE_EAB"), "true")

	acmeProv := &provisioner.ACME{
		Type:       "ACME",
		Name:       acmeProvisionerName,
		Challenges: challenges,
		RequireEAB: requireEAB,
	}

	if deviceAttest {
		acmeProv.AttestationFormats = defaultACMEAttestationFormats()
		if roots, err := loadACMEAttestationRoots(); err != nil {
			return nil, fmt.Errorf("load ACME attestation roots: %w", err)
		} else if len(roots) > 0 {
			acmeProv.AttestationRoots = roots
		}
	}

	return acmeProv, nil
}

func parseACMEChallenges(names []string, deviceAttest bool) ([]provisioner.ACMEChallenge, error) {
	aliases := map[string]provisioner.ACMEChallenge{
		"http-01":          provisioner.HTTP_01,
		"dns-01":           provisioner.DNS_01,
		"tls-alpn-01":      provisioner.TLS_ALPN_01,
		"device-attest-01": provisioner.DEVICE_ATTEST_01,
	}

	if len(names) == 0 {
		names = []string{"http-01", "dns-01", "tls-alpn-01"}
	}

	seen := make(map[provisioner.ACMEChallenge]struct{})
	out := make([]provisioner.ACMEChallenge, 0, len(names)+1)
	for _, raw := range names {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		challenge, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unsupported ACME challenge %q", raw)
		}
		if _, exists := seen[challenge]; exists {
			continue
		}
		seen[challenge] = struct{}{}
		out = append(out, challenge)
	}

	if deviceAttest {
		if _, exists := seen[provisioner.DEVICE_ATTEST_01]; !exists {
			out = append(out, provisioner.DEVICE_ATTEST_01)
		}
	}

	if len(out) == 0 {
		return nil, errors.New("at least one ACME challenge must be enabled")
	}
	return out, nil
}

func buildSCEPProvisioner(basePath string, password []byte, scepCfg config.SCEPProvisionerConfig) (*provisioner.SCEP, error) {
	challenge := strings.TrimSpace(scepCfg.ChallengePassword)
	if challenge == "" {
		challenge = strings.TrimSpace(os.Getenv("CA_API_SCEP_CHALLENGE"))
	}
	if challenge == "" {
		pass, err := generateRandomPassword(24)
		if err != nil {
			return nil, fmt.Errorf("generate SCEP challenge password: %w", err)
		}
		challenge = string(pass)
		log.Printf("SCEP: generated challenge password; set ca.provisioners.scep.challenge_password in server.yaml to pin it across restarts")
	}

	decrypterCert, decrypterKey, err := loadOrCreateSCEPDecrypter(basePath, password)
	if err != nil {
		return nil, fmt.Errorf("configure SCEP decrypter: %w", err)
	}

	scepProv := &provisioner.SCEP{
		Type:                   "SCEP",
		Name:                   scepProvisionerName,
		ChallengePassword:      challenge,
		MinimumPublicKeyLength: 2048,
		DecrypterCertificate:   decrypterCert,
		DecrypterKeyPEM:        decrypterKey,
		DecrypterKeyPassword:   string(password),
		Capabilities: []string{
			"SCEPStandard",
			"POSTPKIOperation",
			"SHA-256",
			"AES",
			"DES3",
		},
	}
	if minLen := strings.TrimSpace(os.Getenv("CA_API_SCEP_MIN_KEY_LENGTH")); minLen != "" {
		var n int
		if _, err := fmt.Sscanf(minLen, "%d", &n); err == nil && n > 0 {
			scepProv.MinimumPublicKeyLength = n
		}
	}
	return scepProv, nil
}

func writeCAConfig(configPath string, cfg *authconfig.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated CA configuration: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("write updated CA configuration: %w", err)
	}
	return nil
}
