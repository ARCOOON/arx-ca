package ca

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

const (
	// Paths after stripping the /certsrv mount prefix (AD CS compatible layout).
	ndesMSCEPPath      = "/mscep/mscep.dll"
	ndesMSCEPAdminPath = "/mscep_admin/mscep_admin.dll"
)

// NDESConnector translates NDES-style enrollment requests to a CA backend.
// Implementations act as drop-in modules for Microsoft AD CS migration paths.
type NDESConnector interface {
	Name() string
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NDESRegistry holds named NDES connectors and routes AD CS compatible paths.
type NDESRegistry struct {
	mu          sync.RWMutex
	connectors  map[string]NDESConnector
	defaultName string
	adminSecret string
}

// NewNDESRegistry creates an empty connector registry.
func NewNDESRegistry() *NDESRegistry {
	return &NDESRegistry{
		connectors: make(map[string]NDESConnector),
	}
}

// Register adds or replaces a connector by name.
func (r *NDESRegistry) Register(c NDESConnector) {
	if r == nil || c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectors[c.Name()] = c
	if r.defaultName == "" {
		r.defaultName = c.Name()
	}
}

// SetDefault selects the connector used for standard mscep.dll traffic.
func (r *NDESRegistry) SetDefault(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultName = name
}

// SetAdminSecret configures the shared secret required for mscep_admin password retrieval.
func (r *NDESRegistry) SetAdminSecret(secret string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adminSecret = secret
}

// Names returns registered connector identifiers.
func (r *NDESRegistry) Names() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.connectors))
	for name := range r.connectors {
		names = append(names, name)
	}
	return names
}

func (r *NDESRegistry) connector(name string) NDESConnector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.connectors[name]; ok {
		return c
	}
	if r.defaultName != "" {
		return r.connectors[r.defaultName]
	}
	return nil
}

// ServeHTTP routes Microsoft NDES URL paths to registered connectors.
func (r *NDESRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r == nil {
		http.NotFound(w, req)
		return
	}

	path := strings.ToLower(req.URL.Path)
	switch {
	case strings.HasSuffix(path, ndesMSCEPPath):
		conn := r.connector(r.defaultName)
		if conn == nil {
			http.Error(w, "NDES SCEP connector is not configured", http.StatusServiceUnavailable)
			return
		}
		conn.ServeHTTP(w, rewriteNDESPath(req, "/"+scepRoutePrefix+"/"+scepProvisionerName))
	case strings.HasSuffix(path, ndesMSCEPAdminPath):
		r.serveAdminPassword(w, req)
	default:
		http.NotFound(w, req)
	}
}

