package ca

import (
	"context"
	"crypto/sha256"
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
	defaultSSHUserCertTTL  = 16 * time.Hour
	defaultSSHHostCertTTL  = 30 * 24 * time.Hour
	maxSSHUserCertTTL      = 24 * time.Hour
	maxSSHHostCertTTL      = 30 * 24 * time.Hour
	sshProvisionerTokenTTL = 5 * time.Minute
)

type sshStepPayload struct {
	SSH *provisioner.SignSSHOptions `json:"ssh,omitempty"`
}

// SignSSHUserCertificate signs an SSH user certificate using a provisioner token.
func (e *PKIEngine) SignSSHUserCertificate(ctx context.Context, req models.SignSSHUserRequest, token string) (*models.SSHCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if !e.SSHEnabled() {
		return nil, errors.New("SSH CA is not configured")
	}

	pub, err := parseSSHPublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	principals, err := resolveSSHPrincipals(req.Principal, req.Principals)
	if err != nil {
		return nil, err
	}

	ttl, err := parseSSHUserTTL(req.TTL)
	if err != nil {
		return nil, err
	}

	opts := provisioner.SignSSHOptions{
		CertType:   provisioner.SSHUserCert,
		KeyID:      principals[0],
		Principals: principals,
	}
	if ttl > 0 {
		var validBefore provisioner.TimeDuration
		validBefore.SetDuration(ttl)
		opts.ValidBefore = validBefore
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("signing token is required")
	}

	cert, err := e.signSSHWithToken(ctx, pub, opts, token)
	if err != nil {
		return nil, err
	}

	return sshCertificateResponse(cert), nil
}

// SignSSHHostCertificate signs an SSH host certificate using the default JWK provisioner.
func (e *PKIEngine) SignSSHHostCertificate(ctx context.Context, req models.SignSSHHostRequest) (*models.SSHCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if !e.SSHEnabled() {
		return nil, errors.New("SSH CA is not configured")
	}

	pub, err := parseSSHPublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	principals, err := resolveSSHPrincipals(req.Hostname, req.Principals)
	if err != nil {
		return nil, err
	}

	ttl, err := parseSSHHostTTL(req.TTL)
	if err != nil {
		return nil, err
	}

	provisionerName := strings.TrimSpace(req.Provisioner)
	if provisionerName == "" {
		provisionerName = defaultProvisioner
	}

	sshOpts := &provisioner.SignSSHOptions{
		CertType:   provisioner.SSHHostCert,
		KeyID:      principals[0],
		Principals: principals,
	}
	if ttl > 0 {
		var validBefore provisioner.TimeDuration
		validBefore.SetDuration(ttl)
		sshOpts.ValidBefore = validBefore
	}

	token, err := e.createSSHSignToken(provisionerName, principals[0], sshOpts, sshProvisionerTokenTTL)
	if err != nil {
		return nil, err
	}

	opts := provisioner.SignSSHOptions{
		CertType:   provisioner.SSHHostCert,
		KeyID:      principals[0],
		Principals: principals,
	}
	if ttl > 0 {
		opts.ValidBefore = sshOpts.ValidBefore
	}

	cert, err := e.signSSHWithToken(ctx, pub, opts, token)
	if err != nil {
		return nil, err
	}

	return sshCertificateResponse(cert), nil
}

// MintSSHUserSignToken creates a provisioner token for SSH user certificate signing.
func (e *PKIEngine) MintSSHUserSignToken(req models.SignSSHUserRequest) (string, error) {
	if e == nil || e.auth == nil {
		return "", errors.New("CA engine is not initialized")
	}

	principals, err := resolveSSHPrincipals(req.Principal, req.Principals)
	if err != nil {
		return "", err
	}

	ttl, err := parseSSHUserTTL(req.TTL)
	if err != nil {
		return "", err
	}

	provisionerName := strings.TrimSpace(req.Provisioner)
	if provisionerName == "" {
		provisionerName = defaultProvisioner
	}

	sshOpts := &provisioner.SignSSHOptions{
		CertType:   provisioner.SSHUserCert,
		KeyID:      principals[0],
		Principals: principals,
	}
	if ttl > 0 {
		var validBefore provisioner.TimeDuration
		validBefore.SetDuration(ttl)
		sshOpts.ValidBefore = validBefore
	}

	return e.createSSHSignToken(provisionerName, principals[0], sshOpts, sshProvisionerTokenTTL)
}

