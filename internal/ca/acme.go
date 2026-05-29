package ca

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"github.com/smallstep/certificates/acme"
	acmeAPI "github.com/smallstep/certificates/acme/api"
	acmeNoSQL "github.com/smallstep/certificates/acme/db/nosql"
	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/certificates/scep"
	"github.com/smallstep/nosql"
)

const (
	acmeProvisionerName = "acme"
	acmeRoutePrefix     = "acme"
)

// ACMEEnabled reports whether the ACME HTTP handler is configured.
func (e *PKIEngine) ACMEEnabled() bool {
	return e != nil && e.acmeHandler != nil
}

// ACMEHandler returns the step-ca ACME HTTP handler. The handler must be mounted
// under the /acme path prefix (for example /acme/acme/directory).
func (e *PKIEngine) ACMEHandler() http.Handler {
	if e == nil || e.acmeHandler == nil {
		return http.NotFoundHandler()
	}
	return e.acmeHandler
}

// BaseContext returns the server base context with authority and ACME dependencies.
func (e *PKIEngine) BaseContext() context.Context {
	if e == nil || e.baseCtx == nil {
		return context.Background()
	}
	return e.baseCtx
}

// ACMEDNS returns the DNS name used to generate ACME directory links.
func (e *PKIEngine) ACMEDNS() string {
	if e == nil {
		return ""
	}
	return e.acmeDNS
}

func normalizeListenHost(listenHost string) string {
	host := strings.TrimSpace(listenHost)
	if host == "" {
		host = ":8080"
	}
	if !strings.Contains(host, "://") {
		if strings.HasPrefix(host, ":") {
			return "http://localhost" + host
		}
		return "http://" + host
	}
	return host
}

