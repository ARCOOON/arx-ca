package database

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	stepacme "github.com/smallstep/certificates/acme"
	"go.step.sm/crypto/jose"
	"go.step.sm/crypto/randutil"
)

const acmeIDLength = 32

// ACMEStore implements step-ca's acme.DB interface using the application database.
type ACMEStore struct {
	db *sql.DB

	accountMu sync.Mutex
	eabMu     sync.RWMutex
	ordersMu  sync.Mutex
}

// NewACMEStore returns an ACME persistence layer backed by SQLite or PostgreSQL.
func NewACMEStore(db *sql.DB) *ACMEStore {
	return &ACMEStore{db: db}
}

func acmeNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func acmeRandID() (string, error) {
	id, err := randutil.Alphanumeric(acmeIDLength)
	if err != nil {
		return "", errors.Wrap(err, "generate acme id")
	}
	return id, nil
}

func (s *ACMEStore) ph(qSQLite, qPostgres string) string {
	if isPostgreSQL(s.db) {
		return qPostgres
	}
	return qSQLite
}

type dbAccount struct {
	ID              string           `json:"id"`
	Key             *jose.JSONWebKey `json:"key"`
	Contact         []string         `json:"contact,omitempty"`
	Status          stepacme.Status  `json:"status"`
	LocationPrefix  string           `json:"locationPrefix"`
	ProvisionerID   string           `json:"provisionerID,omitempty"`
	ProvisionerName string           `json:"provisionerName"`
	CreatedAt       time.Time        `json:"createdAt"`
	DeactivatedAt   time.Time        `json:"deactivatedAt"`
}

type dbOrder struct {
	ID               string                `json:"id"`
	AccountID        string                `json:"accountID"`
	ProvisionerID    string                `json:"provisionerID"`
	Identifiers      []stepacme.Identifier `json:"identifiers"`
	AuthorizationIDs []string              `json:"authorizationIDs"`
	Status           stepacme.Status       `json:"status"`
	NotBefore        time.Time             `json:"notBefore,omitempty"`
	NotAfter         time.Time             `json:"notAfter,omitempty"`
	CreatedAt        time.Time             `json:"createdAt"`
	ExpiresAt        time.Time             `json:"expiresAt,omitempty"`
	CertificateID    string                `json:"certificate,omitempty"`
	Error            *stepacme.Error       `json:"error,omitempty"`
}

type dbAuthz struct {
	ID           string              `json:"id"`
	AccountID    string              `json:"accountID"`
	Identifier   stepacme.Identifier `json:"identifier"`
	Status       stepacme.Status     `json:"status"`
	Token        string              `json:"token"`
	Fingerprint  string              `json:"fingerprint,omitempty"`
	ChallengeIDs []string            `json:"challengeIDs"`
	Wildcard     bool                `json:"wildcard"`
	CreatedAt    time.Time           `json:"createdAt"`
	ExpiresAt    time.Time           `json:"expiresAt"`
	Error        *stepacme.Error     `json:"error"`
}

type dbChallenge struct {
	ID          string                 `json:"id"`
	AccountID   string                 `json:"accountID"`
	Type        stepacme.ChallengeType `json:"type"`
	Status      stepacme.Status        `json:"status"`
	Token       string                 `json:"token"`
	Value       string                 `json:"value"`
	Target      string                 `json:"target,omitempty"`
	ValidatedAt string                 `json:"validatedAt"`
	CreatedAt   time.Time              `json:"createdAt"`
	Error       *stepacme.Error        `json:"error"`
}

type dbCert struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	AccountID     string    `json:"accountID"`
	OrderID       string    `json:"orderID"`
	Leaf          []byte    `json:"leaf"`
	Intermediates []byte    `json:"intermediates"`
}

type dbExternalAccountKey struct {
	ID            string    `json:"id"`
	ProvisionerID string    `json:"provisionerID"`
	Reference     string    `json:"reference"`
	AccountID     string    `json:"accountID,omitempty"`
	HmacKey       []byte    `json:"key"`
	CreatedAt     time.Time `json:"createdAt"`
	BoundAt       time.Time `json:"boundAt"`
}

func (s *ACMEStore) marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal acme record: %w", err)
	}
	return b, nil
}

