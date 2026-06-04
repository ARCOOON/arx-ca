package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"time"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

// prepareWebUITLS loads or generates TLS material for the WebUI listener.
// When TLS is enabled and certificate files are absent, an ephemeral ECDSA P-256
// certificate is generated with SANs for localhost and detected host addresses.
func prepareWebUITLS(cfg arxconfig.WebUIConfig, log *slog.Logger) (*tls.Config, error) {
	if !cfg.TLS.Enabled {
		return nil, nil
	}
	if log == nil {
		log = slog.Default()
	}

	certFile, err := cfg.ResolvedTLSCertFile()
	if err != nil {
		return nil, err
	}
	keyFile, err := cfg.ResolvedTLSKeyFile()
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	if certFile != "" && keyFile != "" {
		if _, statErr := os.Stat(certFile); statErr == nil {
			if _, keyErr := os.Stat(keyFile); keyErr == nil {
				cert, err = tls.LoadX509KeyPair(certFile, keyFile)
				if err != nil {
					return nil, fmt.Errorf("load webui tls key pair: %w", err)
				}
				log.Info("WebUI TLS using configured certificate files",
					slog.String("cert_file", certFile),
					slog.String("key_file", keyFile),
				)
				return buildWebUITLSConfig(cert), nil
			}
		}
	}

	generated, err := generateWebUIServerCertificate()
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral webui tls certificate: %w", err)
	}
	cert = generated
	log.Warn("WebUI TLS certificate files missing or unreadable; using auto-generated ephemeral certificate with SANs",
		slog.String("cert_file", certFile),
		slog.String("key_file", keyFile),
	)
	return buildWebUITLSConfig(cert), nil
}

func buildWebUITLSConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequestClientCert,
	}
}

func generateWebUIServerCertificate() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate ecdsa key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial: %w", err)
	}

	dnsNames, ipAddresses := webUICertificateSANs()
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "arx-ca-webui",
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func webUICertificateSANs() (dnsNames []string, ipAddresses []net.IP) {
	dnsNames = []string{"localhost"}
	ipSet := map[string]net.IP{}

	addIP := func(ip net.IP) {
		if ip == nil {
			return
		}
		ip = ip.To16()
		if ip == nil || ip.IsUnspecified() {
			return
		}
		if ip.IsLinkLocalMulticast() || ip.IsMulticast() {
			return
		}
		ipSet[ip.String()] = ip
	}

	addIP(net.ParseIP("127.0.0.1"))
	addIP(net.ParseIP("::1"))

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, addrErr := iface.Addrs()
			if addrErr != nil {
				continue
			}
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					addIP(v.IP)
				case *net.IPAddr:
					addIP(v.IP)
				}
			}
		}
	}

	ipAddresses = make([]net.IP, 0, len(ipSet))
	for _, ip := range ipSet {
		ipAddresses = append(ipAddresses, ip)
	}
	return dnsNames, ipAddresses
}
