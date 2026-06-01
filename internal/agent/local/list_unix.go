//go:build !windows

package local

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func listStore(kind StoreKind) ([]InstalledCertificate, error) {
	switch kind {
	case StoreSystem:
		return listPEMDirectory(StoreSystem, "system", []string{
			"/etc/ssl/certs",
			"/usr/local/share/ca-certificates",
			"/etc/pki/ca-trust/source/anchors",
		})
	case StoreUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		return listPEMDirectory(StoreUser, "user", []string{
			filepath.Join(home, ".local", "share", "ca-certificates"),
			filepath.Join(home, ".pki", "nssdb"),
		})
	case StoreBrowser:
		return listFirefoxCerts()
	default:
		return nil, fmt.Errorf("unsupported store %q", kind)
	}
}

func listPEMDirectory(store StoreKind, label string, dirs []string) ([]InstalledCertificate, error) {
	var out []InstalledCertificate
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(path), ".pem") &&
				!strings.HasSuffix(strings.ToLower(path), ".crt") &&
				!strings.HasSuffix(strings.ToLower(path), ".cer") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, cert := range parsePEMCerts(data) {
				cert.Store = store
				cert.StoreName = label + ":" + filepath.Base(path)
				out = append(out, cert)
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return out, nil
}

func parsePEMCerts(data []byte) []InstalledCertificate {
	var out []InstalledCertificate
	rest := data
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(cert.Raw)
		thumb := hex.EncodeToString(sum[:])
		out = append(out, InstalledCertificate{
			ID:         thumb,
			Subject:    cert.Subject.String(),
			Issuer:     cert.Issuer.String(),
			Serial:     cert.SerialNumber.String(),
			Thumbprint: thumb,
			NotBefore:  cert.NotBefore,
			NotAfter:   cert.NotAfter,
			DNSNames:   append([]string(nil), cert.DNSNames...),
			IsCA:       cert.IsCA,
		})
	}
	return out
}