func (s *ACMEStore) getAccountRecord(ctx context.Context, id string) (*dbAccount, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_accounts WHERE id = ?`,
		`SELECT data FROM acme_accounts WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load acme account %s: %w", id, err)
	}
	var acc dbAccount
	if err := json.Unmarshal([]byte(data), &acc); err != nil {
		return nil, fmt.Errorf("unmarshal acme account %s: %w", id, err)
	}
	return &acc, nil
}

// CreateAccount stores a new ACME account.
func (s *ACMEStore) CreateAccount(ctx context.Context, acc *stepacme.Account) error {
	var err error
	acc.ID, err = acmeRandID()
	if err != nil {
		return err
	}

	kid, err := stepacme.KeyToID(acc.Key)
	if err != nil {
		return err
	}

	dba := &dbAccount{
		ID:              acc.ID,
		Key:             acc.Key,
		Contact:         acc.Contact,
		Status:          acc.Status,
		CreatedAt:       acmeNow(),
		LocationPrefix:  acc.LocationPrefix,
		ProvisionerID:   acc.ProvisionerID,
		ProvisionerName: acc.ProvisionerName,
	}
	data, err := s.marshal(dba)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if isPostgreSQL(s.db) {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_accounts (id, key_id, data) VALUES ($1, $2, $3)`,
			acc.ID, kid, string(data),
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_accounts (id, key_id, data) VALUES (?, ?, ?)`,
			acc.ID, kid, string(data),
		)
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return errors.New("key-id to account-id index already exists")
		}
		return fmt.Errorf("insert acme account: %w", err)
	}
	return tx.Commit()
}

// GetAccount returns an ACME account by ID.
func (s *ACMEStore) GetAccount(ctx context.Context, id string) (*stepacme.Account, error) {
	dba, err := s.getAccountRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return &stepacme.Account{
		Status:          dba.Status,
		Contact:         dba.Contact,
		Key:             dba.Key,
		ID:              dba.ID,
		LocationPrefix:  dba.LocationPrefix,
		ProvisionerID:   dba.ProvisionerID,
		ProvisionerName: dba.ProvisionerName,
	}, nil
}

