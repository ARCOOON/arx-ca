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
