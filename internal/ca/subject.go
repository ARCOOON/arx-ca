package ca

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"

	"go.step.sm/crypto/x509util"

	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/ARCOOON/arx-ca/internal/models"
)

// certificateSubjectInput carries optional X.509 distinguished name fields for issuance.
type certificateSubjectInput struct {
	Organization       string
	OrganizationalUnit string
	Country            string
	State              string
	Locality           string
}

// certificateKeyUsageInput carries optional extended key usage flags for issuance.
type certificateKeyUsageInput struct {
	ServerAuth bool
	ClientAuth bool
}

func subjectInputFromIssueRequest(req models.IssueCertificateRequest) certificateSubjectInput {
	return certificateSubjectInput{
		Organization:       strings.TrimSpace(req.Organization),
		OrganizationalUnit: strings.TrimSpace(req.OrganizationalUnit),
		Country:            strings.TrimSpace(req.Country),
		State:              strings.TrimSpace(req.State),
		Locality:           strings.TrimSpace(req.Locality),
	}
}

func subjectInputFromGenerateRequest(req models.GenerateCertificateRequest) certificateSubjectInput {
	return certificateSubjectInput{
		Organization:       strings.TrimSpace(req.Organization),
		OrganizationalUnit: strings.TrimSpace(req.OrganizationalUnit),
		Country:            strings.TrimSpace(req.Country),
		State:              strings.TrimSpace(req.State),
		Locality:           strings.TrimSpace(req.Locality),
	}
}

func keyUsageInputFromIssueRequest(req models.IssueCertificateRequest) certificateKeyUsageInput {
	return certificateKeyUsageInput{
		ServerAuth: req.IsServerAuth,
		ClientAuth: req.IsClientAuth,
	}
}

func keyUsageInputFromGenerateRequest(req models.GenerateCertificateRequest) certificateKeyUsageInput {
	return certificateKeyUsageInput{
		ServerAuth: req.IsServerAuth,
		ClientAuth: req.IsClientAuth,
	}
}

func (s certificateSubjectInput) isZero() bool {
	return s.Organization == "" &&
		s.OrganizationalUnit == "" &&
		s.Country == "" &&
		s.State == "" &&
		s.Locality == ""
}

func (s certificateSubjectInput) hasAny() bool {
	return !s.isZero()
}

func (k certificateKeyUsageInput) hasAny() bool {
	return k.ServerAuth || k.ClientAuth
}

func buildPKIXName(commonName string, subject certificateSubjectInput) pkix.Name {
	name := pkix.Name{CommonName: strings.TrimSpace(commonName)}
	if subject.Organization != "" {
		name.Organization = []string{subject.Organization}
	}
	if subject.OrganizationalUnit != "" {
		name.OrganizationalUnit = []string{subject.OrganizationalUnit}
	}
	if subject.Country != "" {
		name.Country = []string{subject.Country}
	}
	if subject.State != "" {
		name.Province = []string{subject.State}
	}
	if subject.Locality != "" {
		name.Locality = []string{subject.Locality}
	}
	return name
}

func createCertificateRequest(commonName string, sans []string, subject certificateSubjectInput, signer crypto.Signer) (*x509.CertificateRequest, error) {
	name := buildPKIXName(commonName, subject)
	dnsNames, ips, emails, uris := x509util.SplitSANs(sans)
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:        name,
		DNSNames:       dnsNames,
		IPAddresses:    ips,
		EmailAddresses: emails,
		URIs:           uris,
	}, signer)
	if err != nil {
		return nil, fmt.Errorf("create certificate signing request: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return nil, fmt.Errorf("parse certificate signing request: %w", err)
	}
	return csr, nil
}

func certificateSignOptions(subject certificateSubjectInput, keyUsage certificateKeyUsageInput) ([]provisioner.SignOption, error) {
	var opts []provisioner.SignOption
	if subject.hasAny() {
		opts = append(opts, subjectModifier(subject))
	}
	if keyUsage.hasAny() {
		opts = append(opts, extKeyUsageModifier(keyUsage))
	}
	return opts, nil
}

func subjectModifier(subject certificateSubjectInput) provisioner.SignOption {
	return provisioner.CertificateModifierFunc(func(cert *x509.Certificate, _ provisioner.SignOptions) error {
		if cert == nil {
			return nil
		}
		if subject.Organization != "" {
			cert.Subject.Organization = []string{subject.Organization}
		}
		if subject.OrganizationalUnit != "" {
			cert.Subject.OrganizationalUnit = []string{subject.OrganizationalUnit}
		}
		if subject.Country != "" {
			cert.Subject.Country = []string{subject.Country}
		}
		if subject.State != "" {
			cert.Subject.Province = []string{subject.State}
		}
		if subject.Locality != "" {
			cert.Subject.Locality = []string{subject.Locality}
		}
		return nil
	})
}

func extKeyUsageModifier(keyUsage certificateKeyUsageInput) provisioner.SignOption {
	return provisioner.CertificateModifierFunc(func(cert *x509.Certificate, _ provisioner.SignOptions) error {
		if cert == nil {
			return nil
		}
		var ekus []x509.ExtKeyUsage
		if keyUsage.ServerAuth {
			ekus = append(ekus, x509.ExtKeyUsageServerAuth)
		}
		if keyUsage.ClientAuth {
			ekus = append(ekus, x509.ExtKeyUsageClientAuth)
		}
		cert.ExtKeyUsage = mergeExtKeyUsage(cert.ExtKeyUsage, ekus)
		return nil
	})
}

func mergeExtKeyUsage(existing, extra []x509.ExtKeyUsage) []x509.ExtKeyUsage {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[x509.ExtKeyUsage]struct{}, len(existing)+len(extra))
	out := make([]x509.ExtKeyUsage, 0, len(existing)+len(extra))
	for _, usage := range existing {
		if _, ok := seen[usage]; ok {
			continue
		}
		seen[usage] = struct{}{}
		out = append(out, usage)
	}
	for _, usage := range extra {
		if _, ok := seen[usage]; ok {
			continue
		}
		seen[usage] = struct{}{}
		out = append(out, usage)
	}
	return out
}
