package config

import (
	kmsapi "go.step.sm/crypto/kms/apiv1"
)

// CABootstrapConfig holds subject and key parameters used when generating a new PKI tree.
type CABootstrapConfig struct {
	RootCN         string `mapstructure:"root_cn" yaml:"RootCN"`
	IntermediateCN string `mapstructure:"intermediate_cn" yaml:"IntermediateCN"`
	Organization   string `mapstructure:"organization" yaml:"Organization"`
	Country        string `mapstructure:"country" yaml:"Country"`
	KeySize        int    `mapstructure:"key_size" yaml:"KeySize"`
}

// DefaultCABootstrapConfig returns safe defaults when CABootstrap is omitted from server.yaml.
func DefaultCABootstrapConfig() CABootstrapConfig {
	return CABootstrapConfig{
		RootCN:         "Arx CA Root CA",
		IntermediateCN: "Arx CA Intermediate CA",
		Organization:   "Arx CA",
		Country:        "",
		KeySize:        256,
	}
}

// EffectiveCABootstrap merges configured values with defaults for missing fields.
func (c ServerConfig) EffectiveCABootstrap() CABootstrapConfig {
	def := DefaultCABootstrapConfig()
	b := c.CABootstrap

	if b.RootCN == "" {
		b.RootCN = def.RootCN
	}
	if b.IntermediateCN == "" {
		b.IntermediateCN = def.IntermediateCN
	}
	if b.Organization == "" {
		b.Organization = def.Organization
	}
	if b.KeySize <= 0 {
		b.KeySize = def.KeySize
	}
	return b
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
