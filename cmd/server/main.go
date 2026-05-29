package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/arx-ca/internal/api/handlers"
	"github.com/your-org/arx-ca/internal/api/middleware"
	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/ca"
)

const (
	defaultListenAddr   = ":8080"
	defaultCAConfigPath = ".pki/config/ca.json"
	shutdownTimeout     = 10 * time.Second
)

func main() {
	listenAddr := envOrDefault("CA_API_LISTEN_ADDR", defaultListenAddr)
	caConfigPath := envOrDefault("CA_API_CA_CONFIG", defaultCAConfigPath)

	pkiEngine, err := ca.InitCA(caConfigPath)
	if err != nil {
		log.Fatalf("CA initialization failed: %v", err)
	}
	defer func() {
		if err := pkiEngine.Shutdown(); err != nil {
			log.Printf("CA shutdown error: %v", err)
		}
	}()

	startTime := time.Now()
	healthHandler := handlers.NewHealthHandler(startTime, pkiEngine)
	caHandler := handlers.NewCAHandler(pkiEngine)
	certHandler := handlers.NewCertificateHandler(pkiEngine)
	provisionerHandler := handlers.NewProvisionerHandler(pkiEngine)
	renewalHandler := handlers.NewRenewalHandler(pkiEngine, listenAddr)

	jwtManager, err := auth.LoadJWTManagerFromEnv()
	if err != nil {
		log.Fatalf("jwt configuration error: %v", err)
	}
	apiKeyStore := auth.NewAPIKeyStore()
	authHandler := handlers.NewAuthHandler(jwtManager, apiKeyStore)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", healthHandler)
	mux.Handle("GET /api/v1/ca/root", caHandler.RootCert())
	mux.Handle("POST /api/v1/auth/login", authHandler.Login())
	mux.Handle(
		"POST /api/v1/auth/service-accounts",
		middleware.RequireAdmin(jwtManager, authHandler.CreateServiceAccount()),
	)

	certAuth := func(h http.Handler) http.Handler {
		return middleware.RequireServiceAccountOrAdmin(jwtManager, apiKeyStore, h)
	}
	mux.Handle("POST /api/v1/certificates/issue", certAuth(certHandler.Issue()))
	mux.Handle("POST /api/v1/certificates/issue-with-token", certAuth(certHandler.IssueWithToken()))
	mux.Handle("POST /api/v1/certificates/auto", certAuth(certHandler.Auto()))
	mux.Handle("POST /api/v1/certificates/revoke", certAuth(certHandler.Revoke()))
	mux.Handle("POST /api/v1/certificates/lint", certAuth(certHandler.Lint()))
	mux.Handle("GET /api/v1/certificates", certAuth(certHandler.List()))
	mux.Handle("POST /api/v1/provisioners/token", certAuth(provisionerHandler.Token()))
	mux.Handle("POST /api/v1/certificates/renew", certAuth(renewalHandler.Renew()))
	mux.Handle("POST /api/v1/certificates/rekey", certAuth(renewalHandler.Rekey()))
	mux.Handle("GET /api/v1/acme/status", certAuth(renewalHandler.ACMEStatus()))

	if pkiEngine.ACMEEnabled() {
		mux.Handle("/acme/", http.StripPrefix("/acme", pkiEngine.ACMEHandler()))
		log.Printf("ACME enabled; directory available at /acme/acme/directory")
	}

	handler := middleware.Logger(mux)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return pkiEngine.BaseContext()
		},
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("arx-ca server listening on %s", listenAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
