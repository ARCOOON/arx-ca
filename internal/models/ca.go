package models

// RootCertResponse carries the Root CA certificate in PEM encoding.
type RootCertResponse struct {
	PEM string `json:"pem"`
}

// IntermediateCertResponse carries the Intermediate CA certificate in PEM encoding.
type IntermediateCertResponse struct {
	PEM string `json:"pem"`
}

// PublicCertificateSummary is a read-only view of an issued certificate for unauthenticated clients.
type PublicCertificateSummary struct {
	Serial      string   `json:"serial"`
	Subject     string   `json:"subject"`
	DNSNames    []string `json:"dns_names,omitempty"`
	IPAddresses []string `json:"ip_addresses,omitempty"`
	NotBefore   string   `json:"not_before"`
	NotAfter    string   `json:"not_after"`
	Revoked     bool     `json:"revoked"`
}

// PublicListCertificatesResponse returns public certificate metadata from the CA.
type PublicListCertificatesResponse struct {
	Certificates []PublicCertificateSummary `json:"certificates"`
	Total        int                        `json:"total"`
}
