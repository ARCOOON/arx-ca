package ca

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.step.sm/crypto/pemutil"

	"github.com/smallstep/certificates/cas"
	"github.com/smallstep/certificates/cas/apiv1"
	"github.com/smallstep/certificates/pki"

	"github.com/ARCOOON/arx-ca/internal/config"
)

func bootstrapCACreator() (apiv1.CertificateAuthorityCreator, error) {
	svc, err := cas.New(context.Background(), apiv1.Options{
		Type:      apiv1.SoftCAS,
		IsCreator: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create softcas: %w", err)
	}
	creator, ok := svc.(apiv1.CertificateAuthorityCreator)
	if !ok {
		return nil, errors.New("softcas does not implement CertificateAuthorityCreator")
	}
	return creator, nil
}

func generateBootstrapRoot(
	p *pki.PKI,
	creator apiv1.CertificateAuthorityCreator,
	boot config.CABootstrapConfig,
	resource string,
	pass []byte,
) (*apiv1.CreateCertificateAuthorityResponse, error) {
	keyReq := boot.KeyCreateRequest()
	subject := bootstrapSubject(boot.RootCN, boot.Organization, boot.Country)

	rootName := strings.TrimSpace(boot.RootCN)
	if rootName == "" {
		rootName = resource + "-Root-CA"
	}

	resp, err := creator.CreateCertificateAuthority(&apiv1.CreateCertificateAuthorityRequest{
		Name:     rootName,
		Type:     apiv1.RootCA,
		Lifetime: 10 * 365 * 24 * time.Hour,
		CreateKey: &apiv1.CreateKeyRequest{
			Name:               p.RootKey[0],
			SignatureAlgorithm: keyReq.SignatureAlgorithm,
			Bits:               keyReq.Bits,
		},
		Template: &x509.Certificate{
			Subject:               subject,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            1,
			MaxPathLenZero:        false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create root certificate authority: %w", err)
	}

	if resp.KeyName != "" {
		p.RootKey[0] = resp.KeyName
	}
	if err := p.WriteRootCertificate(resp.Certificate, resp.PrivateKey, pass); err != nil {
		return nil, fmt.Errorf("write root certificate: %w", err)
	}
	return resp, nil
}

func generateBootstrapIntermediate(
	p *pki.PKI,
	creator apiv1.CertificateAuthorityCreator,
	boot config.CABootstrapConfig,
	resource string,
	parent *apiv1.CreateCertificateAuthorityResponse,
	pass []byte,
) error {
	keyReq := boot.KeyCreateRequest()
	subject := bootstrapSubject(boot.IntermediateCN, boot.Organization, boot.Country)

	intermediateName := strings.TrimSpace(boot.IntermediateCN)
	if intermediateName == "" {
		intermediateName = resource + "-Intermediate-CA"
	}

	resp, err := creator.CreateCertificateAuthority(&apiv1.CreateCertificateAuthorityRequest{
		Name:     intermediateName,
		Type:     apiv1.IntermediateCA,
		Lifetime: 10 * 365 * 24 * time.Hour,
		CreateKey: &apiv1.CreateKeyRequest{
			Name:               p.IntermediateKey,
			SignatureAlgorithm: keyReq.SignatureAlgorithm,
			Bits:               keyReq.Bits,
		},
		Template: &x509.Certificate{
			Subject:               subject,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            0,
			MaxPathLenZero:        true,
		},
		Parent: parent,
	})
	if err != nil {
		return fmt.Errorf("create intermediate certificate authority: %w", err)
	}

	p.Files[p.Intermediate] = encodeCertificatePEM(resp.Certificate)

	if resp.KeyName != "" {
		p.IntermediateKey = resp.KeyName
	}
	if resp.PrivateKey != nil {
		keyPEM, err := encodeEncryptedPrivateKey(resp.PrivateKey, pass)
		if err != nil {
			return fmt.Errorf("encode intermediate private key: %w", err)
		}
		p.Files[p.IntermediateKey] = keyPEM
	}
	return nil
}

func bootstrapSubject(commonName, organization, country string) pkix.Name {
	subject := pkix.Name{
		CommonName: strings.TrimSpace(commonName),
	}
	if org := strings.TrimSpace(organization); org != "" {
		subject.Organization = []string{org}
	}
	if c := strings.TrimSpace(country); c != "" {
		subject.Country = []string{c}
	}
	return subject
}

func encodeEncryptedPrivateKey(key interface{}, pass []byte) ([]byte, error) {
	block, err := pemutil.Serialize(key, pemutil.WithPassword(pass))
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(block), nil
}
