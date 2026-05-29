package ca

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/authority"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/nosql"
	"github.com/smallstep/nosql/database"
	"golang.org/x/crypto/ocsp"
)

// revokedCertsBucket is the step-ca database bucket for revoked X.509 certificates.
var revokedCertsBucket = []byte("revoked_x509_certs")

// RespondOCSP processes a DER-encoded OCSP request and returns a signed DER OCSP response.
// Certificate status is resolved live against the configured certificate database.
func (e *PKIEngine) RespondOCSP(ctx context.Context, requestDER []byte) ([]byte, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}
	if len(requestDER) == 0 {
		return nil, errors.New("OCSP request body is empty")
	}

	req, err := ocsp.ParseRequest(requestDER)
	if err != nil {
		return nil, fmt.Errorf("parse OCSP request: %w", err)
	}
	if req.SerialNumber == nil {
		return nil, errors.New("OCSP request is missing serial number")
	}

	issuer := e.auth.GetIntermediateCertificate()
	if issuer == nil {
		return nil, errors.New("intermediate CA certificate is not available")
	}
	signer, err := e.auth.GetX509Signer()
	if err != nil {
		return nil, fmt.Errorf("load OCSP signing key: %w", err)
	}

	serial := req.SerialNumber.String()
	status, revokedAt, reason := e.lookupOCSPStatus(serial)
	if !issuerMatchesRequest(issuer, req) {
		status = ocsp.Unknown
		revokedAt = time.Time{}
		reason = 0
	}

	hashAlgo := req.HashAlgorithm
	if hashAlgo == 0 {
		hashAlgo = crypto.SHA1
	}

	template := ocsp.Response{
		Status:       status,
		SerialNumber: req.SerialNumber,
		IssuerHash:   hashAlgo,
		ThisUpdate:   time.Now().UTC(),
		NextUpdate:   time.Now().UTC().Add(24 * time.Hour),
	}
	if status == ocsp.Revoked {
		template.RevokedAt = revokedAt
		template.RevocationReason = reason
	}

	respDER, err := ocsp.CreateResponse(issuer, issuer, template, signer)
	if err != nil {
		return nil, fmt.Errorf("create OCSP response: %w", err)
	}
	return respDER, nil
}

func issuerMatchesRequest(issuer *x509.Certificate, req *ocsp.Request) bool {
	if issuer == nil || req == nil {
		return false
	}
	hashAlgo := req.HashAlgorithm
	if hashAlgo == 0 {
		hashAlgo = crypto.SHA1
	}
	if !hashAlgo.Available() {
		return false
	}

	var publicKeyInfo struct {
		Algorithm asn1.RawValue
		PublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(issuer.RawSubjectPublicKeyInfo, &publicKeyInfo); err != nil {
		return false
	}

	h := hashAlgo.New()
	h.Write(publicKeyInfo.PublicKey.RightAlign())
	if string(h.Sum(nil)) != string(req.IssuerKeyHash) {
		return false
	}

	h.Reset()
	h.Write(issuer.RawSubject)
	return string(h.Sum(nil)) == string(req.IssuerNameHash)
}

func (e *PKIEngine) lookupOCSPStatus(serial string) (status int, revokedAt time.Time, reason int) {
	status = ocsp.Good

	revoked, err := e.auth.IsRevoked(serial)
	if err != nil {
		return ocsp.Unknown, time.Time{}, 0
	}
	if !revoked {
		cert, certErr := e.auth.GetDatabase().GetCertificate(serial)
		if certErr != nil {
			if database.IsErrNotFound(certErr) {
				return ocsp.Unknown, time.Time{}, 0
			}
			return ocsp.Unknown, time.Time{}, 0
		}
		if cert != nil && time.Now().After(cert.NotAfter) {
			return ocsp.Good, time.Time{}, 0
		}
		return ocsp.Good, time.Time{}, 0
	}

	rci, err := e.getRevokedCertificateInfo(serial)
	if err != nil || rci == nil {
		return ocsp.Revoked, time.Now().UTC(), ocsp.Unspecified
	}

	reason = rci.ReasonCode
	if reason < ocsp.Unspecified || reason > ocsp.AACompromise {
		reason = ocsp.Unspecified
	}
	return ocsp.Revoked, rci.RevokedAt.UTC(), reason
}

func (e *PKIEngine) getRevokedCertificateInfo(serial string) (*db.RevokedCertificateInfo, error) {
	store, ok := e.auth.GetDatabase().(nosql.DB)
	if !ok {
		return nil, errors.New("database does not support revocation lookups")
	}

	raw, err := store.Get(revokedCertsBucket, []byte(serial))
	if err != nil {
		if nosql.IsErrNotFound(err) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	var info db.RevokedCertificateInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("decode revoked certificate info: %w", err)
	}
	return &info, nil
}

// GetCRL returns the current certificate revocation list from the authority database.
func (e *PKIEngine) GetCRL(ctx context.Context) (*authority.CertificateRevocationListInfo, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	crlInfo, err := e.auth.GetCertificateRevocationList()
	if err != nil || crlInfo == nil || len(crlInfo.Data) == 0 {
		if genErr := e.auth.GenerateCertificateRevocationList(); genErr != nil {
			return nil, fmt.Errorf("generate CRL: %w", genErr)
		}
		crlInfo, err = e.auth.GetCertificateRevocationList()
		if err != nil {
			return nil, err
		}
	}
	if crlInfo == nil || len(crlInfo.Data) == 0 {
		return nil, errors.New("no CRL available")
	}
	return crlInfo, nil
}

// EncodeCRLPEM wraps DER CRL bytes in PEM encoding.
func EncodeCRLPEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: der,
	})
}

// SerialFromOCSPRequest extracts the certificate serial number from a DER OCSP request.
func SerialFromOCSPRequest(requestDER []byte) (*big.Int, error) {
	req, err := ocsp.ParseRequest(requestDER)
	if err != nil {
		return nil, err
	}
	if req.SerialNumber == nil {
		return nil, errors.New("OCSP request is missing serial number")
	}
	return req.SerialNumber, nil
}
