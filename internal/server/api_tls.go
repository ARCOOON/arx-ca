package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/your-org/arx-ca/internal/ca"
	arxconfig "github.com/your-org/arx-ca/internal/config"
)

// BuildAPITLSConfig constructs TLS settings for the API listener when TLS is enabled.
// Client certificates are verified when presented, enabling mTLS on selected routes.
func BuildAPITLSConfig(engine *ca.PKIEngine, tlsCfg arxconfig.ServerTLSConfig) (*tls.Config, error) {
	if !tlsCfg.Enabled {
		return nil, nil
	}

	validator, err := ca.NewClientCertValidator(engine)
	if err != nil {
		return nil, fmt.Errorf("initialize client certificate validator: %w", err)
	}

	clientCAs := x509.NewCertPool()
	rootPEM := engine.RootCertPEM()
	if len(rootPEM) > 0 {
		clientCAs.AppendCertsFromPEM(rootPEM)
	}
	if intermediatePEM := engine.IntermediateCertPEM(); len(intermediatePEM) > 0 {
		clientCAs.AppendCertsFromPEM(intermediatePEM)
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  clientCAs,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return nil
			}
			return validator.Validate(state.PeerCertificates[0])
		},
	}, nil
}

// ValidateAPITLSCredentials ensures TLS cert and key files exist when TLS is enabled.
func ValidateAPITLSCredentials(settings arxconfig.ServerSettings) (certFile, keyFile string, err error) {
	if !settings.TLS.Enabled {
		return "", "", nil
	}

	certFile, err = settings.ResolvedTLSCertFile()
	if err != nil {
		return "", "", fmt.Errorf("resolve api tls cert_file: %w", err)
	}
	keyFile, err = settings.ResolvedTLSKeyFile()
	if err != nil {
		return "", "", fmt.Errorf("resolve api tls key_file: %w", err)
	}
	if certFile == "" || keyFile == "" {
		return "", "", fmt.Errorf("server tls is enabled but cert_file and key_file must be set")
	}
	if _, err := os.Stat(certFile); err != nil {
		return "", "", fmt.Errorf("api tls cert_file %s: %w", certFile, err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return "", "", fmt.Errorf("api tls key_file %s: %w", keyFile, err)
	}
	return certFile, keyFile, nil
}
