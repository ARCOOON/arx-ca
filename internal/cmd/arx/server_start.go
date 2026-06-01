package arxcmd

import (
	"context"
	"errors"
	"fmt"
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
	arxconfig "github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/database"
	"github.com/your-org/arx-ca/internal/telemetry"
)

func runServer() error {
	serverCfg := arxconfig.ServerConfigFromViper()

	appDB, err := database.Open(serverCfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := appDB.Close(); err != nil {
			log.Printf("application database close error: %v", err)
		}
	}()
	if err := database.Migrate(appDB); err != nil {
		return err
	}
	if err := database.SeedInitialAdmin(appDB, serverCfg.Bootstrap); err != nil {
		return err
	}

	listenAddr := serverCfg.ListenAddress()
	caConfigPath := serverCfg.CA.ConfigPath()

	otelShutdown, err := telemetry.Init(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Printf("OpenTelemetry shutdown error: %v", err)
		}
	}()

	pkiEngine, err := ca.InitCA(caConfigPath)
	if err != nil {
		return err
	}
	pkiEngine.SetApplicationDatabase(appDB)
	if err := pkiEngine.InitACMEServer(); err != nil {
		return fmt.Errorf("initialize ACME: %w", err)
	}
	defer func() {
		if err := pkiEngine.Shutdown(); err != nil {
			log.Printf("CA shutdown error: %v", err)
		}
	}()

	startTime := time.Now()
	healthHandler := handlers.NewHealthHandler(startTime, pkiEngine)
	caHandler := handlers.NewCAHandler(pkiEngine)
	publicHandler := handlers.NewPublicHandler(pkiEngine)
	ocspHandler := handlers.NewOCSPHandler(pkiEngine)
	certHandler := handlers.NewCertificateHandler(pkiEngine)
	provisionerHandler := handlers.NewProvisionerHandler(pkiEngine)
	renewalHandler := handlers.NewRenewalHandler(pkiEngine, listenAddr)
	acmeHandler := handlers.NewACMEHandler(pkiEngine, listenAddr)

	jwtManager, err := auth.LoadJWTManagerFromConfig(serverCfg.Security)
	if err != nil {
		return err
	}
	apiKeyStore := auth.NewAPIKeyStore()
	authHandler := handlers.NewAuthHandler(jwtManager, apiKeyStore)
	sshHandler := handlers.NewSSHHandler(pkiEngine, jwtManager, apiKeyStore)
	templateHandler := handlers.NewTemplateHandler(pkiEngine)

	adminPerm := func(perm auth.Permission, h http.Handler) http.Handler {
		return middleware.RequireAdmin(jwtManager, middleware.RequirePermission(perm, h))
	}
	certPerm := func(perm auth.Permission, h http.Handler) http.Handler {
		return middleware.RequireServiceAccountOrAdmin(jwtManager, apiKeyStore, middleware.RequirePermission(perm, h))
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", healthHandler)
	mux.Handle("GET /api/v1/ca/root", caHandler.RootCert())
	mux.Handle("GET /api/v1/ca/crl", caHandler.CRL())
	mux.Handle("GET /api/v1/public/ca/intermediate", publicHandler.IntermediateCert())
	mux.Handle("GET /api/v1/public/certificates", publicHandler.ListCertificates())
	mux.Handle("GET /api/v1/public/certificates/{serial}", publicHandler.GetCertificate())
	mux.Handle("POST /ocsp", ocspHandler.Post())
	mux.Handle("GET /ocsp/{request}", ocspHandler.Get())
	mux.Handle("POST /api/v1/auth/login", authHandler.Login())
	mux.Handle("POST /api/v1/auth/service-accounts", adminPerm(auth.PermServiceAccounts, authHandler.CreateServiceAccount()))

	mux.Handle("POST /api/v1/certificates/issue", certPerm(auth.PermCertificatesIssue, certHandler.Issue()))
	mux.Handle("POST /api/v1/certificates/issue-with-token", certPerm(auth.PermCertificatesIssue, certHandler.IssueWithToken()))
	mux.Handle("POST /api/v1/certificates/auto", certPerm(auth.PermCertificatesIssue, certHandler.Auto()))
	mux.Handle("POST /api/v1/certificates/revoke", certPerm(auth.PermCertificatesRevoke, certHandler.Revoke()))
	mux.Handle("POST /api/v1/certificates/lint", certPerm(auth.PermCertificatesLint, certHandler.Lint()))
	mux.Handle("GET /api/v1/certificates", certPerm(auth.PermCertificatesRead, certHandler.List()))
	mux.Handle("POST /api/v1/provisioners/token", certPerm(auth.PermProvisionersToken, provisionerHandler.Token()))
	mux.Handle("GET /api/v1/k8s/status", certPerm(auth.PermEnrollmentStatus, provisionerHandler.K8sStatus()))
	mux.Handle("POST /api/v1/certificates/renew", certPerm(auth.PermCertificatesRenew, renewalHandler.Renew()))
	mux.Handle("POST /api/v1/certificates/rekey", certPerm(auth.PermCertificatesRenew, renewalHandler.Rekey()))
	mux.Handle("GET /api/v1/acme/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.Status()))
	mux.Handle("POST /api/v1/acme/eab-keys", certPerm(auth.PermACMEEAB, acmeHandler.CreateEABKey()))
	mux.Handle("GET /api/v1/scep/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.SCEPStatus()))
	mux.Handle("GET /api/v1/ndes/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.NDESStatus()))

	mux.Handle("POST /api/v1/templates", certPerm(auth.PermTemplatesManage, templateHandler.Create()))
	mux.Handle("GET /api/v1/templates", certPerm(auth.PermTemplatesRead, templateHandler.List()))

	mux.Handle("POST /api/v1/ssh/sign-user", sshHandler.SignUser())
	mux.Handle("POST /api/v1/ssh/sign-host", certPerm(auth.PermSSHSignHost, sshHandler.SignHost()))
	mux.Handle("POST /api/v1/ssh/inspect", certPerm(auth.PermSSHInspect, sshHandler.Inspect()))
	mux.Handle("GET /api/v1/ssh/roots", sshHandler.Roots())

	if pkiEngine.ACMEEnabled() {
		mux.Handle("/acme/", http.StripPrefix("/acme", pkiEngine.ACMEHandler()))
		log.Printf("ACME enabled; directory available at /acme/directory")
	}
	if pkiEngine.SCEPEnabled() {
		mux.Handle("/scep/", http.StripPrefix("/scep", pkiEngine.SCEPHandler()))
		log.Printf("SCEP enabled; endpoint available at /scep/%s", pkiEngine.SCEPProvisionerName())
	}
	if pkiEngine.NDESEnabled() {
		mux.Handle("/certsrv/", http.StripPrefix("/certsrv", pkiEngine.NDESHandler()))
		ca.LogNDESConnectors(pkiEngine.NDESRegistryRef())
	}

	handler := telemetry.HTTPMiddleware(middleware.Logger(mux))

	idleTimeout := serverCfg.Server.ReadTimeout + serverCfg.Server.WriteTimeout
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  serverCfg.Server.ReadTimeout,
		WriteTimeout: serverCfg.Server.WriteTimeout,
		IdleTimeout:  idleTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return pkiEngine.BaseContext()
		},
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("arx server listening on %s", listenAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), serverCfg.Server.WriteTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}
	log.Println("server stopped")
	return nil
}