// GetAccountByKeyID returns an ACME account by JWK thumbprint.
func (s *ACMEStore) GetAccountByKeyID(ctx context.Context, kid string) (*stepacme.Account, error) {
	var id string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT id FROM acme_accounts WHERE key_id = ?`,
		`SELECT id FROM acme_accounts WHERE key_id = $1`,
	), kid).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, stepacme.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup acme account by key id: %w", err)
	}
	return s.GetAccount(ctx, id)
}

// UpdateAccount updates an ACME account.
func (s *ACMEStore) UpdateAccount(ctx context.Context, acc *stepacme.Account) error {
	s.accountMu.Lock()
	defer s.accountMu.Unlock()

	old, err := s.getAccountRecord(ctx, acc.ID)
	if err != nil {
		return err
	}

	nu := *old
	nu.Contact = acc.Contact
	nu.Status = acc.Status
	if acc.Status == stepacme.StatusDeactivated && old.Status != stepacme.StatusDeactivated {
		nu.DeactivatedAt = acmeNow()
	}

	data, err := s.marshal(&nu)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, s.ph(
		`UPDATE acme_accounts SET data = ? WHERE id = ?`,
		`UPDATE acme_accounts SET data = $1 WHERE id = $2`,
	), string(data), acc.ID)
	if err != nil {
		return fmt.Errorf("update acme account: %w", err)
	}
	return nil
}

// CreateNonce creates a single-use ACME replay nonce.
func (s *ACMEStore) CreateNonce(ctx context.Context) (stepacme.Nonce, error) {
	raw, err := acmeRandID()
	if err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(raw))
	createdAt := acmeNow().Format(time.RFC3339)

	_, err = s.db.ExecContext(ctx, s.ph(
		`INSERT INTO acme_nonces (id, created_at) VALUES (?, ?)`,
		`INSERT INTO acme_nonces (id, created_at) VALUES ($1, $2)`,
	), id, createdAt)
	if err != nil {
		return "", fmt.Errorf("insert acme nonce: %w", err)
	}
	return stepacme.Nonce(id), nil
}

// DeleteNonce consumes a replay nonce.
func (s *ACMEStore) DeleteNonce(ctx context.Context, nonce stepacme.Nonce) error {
	res, err := s.db.ExecContext(ctx, s.ph(
		`DELETE FROM acme_nonces WHERE id = ?`,
		`DELETE FROM acme_nonces WHERE id = $1`,
	), string(nonce))
	if err != nil {
		return fmt.Errorf("delete acme nonce: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return stepacme.NewError(stepacme.ErrorBadNonceType, "nonce %s not found", string(nonce))
	}
	return nil
}

func (s *ACMEStore) getOrderRecord(ctx context.Context, id string) (*dbOrder, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_orders WHERE id = ?`,
		`SELECT data FROM acme_orders WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.NewError(stepacme.ErrorMalformedType, "order %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("load acme order %s: %w", id, err)
	}
	var o dbOrder
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return nil, fmt.Errorf("unmarshal acme order %s: %w", id, err)
	}
	return &o, nil
}

// CreateOrder stores a new ACME order.
func (s *ACMEStore) CreateOrder(ctx context.Context, o *stepacme.Order) error {
	var err error
	o.ID, err = acmeRandID()
	if err != nil {
		return err
	}

	dbo := &dbOrder{
		ID:               o.ID,
		AccountID:        o.AccountID,
		ProvisionerID:    o.ProvisionerID,
		Status:           o.Status,
		CreatedAt:        acmeNow(),
		ExpiresAt:        o.ExpiresAt,
		Identifiers:      o.Identifiers,
		NotBefore:        o.NotBefore,
		NotAfter:         o.NotAfter,
		AuthorizationIDs: o.AuthorizationIDs,
	}
	data, err := s.marshal(dbo)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if isPostgreSQL(s.db) {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_orders (id, account_id, data) VALUES ($1, $2, $3)`,
			o.ID, o.AccountID, string(data),
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_orders (id, account_id, data) VALUES (?, ?, ?)`,
			o.ID, o.AccountID, string(data),
		)
	}
	if err != nil {
		return fmt.Errorf("insert acme order: %w", err)
	}
	if _, err := s.updateAccountOrdersTx(ctx, tx, o.AccountID, false, o.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// GetOrder returns an ACME order by ID.
func (s *ACMEStore) GetOrder(ctx context.Context, id string) (*stepacme.Order, error) {
	dbo, err := s.getOrderRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return &stepacme.Order{
		ID:               dbo.ID,
		AccountID:        dbo.AccountID,
		ProvisionerID:    dbo.ProvisionerID,
		CertificateID:    dbo.CertificateID,
		Status:           dbo.Status,
		ExpiresAt:        dbo.ExpiresAt,
		Identifiers:      dbo.Identifiers,
		NotBefore:        dbo.NotBefore,
		NotAfter:         dbo.NotAfter,
		AuthorizationIDs: dbo.AuthorizationIDs,
		Error:            dbo.Error,
	}, nil
}

// UpdateOrder updates an ACME order.
func (s *ACMEStore) UpdateOrder(ctx context.Context, o *stepacme.Order) error {
	old, err := s.getOrderRecord(ctx, o.ID)
	if err != nil {
		return err
	}
	nu := *old
	nu.Status = o.Status
	nu.Error = o.Error
	nu.CertificateID = o.CertificateID
	data, err := s.marshal(&nu)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`UPDATE acme_orders SET data = ? WHERE id = ?`,
		`UPDATE acme_orders SET data = $1 WHERE id = $2`,
	), string(data), o.ID)
	if err != nil {
		return fmt.Errorf("update acme order: %w", err)
	}
	return nil
}

func (s *ACMEStore) loadAccountOrderIDs(ctx context.Context, accountID string) ([]string, error) {
	var raw sql.NullString
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT order_ids FROM acme_account_orders WHERE account_id = ?`,
		`SELECT order_ids FROM acme_account_orders WHERE account_id = $1`,
	), accountID).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw.String), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *ACMEStore) saveAccountOrderIDs(ctx context.Context, accountID string, ids []string) error {
	payload, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	if isPostgreSQL(s.db) {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO acme_account_orders (account_id, order_ids) VALUES ($1, $2)
			ON CONFLICT (account_id) DO UPDATE SET order_ids = EXCLUDED.order_ids`,
			accountID, string(payload),
		)
	} else {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO acme_account_orders (account_id, order_ids) VALUES (?, ?)
			ON CONFLICT(account_id) DO UPDATE SET order_ids = excluded.order_ids`,
			accountID, string(payload),
		)
	}
	return err
}

