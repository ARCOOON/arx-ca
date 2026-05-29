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

// ACMEDirectoryURL returns the local ACME directory URL for the default provisioner.
func (e *PKIEngine) ACMEDirectoryURL(listenHost string) string {
	host := strings.TrimSpace(listenHost)
	if host == "" {
		host = ":8080"
	}
	if !strings.Contains(host, "://") {
		if strings.HasPrefix(host, ":") {
			host = "http://localhost" + host
		} else {
			host = "http://" + host
		}
	}
	u, err := url.Parse(host)
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

	e.acmeHandler = &acmeContextHandler{
		base:   e.buildACMEContext(),
		router: router,
	}
	e.baseCtx = e.buildACMEContext()

	return nil
}

func (e *PKIEngine) buildACMEContext() context.Context {
	ctx := authority.NewContext(context.Background(), e.auth)
	if authDB := e.auth.GetDatabase(); authDB != nil {
		ctx = db.NewContext(ctx, authDB)
	}
	if e.acmeDB != nil && e.acmeLinker != nil {
		ctx = acme.NewContext(ctx, e.acmeDB, acme.NewClient(), e.acmeLinker, nil)
	}
	return ctx
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

	cfg.AuthorityConfig.Provisioners = append(cfg.AuthorityConfig.Provisioners, &provisioner.ACME{
		Type: "ACME",
		Name: acmeProvisionerName,
		Challenges: []provisioner.ACMEChallenge{
			provisioner.HTTP_01,
			provisioner.DNS_01,
			provisioner.TLS_ALPN_01,
		},
	})

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

type acmeContextHandler struct {
	base   context.Context
	router chi.Router
}

func (h *acmeContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(h.base)
	h.router.ServeHTTP(w, r)
}
