package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	arxconfig "github.com/your-org/arx-ca/internal/config"
)

// WebUIServer serves static WebUI assets on a dedicated listener.
type WebUIServer struct {
	cfg    arxconfig.WebUIConfig
	server *http.Server
	log    *slog.Logger
}

// NewWebUIServer validates configuration and builds the WebUI HTTP server.
func NewWebUIServer(cfg arxconfig.WebUIConfig, log *slog.Logger) (*WebUIServer, error) {
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

	if cfg.TLS.Enabled {
		certFile, err := cfg.ResolvedTLSCertFile()
		if err != nil {
			return nil, err
		}
		keyFile, err := cfg.ResolvedTLSKeyFile()
		if err != nil {
			return nil, err
		}
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("webui tls is enabled but cert_file and key_file must be set")
		}
		if _, err := os.Stat(certFile); err != nil {
			return nil, fmt.Errorf("webui tls cert_file %s: %w", certFile, err)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return nil, fmt.Errorf("webui tls key_file %s: %w", keyFile, err)
		}
	}

	indexPath := filepath.Join(uiDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return nil, fmt.Errorf("webui index.html in %s: %w", uiDir, err)
	}

	handler := buildWebUIHandler(cfg, uiDir, indexPath)
	mux := http.NewServeMux()
	registerWebUIRoutes(mux, cfg.NormalizedPathPrefix(), handler)

	idleTimeout := readTimeout + writeTimeout
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}

	return &WebUIServer{
		cfg: cfg,
		log: log,
		server: &http.Server{
			Addr:         cfg.EffectiveListenAddress(),
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}, nil
}

// Start runs the WebUI listener in a background goroutine.
func (s *WebUIServer) Start() {
	go func() {
		s.log.Info("WebUI server starting", slog.String("url", s.cfg.StartupURL()))
		var err error
		if s.cfg.TLS.Enabled {
			certFile, certErr := s.cfg.ResolvedTLSCertFile()
			if certErr != nil {
				s.log.Error("WebUI server TLS cert resolution failed", slog.Any("error", certErr))
				return
			}
			keyFile, keyErr := s.cfg.ResolvedTLSKeyFile()
			if keyErr != nil {
				s.log.Error("WebUI server TLS key resolution failed", slog.Any("error", keyErr))
				return
			}
			err = s.server.ListenAndServeTLS(certFile, keyFile)
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
	handler = corsMiddleware(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, handler)
	return handler
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

func corsMiddleware(allowedOrigins, allowedMethods []string, next http.Handler) http.Handler {
	origins := append([]string(nil), allowedOrigins...)
	methods := append([]string(nil), allowedMethods...)
	if len(methods) == 0 {
		methods = []string{"GET", "OPTIONS"}
	}
	allowMethods := strings.Join(methods, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && corsOriginAllowed(origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", corsAllowOriginValue(origins, origin))
			w.Header().Add("Vary", "Origin")
		} else if corsAllowsAnyOrigin(origins) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		if acrh := r.Header.Get("Access-Control-Request-Headers"); acrh != "" {
			w.Header().Set("Access-Control-Allow-Headers", acrh)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsAllowsAnyOrigin(origins []string) bool {
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}

func corsOriginAllowed(origins []string, requestOrigin string) bool {
	if corsAllowsAnyOrigin(origins) {
		return true
	}
	for _, o := range origins {
		if strings.EqualFold(strings.TrimSpace(o), requestOrigin) {
			return true
		}
	}
	return false
}

func corsAllowOriginValue(origins []string, requestOrigin string) string {
	if corsAllowsAnyOrigin(origins) {
		return "*"
	}
	return requestOrigin
}