func (s *ACMEStore) updateAccountOrdersTx(ctx context.Context, tx *sql.Tx, accountID string, includeReady bool, addIDs ...string) ([]string, error) {
	s.ordersMu.Lock()
	defer s.ordersMu.Unlock()

	var raw sql.NullString
	var err error
	if isPostgreSQL(s.db) {
		err = tx.QueryRowContext(ctx,
			`SELECT order_ids FROM acme_account_orders WHERE account_id = $1 FOR UPDATE`,
			accountID,
		).Scan(&raw)
	} else {
		err = tx.QueryRowContext(ctx,
			`SELECT order_ids FROM acme_account_orders WHERE account_id = ?`,
			accountID,
		).Scan(&raw)
	}
	oldIDs := []string{}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && raw.Valid && raw.String != "" {
		if err := json.Unmarshal([]byte(raw.String), &oldIDs); err != nil {
			return nil, err
		}
	}

	pending := make([]string, 0, len(oldIDs)+len(addIDs))
	for _, oid := range oldIDs {
		order, err := s.GetOrder(ctx, oid)
		if err != nil {
			return nil, stepacme.WrapErrorISE(err, "load order %s for account %s", oid, accountID)
		}
		if err = order.UpdateStatus(ctx, s); err != nil {
			return nil, stepacme.WrapErrorISE(err, "update order %s for account %s", oid, accountID)
		}
		if order.Status == stepacme.StatusPending || (order.Status == stepacme.StatusReady && includeReady) {
			pending = append(pending, oid)
		}
	}
	pending = append(pending, addIDs...)

	payload, err := json.Marshal(pending)
	if err != nil {
		return nil, err
	}
	if isPostgreSQL(s.db) {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO acme_account_orders (account_id, order_ids) VALUES ($1, $2)
			ON CONFLICT (account_id) DO UPDATE SET order_ids = EXCLUDED.order_ids`,
			accountID, string(payload),
		)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO acme_account_orders (account_id, order_ids) VALUES (?, ?)
			ON CONFLICT(account_id) DO UPDATE SET order_ids = excluded.order_ids`,
			accountID, string(payload),
		)
	}
	if err != nil {
		for _, oid := range addIDs {
			_, _ = tx.ExecContext(ctx, s.ph(`DELETE FROM acme_orders WHERE id = ?`, `DELETE FROM acme_orders WHERE id = $1`), oid)
		}
		return nil, fmt.Errorf("save account order index: %w", err)
	}
	return pending, nil
}

// GetOrdersByAccountID returns pending order IDs for an account.
func (s *ACMEStore) GetOrdersByAccountID(ctx context.Context, accountID string) ([]string, error) {
	s.ordersMu.Lock()
	defer s.ordersMu.Unlock()

	ids, err := s.loadAccountOrderIDs(ctx, accountID)
	if err != nil {
		return nil, err
	}
	pending := make([]string, 0, len(ids))
	for _, oid := range ids {
		order, err := s.GetOrder(ctx, oid)
		if err != nil {
			return nil, err
		}
		if err = order.UpdateStatus(ctx, s); err != nil {
			return nil, err
		}
		if order.Status == stepacme.StatusPending {
			pending = append(pending, oid)
		}
	}
	if err := s.saveAccountOrderIDs(ctx, accountID, pending); err != nil {
		return nil, err
	}
	return pending, nil
}

func (s *ACMEStore) getAuthzRecord(ctx context.Context, id string) (*dbAuthz, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_authorizations WHERE id = ?`,
		`SELECT data FROM acme_authorizations WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.NewError(stepacme.ErrorMalformedType, "authz %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	var az dbAuthz
	if err := json.Unmarshal([]byte(data), &az); err != nil {
		return nil, err
	}
	return &az, nil
}

// CreateAuthorization stores a new authorization.
func (s *ACMEStore) CreateAuthorization(ctx context.Context, az *stepacme.Authorization) error {
	var err error
	az.ID, err = acmeRandID()
	if err != nil {
		return err
	}
	chIDs := make([]string, len(az.Challenges))
	for i, ch := range az.Challenges {
		chIDs[i] = ch.ID
	}
	dbaz := &dbAuthz{
		ID:           az.ID,
		AccountID:    az.AccountID,
		Status:       az.Status,
		CreatedAt:    acmeNow(),
		ExpiresAt:    az.ExpiresAt,
		Identifier:   az.Identifier,
		ChallengeIDs: chIDs,
		Token:        az.Token,
		Fingerprint:  az.Fingerprint,
		Wildcard:     az.Wildcard,
	}
	data, err := s.marshal(dbaz)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`INSERT INTO acme_authorizations (id, account_id, data) VALUES (?, ?, ?)`,
		`INSERT INTO acme_authorizations (id, account_id, data) VALUES ($1, $2, $3)`,
	), az.ID, az.AccountID, string(data))
	return err
}

