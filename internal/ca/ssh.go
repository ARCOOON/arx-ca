package ca

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.step.sm/crypto/jose"
	"go.step.sm/crypto/randutil"
	"golang.org/x/crypto/ssh"

	"github.com/smallstep/certificates/authority"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/your-org/arx-ca/internal/models"
)

const (
	defaultSSHUserTTL = "8h"
	sshUserCertType   = provisioner.SSHUserCert
	sshHostCertType   = provisioner.SSHHostCert
)

type sshStepPayload struct {
	SSH *provisioner.SignSSHOptions `json:"ssh,omitempty"`
}

// SignSSHUser issues a short-lived SSH user certificate for the given public key and principal.
// When oidcToken is non-empty, the OIDC provisioner authorizes the request (SSO path).
func (e *PKIEngine) SignSSHUser(ctx context.Context, req models.SignSSHUserRequest, oidcToken string) (*models.SSHCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if e.config == nil || e.config.SSH == nil {
		return nil, errors.New("SSH certificate authority is not configured")
	}

	publicKey, err := parseSSHPublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	principal := strings.TrimSpace(req.Principal)
	if principal == "" {
		return nil, errors.New("principal is required")
	}

	ttl := strings.TrimSpace(req.TTL)
	if ttl == "" {
		ttl = defaultSSHUserTTL
	}

	signOpts := provisioner.SignSSHOptions{
		CertType:   sshUserCertType,
		KeyID:      principal,
		Principals: []string{principal},
	}

	var token string
	switch {
	case strings.TrimSpace(oidcToken) != "":
		token = strings.TrimSpace(oidcToken)
		if _, err := e.findOIDCProvisioner(); err != nil {
			return nil, err
		}
	default:
		token, err = e.createSSHSignToken(principal, &provisioner.SignSSHOptions{
			CertType:   sshUserCertType,
			Principals: []string{principal},
		})
		if err != nil {
			return nil, fmt.Errorf("create SSH signing token: %w", err)
		}
	}

	cert, err := e.signSSH(ctx, publicKey, signOpts, token, ttl)
	if err != nil {
		return nil, err
	}

	return sshCertificateResponse(cert), nil
}

// SignSSHHost issues an SSH host certificate for the given public key and hostname.
func (e *PKIEngine) SignSSHHost(ctx context.Context, req models.SignSSHHostRequest) (*models.SSHCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if e.config == nil || e.config.SSH == nil {
		return nil, errors.New("SSH certificate authority is not configured")
	}

	publicKey, err := parseSSHPublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	hostname := strings.TrimSpace(req.Hostname)
	if hostname == "" {
		return nil, errors.New("hostname is required")
	}

	ttl := strings.TrimSpace(req.TTL)
	if ttl == "" {
		ttl = ""
	}

	signOpts := provisioner.SignSSHOptions{
		CertType:   sshHostCertType,
		KeyID:      hostname,
		Principals: []string{hostname},
	}

	token, err := e.createSSHSignToken(hostname, &provisioner.SignSSHOptions{
		CertType:   sshHostCertType,
		Principals: []string{hostname},
	})
	if err != nil {
		return nil, fmt.Errorf("create SSH signing token: %w", err)
	}

	cert, err := e.signSSH(ctx, publicKey, signOpts, token, ttl)
	if err != nil {
		return nil, err
	}

	return sshCertificateResponse(cert), nil
}

// InspectSSHCertificate decodes an SSH certificate and returns its metadata.
func (e *PKIEngine) InspectSSHCertificate(_ context.Context, certificate string) (*models.SSHCertificateInspection, error) {
	cert, err := parseSSHCertificate(certificate)
	if err != nil {
		return nil, err
	}

	inspection := &models.SSHCertificateInspection{
		KeyID:       cert.KeyId,
		Principals:  append([]string(nil), cert.ValidPrincipals...),
		ValidAfter:  time.Unix(int64(cert.ValidAfter), 0).UTC().Format(time.RFC3339),
		ValidBefore: time.Unix(int64(cert.ValidBefore), 0).UTC().Format(time.RFC3339),
		CertType:    sshCertTypeName(cert.CertType),
		Serial:      cert.Serial,
	}

	if cert.SignatureKey != nil {
		inspection.SignatureKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(cert.SignatureKey)))
	}

	if len(cert.CriticalOptions) > 0 {
		inspection.CriticalOptions = make(map[string]string, len(cert.CriticalOptions))
		for k, v := range cert.CriticalOptions {
			inspection.CriticalOptions[k] = v
		}
	}

	if len(cert.Extensions) > 0 {
		inspection.Extensions = make(map[string]string, len(cert.Extensions))
		for k, v := range cert.Extensions {
			inspection.Extensions[k] = v
		}
	}

	return inspection, nil
}

