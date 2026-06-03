package ca

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.step.sm/crypto/jose"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/pemutil"
	"go.step.sm/crypto/randutil"
	"go.step.sm/crypto/x509util"
	"golang.org/x/crypto/ocsp"

	"github.com/smallstep/certificates/authority"
	"github.com/smallstep/certificates/authority/provisioner"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/certificates/errs"
	"github.com/smallstep/nosql/database"

	"github.com/your-org/arx-ca/internal/models"
)

// x509CertsTable is the step-ca database bucket for issued X.509 certificates.
var x509CertsTable = []byte("x509_certs")

// IssueCertificateWithToken validates a provisioner token and signs the provided CSR.
func (e *PKIEngine) IssueCertificateWithToken(ctx context.Context, token, csrPEM, ttl, templateID string, metadata map[string]any) (*models.CertificatePEMResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("token is required")
	}

	csr, err := pemutil.ParseCertificateRequest([]byte(strings.TrimSpace(csrPEM)))
	if err != nil {
		return nil, fmt.Errorf("parse certificate signing request: %w", err)
	}

	signOpts, err := e.buildSignOptions(ttl)
	if err != nil {
		return nil, err
	}

	templateOpts, err := e.templateSignOptions(templateID, metadata, csr, csr.Subject.CommonName)
	if err != nil {
		return nil, err
	}

	chain, err := e.signCSRWithToken(ctx, csr, signOpts, token, templateOpts...)
	if err != nil {
		return nil, err
	}

	return certificateResponse(chain[0]), nil
}

// IssueCertificate signs a PEM-encoded CSR using the intermediate CA via step-ca SignWithContext.
func (e *PKIEngine) IssueCertificate(ctx context.Context, csrPEM, ttl, templateID string, metadata map[string]any) (*models.CertificatePEMResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	csr, err := pemutil.ParseCertificateRequest([]byte(csrPEM))
	if err != nil {
		return nil, fmt.Errorf("parse certificate signing request: %w", err)
	}

	signOpts, err := e.buildSignOptions(ttl)
	if err != nil {
		return nil, err
	}

	templateOpts, err := e.templateSignOptions(templateID, metadata, csr, csr.Subject.CommonName)
	if err != nil {
		return nil, err
	}

	chain, err := e.signCSR(ctx, csr, signOpts, templateOpts...)
	if err != nil {
		return nil, err
	}

	return certificateResponse(chain[0]), nil
}

// AutoCertificate generates an ECDSA P-384 key and CSR in memory, signs the CSR with the
// intermediate CA, and returns both the private key and certificate in PEM format.
func (e *PKIEngine) AutoCertificate(ctx context.Context, req models.AutoCertificateRequest) (*models.AutoCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	cn := strings.TrimSpace(req.CommonName)
	if cn == "" {
		return nil, errors.New("common_name is required")
	}

	sans, err := buildSANs(cn, req.DNSSANs, req.IPSANs)
	if err != nil {
		return nil, err
	}

	signer, err := keyutil.GenerateSigner("EC", "P-384", 0)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	csr, err := x509util.CreateCertificateRequest(cn, sans, signer)
	if err != nil {
		return nil, fmt.Errorf("create certificate signing request: %w", err)
	}

	signOpts, err := e.buildSignOptions(req.TTL)
	if err != nil {
		return nil, err
	}

	templateOpts, err := e.templateSignOptions(req.TemplateID, req.Metadata, csr, cn)
	if err != nil {
		return nil, err
	}

	chain, err := e.signCSR(ctx, csr, signOpts, templateOpts...)
	if err != nil {
		return nil, err
	}

	keyPEM, err := encodePrivateKeyPEM(signer)
	if err != nil {
		return nil, err
	}

	certResp := certificateResponse(chain[0])
	return &models.AutoCertificateResponse{
		CertificatePEM: certResp.CertificatePEM,
		PrivateKeyPEM:  keyPEM,
		Serial:         certResp.Serial,
		NotBefore:      certResp.NotBefore,
		NotAfter:       certResp.NotAfter,
	}, nil
}