// GetAuthorization returns an authorization with challenges.
func (s *ACMEStore) GetAuthorization(ctx context.Context, id string) (*stepacme.Authorization, error) {
	dbaz, err := s.getAuthzRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	chs := make([]*stepacme.Challenge, len(dbaz.ChallengeIDs))
	for i, chID := range dbaz.ChallengeIDs {
		chs[i], err = s.GetChallenge(ctx, chID, id)
		if err != nil {
			return nil, err
		}
	}
	return &stepacme.Authorization{
		ID:          dbaz.ID,
		AccountID:   dbaz.AccountID,
		Identifier:  dbaz.Identifier,
		Status:      dbaz.Status,
		Challenges:  chs,
		Wildcard:    dbaz.Wildcard,
		ExpiresAt:   dbaz.ExpiresAt,
		Token:       dbaz.Token,
		Fingerprint: dbaz.Fingerprint,
		Error:       dbaz.Error,
	}, nil
}

// UpdateAuthorization updates an authorization.
func (s *ACMEStore) UpdateAuthorization(ctx context.Context, az *stepacme.Authorization) error {
	old, err := s.getAuthzRecord(ctx, az.ID)
	if err != nil {
		return err
	}
	nu := *old
	nu.Status = az.Status
	nu.Fingerprint = az.Fingerprint
	nu.Error = az.Error
	data, err := s.marshal(&nu)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`UPDATE acme_authorizations SET data = ? WHERE id = ?`,
		`UPDATE acme_authorizations SET data = $1 WHERE id = $2`,
	), string(data), az.ID)
	return err
}

// GetAuthorizationsByAccountID lists authorizations for an account.
func (s *ACMEStore) GetAuthorizationsByAccountID(ctx context.Context, accountID string) ([]*stepacme.Authorization, error) {
	rows, err := s.db.QueryContext(ctx, s.ph(
		`SELECT data FROM acme_authorizations WHERE account_id = ?`,
		`SELECT data FROM acme_authorizations WHERE account_id = $1`,
	), accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*stepacme.Authorization
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var dbaz dbAuthz
		if err := json.Unmarshal([]byte(data), &dbaz); err != nil {
			return nil, err
		}
		out = append(out, &stepacme.Authorization{
			ID:          dbaz.ID,
			AccountID:   dbaz.AccountID,
			Identifier:  dbaz.Identifier,
			Status:      dbaz.Status,
			Wildcard:    dbaz.Wildcard,
			ExpiresAt:   dbaz.ExpiresAt,
			Token:       dbaz.Token,
			Fingerprint: dbaz.Fingerprint,
			Error:       dbaz.Error,
		})
	}
	return out, rows.Err()
}

func (s *ACMEStore) getChallengeRecord(ctx context.Context, id string) (*dbChallenge, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_challenges WHERE id = ?`,
		`SELECT data FROM acme_challenges WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.NewError(stepacme.ErrorMalformedType, "challenge %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	var ch dbChallenge
	if err := json.Unmarshal([]byte(data), &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// CreateChallenge stores a new challenge.
func (s *ACMEStore) CreateChallenge(ctx context.Context, ch *stepacme.Challenge) error {
	var err error
	ch.ID, err = acmeRandID()
	if err != nil {
		return err
	}
	dbch := &dbChallenge{
		ID:        ch.ID,
		AccountID: ch.AccountID,
		Value:     ch.Value,
		Status:    stepacme.StatusPending,
		Token:     ch.Token,
		CreatedAt: acmeNow(),
		Type:      ch.Type,
		Target:    ch.Target,
	}
	data, err := s.marshal(dbch)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`INSERT INTO acme_challenges (id, authz_id, data) VALUES (?, ?, ?)`,
		`INSERT INTO acme_challenges (id, authz_id, data) VALUES ($1, $2, $3)`,
	), ch.ID, "", string(data))
	return err
}

// GetChallenge returns a challenge by ID.
func (s *ACMEStore) GetChallenge(ctx context.Context, id, authzID string) (*stepacme.Challenge, error) {
	_ = authzID
	dbch, err := s.getChallengeRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return &stepacme.Challenge{
		ID:          dbch.ID,
		AccountID:   dbch.AccountID,
		Type:        dbch.Type,
		Value:       dbch.Value,
		Status:      dbch.Status,
		Token:       dbch.Token,
		Error:       dbch.Error,
		ValidatedAt: dbch.ValidatedAt,
		Target:      dbch.Target,
	}, nil
}

// UpdateChallenge updates a challenge.
func (s *ACMEStore) UpdateChallenge(ctx context.Context, ch *stepacme.Challenge) error {
	old, err := s.getChallengeRecord(ctx, ch.ID)
	if err != nil {
		return err
	}
	nu := *old
	nu.Status = ch.Status
	nu.Error = ch.Error
	nu.ValidatedAt = ch.ValidatedAt
	data, err := s.marshal(&nu)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`UPDATE acme_challenges SET data = ? WHERE id = ?`,
		`UPDATE acme_challenges SET data = $1 WHERE id = $2`,
	), string(data), ch.ID)
	return err
}

