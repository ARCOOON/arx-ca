package local

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func listFirefoxCerts() ([]InstalledCertificate, error) {
	profiles, err := firefoxProfileDirs()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}

	certutil, err := exec.LookPath("certutil")
	if err != nil {
		return nil, nil
	}

	var out []InstalledCertificate
	for _, profile := range profiles {
		certs, listErr := listFirefoxProfile(certutil, profile)
		if listErr != nil {
			continue
		}
		out = append(out, certs...)
	}
	return out, nil
}

func listFirefoxProfile(certutil, profileDir string) ([]InstalledCertificate, error) {
	dbPath := filepath.Join(profileDir, "cert9.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}

	cmd := exec.Command(certutil, "-L", "-d", "sql:"+profileDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var out []InstalledCertificate
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Certificate Nickname") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		nickname := fields[0]
		dump := exec.Command(certutil, "-L", "-d", "sql:"+profileDir, "-a", "-n", nickname)
		pemBytes, dumpErr := dump.Output()
		if dumpErr != nil {
			continue
		}
		block, _ := pem.Decode(pemBytes)
		if block == nil || block.Type != "CERTIFICATE" {
			continue
		}
		cert, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			continue
		}
		sum := sha256.Sum256(cert.Raw)
		thumb := hex.EncodeToString(sum[:])
		out = append(out, InstalledCertificate{
			ID:         thumb,
			Store:      StoreBrowser,
			StoreName:  "Firefox:" + filepath.Base(profileDir),
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
	return out, nil
}

func firefoxProfileDirs() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	candidates := []string{
		filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles"),
		filepath.Join(home, ".mozilla", "firefox"),
	}

	var profiles []string
	for _, base := range candidates {
		entries, readErr := os.ReadDir(base)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			profiles = append(profiles, filepath.Join(base, entry.Name()))
		}
	}
	return profiles, nil
}