// RevokeCertificate revokes a certificate by serial number in the step-ca database (passive revocation).
func (e *PKIEngine) RevokeCertificate(ctx context.Context, serial, reason string, reasonCode int) (*models.RevokeCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	normalizedSerial, err := normalizeSerial(serial)
	if err != nil {
		return nil, err
	}

	if reasonCode < ocsp.Unspecified || reasonCode > ocsp.AACompromise {
		return nil, fmt.Errorf("reason_code must be between %d and %d", ocsp.Unspecified, ocsp.AACompromise)
	}

	cert, err := e.auth.GetDatabase().GetCertificate(normalizedSerial)
	if err != nil {
		return nil, fmt.Errorf("certificate not found")
	}

	ctx = provisioner.NewContextWithMethod(ctx, provisioner.RevokeMethod)
	ctx = authority.NewContext(ctx, e.auth)

	revokeOpts := &authority.RevokeOptions{
		Serial:      normalizedSerial,
		Reason:      reason,
		ReasonCode:  reasonCode,
		PassiveOnly: true,
		MTLS:        true,
		Crt:         cert,
	}

	if err := e.auth.Revoke(ctx, revokeOpts); err != nil {
		return nil, err
	}

	return &models.RevokeCertificateResponse{
		Serial:    normalizedSerial,
		RevokedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ListCertificates returns all issued certificates stored in the step-ca database.
func (e *PKIEngine) ListCertificates(ctx context.Context) (*models.ListCertificatesResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	lister, ok := e.auth.GetDatabase().(interface {
		List(bucket []byte) ([]*database.Entry, error)
	})
	if !ok {
		return nil, errors.New("certificate listing is not supported by the configured database")
	}

	entries, err := lister.List(x509CertsTable)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}

	summaries := make([]models.CertificateSummary, 0, len(entries))
	for _, entry := range entries {
		cert, parseErr := x509.ParseCertificate(entry.Value)
		if parseErr != nil {
			continue
		}

		serial := string(entry.Key)
		if serial == "" {
			serial = cert.SerialNumber.String()
		}

		revoked, _ := e.auth.IsRevoked(serial)

		summary := models.CertificateSummary{
			Serial:    serial,
			Subject:   cert.Subject.String(),
			DNSNames:  append([]string(nil), cert.DNSNames...),
			NotBefore: cert.NotBefore.UTC(),
			NotAfter:  cert.NotAfter.UTC(),
			Revoked:   revoked,
		}

		for _, ip := range cert.IPAddresses {
			summary.IPAddresses = append(summary.IPAddresses, ip.String())
		}

		if reader, ok := e.auth.GetDatabase().(interface {
			GetCertificateData(serialNumber string) (*db.CertificateData, error)
		}); ok {
			if data, dataErr := reader.GetCertificateData(serial); dataErr == nil && data != nil && data.Provisioner != nil {
				summary.ProvisionerID = data.Provisioner.ID
				summary.Provisioner = data.Provisioner.Name
			}
		}

		summaries = append(summaries, summary)
	}

	return &models.ListCertificatesResponse{
		Certificates: summaries,
		Total:        len(summaries),
	}, nil
}

// ListPublicCertificates returns read-only certificate metadata for unauthenticated clients.
func (e *PKIEngine) ListPublicCertificates(ctx context.Context) (*models.PublicListCertificatesResponse, error) {
	list, err := e.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	public := make([]models.PublicCertificateSummary, 0, len(list.Certificates))
	for _, c := range list.Certificates {
		public = append(public, models.PublicCertificateSummary{
			Serial:      c.Serial,
			Subject:     c.Subject,
			DNSNames:    c.DNSNames,
			IPAddresses: c.IPAddresses,
			NotBefore:   c.NotBefore.UTC().Format(time.RFC3339),
			NotAfter:    c.NotAfter.UTC().Format(time.RFC3339),
			Revoked:     c.Revoked,
		})
	}

	return &models.PublicListCertificatesResponse{
		Certificates: public,
		Total:        list.Total,
	}, nil
}

// GetCertificatePEM returns a single issued certificate in PEM encoding by serial number.
func (e *PKIEngine) GetCertificatePEM(ctx context.Context, serial string) (string, error) {
	_ = ctx
	if e == nil || e.auth == nil {
		return "", errors.New("CA engine is not initialized")
	}

	normalizedSerial, err := normalizeSerial(serial)
	if err != nil {
		return "", err
	}

	cert, err := e.auth.GetDatabase().GetCertificate(normalizedSerial)
	if err != nil {
		return "", fmt.Errorf("certificate not found")
	}

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return string(pem.EncodeToMemory(block)), nil
}

func (e *PKIEngine) signCSR(ctx context.Context, csr *x509.CertificateRequest, signOpts provisioner.SignOptions, extraOpts ...provisioner.SignOption) ([]*x509.Certificate, error) {
	sans := collectCSRSubjectAlternativeNames(csr)
	token, _, _, err := e.createProvisionerSignToken(defaultProvisioner, csr.Subject.CommonName, sans, defaultTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("create signing token: %w", err)
	}
	return e.signCSRWithToken(ctx, csr, signOpts, token, extraOpts...)
}

func (e *PKIEngine) signCSRWithToken(ctx context.Context, csr *x509.CertificateRequest, signOpts provisioner.SignOptions, token string, extraOpts ...provisioner.SignOption) ([]*x509.Certificate, error) {
	token, err := e.prepareEnrollmentToken(ctx, csr, token)
	if err != nil {
		return nil, err
	}

	ctx = provisioner.NewContextWithMethod(ctx, provisioner.SignMethod)
	ctx = authority.NewContext(ctx, e.auth)

	authOpts, err := e.auth.Authorize(ctx, token)
	if err != nil {
		return nil, err
	}

	signArgs := make([]provisioner.SignOption, 0, len(authOpts)+len(extraOpts))
	signArgs = append(signArgs, authOpts...)
	signArgs = append(signArgs, extraOpts...)

	chain, err := e.auth.SignWithContext(ctx, csr, signOpts, signArgs...)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return nil, errors.New("signing produced an empty certificate chain")
	}

	return chain, nil
}

