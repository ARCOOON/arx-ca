package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// KMSType identifies the key management integration.
type KMSType string

const (
	KMSTypeLocal   KMSType = "local"
	KMSTypePKCS11  KMSType = "pkcs11"
	KMSTypeAWSKMS  KMSType = "awskms"
	KMSTypeGCPKMS  KMSType = "cloudkms"
	KMSTypeVaultRA KMSType = "vault"
)

// KMSConfig holds PKCS #11 and future cloud KMS settings.
type KMSConfig struct {
	Type KMSType

	// PKCS #11
	ModulePath         string
	TokenLabel         string
	PIN                string
	PINSource          string
	RootKeyURI         string
	IntermediateKeyURI string
}

// LoadKMSFromEnv reads KMS-related environment variables.
func LoadKMSFromEnv() KMSConfig {
	typ := KMSType(strings.ToLower(strings.TrimSpace(os.Getenv(EnvKMSType))))
	if typ == "" {
		typ = KMSTypeLocal
	}

	return KMSConfig{
		Type:               typ,
		ModulePath:         strings.TrimSpace(os.Getenv(EnvPKCS11ModulePath)),
		TokenLabel:         strings.TrimSpace(os.Getenv("PKCS11_TOKEN")),
		PIN:                strings.TrimSpace(os.Getenv("PKCS11_PIN")),
		PINSource:          strings.TrimSpace(os.Getenv("PKCS11_PIN_SOURCE")),
		RootKeyURI:         strings.TrimSpace(os.Getenv("PKCS11_ROOT_KEY_URI")),
		IntermediateKeyURI: strings.TrimSpace(os.Getenv("PKCS11_INTERMEDIATE_KEY_URI")),
	}
}

// Enabled reports whether a non-local KMS backend is selected.
func (k KMSConfig) Enabled() bool {
	return k.Type != "" && k.Type != KMSTypeLocal
}

// PKCS11Enabled reports whether PKCS #11 hardware/module signing is requested.
func (k KMSConfig) PKCS11Enabled() bool {
	return strings.EqualFold(string(k.Type), string(KMSTypePKCS11))
}

// Validate checks PKCS #11 settings when that backend is active.
func (k KMSConfig) Validate() error {
	if !k.PKCS11Enabled() {
		return nil
	}
	if k.ModulePath == "" {
		return fmt.Errorf("%s is required when %s=pkcs11", EnvPKCS11ModulePath, EnvKMSType)
	}
	if k.IntermediateKeyURI == "" {
		return fmt.Errorf("PKCS11_INTERMEDIATE_KEY_URI is required when %s=pkcs11", EnvKMSType)
	}
	return nil
}

// PKCS11KMSURI returns the step-ca kms.uri value for ca.json.
func (k KMSConfig) PKCS11KMSURI() string {
	if k.ModulePath == "" {
		return ""
	}
	v := url.Values{}
	v.Set("module-path", k.ModulePath)
	if k.TokenLabel != "" {
		v.Set("token", k.TokenLabel)
	}
	query := v.Encode()
	uri := "pkcs11:" + query
	if k.PIN != "" {
		uri += "?pin-value=" + url.QueryEscape(k.PIN)
	} else if k.PINSource != "" {
		uri += "?pin-source=" + url.QueryEscape(k.PINSource)
	}
	return uri
}

// IntermediateKeyRef returns the key reference written to ca.json "key" when using PKCS #11.
func (k KMSConfig) IntermediateKeyRef() string {
	if k.IntermediateKeyURI != "" {
		return k.IntermediateKeyURI
	}
	return ""
}

// RootKeyRef returns the optional root key reference for PKCS #11 root signing operations.
func (k KMSConfig) RootKeyRef() string {
	return k.RootKeyURI
}
