package config

import "testing"

func TestManagedCertValidate(t *testing.T) {
	t.Parallel()

	validAPI := ManagedCert{
		Protocol:   AgentProtocolAPI,
		CertPath:   "/tmp/cert.pem",
		KeyPath:    "/tmp/key.pem",
		CommonName: "app.example.com",
	}
	if err := validAPI.Validate(); err != nil {
		t.Fatalf("api cert: %v", err)
	}

	validACME := ManagedCert{
		Protocol:         AgentProtocolACME,
		CertPath:         "/tmp/cert.pem",
		KeyPath:          "/tmp/key.pem",
		CommonName:       "app.example.com",
		ACMEDirectoryURL: "https://ca.example.com/acme/directory",
		ACMEEmail:        "admin@example.com",
		Webroot:          "/var/www/html",
	}
	if err := validACME.Validate(); err != nil {
		t.Fatalf("acme cert: %v", err)
	}

	invalidACME := ManagedCert{
		Protocol:   AgentProtocolACME,
		CertPath:   "/tmp/cert.pem",
		KeyPath:    "/tmp/key.pem",
		CommonName: "app.example.com",
	}
	if err := invalidACME.Validate(); err == nil {
		t.Fatal("expected acme validation error")
	}
}
