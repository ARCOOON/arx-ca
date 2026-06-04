package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/api/middleware"
	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

// WebUIServer serves static WebUI assets on a dedicated listener.
type WebUIServer struct {
	cfg       arxconfig.WebUIConfig
	server    *http.Server
	tlsConfig *tls.Config
	log       *slog.Logger
}

// NewWebUIServer validates configuration and builds the WebUI HTTP server.
// When cfg.ProxyAPIEnabled() is true, apiUpstream must be the loopback URL of the API listener.
func NewWebUIServer(cfg arxconfig.WebUIConfig, apiUpstream *url.URL, log *slog.Logger) (*WebUIServer, error) {
	if log == nil {
		log = slog.Default()
	}

	uiDir, err := cfg.ResolvedUIDir()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(uiDir)
	if err != nil {
		return nil, fmt.Errorf("webui ui_dir %s: %w", uiDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("webui ui_dir %s is not a directory", uiDir)
	}

	readTimeout, err := cfg.ReadTimeoutDuration()
	if err != nil {
		return nil, fmt.Errorf("webui read_timeout: %w", err)
	}
	writeTimeout, err := cfg.WriteTimeoutDuration()
	if err != nil {
		return nil, fmt.Errorf("webui write_timeout: %w", err)
	}

	var tlsConfig *tls.Config
	if cfg.TLS.Enabled {
		tlsConfig, err = prepareWebUITLS(cfg, log)
		if err != nil {
			return nil, err
		}
	}

	indexPath := filepath.Join(uiDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return nil, fmt.Errorf("webui index.html in %s: %w", uiDir, err)
	}

	handler := buildWebUIHandler(cfg, uiDir, indexPath)
	mux := http.NewServeMux()
	if cfg.ProxyAPIEnabled() {
		if apiUpstream == nil {
			return nil, fmt.Errorf("webui proxy_api is enabled but API upstream URL is nil")
		}
		registerAPIProxyRoutes(mux, apiUpstream, apiUpstream.Scheme == "https")
	}
	registerWebUIRoutes(mux, cfg.NormalizedPathPrefix(), handler)

	idleTimeout := readTimeout + writeTimeout
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}

	httpServer := &http.Server{
		Addr:         cfg.EffectiveListenAddress(),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	if tlsConfig != nil {
		httpServer.TLSConfig = tlsConfig
	}

	return &WebUIServer{
		cfg:       cfg,
		log:       log,
		server:    httpServer,
		tlsConfig: tlsConfig,
	}, nil
}

// Start runs the WebUI listener in a background goroutine.
func (s *WebUIServer) Start() {
	go func() {
		s.log.Info("WebUI server starting", slog.String("url", s.cfg.StartupURL()))
		var err error
		if s.tlsConfig != nil {
			listener, listenErr := tls.Listen("tcp", s.server.Addr, s.tlsConfig)
			if listenErr != nil {
				s.log.Error("WebUI server TLS listen failed", slog.Any("error", listenErr))
				return
			}
			err = s.server.Serve(listener)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("WebUI server listen error", slog.Any("error", err))
		}
	}()
}

// Shutdown gracefully stops the WebUI listener.
func (s *WebUIServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("webui shutdown: %w", err)
	}
	s.log.Info("WebUI server stopped")
	return nil
}

func buildWebUIHandler(cfg arxconfig.WebUIConfig, uiDir, indexPath string) http.Handler {
	fileHandler := &spaFileHandler{
		uiDir:     uiDir,
		indexPath: indexPath,
	}
	var handler http.Handler = fileHandler
	handler = maxBodySizeMiddleware(cfg.MaxBodySize, handler)
	corsOpts := middleware.CORSOptions{
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		AllowedMethods: cfg.CORS.AllowedMethods,
		AllowedHeaders: cfg.CORS.AllowedHeaders,
	}
	handler = middleware.CORS(corsOpts, handler)
	return handler
}

func registerAPIProxyRoutes(mux *http.ServeMux, upstream *url.URL, apiTLS bool) {
	transport := APIProxyTransport(upstream, apiTLS)
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.Transport = transport

	proxyHandler := wrapAPIProxyHandler(proxy)

	mux.Handle("/api/", proxyHandler)
	mux.Handle("/ocsp", proxyHandler)
	mux.Handle("/ocsp/", proxyHandler)
	mux.Handle("/acme/", proxyHandler)
	mux.Handle("/scep/", proxyHandler)
	mux.Handle("/certsrv/", proxyHandler)
}

func registerWebUIRoutes(mux *http.ServeMux, prefix string, handler http.Handler) {
	if prefix == "/" {
		mux.Handle("/", handler)
		return
	}
	mux.Handle(prefix+"/", http.StripPrefix(prefix, handler))
	mux.Handle(prefix, http.RedirectHandler(prefix+"/", http.StatusPermanentRedirect))
}

type spaFileHandler struct {
	uiDir     string
	indexPath string
}

func (h *spaFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	if path == "" || path == "/" {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	candidate := filepath.Join(h.uiDir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
	rel, err := filepath.Rel(h.uiDir, candidate)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	http.ServeFile(w, r, candidate)
}

func maxBodySizeMiddleware(limit int64, next http.Handler) http.Handler {
	if limit <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