// CreateCertificate stores an issued certificate chain.
func (s *ACMEStore) CreateCertificate(ctx context.Context, cert *stepacme.Certificate) error {
	var err error
	cert.ID, err = acmeRandID()
	if err != nil {
		return err
	}

	leaf := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Leaf.Raw})
	var intermediates []byte
	for _, c := range cert.Intermediates {
		intermediates = append(intermediates, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})...)
	}

	dbch := &dbCert{
		ID:            cert.ID,
		AccountID:     cert.AccountID,
		OrderID:       cert.OrderID,
		Leaf:          leaf,
		Intermediates: intermediates,
		CreatedAt:     time.Now().UTC(),
	}
	data, err := s.marshal(dbch)
	if err != nil {
		return err
	}
	serial := cert.Leaf.SerialNumber.String()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if isPostgreSQL(s.db) {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_certificates (id, serial, data) VALUES ($1, $2, $3)`,
			cert.ID, serial, string(data),
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acme_certificates (id, serial, data) VALUES (?, ?, ?)`,
			cert.ID, serial, string(data),
		)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetCertificate returns a stored certificate by ID.
func (s *ACMEStore) GetCertificate(ctx context.Context, id string) (*stepacme.Certificate, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_certificates WHERE id = ?`,
		`SELECT data FROM acme_certificates WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.NewError(stepacme.ErrorMalformedType, "certificate %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	var dbC dbCert
	if err := json.Unmarshal([]byte(data), &dbC); err != nil {
		return nil, err
	}
	certs, err := parseACMECertBundle(append(dbC.Leaf, dbC.Intermediates...))
	if err != nil {
		return nil, err
	}
	return &stepacme.Certificate{
		ID:            dbC.ID,
		AccountID:     dbC.AccountID,
		OrderID:       dbC.OrderID,
		Leaf:          certs[0],
		Intermediates: certs[1:],
	}, nil
}

// GetCertificateBySerial returns a certificate by serial number.
func (s *ACMEStore) GetCertificateBySerial(ctx context.Context, serial string) (*stepacme.Certificate, error) {
	var id string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT id FROM acme_certificates WHERE serial = ?`,
		`SELECT id FROM acme_certificates WHERE serial = $1`,
	), serial).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, stepacme.NewError(stepacme.ErrorMalformedType, "certificate with serial %s not found", serial)
	}
	if err != nil {
		return nil, err
	}
	return s.GetCertificate(ctx, id)
}

func parseACMECertBundle(b []byte) ([]*x509.Certificate, error) {
	var bundle []*x509.Certificate
	for len(b) > 0 {
		block, rest := pem.Decode(b)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return nil, errors.New("pem block is not a certificate")
		}
		crt, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		bundle = append(bundle, crt)
		b = rest
	}
	if len(b) > 0 {
		return nil, errors.New("unexpected trailing pem data")
	}
	return bundle, nil
}

func eabReferenceKey(provisionerID, reference string) string {
	return provisionerID + ":" + reference
}

func (s *ACMEStore) getEABRecord(ctx context.Context, id string) (*dbExternalAccountKey, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT data FROM acme_eab_keys WHERE id = ?`,
		`SELECT data FROM acme_eab_keys WHERE id = $1`,
	), id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, stepacme.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var eak dbExternalAccountKey
	if err := json.Unmarshal([]byte(data), &eak); err != nil {
		return nil, err
	}
	return &eak, nil
}