// ACMEDirectoryURL returns the local ACME directory URL for the default provisioner.
func (e *PKIEngine) ACMEDirectoryURL(listenHost string) string {
	u, err := url.Parse(normalizeListenHost(listenHost))
	if err != nil {
		return ""
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + acmeRoutePrefix + "/" + acmeProvisionerName + "/directory"
	return u.String()
}

func (e *PKIEngine) initACME() error {
	if e == nil || e.auth == nil || e.config == nil {
		return nil
	}

	configureACMEChallengePorts()

	if e.config.DB == nil {
		if e.auth.HasACMEProvisioner() {
			log.Println("WARNING: ACME provisioner is configured but no database is available; ACME is disabled")
		}
		return nil
	}
	if !e.auth.HasACMEProvisioner() {
		return nil
	}

	nosqlDB, ok := e.auth.GetDatabase().(nosql.DB)
	if !ok {
		return errors.New("ACME requires a nosql-compatible database backend")
	}

	acmeDB, err := acmeNoSQL.New(nosqlDB)
	if err != nil {
		return fmt.Errorf("configure ACME database: %w", err)
	}

	dns := resolveACMEDNS(e.config)
	e.acmeLinker = acme.NewLinker(dns, acmeRoutePrefix)
	e.acmeDB = acmeDB
	e.acmeDNS = dns

	router := chi.NewRouter()
	acmeAPI.Route(router)

	e.rebuildBaseContext()
	e.acmeHandler = newChiProtocolHandler(e, router)

	return nil
}

// buildBaseContext returns the server base context with authority, ACME, and SCEP dependencies.
func (e *PKIEngine) buildBaseContext() context.Context {
	if e == nil || e.auth == nil {
		return context.Background()
	}
	ctx := authority.NewContext(context.Background(), e.auth)
	if authDB := e.auth.GetDatabase(); authDB != nil {
		ctx = db.NewContext(ctx, authDB)
	}
	if scepAuth := e.auth.GetSCEP(); scepAuth != nil {
		ctx = scep.NewContext(ctx, scepAuth)
	}
	if e.acmeDB != nil && e.acmeLinker != nil {
		ctx = acme.NewContext(ctx, e.acmeDB, acme.NewClient(), e.acmeLinker, nil)
	}
	return ctx
}

func (e *PKIEngine) rebuildBaseContext() {
	e.baseCtx = e.buildBaseContext()
}

func configureACMEChallengePorts() {
	if value := strings.TrimSpace(os.Getenv("CA_API_ACME_HTTP_PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 {
			log.Printf("WARNING: invalid CA_API_ACME_HTTP_PORT %q; using default port 80 for http-01", value)
		} else {
			acme.InsecurePortHTTP01 = port
		}
	}
	if value := strings.TrimSpace(os.Getenv("CA_API_ACME_TLS_PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 {
			log.Printf("WARNING: invalid CA_API_ACME_TLS_PORT %q; using default port 443 for tls-alpn-01", value)
		} else {
			acme.InsecurePortTLSALPN01 = port
		}
	}
	if strings.EqualFold(os.Getenv("CA_API_ACME_STRICT_FQDN"), "true") {
		acme.StrictFQDN = true
	}
}

func resolveACMEDNS(cfg *authconfig.Config) string {
	if value := strings.TrimSpace(os.Getenv("CA_API_ACME_DNS")); value != "" {
		return value
	}
	if cfg == nil {
		return defaultCADNS
	}
	if len(cfg.DNSNames) > 0 && strings.TrimSpace(cfg.DNSNames[0]) != "" {
		return strings.TrimSpace(cfg.DNSNames[0])
	}
	if addr := strings.TrimSpace(cfg.Address); addr != "" {
		if host, _, err := netSplitHostPort(addr); err == nil && host != "" {
			return host
		}
		return strings.TrimPrefix(strings.TrimPrefix(addr, "https://"), "http://")
	}
	return defaultCADNS
}

func netSplitHostPort(addr string) (host, port string, err error) {
	if strings.Contains(addr, "://") {
		u, parseErr := url.Parse(addr)
		if parseErr != nil {
			return "", "", parseErr
		}
		return u.Hostname(), u.Port(), nil
	}
	return splitHostPort(addr)
}

func splitHostPort(addr string) (host, port string, err error) {
	if h, p, ok := strings.Cut(addr, ":"); ok {
		return h, p, nil
	}
	return addr, "", nil
}

func deviceAttestEnabled() bool {
	return strings.EqualFold(os.Getenv("CA_API_ACME_DEVICE_ATTEST"), "true")
}

func resolveACMEAttestationFormats() []provisioner.ACMEAttestationFormat {
	raw := strings.TrimSpace(os.Getenv("CA_API_ACME_ATTESTATION_FORMATS"))
	if raw == "" {
		return []provisioner.ACMEAttestationFormat{
			provisioner.APPLE,
			provisioner.STEP,
			provisioner.TPM,
		}
	}
	parts := strings.Split(raw, ",")
	out := make([]provisioner.ACMEAttestationFormat, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		format := provisioner.ACMEAttestationFormat(part)
		if err := format.Validate(); err != nil {
			log.Printf("WARNING: skipping invalid attestation format %q: %v", part, err)
			continue
		}
		out = append(out, format)
	}
	if len(out) == 0 {
		return []provisioner.ACMEAttestationFormat{
			provisioner.APPLE,
			provisioner.TPM,
			provisioner.STEP,
		}
	}
	return out
}

func loadACMEAttestationRoots() ([]byte, error) {
	path := strings.TrimSpace(os.Getenv("CA_API_ACME_ATTESTATION_ROOTS"))
	if path == "" {
		return nil, nil
	}
	return os.ReadFile(path)
}

// ensureACMEProvisioner adds the default ACME provisioner to ca.json when missing.
func ensureACMEProvisioner(configPath string) error {
	if strings.EqualFold(os.Getenv("CA_API_ACME_DISABLED"), "true") {
		return nil
	}

	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for ACME provisioner: %w", err)
	}
	if cfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	for _, p := range cfg.AuthorityConfig.Provisioners {
		if p.GetName() == acmeProvisionerName && p.GetType() == provisioner.TypeACME {
			return nil
		}
	}

	acmeProv := &provisioner.ACME{
		Type: "ACME",
		Name: acmeProvisionerName,
		Challenges: []provisioner.ACMEChallenge{
			provisioner.HTTP_01,
			provisioner.DNS_01,
			provisioner.TLS_ALPN_01,
		},
	}
	if deviceAttestEnabled() {
		acmeProv.Challenges = append(acmeProv.Challenges, provisioner.DEVICE_ATTEST_01)
		acmeProv.AttestationFormats = resolveACMEAttestationFormats()
		if roots, err := loadACMEAttestationRoots(); err != nil {
			return fmt.Errorf("load ACME attestation roots: %w", err)
		} else if len(roots) > 0 {
			acmeProv.AttestationRoots = roots
		}
	}
	if strings.EqualFold(os.Getenv("CA_API_ACME_REQUIRE_EAB"), "true") {
		acmeProv.RequireEAB = true
	}

	cfg.AuthorityConfig.Provisioners = append(cfg.AuthorityConfig.Provisioners, acmeProv)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated CA configuration: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("write updated CA configuration: %w", err)
	}

	return nil
}
