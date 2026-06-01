package local

import "time"

// StoreKind identifies where a certificate is installed.
type StoreKind string

const (
	StoreSystem  StoreKind = "system"
	StoreUser    StoreKind = "user"
	StoreBrowser StoreKind = "browser"
)

// InstalledCertificate is a read-only view of a certificate on the local machine.
type InstalledCertificate struct {
	ID         string    `json:"id"`
	Store      StoreKind `json:"store"`
	StoreName  string    `json:"store_name"`
	Subject    string    `json:"subject"`
	Issuer     string    `json:"issuer"`
	Serial     string    `json:"serial"`
	Thumbprint string    `json:"thumbprint"`
	NotBefore  time.Time `json:"not_before"`
	NotAfter   time.Time `json:"not_after"`
	DNSNames   []string  `json:"dns_names,omitempty"`
	IsCA       bool      `json:"is_ca"`
}