func (e *PKIEngine) buildSignOptions(ttl string) (provisioner.SignOptions, error) {
	opts := provisioner.SignOptions{}
	if strings.TrimSpace(ttl) == "" {
		return opts, nil
	}

	notAfter, err := provisioner.ParseTimeDuration(ttl)
	if err != nil {
		return opts, fmt.Errorf("invalid ttl: %w", err)
	}
	opts.NotAfter = notAfter
	return opts, nil
}

func decryptProvisionerKey(kid string, encryptedKey []byte, password []byte) (jose.Signer, error) {
	plaintext, err := jose.Decrypt(
		encryptedKey,
		jose.WithPassword(password),
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt provisioner key: %w", err)
	}

	var jwk jose.JSONWebKey
	if err := json.Unmarshal(plaintext, &jwk); err != nil {
		return nil, fmt.Errorf("parse provisioner key: %w", err)
	}

	signerKey, ok := jwk.Key.(crypto.Signer)
	if !ok {
		return nil, errors.New("provisioner key is not a crypto.Signer")
	}

	opts := new(jose.SignerOptions)
	opts.WithType("JWT")
	opts.WithHeader("kid", kid)

	return jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: signerKey}, opts)
}

func buildProvisionerToken(subject, issuer, audience string, sans []string, signer jose.Signer, tokenTTL time.Duration) (string, error) {
	if tokenTTL <= 0 {
		tokenTTL = defaultTokenTTL
	}

	id, err := randutil.ASCII(64)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claims := struct {
		jose.Claims
		SANS []string `json:"sans,omitempty"`
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
		SANS: sans,
	}

	return jose.Signed(signer).Claims(claims).CompactSerialize()
}

func buildSANs(cn string, dnsSANs, ipSANs []string) ([]string, error) {
	seen := make(map[string]struct{})
	out := make([]string, 0, 1+len(dnsSANs)+len(ipSANs))

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, exists := seen[s]; exists {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	add(cn)
	for _, name := range dnsSANs {
		add(name)
	}
	for _, ip := range ipSANs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if parsed := net.ParseIP(ip); parsed == nil {
			return nil, fmt.Errorf("invalid ip_sans entry %q", ip)
		}
		add(ip)
	}

	return out, nil
}

func collectCSRSubjectAlternativeNames(csr *x509.CertificateRequest) []string {
	if csr == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var sans []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, exists := seen[s]; exists {
			return
		}
		seen[s] = struct{}{}
		sans = append(sans, s)
	}

	add(csr.Subject.CommonName)
	for _, name := range csr.DNSNames {
		add(name)
	}
	for _, ip := range csr.IPAddresses {
		add(ip.String())
	}
	for _, email := range csr.EmailAddresses {
		add(email)
	}
	for _, uri := range csr.URIs {
		if uri != nil {
			add(uri.String())
		}
	}

	return sans
}

func certificateResponse(cert *x509.Certificate) *models.CertificatePEMResponse {
	return &models.CertificatePEMResponse{
		CertificatePEM: string(encodeCertificatePEM(cert)),
		Serial:         cert.SerialNumber.String(),
		NotBefore:      cert.NotBefore.UTC().Format(time.RFC3339),
		NotAfter:       cert.NotAfter.UTC().Format(time.RFC3339),
	}
}

func encodePrivateKeyPEM(signer crypto.Signer) (string, error) {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(signer)
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

func normalizeSerial(serial string) (string, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return "", errors.New("serial is required")
	}

	sn, ok := new(big.Int).SetString(serial, 0)
	if !ok {
		return "", errors.New("serial is not a valid number")
	}
	return sn.String(), nil
}

// MapCAError converts step-ca authority errors into HTTP status codes and client-safe messages.
func MapCAError(err error) (status int, message string) {
	if err == nil {
		return http.StatusInternalServerError, "unknown error"
	}

	var stepErr *errs.Error
	if errors.As(err, &stepErr) {
		return stepErr.StatusCode(), stepErr.Error()
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"), strings.Contains(msg, "certificate not found"):
		return http.StatusNotFound, "certificate not found"
	case strings.Contains(msg, "already revoked"):
		return http.StatusConflict, "certificate is already revoked"
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "malformed"), strings.Contains(msg, "parse"):
		return http.StatusBadRequest, msg
	case strings.Contains(msg, "forbidden"), strings.Contains(msg, "not allowed"):
		return http.StatusForbidden, msg
	case strings.Contains(msg, "unauthorized"):
		return http.StatusUnauthorized, msg
	default:
		return http.StatusInternalServerError, "certificate operation failed"
	}
}