// InspectSSHCertificate decodes an SSH certificate and returns its metadata.
func (e *PKIEngine) InspectSSHCertificate(certificate string) (*models.SSHCertificateInspection, error) {
	cert, err := parseSSHCertificate(certificate)
	if err != nil {
		return nil, err
	}

	certType := "unknown"
	switch cert.CertType {
	case ssh.UserCert:
		certType = provisioner.SSHUserCert
	case ssh.HostCert:
		certType = provisioner.SSHHostCert
	}

	pubKeyType := ""
	if cert.Key != nil {
		pubKeyType = cert.Key.Type()
	}

	inspection := &models.SSHCertificateInspection{
		CertificateType: certType,
		KeyID:           cert.KeyId,
		Principals:      append([]string(nil), cert.ValidPrincipals...),
		Serial:          cert.Serial,
		ValidAfter:      time.Unix(int64(cert.ValidAfter), 0).UTC().Format(time.RFC3339),
		ValidBefore:     time.Unix(int64(cert.ValidBefore), 0).UTC().Format(time.RFC3339),
		PublicKeyType:   pubKeyType,
		CriticalOptions: copyStringMap(cert.CriticalOptions),
		Extensions:      copyStringMap(cert.Extensions),
	}

	if cert.SignatureKey != nil {
		inspection.SignatureKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(cert.SignatureKey)))
	}

	return inspection, nil
}

// GetSSHRoots returns SSH CA public keys for user and host trust configuration.
func (e *PKIEngine) GetSSHRoots(ctx context.Context) (*models.SSHRootsResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if !e.SSHEnabled() {
		return nil, errors.New("SSH CA is not configured")
	}

	keys, err := e.auth.GetSSHRoots(ctx)
	if err != nil {
		return nil, fmt.Errorf("get SSH roots: %w", err)
	}

	resp := &models.SSHRootsResponse{
		UserKeys: make([]models.SSHRootKey, 0, len(keys.UserKeys)),
		HostKeys: make([]models.SSHRootKey, 0, len(keys.HostKeys)),
	}

	for _, key := range keys.UserKeys {
		resp.UserKeys = append(resp.UserKeys, sshRootKey(key))
	}
	for _, key := range keys.HostKeys {
		resp.HostKeys = append(resp.HostKeys, sshRootKey(key))
	}

	if len(resp.UserKeys) == 0 && len(resp.HostKeys) == 0 {
		return nil, errors.New("no SSH CA public keys found")
	}

	return resp, nil
}

func (e *PKIEngine) createSSHSignToken(provisionerName, subject string, sshOpts *provisioner.SignSSHOptions, tokenTTL time.Duration) (string, error) {
	prov, err := e.loadProvisionerByName(provisionerName)
	if err != nil {
		return "", err
	}

	switch p := prov.(type) {
	case *provisioner.JWK:
		kid, encryptedKey, ok := p.GetEncryptedKey()
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

		return buildSSHProvisionerToken(subject, prov.GetName(), audiences[0], sshOpts, signer, tokenTTL)
	default:
		return "", fmt.Errorf("provisioner %q (type %s) cannot mint SSH signing tokens; use a JWK provisioner", prov.GetName(), prov.GetType().String())
	}
}

func (e *PKIEngine) signSSHWithToken(ctx context.Context, pub ssh.PublicKey, opts provisioner.SignSSHOptions, token string) (*ssh.Certificate, error) {
	ctx = provisioner.NewContextWithMethod(ctx, provisioner.SSHSignMethod)
	ctx = provisioner.NewContextWithToken(ctx, token)
	ctx = provisioner.NewContextWithCertType(ctx, opts.CertType)
	ctx = authority.NewContext(ctx, e.auth)

	signOpts, err := e.auth.Authorize(ctx, token)
	if err != nil {
		return nil, err
	}

	return e.auth.SignSSH(ctx, pub, opts, signOpts...)
}

