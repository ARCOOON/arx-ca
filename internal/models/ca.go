package models

// RootCertResponse carries the Root CA certificate in PEM encoding.
type RootCertResponse struct {
	PEM string `json:"pem"`
}
