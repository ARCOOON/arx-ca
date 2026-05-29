package models

// CreateACMEEABKeyRequest creates an ACME External Account Binding credential.
type CreateACMEEABKeyRequest struct {
	Provisioner string `json:"provisioner,omitempty"`
	Reference   string `json:"reference,omitempty"`
}

// ACMEEABKeyResponse returns EAB credentials for ACME account registration.
type ACMEEABKeyResponse struct {
	KeyID       string `json:"key_id"`
	Provisioner string `json:"provisioner"`
	HmacKey     string `json:"hmac_key"`
	Reference   string `json:"reference,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// ACMEStatusResponse exposes ACME directory metadata for operators.
type ACMEStatusResponse struct {
	Enabled             bool     `json:"enabled"`
	DirectoryURL        string   `json:"directory_url,omitempty"`
	Provisioner         string   `json:"provisioner,omitempty"`
	Challenges          []string `json:"challenges,omitempty"`
	DNSName             string   `json:"dns_name,omitempty"`
	RequireEAB          bool     `json:"require_eab"`
	DeviceAttestEnabled bool     `json:"device_attest_enabled"`
	AttestationFormats  []string `json:"attestation_formats,omitempty"`
}

// SCEPStatusResponse exposes SCEP endpoint metadata.
type SCEPStatusResponse struct {
	Enabled       bool   `json:"enabled"`
	BaseURL       string `json:"base_url,omitempty"`
	Provisioner   string `json:"provisioner,omitempty"`
	ChallengeHint string `json:"challenge_hint,omitempty"`
}

// K8sProvisionerStatusResponse exposes Kubernetes Service Account provisioner metadata.
type K8sProvisionerStatusResponse struct {
	Enabled     bool   `json:"enabled"`
	Provisioner string `json:"provisioner,omitempty"`
	ReviewMode  string `json:"review_mode,omitempty"`
	HasPubKeys  bool   `json:"has_public_keys"`
	UsesAPI     bool   `json:"uses_token_review_api"`
}

// NDESStatusResponse exposes NDES connector metadata for AD CS migrations.
type NDESStatusResponse struct {
	Enabled        bool     `json:"enabled"`
	SCEPEndpoint   string   `json:"scep_endpoint,omitempty"`
	AdminEndpoint  string   `json:"admin_endpoint,omitempty"`
	Connectors     []string `json:"connectors,omitempty"`
	ADCSCompatible bool     `json:"adcs_compatible"`
}