func buildSSHProvisionerToken(subject, issuer, audience string, sshOpts *provisioner.SignSSHOptions, signer jose.Signer, tokenTTL time.Duration) (string, error) {
	if tokenTTL <= 0 {
		tokenTTL = sshProvisionerTokenTTL
	}

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
			Expiry:    jose.NewNumericDate(now.Add(tokenTTL)),
			Audience:  []string{audience},
		},
		Step: &sshStepPayload{SSH: sshOpts},
	}

	return jose.Signed(signer).Claims(claims).CompactSerialize()
}

func parseSSHPublicKey(raw string) (ssh.PublicKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("public_key is required")
	}

	if data, err := base64.StdEncoding.DecodeString(raw); err == nil {
		if pub, err := ssh.ParsePublicKey(data); err == nil {
			return pub, nil
		}
	}

	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(raw))
	if err == nil {
		return pub, nil
	}

	if pub, err := ssh.ParsePublicKey([]byte(raw)); err == nil {
		return pub, nil
	}

	return nil, fmt.Errorf("parse ssh public key: %w", err)
}

func parseSSHCertificate(raw string) (*ssh.Certificate, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("certificate is required")
	}

	pub, err := parseSSHPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse ssh certificate: %w", err)
	}

	cert, ok := pub.(*ssh.Certificate)
	if !ok {
		return nil, errors.New("provided key is not an SSH certificate")
	}

	return cert, nil
}

func resolveSSHPrincipals(primary string, extras []string) ([]string, error) {
	primary = strings.TrimSpace(primary)
	if primary == "" && len(extras) == 0 {
		return nil, errors.New("at least one principal is required")
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, 1+len(extras))

	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	add(primary)
	for _, principal := range extras {
		add(principal)
	}

	if len(out) == 0 {
		return nil, errors.New("at least one principal is required")
	}

	return out, nil
}

func parseSSHUserTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSSHUserCertTTL, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid ttl: %w", err)
	}
	if d <= 0 {
		return 0, errors.New("ttl must be greater than zero")
	}
	if d > maxSSHUserCertTTL {
		return 0, fmt.Errorf("ttl must not exceed %s", maxSSHUserCertTTL)
	}
	return d, nil
}

func parseSSHHostTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSSHHostCertTTL, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid ttl: %w", err)
	}
	if d <= 0 {
		return 0, errors.New("ttl must be greater than zero")
	}
	if d > maxSSHHostCertTTL {
		return 0, fmt.Errorf("ttl must not exceed %s", maxSSHHostCertTTL)
	}
	return d, nil
}

func sshCertificateResponse(cert *ssh.Certificate) *models.SSHCertificateResponse {
	certType := provisioner.SSHUserCert
	if cert.CertType == ssh.HostCert {
		certType = provisioner.SSHHostCert
	}

	return &models.SSHCertificateResponse{
		Certificate:     strings.TrimSpace(string(ssh.MarshalAuthorizedKey(cert))),
		CertificateType: certType,
		KeyID:           cert.KeyId,
		Principals:      append([]string(nil), cert.ValidPrincipals...),
		Serial:          cert.Serial,
		ValidAfter:      time.Unix(int64(cert.ValidAfter), 0).UTC().Format(time.RFC3339),
		ValidBefore:     time.Unix(int64(cert.ValidBefore), 0).UTC().Format(time.RFC3339),
	}
}

func sshRootKey(key ssh.PublicKey) models.SSHRootKey {
	if key == nil {
		return models.SSHRootKey{}
	}

	fingerprint := sha256.Sum256(key.Marshal())
	return models.SSHRootKey{
		PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))),
		KeyType:     key.Type(),
		Fingerprint: "SHA256:" + base64.RawStdEncoding.EncodeToString(fingerprint[:]),
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