func (r *NDESRegistry) serveAdminPassword(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.mu.RLock()
	secret := r.adminSecret
	r.mu.RUnlock()

	if secret != "" {
		provided := strings.TrimSpace(req.Header.Get("X-NDES-Admin-Secret"))
		if provided == "" {
			provided = strings.TrimSpace(req.URL.Query().Get("secret"))
		}
		if subtleConstantTimeCompare(provided, secret) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	challenge := strings.TrimSpace(os.Getenv("CA_API_SCEP_CHALLENGE"))
	if challenge == "" {
		http.Error(w, "SCEP challenge password is not configured", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(challenge))
}

// SCEPNDESConnector forwards NDES-normalized requests to the step-ca SCEP handler.
type SCEPNDESConnector struct {
	name    string
	handler http.Handler
}

// NewSCEPNDESConnector wraps an SCEP HTTP handler as an NDES connector.
func NewSCEPNDESConnector(name string, handler http.Handler) *SCEPNDESConnector {
	if strings.TrimSpace(name) == "" {
		name = "scep"
	}
	return &SCEPNDESConnector{name: name, handler: handler}
}

// Name implements NDESConnector.
func (c *SCEPNDESConnector) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// ServeHTTP implements NDESConnector.
func (c *SCEPNDESConnector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c == nil || c.handler == nil {
		http.Error(w, "SCEP handler is not configured", http.StatusServiceUnavailable)
		return
	}
	c.handler.ServeHTTP(w, r)
}

// NDESEnabled reports whether the NDES router is configured.
func (e *PKIEngine) NDESEnabled() bool {
	return e != nil && e.ndesHandler != nil
}

// NDESHandler returns the NDES HTTP handler for AD CS compatible paths.
func (e *PKIEngine) NDESHandler() http.Handler {
	if e == nil || e.ndesHandler == nil {
		return http.NotFoundHandler()
	}
	return e.ndesHandler
}

// NDESRegistryRef exposes the connector registry for extension and testing.
func (e *PKIEngine) NDESRegistryRef() *NDESRegistry {
	if e == nil {
		return nil
	}
	return e.ndesRegistry
}

func (e *PKIEngine) initNDES() error {
	if e == nil {
		return nil
	}
	if strings.EqualFold(os.Getenv("CA_API_NDES_DISABLED"), "true") {
		return nil
	}
	if !e.SCEPEnabled() {
		if strings.EqualFold(os.Getenv("CA_API_NDES_REQUIRED"), "true") {
			return errors.New("NDES requires SCEP to be enabled")
		}
		return nil
	}

	registry := NewNDESRegistry()
	registry.Register(NewSCEPNDESConnector("scep", e.SCEPHandler()))
	registry.SetDefault("scep")

	if secret := strings.TrimSpace(os.Getenv("CA_API_NDES_ADMIN_SECRET")); secret != "" {
		registry.SetAdminSecret(secret)
	}

	e.ndesRegistry = registry
	e.ndesHandler = &protocolContextHandler{
		engine: e,
		router: ndesRouter{registry: registry},
	}
	return nil
}

type ndesRouter struct {
	registry *NDESRegistry
}

func (r ndesRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.registry.ServeHTTP(w, req)
}

// rewriteNDESPath clones the request with an SCEP path while preserving query parameters.
func rewriteNDESPath(r *http.Request, scepPath string) *http.Request {
	clone := r.Clone(r.Context())
	u := *r.URL
	u.Path = scepPath
	u.RawPath = scepPath
	clone.URL = &u
	clone.RequestURI = u.RequestURI()
	return clone
}

func subtleConstantTimeCompare(a, b string) int {
	if len(a) != len(b) {
		return 0
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	if v == 0 {
		return 1
	}
	return 0
}

// NDESContext carries request-scoped NDES metadata for connectors.
type NDESContext struct {
	Template    string
	RAProfile   string
	DeviceID    string
	ConnectorID string
}

type ndesContextKey struct{}

// WithNDESContext attaches NDES metadata to a context.
func WithNDESContext(ctx context.Context, meta NDESContext) context.Context {
	return context.WithValue(ctx, ndesContextKey{}, meta)
}

// NDESContextFromRequest extracts optional NDES query parameters.
func NDESContextFromRequest(r *http.Request) NDESContext {
	if r == nil {
		return NDESContext{}
	}
	q := r.URL.Query()
	return NDESContext{
		Template:    strings.TrimSpace(q.Get("template")),
		RAProfile:   strings.TrimSpace(q.Get("ra")),
		DeviceID:    strings.TrimSpace(q.Get("deviceid")),
		ConnectorID: strings.TrimSpace(q.Get("connector")),
	}
}

// NDESAdminURL returns the mscep_admin password endpoint for a listen host.
func (e *PKIEngine) NDESAdminURL(listenHost string) string {
	host := normalizeListenHost(listenHost)
	if host == "" {
		return ""
	}
	u, err := url.Parse(host)
	if err != nil {
		return ""
	}
	u.Path = "/certsrv" + ndesMSCEPAdminPath
	return u.String()
}

// NDESMSCEPURL builds an AD CS compatible SCEP URL for a given host.
func NDESMSCEPURL(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return ""
	}
	u.Path = "/certsrv" + ndesMSCEPPath
	return u.String()
}

// LogNDESConnectors logs registered connectors at startup.
func LogNDESConnectors(registry *NDESRegistry) {
	if registry == nil {
		return
	}
	names := registry.Names()
	if len(names) == 0 {
		return
	}
	log.Printf("NDES enabled; connectors=%s; mscep=%s", strings.Join(names, ","), ndesMSCEPPath)
}
