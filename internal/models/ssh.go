package models

// SignSSHUserRequest carries the SSH public key and principal for a user certificate.
type SignSSHUserRequest struct {
	PublicKey string `json:"public_key"`
	Principal string `json:"principal"`
	TTL       string `json:"ttl,omitempty"`
	OIDCToken string `json:"oidc_token,omitempty"`
}

// SignSSHHostRequest carries the SSH public key and hostname for a host certificate.
type SignSSHHostRequest struct {
	PublicKey string `json:"public_key"`
	Hostname  string `json:"hostname"`
	TTL       string `json:"ttl,omitempty"`
}

// InspectSSHCertificateRequest carries an SSH certificate to decode.
type InspectSSHCertificateRequest struct {
	Certificate string `json:"certificate"`
}

// SSHCertificateResponse returns a signed SSH certificate in OpenSSH wire format.
type SSHCertificateResponse struct {
	Certificate string `json:"certificate"`
	KeyID       string `json:"key_id"`
	Principals  []string `json:"principals"`
	NotBefore   string `json:"not_before"`
	NotAfter    string `json:"not_after"`
	CertType    string `json:"cert_type"`
}

// SSHCertificateInspection describes decoded SSH certificate metadata.
type SSHCertificateInspection struct {
	KeyID            string            `json:"key_id"`
	Principals       []string          `json:"principals"`
	ValidAfter       string            `json:"valid_after"`
	ValidBefore      string            `json:"valid_before"`
	CertType         string            `json:"cert_type"`
	Serial           uint64            `json:"serial"`
	SignatureKey     string            `json:"signature_key,omitempty"`
	CriticalOptions  map[string]string `json:"critical_options,omitempty"`
	Extensions       map[string]string `json:"extensions,omitempty"`
	Provisioner      string            `json:"provisioner,omitempty"`
}

// SSHRootsResponse returns SSH CA public keys for trust configuration.
type SSHRootsResponse struct {
	UserCAKeys []string `json:"user_ca_keys"`
	HostCAKeys []string `json:"host_ca_keys"`
}
