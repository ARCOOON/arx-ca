package acmeprotocol

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.step.sm/crypto/jose"

	stepacme "github.com/smallstep/certificates/acme"
)

// Validator performs RFC 8555 / RFC 8737 challenge validation with explicit state transitions.
type Validator struct {
	HTTP *http.Client
	TLS  TLSDialer
}

// NewValidator returns a challenge validator using default outbound clients.
func NewValidator() *Validator {
	c := NewChallengeClient()
	return &Validator{
		HTTP: c.http,
		TLS:  defaultTLSDialer{},
	}
}

// ValidateChallenge validates an ACME challenge and persists status transitions:
// pending -> processing -> valid | invalid.
func (v *Validator) ValidateChallenge(ctx context.Context, db stepacme.DB, ch *stepacme.Challenge, jwk *jose.JSONWebKey) error {
	if ch.Status != stepacme.StatusPending && ch.Status != StatusProcessing {
		return nil
	}

	if err := setChallengeStatus(ctx, db, ch, StatusProcessing, nil); err != nil {
		return err
	}

	keyAuth, err := KeyAuthorization(ch.Token, jwk)
	if err != nil {
		return storeChallengeError(ctx, db, ch, true, err)
	}

	var validateErr error
	switch ch.Type {
	case stepacme.HTTP01:
		validateErr = VerifyHTTP01(ctx, v.HTTP, ch.Value, ch.Token, keyAuth)
	case stepacme.DNS01:
		validateErr = VerifyDNS01(ctx, ch.Value, keyAuth)
	case stepacme.TLSALPN01:
		validateErr = VerifyTLSALPN01(ctx, v.TLS, ch.Value, keyAuth)
	default:
		validateErr = fmt.Errorf("unsupported challenge type %q", ch.Type)
	}

	if validateErr != nil {
		acmeErr := stepacme.NewError(stepacme.ErrorConnectionType, "%s", validateErr.Error())
		if ch.Type == stepacme.DNS01 {
			acmeErr = stepacme.NewError(stepacme.ErrorDNSType, "%s", validateErr.Error())
		}
		markInvalid := false
		if isRejectedIdentifier(validateErr) {
			acmeErr = stepacme.NewError(stepacme.ErrorRejectedIdentifierType, "%s", validateErr.Error())
			markInvalid = true
		}
		return storeChallengeError(ctx, db, ch, markInvalid, acmeErr)
	}

	ch.Status = stepacme.StatusValid
	ch.Error = nil
	ch.ValidatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := db.UpdateChallenge(ctx, ch); err != nil {
		return fmt.Errorf("update challenge: %w", err)
	}
	return nil
}

func setChallengeStatus(ctx context.Context, db stepacme.DB, ch *stepacme.Challenge, status stepacme.Status, acmeErr *stepacme.Error) error {
	ch.Status = status
	ch.Error = acmeErr
	return db.UpdateChallenge(ctx, ch)
}

func storeChallengeError(ctx context.Context, db stepacme.DB, ch *stepacme.Challenge, markInvalid bool, err error) error {
	var acmeErr *stepacme.Error
	switch e := err.(type) {
	case *stepacme.Error:
		acmeErr = e
	default:
		acmeErr = stepacme.NewError(stepacme.ErrorConnectionType, "%s", err.Error())
	}
	ch.Error = acmeErr
	switch {
	case markInvalid:
		ch.Status = stepacme.StatusInvalid
	default:
		ch.Status = stepacme.StatusPending
	}
	if updateErr := db.UpdateChallenge(ctx, ch); updateErr != nil {
		return fmt.Errorf("persist challenge error: %w", updateErr)
	}
	return nil
}

func isRejectedIdentifier(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, sub := range []string{"mismatch", "not found", "malformed", "incorrect", "missing", "obsolete", "ALPN"} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}
