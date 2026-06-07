package config

import "strings"

// CAProvisionersConfig groups enrollment provisioners controlled from server.yaml.
type CAProvisionersConfig struct {
	ACME ACMEProvisionerConfig `json:"acme" mapstructure:"acme" yaml:"acme"`
	SCEP SCEPProvisionerConfig `json:"scep" mapstructure:"scep" yaml:"scep"`
}

// ACMEProvisionerConfig controls the step-ca ACME provisioner entry in ca.json.
type ACMEProvisionerConfig struct {
	Enabled           *bool    `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	RequireEAB        bool     `json:"require_eab" mapstructure:"require_eab" yaml:"require_eab"`
	Challenges        []string `json:"challenges" mapstructure:"challenges" yaml:"challenges"`
	DeviceAttestation bool     `json:"device_attestation" mapstructure:"device_attestation" yaml:"device_attestation"`
}

// SCEPProvisionerConfig controls the step-ca SCEP provisioner entry in ca.json.
type SCEPProvisionerConfig struct {
	Enabled           *bool  `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	DeviceAttestation bool   `json:"device_attestation" mapstructure:"device_attestation" yaml:"device_attestation"`
	ChallengePassword string `json:"challenge_password" mapstructure:"challenge_password" yaml:"challenge_password"`
}

// DefaultCAProvisionersConfig returns enrollment defaults when server.yaml omits the block.
func DefaultCAProvisionersConfig() CAProvisionersConfig {
	return CAProvisionersConfig{
		ACME: ACMEProvisionerConfig{
			Enabled:    boolPtr(true),
			RequireEAB: false,
			Challenges: []string{"http-01", "dns-01", "tls-alpn-01"},
		},
		SCEP: SCEPProvisionerConfig{
			Enabled: boolPtr(false),
		},
	}
}

// EffectiveProvisioners merges configured values with defaults for unset fields.
func (c CAConfig) EffectiveProvisioners() CAProvisionersConfig {
	def := DefaultCAProvisionersConfig()
	p := c.Provisioners

	p.ACME.Enabled = boolPtr(boolOrDefault(p.ACME.Enabled, def.ACME.Enabled))
	p.SCEP.Enabled = boolPtr(boolOrDefault(p.SCEP.Enabled, def.SCEP.Enabled))

	if len(p.ACME.Challenges) == 0 {
		p.ACME.Challenges = append([]string(nil), def.ACME.Challenges...)
	}
	return p
}

func boolPtr(v bool) *bool {
	return &v
}

func boolOrDefault(value *bool, fallback *bool) bool {
	if value != nil {
		return *value
	}
	if fallback != nil {
		return *fallback
	}
	return false
}

// ACMEEnabled reports whether the ACME provisioner should be active after normalization.
func (p CAProvisionersConfig) ACMEEnabled() bool {
	return boolOrDefault(p.ACME.Enabled, boolPtr(true))
}

// SCEPEnabled reports whether the SCEP provisioner should be active after normalization.
func (p CAProvisionersConfig) SCEPEnabled() bool {
	return boolOrDefault(p.SCEP.Enabled, boolPtr(false))
}

func applyProvisionerRuntimeEnv(prov CAProvisionersConfig) {
	if !prov.ACMEEnabled() {
		setEnvIfEmpty("CA_API_ACME_DISABLED", "true")
	}
	if prov.ACME.RequireEAB {
		setEnvIfEmpty("CA_API_ACME_REQUIRE_EAB", "true")
	}
	if prov.ACME.DeviceAttestation {
		setEnvIfEmpty("CA_API_ACME_DEVICE_ATTEST", "true")
	}
	if !prov.SCEPEnabled() {
		setEnvIfEmpty("CA_API_SCEP_DISABLED", "true")
	}
	if challenge := strings.TrimSpace(prov.SCEP.ChallengePassword); challenge != "" {
		setEnvIfEmpty("CA_API_SCEP_CHALLENGE", challenge)
	}
}
