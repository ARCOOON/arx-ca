package agent

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"

	"github.com/ARCOOON/arx-ca/internal/config"
)

const acmeUserAgent = "arx-agent/1.0"

// ACMERenewer obtains and renews certificates using ACMEv2 (RFC 8555).
type ACMERenewer struct {
	HTTPClient *http.Client
}

// NewACMERenewer returns an ACME renewer with optional HTTP client override.
func NewACMERenewer(httpClient *http.Client) *ACMERenewer {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ACMERenewer{HTTPClient: httpClient}
}

type acmeAccountState struct {
	KID string `json:"kid"`
	URI string `json:"uri"`
}

// Renew registers or reuses an ACME account, completes http-01 validation, and writes PEM files.
func (r *ACMERenewer) Renew(ctx context.Context, managed config.ManagedCert) error {
	if r == nil {
		return fmt.Errorf("ACME renewer is not configured")
	}
	if err := managed.Validate(); err != nil {
		return err
	}
	if managed.ProtocolName() != config.AgentProtocolACME {
		return fmt.Errorf("managed cert protocol is %q, not acme", managed.ProtocolName())
	}

	directoryURL := strings.TrimSpace(managed.ACMEDirectoryURL)
	email := strings.TrimSpace(managed.ACMEEmail)
	commonName := strings.TrimSpace(managed.CommonName)

	accountKey, accountKeyPath, err := loadOrCreateAccountKey(directoryURL, email)
	if err != nil {
		return err
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directoryURL,
		HTTPClient:   r.HTTPClient,
		UserAgent:    acmeUserAgent,
	}

	statePath, err := accountStatePath(directoryURL, email)
	if err != nil {
		return err
	}
	if state, err := loadAccountState(statePath); err == nil && state.KID != "" {
		client.KID = acme.KeyID(strings.TrimSpace(state.KID))
	}

	if _, err := client.Discover(ctx); err != nil {
		return fmt.Errorf("discover ACME directory: %w", err)
	}

	acct := &acme.Account{Contact: []string{"mailto:" + email}}
	reg, err := client.Register(ctx, acct, acceptACMETOS)
	if err != nil {
		return fmt.Errorf("register ACME account: %w", err)
	}
	kid := strings.TrimSpace(string(reg.URI))
	if kid == "" && client.KID != "" {
		kid = string(client.KID)
	}
	if kid != "" {
		if err := saveAccountState(statePath, acmeAccountState{KID: kid, URI: reg.URI}); err != nil {
			return fmt.Errorf("persist ACME account state: %w", err)
		}
		client.KID = acme.KeyID(kid)
	}
	_ = accountKeyPath

	certKey, err := loadOrCreateCertificateKey(strings.TrimSpace(managed.KeyPath))
	if err != nil {
		return fmt.Errorf("load certificate key: %w", err)
	}

	identifiers := []acme.AuthzID{{Type: "dns", Value: commonName}}
	order, err := client.AuthorizeOrder(ctx, identifiers)
	if err != nil {
		return fmt.Errorf("create ACME order: %w", err)
	}

	solver, cleanup, err := newHTTP01Solver(managed)
	if err != nil {
		return err
	}
	defer cleanup()

	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return fmt.Errorf("get authorization %s: %w", authzURL, err)
		}
		if authz.Status != acme.StatusPending {
			continue
		}
		for _, challenge := range authz.Challenges {
			if challenge.Status != acme.StatusPending {
				continue
			}
			if challenge.Type != config.AgentChallengeHTTP01 {
				continue
			}
			if err := solver.Provision(client, authz.Identifier, challenge); err != nil {
				return fmt.Errorf("provision http-01 challenge: %w", err)
			}
			if _, err := client.Accept(ctx, challenge); err != nil {
				return fmt.Errorf("accept http-01 challenge: %w", err)
			}
			break
		}
	}

	order, err = client.WaitOrder(ctx, order.URI)
	if err != nil {
		var orderErr *acme.OrderError
		if errors.As(err, &orderErr) {
			return fmt.Errorf("wait for ACME order: %s", orderErr)
		}
		return fmt.Errorf("wait for ACME order: %w", err)
	}
	if order.Status != acme.StatusReady {
		return fmt.Errorf("ACME order not ready: status %s", order.Status)
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: commonName},
		DNSNames: []string{commonName},
	}, certKey)
	if err != nil {
		return fmt.Errorf("create CSR: %w", err)
	}

	chainDER, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csrDER, true)
	if err != nil {
		return fmt.Errorf("finalize ACME order: %w", err)
	}

	certPEM, err := encodeCertificateChainPEM(chainDER)
	if err != nil {
		return err
	}
	keyPEM, err := encodePrivateKeyPEM(certKey)
	if err != nil {
		return err
	}

	certPath := strings.TrimSpace(managed.CertPath)
	keyPath := strings.TrimSpace(managed.KeyPath)

	if err := writePEMFile(certPath, certPEM); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}
	if err := writePEMFile(keyPath, keyPEM); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	return runPostHookIfSet(managed)
}

func acceptACMETOS(tosURL string) bool {
	slog.Info("accepting ACME terms of service", "url", tosURL)
	return true
}

