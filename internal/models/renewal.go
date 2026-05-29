package models

// RenewCertificateRequest renews an existing certificate for non-ACME clients.
type RenewCertificateRequest struct {
	CertificatePEM string `json:"certificate_pem,omitempty"`
	RenewToken     string `json:"renew_token,omitempty"`
}

// RekeyCertificateRequest rekeys an existing certificate using a new CSR.
type RekeyCertificateRequest struct {
	CertificatePEM string `json:"certificate_pem,omitempty"`
	RenewToken     string `json:"renew_token,omitempty"`
	CSR            string `json:"csr"`
}

// ACMEStatusResponse exposes ACME directory metadata for operators.
type ACMEStatusResponse struct {
	Enabled       bool     `json:"enabled"`
	DirectoryURL  string   `json:"directory_url,omitempty"`
	Provisioner   string   `json:"provisioner,omitempty"`
	Challenges    []string `json:"challenges,omitempty"`
	DNSName       string   `json:"dns_name,omitempty"`
}
