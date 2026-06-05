package config

import (
	"strings"

	kmsapi "go.step.sm/crypto/kms/apiv1"
)

// CABootstrapConfig holds subject and key parameters used when generating a new PKI tree.
type CABootstrapConfig struct {
	RootCN         string `mapstructure:"root_cn" yaml:"root_cn"`
	IntermediateCN string `mapstructure:"intermediate_cn" yaml:"intermediate_cn"`
	Organization   string `mapstructure:"organization" yaml:"organization"`
	Country        string `mapstructure:"country" yaml:"country"`
	KeySize        int    `mapstructure:"key_size" yaml:"key_size"`
}

// CABootstrapFromMap parses a CABootstrap block from server.yaml keys (snake_case or PascalCase).
func CABootstrapFromMap(raw map[string]any) CABootstrapConfig {
	if len(raw) == 0 {
		return CABootstrapConfig{}
	}
	return CABootstrapConfig{
		RootCN:         bootstrapString(raw, "root_cn", "RootCN"),
		IntermediateCN: bootstrapString(raw, "intermediate_cn", "IntermediateCN"),
		Organization:   bootstrapString(raw, "organization", "Organization"),
		Country:        bootstrapString(raw, "country", "Country"),
		KeySize:        bootstrapInt(raw, "key_size", "KeySize"),
	}
}

func bootstrapString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func bootstrapInt(raw map[string]any, keys ...string) int {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}
	return 0
}

// DefaultCABootstrapConfig returns safe defaults when CABootstrap is omitted from server.yaml.
func DefaultCABootstrapConfig() CABootstrapConfig {
	return CABootstrapConfig{
		RootCN:         "Arx CA Root CA",
		IntermediateCN: "Arx CA Intermediate CA",
		Organization:   "Arx CA",
		Country:        "",
		KeySize:        4096,
	}
}

// WithDefaults merges configured values with defaults for missing fields.
func (b CABootstrapConfig) WithDefaults() CABootstrapConfig {
	def := DefaultCABootstrapConfig()
	if strings.TrimSpace(b.RootCN) == "" {
		b.RootCN = def.RootCN
	}
	if strings.TrimSpace(b.IntermediateCN) == "" {
		b.IntermediateCN = def.IntermediateCN
	}
	if strings.TrimSpace(b.Organization) == "" {
		b.Organization = def.Organization
	}
	if b.KeySize <= 0 {
		b.KeySize = def.KeySize
	}
	return b
}

// EffectiveCABootstrap merges configured values with defaults for missing fields.
func (c ServerConfig) EffectiveCABootstrap() CABootstrapConfig {
	return c.CABootstrap.WithDefaults()
}

// KeyCreateRequest maps KeySize to a KMS create-key request for PKI bootstrap.
func (b CABootstrapConfig) KeyCreateRequest() *kmsapi.CreateKeyRequest {
	switch {
	case b.KeySize >= 4096:
		return &kmsapi.CreateKeyRequest{
			SignatureAlgorithm: kmsapi.SHA256WithRSA,
			Bits:               4096,
		}
	case b.KeySize >= 2048:
		return &kmsapi.CreateKeyRequest{
			SignatureAlgorithm: kmsapi.SHA256WithRSA,
			Bits:               b.KeySize,
		}
	default:
		return &kmsapi.CreateKeyRequest{
			SignatureAlgorithm: kmsapi.ECDSAWithSHA256,
		}
	}
}