func loadOrCreateAccountKey(directoryURL, email string) (crypto.Signer, string, error) {
	keyPath, err := accountKeyPath(directoryURL, email)
	if err != nil {
		return nil, "", err
	}
	if key, err := loadSignerFromPEM(keyPath); err == nil {
		return key, keyPath, nil
	} else if !os.IsNotExist(err) {
		return nil, keyPath, err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, keyPath, fmt.Errorf("generate ACME account key: %w", err)
	}
	if err := saveSignerPEM(keyPath, key); err != nil {
		return nil, keyPath, err
	}
	return key, keyPath, nil
}

func loadOrCreateCertificateKey(keyPath string) (crypto.Signer, error) {
	if key, err := loadSignerFromPEM(keyPath); err == nil {
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate certificate key: %w", err)
	}
	return key, nil
}

func loadSignerFromPEM(path string) (crypto.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode PEM from %s", path)
	}
	switch block.Type {
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key in %s is not a crypto.Signer", path)
		}
		return signer, nil
	default:
		return nil, fmt.Errorf("unsupported private key type %q in %s", block.Type, path)
	}
}

func saveSignerPEM(path string, key crypto.Signer) error {
	pemBytes, err := encodePrivateKeyPEM(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(pemBytes), 0o600)
}

func encodePrivateKeyPEM(key crypto.Signer) (string, error) {
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		der, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return "", fmt.Errorf("marshal EC private key: %w", err)
		}
		return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), nil
	default:
		return "", fmt.Errorf("unsupported private key type %T", key)
	}
}

func encodeCertificateChainPEM(chainDER [][]byte) (string, error) {
	var b strings.Builder
	for i, der := range chainDER {
		if len(der) == 0 {
			continue
		}
		if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
			return "", fmt.Errorf("encode certificate %d: %w", i, err)
		}
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("ACME response did not include certificate data")
	}
	return b.String(), nil
}

func acmeStateBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".arx-cert-service", "acme"), nil
}

func accountStorageID(directoryURL, email string) string {
	id := strings.ToLower(strings.TrimSpace(directoryURL) + "|" + strings.TrimSpace(email))
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func accountKeyPath(directoryURL, email string) (string, error) {
	base, err := acmeStateBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, accountStorageID(directoryURL, email), "account.key"), nil
}

func accountStatePath(directoryURL, email string) (string, error) {
	base, err := acmeStateBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, accountStorageID(directoryURL, email), "account.json"), nil
}

func loadAccountState(path string) (acmeAccountState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return acmeAccountState{}, err
	}
	var state acmeAccountState
	if err := json.Unmarshal(data, &state); err != nil {
		return acmeAccountState{}, fmt.Errorf("parse account state: %w", err)
	}
	return state, nil
}

func saveAccountState(path string, state acmeAccountState) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal account state: %w", err)
	}
	return os.WriteFile(path, raw, 0o600)
}

type http01Solver interface {
	Provision(client *acme.Client, ident acme.AuthzID, chall *acme.Challenge) error
}

func newHTTP01Solver(managed config.ManagedCert) (http01Solver, func(), error) {
	webroot := strings.TrimSpace(managed.Webroot)
	if webroot != "" {
		return &webrootHTTP01{webroot: webroot}, func() {}, nil
	}
	port := managed.ChallengeListenPort
	if port <= 0 {
		port = 80
	}
	listener, err := newChallengeHTTPServer(port)
	if err != nil {
		return nil, func() {}, err
	}
	return listener, func() {
		_ = listener.Shutdown()
	}, nil
}

type webrootHTTP01 struct {
	webroot string
}

func (w *webrootHTTP01) Provision(client *acme.Client, _ acme.AuthzID, chall *acme.Challenge) error {
	tokenPath := client.HTTP01ChallengePath(chall.Token)
	body, err := client.HTTP01ChallengeResponse(chall.Token)
	if err != nil {
		return fmt.Errorf("build http-01 response: %w", err)
	}
	rel := strings.TrimPrefix(tokenPath, "/")
	target := filepath.Join(w.webroot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create webroot challenge directory: %w", err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write challenge token: %w", err)
	}
	slog.Debug("http-01 challenge written to webroot", "path", target)
	return nil
}

type challengeHTTPServer struct {
	server   *http.Server
	mu       sync.RWMutex
	challMap map[string]string
}

func newChallengeHTTPServer(port int) (*challengeHTTPServer, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s for http-01: %w", addr, err)
	}

	srv := &challengeHTTPServer{
		challMap: make(map[string]string),
	}
	srv.server = &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		if err := srv.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http-01 challenge server error", "error", err)
		}
	}()

	slog.Info("http-01 challenge listener started", "address", ln.Addr().String())
	return srv, nil
}

func (c *challengeHTTPServer) Provision(client *acme.Client, _ acme.AuthzID, chall *acme.Challenge) error {
	tokenPath := client.HTTP01ChallengePath(chall.Token)
	body, err := client.HTTP01ChallengeResponse(chall.Token)
	if err != nil {
		return fmt.Errorf("build http-01 response: %w", err)
	}
	c.mu.Lock()
	c.challMap[path.Clean(tokenPath)] = body
	c.mu.Unlock()
	slog.Debug("http-01 challenge registered on listener", "path", tokenPath)
	return nil
}

func (c *challengeHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c.mu.RLock()
	body, ok := c.challMap[path.Clean(r.URL.Path)]
	c.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(body))
}

func (c *challengeHTTPServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.server.Shutdown(ctx)
}