func (s *ACMEStore) loadProvisionerEABIDs(ctx context.Context, provisionerID string) ([]string, error) {
	var raw sql.NullString
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT key_ids FROM acme_eab_provisioner_index WHERE provisioner_id = ?`,
		`SELECT key_ids FROM acme_eab_provisioner_index WHERE provisioner_id = $1`,
	), provisionerID).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !raw.Valid {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw.String), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *ACMEStore) saveProvisionerEABIDs(ctx context.Context, provisionerID string, ids []string) error {
	payload, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	if isPostgreSQL(s.db) {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO acme_eab_provisioner_index (provisioner_id, key_ids) VALUES ($1, $2)
			ON CONFLICT (provisioner_id) DO UPDATE SET key_ids = EXCLUDED.key_ids`,
			provisionerID, string(payload),
		)
	} else {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO acme_eab_provisioner_index (provisioner_id, key_ids) VALUES (?, ?)
			ON CONFLICT(provisioner_id) DO UPDATE SET key_ids = excluded.key_ids`,
			provisionerID, string(payload),
		)
	}
	return err
}

// CreateExternalAccountKey mints an EAB credential.
func (s *ACMEStore) CreateExternalAccountKey(ctx context.Context, provisionerID, reference string) (*stepacme.ExternalAccountKey, error) {
	s.eabMu.Lock()
	defer s.eabMu.Unlock()

	keyID, err := acmeRandID()
	if err != nil {
		return nil, err
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}

	dbeak := &dbExternalAccountKey{
		ID:            keyID,
		ProvisionerID: provisionerID,
		Reference:     reference,
		HmacKey:       random,
		CreatedAt:     acmeNow(),
	}
	data, err := s.marshal(dbeak)
	if err != nil {
		return nil, err
	}

	refKey := ""
	if reference != "" {
		refKey = eabReferenceKey(provisionerID, reference)
	}

	if isPostgreSQL(s.db) {
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO acme_eab_keys (id, provisioner_id, reference_key, data) VALUES ($1, $2, $3, $4)`,
			keyID, provisionerID, nullIfEmpty(refKey), string(data),
		)
	} else {
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO acme_eab_keys (id, provisioner_id, reference_key, data) VALUES (?, ?, ?, ?)`,
			keyID, provisionerID, nullIfEmpty(refKey), string(data),
		)
	}
	if err != nil {
		return nil, err
	}

	ids, err := s.loadProvisionerEABIDs(ctx, provisionerID)
	if err != nil {
		return nil, err
	}
	ids = append(ids, keyID)
	if err := s.saveProvisionerEABIDs(ctx, provisionerID, ids); err != nil {
		return nil, err
	}

	return &stepacme.ExternalAccountKey{
		ID:            dbeak.ID,
		ProvisionerID: dbeak.ProvisionerID,
		Reference:     dbeak.Reference,
		AccountID:     dbeak.AccountID,
		HmacKey:       dbeak.HmacKey,
		CreatedAt:     dbeak.CreatedAt,
		BoundAt:       dbeak.BoundAt,
	}, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetExternalAccountKey returns an EAB key by ID.
func (s *ACMEStore) GetExternalAccountKey(ctx context.Context, provisionerID, keyID string) (*stepacme.ExternalAccountKey, error) {
	s.eabMu.RLock()
	defer s.eabMu.RUnlock()

	dbeak, err := s.getEABRecord(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if dbeak.ProvisionerID != provisionerID {
		return nil, stepacme.NewError(stepacme.ErrorUnauthorizedType, "provisioner does not match provisioner for which the EAB key was created")
	}
	return eabFromDB(dbeak), nil
}

func eabFromDB(dbeak *dbExternalAccountKey) *stepacme.ExternalAccountKey {
	return &stepacme.ExternalAccountKey{
		ID:            dbeak.ID,
		ProvisionerID: dbeak.ProvisionerID,
		Reference:     dbeak.Reference,
		AccountID:     dbeak.AccountID,
		HmacKey:       dbeak.HmacKey,
		CreatedAt:     dbeak.CreatedAt,
		BoundAt:       dbeak.BoundAt,
	}
}

// GetExternalAccountKeys lists EAB keys for a provisioner.
func (s *ACMEStore) GetExternalAccountKeys(ctx context.Context, provisionerID, cursor string, limit int) ([]*stepacme.ExternalAccountKey, string, error) {
	_, _ = cursor, limit
	s.eabMu.RLock()
	defer s.eabMu.RUnlock()

	ids, err := s.loadProvisionerEABIDs(ctx, provisionerID)
	if err != nil {
		return nil, "", err
	}
	keys := make([]*stepacme.ExternalAccountKey, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		eak, err := s.getEABRecord(ctx, id)
		if err != nil {
			if stepacme.IsErrNotFound(err) {
				continue
			}
			return nil, "", err
		}
		keys = append(keys, eabFromDB(eak))
	}
	return keys, "", nil
}

// GetExternalAccountKeyByReference returns an EAB key by reference label.
func (s *ACMEStore) GetExternalAccountKeyByReference(ctx context.Context, provisionerID, reference string) (*stepacme.ExternalAccountKey, error) {
	s.eabMu.RLock()
	defer s.eabMu.RUnlock()

	if reference == "" {
		return nil, nil
	}
	var id string
	err := s.db.QueryRowContext(ctx, s.ph(
		`SELECT id FROM acme_eab_keys WHERE reference_key = ?`,
		`SELECT id FROM acme_eab_keys WHERE reference_key = $1`,
	), eabReferenceKey(provisionerID, reference)).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, stepacme.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	dbeak, err := s.getEABRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return eabFromDB(dbeak), nil
}

// GetExternalAccountKeyByAccountID is not used by the open-source ACME API.
func (s *ACMEStore) GetExternalAccountKeyByAccountID(context.Context, string, string) (*stepacme.ExternalAccountKey, error) {
	return nil, nil
}

// UpdateExternalAccountKey updates EAB metadata (typically account binding).
func (s *ACMEStore) UpdateExternalAccountKey(ctx context.Context, provisionerID string, eak *stepacme.ExternalAccountKey) error {
	s.eabMu.Lock()
	defer s.eabMu.Unlock()

	old, err := s.getEABRecord(ctx, eak.ID)
	if err != nil {
		return err
	}
	if old.ProvisionerID != provisionerID {
		return errors.New("provisioner does not match provisioner for which the EAB key was created")
	}
	if old.ProvisionerID != eak.ProvisionerID {
		return errors.New("cannot change provisioner for an existing ACME EAB Key")
	}
	if old.Reference != eak.Reference {
		return errors.New("cannot change reference for an existing ACME EAB Key")
	}

	nu := dbExternalAccountKey{
		ID:            eak.ID,
		ProvisionerID: eak.ProvisionerID,
		Reference:     eak.Reference,
		AccountID:     eak.AccountID,
		HmacKey:       eak.HmacKey,
		CreatedAt:     eak.CreatedAt,
		BoundAt:       eak.BoundAt,
	}
	data, err := s.marshal(&nu)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`UPDATE acme_eab_keys SET data = ? WHERE id = ?`,
		`UPDATE acme_eab_keys SET data = $1 WHERE id = $2`,
	), string(data), eak.ID)
	return err
}

// DeleteExternalAccountKey removes an EAB key.
func (s *ACMEStore) DeleteExternalAccountKey(ctx context.Context, provisionerID, keyID string) error {
	s.eabMu.Lock()
	defer s.eabMu.Unlock()

	dbeak, err := s.getEABRecord(ctx, keyID)
	if err != nil {
		return err
	}
	if dbeak.ProvisionerID != provisionerID {
		return errors.New("provisioner does not match provisioner for which the EAB key was created")
	}

	if dbeak.Reference != "" {
		_, err = s.db.ExecContext(ctx, s.ph(
			`DELETE FROM acme_eab_keys WHERE reference_key = ?`,
			`DELETE FROM acme_eab_keys WHERE reference_key = $1`,
		), eabReferenceKey(provisionerID, dbeak.Reference))
		if err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, s.ph(
		`DELETE FROM acme_eab_keys WHERE id = ?`,
		`DELETE FROM acme_eab_keys WHERE id = $1`,
	), keyID)
	if err != nil {
		return err
	}

	ids, err := s.loadProvisionerEABIDs(ctx, provisionerID)
	if err != nil {
		return err
	}
	filtered := ids[:0]
	for _, id := range ids {
		if id != keyID {
			filtered = append(filtered, id)
		}
	}
	return s.saveProvisionerEABIDs(ctx, provisionerID, filtered)
}

// Compile-time assertion that ACMEStore implements acme.DB.
var _ stepacme.DB = (*ACMEStore)(nil)
