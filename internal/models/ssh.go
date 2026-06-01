package models

// SignSSHUserRequest issues a short-lived SSH user certificate.
// When token is set, an OIDC or provisioner token is used instead of API credentials.
type SignSSHUserRequest struct {
	PublicKey   string   `json:"public_key"`
	Principal   string   `json:"principal"`
	Principals  []string `json:"principals,omitempty"`
	TTL         string   `json:"ttl,omitempty"`
	Token       string   `json:"token,omitempty"`
	Provisioner string   `json:"provisioner,omitempty"`
}

// SignSSHHostRequest issues an SSH host certificate.
type SignSSHHostRequest struct {
	PublicKey   string   `json:"public_key"`
	Hostname    string   `json:"hostname"`
	Principals  []string `json:"principals,omitempty"`
	TTL         string   `json:"ttl,omitempty"`
	Provisioner string   `json:"provisioner,omitempty"`
}

// InspectSSHCertificateRequest decodes an SSH certificate.
type InspectSSHCertificateRequest struct {
	Certificate string `json:"certificate"`
}

// SSHCertificateResponse returns a signed SSH certificate.
type SSHCertificateResponse struct {
	Certificate     string   `json:"certificate"`
	CertificateType string   `json:"certificate_type"`
	KeyID           string   `json:"key_id"`
	Principals      []string `json:"principals"`
	Serial          uint64   `json:"serial"`
	ValidAfter      string   `json:"valid_after"`
	ValidBefore     string   `json:"valid_before"`
}

// SSHCertificateInspection returns decoded SSH certificate metadata.
type SSHCertificateInspection struct {
	CertificateType string            `json:"certificate_type"`
	KeyID           string            `json:"key_id"`
	Principals      []string          `json:"principals"`
	Serial          uint64            `json:"serial"`
	ValidAfter      string            `json:"valid_after"`
	ValidBefore     string            `json:"valid_before"`
	PublicKeyType   string            `json:"public_key_type"`
	CriticalOptions map[string]string `json:"critical_options,omitempty"`
	Extensions      map[string]string `json:"extensions,omitempty"`
	SignatureKey    string            `json:"signature_key,omitempty"`
}

// SSHRootsResponse returns SSH CA public keys for client trust configuration.
type SSHRootsResponse struct {
	UserKeys []SSHRootKey `json:"user_keys"`
	HostKeys []SSHRootKey `json:"host_keys"`
}

// SSHRootKey is a single SSH CA public key in authorized_keys format.
type SSHRootKey struct {
	PublicKey   string `json:"public_key"`
	KeyType     string `json:"key_type"`
	Fingerprint string `json:"fingerprint"`
}
