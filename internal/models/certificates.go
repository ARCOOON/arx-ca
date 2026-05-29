package models

import "time"

// IssueCertificateRequest carries a PEM-encoded CSR to be signed by the intermediate CA.
type IssueCertificateRequest struct {
	CSR        string         `json:"csr"`
	TTL        string         `json:"ttl,omitempty"`
	TemplateID string         `json:"template_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// AutoCertificateRequest describes a certificate to be generated and signed in one step.
type AutoCertificateRequest struct {
	CommonName string         `json:"common_name"`
	DNSSANs    []string       `json:"dns_sans,omitempty"`
	IPSANs     []string       `json:"ip_sans,omitempty"`
	TTL        string         `json:"ttl,omitempty"`
	TemplateID string         `json:"template_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// RevokeCertificateRequest revokes a previously issued certificate by serial number.
type RevokeCertificateRequest struct {
	Serial     string `json:"serial"`
	Reason     string `json:"reason,omitempty"`
	ReasonCode int    `json:"reason_code,omitempty"`
}

// CertificatePEMResponse returns a signed certificate in PEM encoding.
type CertificatePEMResponse struct {
	CertificatePEM string `json:"certificate_pem"`
	Serial         string `json:"serial"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
}

// AutoCertificateResponse returns the generated key pair and signed certificate.
type AutoCertificateResponse struct {
	CertificatePEM string `json:"certificate_pem"`
	PrivateKeyPEM  string `json:"private_key_pem"`
	Serial         string `json:"serial"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
}

// RevokeCertificateResponse confirms passive revocation in the step-ca database.
type RevokeCertificateResponse struct {
	Serial    string `json:"serial"`
	RevokedAt string `json:"revoked_at"`
}

// CertificateSummary is a compact view of an issued certificate stored in the CA database.
type CertificateSummary struct {
	Serial        string    `json:"serial"`
	Subject       string    `json:"subject"`
	DNSNames      []string  `json:"dns_names,omitempty"`
	IPAddresses   []string  `json:"ip_addresses,omitempty"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	Revoked       bool      `json:"revoked"`
	ProvisionerID string    `json:"provisioner_id,omitempty"`
	Provisioner   string    `json:"provisioner,omitempty"`
}

// ListCertificatesResponse returns all certificates known to the CA database.
type ListCertificatesResponse struct {
	Certificates []CertificateSummary `json:"certificates"`
	Total        int                  `json:"total"`
}

// IssueCertificateWithTokenRequest signs a CSR using a provisioner-issued single-use token.
type IssueCertificateWithTokenRequest struct {
	Token      string         `json:"token"`
	CSR        string         `json:"csr"`
	TTL        string         `json:"ttl,omitempty"`
	TemplateID string         `json:"template_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// LintCertificateRequest carries a PEM-encoded certificate to inspect.
type LintCertificateRequest struct {
	CertificatePEM string `json:"certificate_pem"`
}

// CertificateLintFinding is a single RFC 5280 or CA/Browser Forum lint result.
type CertificateLintFinding struct {
	Lint     string `json:"lint"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"`
}

// LintCertificateSummary aggregates lint outcomes by severity.
type LintCertificateSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Notices  int `json:"notices"`
	Fatals   int `json:"fatals"`
}

// LintCertificateResponse returns compliance findings for a certificate.
type LintCertificateResponse struct {
	Findings []CertificateLintFinding `json:"findings"`
	Summary  LintCertificateSummary   `json:"summary"`
}
