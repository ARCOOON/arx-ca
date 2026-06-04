package models

// ProvisionerTokenRequest describes a single-use signing token to mint for a subject.
type ProvisionerTokenRequest struct {
	Provisioner string   `json:"provisioner,omitempty"`
	CommonName  string   `json:"common_name"`
	DNSSANs     []string `json:"dns_sans,omitempty"`
	IPSANs      []string `json:"ip_sans,omitempty"`
	TokenTTL    string   `json:"token_ttl,omitempty"`
}

// ProvisionerTokenResponse returns a JWK provisioner signing token.
type ProvisionerTokenResponse struct {
	Token           string `json:"token"`
	Provisioner     string `json:"provisioner"`
	ProvisionerType string `json:"provisioner_type"`
	ExpiresIn       int    `json:"expires_in"`
	Audience        string `json:"audience"`
}

// ProvisionerSummary describes a configured step-ca provisioner.
type ProvisionerSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListProvisionersResponse returns provisioners registered with the CA.
type ListProvisionersResponse struct {
	Provisioners []ProvisionerSummary `json:"provisioners"`
	Total        int                  `json:"total"`
}

// CAProvisionerDetail describes a sanitized provisioner entry from ca.json.
type CAProvisionerDetail struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	RequireEAB *bool    `json:"require_eab,omitempty"`
	Challenges []string `json:"challenges,omitempty"`
	Challenge  string   `json:"challenge,omitempty"`
}

// CAProvisionersResponse returns active provisioners parsed from ca.json.
type CAProvisionersResponse struct {
	Provisioners []CAProvisionerDetail `json:"provisioners"`
	Total        int                   `json:"total"`
}