// GetSSHRoots returns the SSH user and host CA public keys.
func (e *PKIEngine) GetSSHRoots(ctx context.Context) (*models.SSHRootsResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if e.config == nil || e.config.SSH == nil {
		return nil, errors.New("SSH certificate authority is not configured")
	}

	keys, err := e.auth.GetSSHRoots(ctx)
	if err != nil {
		return nil, fmt.Errorf("get SSH roots: %w", err)
	}

	resp := &models.SSHRootsResponse{}
	for _, key := range keys.UserKeys {
		if key != nil {
			resp.UserCAKeys = append(resp.UserCAKeys, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
		}
	}
	for _, key := range keys.HostKeys {
		if key != nil {
			resp.HostCAKeys = append(resp.HostCAKeys, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
		}
	}

	if len(resp.UserCAKeys) == 0 && len(resp.HostCAKeys) == 0 {
		return nil, errors.New("no SSH CA public keys found")
	}

	return resp, nil
}

// SSHEnabled reports whether the CA has SSH signing keys configured.
func (e *PKIEngine) SSHEnabled() bool {
	return e != nil && e.config != nil && e.config.SSH != nil
}

func (e *PKIEngine) signSSH(
	ctx context.Context,
	publicKey ssh.PublicKey,
	opts provisioner.SignSSHOptions,
	token string,
	ttl string,
) (*ssh.Certificate, error) {
	ctx = provisioner.NewContextWithMethod(ctx, provisioner.SSHSignMethod)
	ctx = provisioner.NewContextWithToken(ctx, token)
	ctx = provisioner.NewContextWithCertType(ctx, opts.CertType)
	ctx = authority.NewContext(ctx, e.auth)

	authOpts, err := e.auth.Authorize(ctx, token)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ttl) != "" {
		notAfter, err := provisioner.ParseTimeDuration(ttl)
		if err != nil {
			return nil, fmt.Errorf("invalid ttl: %w", err)
		}
		authOpts = append(authOpts, sshTTLModifier{notAfter: notAfter.RelativeTime(time.Now().UTC())})
	}

	cert, err := e.auth.SignSSH(ctx, publicKey, opts, authOpts...)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, errors.New("signing produced a nil SSH certificate")
	}

	return cert, nil
}

// sshTTLModifier caps SSH certificate validity at the requested TTL.
type sshTTLModifier struct {
	notAfter time.Time
}

func (m sshTTLModifier) Modify(cert *ssh.Certificate, _ provisioner.SignSSHOptions) error {
	cert.ValidBefore = uint64(m.notAfter.Unix())
	return nil
}

func (e *PKIEngine) createSSHSignToken(subject string, sshOpts *provisioner.SignSSHOptions) (string, error) {
	prov, err := e.loadDefaultProvisioner()
	if err != nil {
		return "", err
	}

	jwkProv, ok := prov.(*provisioner.JWK)
	if !ok {
		return "", fmt.Errorf("provisioner %q is not a JWK provisioner", prov.GetName())
	}

	kid, encryptedKey, ok := jwkProv.GetEncryptedKey()
	if !ok || len(encryptedKey) == 0 {
		return "", fmt.Errorf("provisioner %q does not have an encrypted signing key", prov.GetName())
	}

	signer, err := decryptProvisionerKey(kid, []byte(encryptedKey), e.password)
	if err != nil {
		return "", err
	}

	audiences := e.config.GetAudiences().SSHSign
	if len(audiences) == 0 {
		return "", errors.New("no SSH sign audiences configured")
	}

	return buildSSHSignToken(subject, prov.GetName(), audiences[0], sshOpts, signer)
}

func buildSSHSignToken(subject, issuer, audience string, sshOpts *provisioner.SignSSHOptions, signer jose.Signer) (string, error) {
	id, err := randutil.ASCII(64)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claims := struct {
		jose.Claims
		Step *sshStepPayload `json:"step,omitempty"`
	}{
		Claims: jose.Claims{
			ID:        id,
			Subject:   subject,
			Issuer:    issuer,
			IssuedAt:  jose.NewNumericDate(now),
			NotBefore: jose.NewNumericDate(now),
			Expiry:    jose.NewNumericDate(now.Add(5 * time.Minute)),
			Audience:  []string{audience},
		},
		Step: &sshStepPayload{SSH: sshOpts},
	}

	return jose.Signed(signer).Claims(claims).CompactSerialize()
}

func (e *PKIEngine) findOIDCProvisioner() (provisioner.Interface, error) {
	provisioners, _, err := e.auth.GetProvisioners("", 0)
	if err != nil {
		return nil, fmt.Errorf("load provisioners: %w", err)
	}

	for _, prov := range provisioners {
		if prov.GetType() == provisioner.TypeOIDC {
			return prov, nil
		}
	}

	return nil, errors.New("no OIDC provisioner configured for SSH user certificate requests")
}

func parseSSHPublicKey(raw string) (ssh.PublicKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("public_key is required")
	}

	if key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(raw)); err == nil {
		return key, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("parse SSH public key: %w", err)
	}

	key, err := ssh.ParsePublicKey(decoded)
	if err != nil {
		return nil, fmt.Errorf("parse SSH public key: %w", err)
	}

	return key, nil
}

func parseSSHCertificate(raw string) (*ssh.Certificate, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("certificate is required")
	}

	if pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(raw)); err == nil {
		if cert, ok := pub.(*ssh.Certificate); ok {
			return cert, nil
		}
		return nil, errors.New("parsed key is not an SSH certificate")
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("parse SSH certificate: %w", err)
	}

	pub, err := ssh.ParsePublicKey(decoded)
	if err != nil {
		return nil, fmt.Errorf("parse SSH certificate: %w", err)
	}

	cert, ok := pub.(*ssh.Certificate)
	if !ok {
		return nil, errors.New("decoded data is not an SSH certificate")
	}

	return cert, nil
}

func sshCertificateResponse(cert *ssh.Certificate) *models.SSHCertificateResponse {
	return &models.SSHCertificateResponse{
		Certificate: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(cert))),
		KeyID:       cert.KeyId,
		Principals:  append([]string(nil), cert.ValidPrincipals...),
		NotBefore:   time.Unix(int64(cert.ValidAfter), 0).UTC().Format(time.RFC3339),
		NotAfter:    time.Unix(int64(cert.ValidBefore), 0).UTC().Format(time.RFC3339),
		CertType:    sshCertTypeName(cert.CertType),
	}
}

func sshCertTypeName(certType uint32) string {
	switch certType {
	case ssh.UserCert:
		return sshUserCertType
	case ssh.HostCert:
		return sshHostCertType
	default:
		return "unknown"
	}
}