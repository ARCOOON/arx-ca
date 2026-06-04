package models

// RootCertResponse carries the Root CA certificate in PEM encoding.
type RootCertResponse struct {
	PEM string `json:"pem"`
}

// IntermediateCertResponse carries the Intermediate CA certificate in PEM encoding.
type IntermediateCertResponse struct {
	PEM string `json:"pem"`
}

// CASubjectInfo describes an X.509 distinguished name.
type CASubjectInfo struct {
	CommonName         string   `json:"common_name"`
	Organization       []string `json:"organization,omitempty"`
	OrganizationalUnit []string `json:"organizational_unit,omitempty"`
	Country            []string `json:"country,omitempty"`
	Province           []string `json:"province,omitempty"`
	Locality           []string `json:"locality,omitempty"`
	StreetAddress      []string `json:"street_address,omitempty"`
	PostalCode         []string `json:"postal_code,omitempty"`
	SerialNumber       string   `json:"serial_number,omitempty"`
}

// CACertificateInfo carries parsed X.509 metadata and the PEM-encoded certificate.
type CACertificateInfo struct {
	Subject            CASubjectInfo `json:"subject"`
	Issuer             CASubjectInfo `json:"issuer"`
	NotBefore          string        `json:"not_before"`
	NotAfter           string        `json:"not_after"`
	SerialNumber       string        `json:"serial_number"`
	SignatureAlgorithm string        `json:"signature_algorithm"`
	KeyUsages          []string      `json:"key_usages,omitempty"`
	ExtKeyUsages       []string      `json:"ext_key_usages,omitempty"`
	Fingerprint        string        `json:"fingerprint"`
	PEM                string        `json:"pem"`
}

// CAInfoResponse returns Root and Intermediate CA certificate metadata.
type CAInfoResponse struct {
	Root         CACertificateInfo `json:"root"`
	Intermediate CACertificateInfo `json:"intermediate"`
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
