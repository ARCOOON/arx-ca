package arxcmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ARCOOON/arx-ca/internal/api/handlers"
	"github.com/ARCOOON/arx-ca/internal/api/middleware"
	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/ca"
	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/database"
	auditdb "github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
	"github.com/ARCOOON/arx-ca/internal/notifications"
	"github.com/ARCOOON/arx-ca/internal/repository"
	arxserver "github.com/ARCOOON/arx-ca/internal/server"
	"github.com/ARCOOON/arx-ca/internal/telemetry"
	"github.com/ARCOOON/arx-ca/internal/updater"
)

func runServer() error {
	if configPath, err := arxconfig.ServerConfigPath(); err == nil {
		configDir := filepath.Dir(configPath)
		if err := os.Chdir(configDir); err != nil {
			return fmt.Errorf("change working directory to %s: %w", configDir, err)
		}
	}

	serverCfg := arxconfig.ServerConfigFromViper()
	logging.Configure(serverCfg.Server.LogLevel)
	log := logging.Logger()

	appDB, err := database.Open(serverCfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := appDB.Close(); err != nil {
			log.Error("application database close error", slog.Any("error", err))
		}
	}()
	if err := database.Migrate(appDB); err != nil {
		return err
	}
	if err := database.EnsureBootstrapAdmin(appDB, serverCfg); err != nil {
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
			log.Error("OpenTelemetry shutdown error", slog.Any("error", err))
		}
	}()

	pkiEngine, err := ca.InitCA(caConfigPath, serverCfg.CA, serverCfg.EffectiveCABootstrap())
	if err != nil {
		return err
	}
	pkiEngine.SetApplicationDatabase(appDB)
	if err := pkiEngine.InitACMEServer(); err != nil {
		return fmt.Errorf("initialize ACME: %w", err)
	}
	defer func() {
		if err := pkiEngine.Shutdown(); err != nil {
			log.Error("CA shutdown error", slog.Any("error", err))
		}
	}()

	startTime := time.Now()
	healthHandler := handlers.NewHealthHandler(startTime, pkiEngine)
	caHandler := handlers.NewCAHandler(pkiEngine)
	publicHandler := handlers.NewPublicHandler(pkiEngine)
	ocspHandler := handlers.NewOCSPHandler(pkiEngine)
	provisionerHandler := handlers.NewProvisionerHandler(pkiEngine)
	renewalHandler := handlers.NewRenewalHandler(pkiEngine, listenAddr)
	acmeHandler := handlers.NewACMEHandler(pkiEngine, listenAddr)

	jwtManager, err := auth.LoadJWTManagerFromConfig(serverCfg.Security)
	if err != nil {
		return err
	}
	apiKeyStore := auth.NewAPIKeyStore()
	clientCertValidator, err := ca.NewClientCertValidator(pkiEngine)
	if err != nil {
		return fmt.Errorf("initialize client certificate validator: %w", err)
	}
	userStore := repository.NewUserStore(appDB)
	certStore := database.NewCertificateStore(appDB)
	certHandler := handlers.NewCertificateHandler(pkiEngine, certStore, userStore)
	sessionCookie := auth.SessionCookieConfigFromSecurity(serverCfg.Security)
	authHandler := handlers.NewAuthHandler(jwtManager, apiKeyStore, userStore, sessionCookie)
	sshCertStore := auditdb.NewSSHCertificateStore(appDB)
	sshHandler := handlers.NewSSHHandler(pkiEngine, jwtManager, apiKeyStore, sshCertStore)
	statsHandler := handlers.NewStatsHandler(pkiEngine, sshCertStore)
	templateHandler := handlers.NewTemplateHandler(pkiEngine)
	auditStore := auditdb.NewAuditStore(appDB)
	auditHandler := handlers.NewAuditHandler(auditStore)
	webhookStore := auditdb.NewWebhookStore(appDB)
	notificationStore := auditdb.NewNotificationStore(appDB)
	webhookDispatcher := notifications.NewDispatcher(webhookStore, notificationStore)
	webhookHandler := handlers.NewWebhookHandler(webhookStore, webhookDispatcher)
	notificationHandler := handlers.NewNotificationHandler(webhookDispatcher, notificationStore)
	configManager, err := arxconfig.NewManager()
	if err != nil {
		return fmt.Errorf("initialize configuration manager: %w", err)
	}
	configHandler := handlers.NewConfigHandler(configManager, auditStore, webhookDispatcher)
	updaterHandler := handlers.NewUpdaterHandler()

	adminPerm := func(perm auth.Permission, h http.Handler) http.Handler {
		return middleware.RequireAdmin(jwtManager, middleware.RequirePermission(perm, h))
	}
	certPerm := func(perm auth.Permission, h http.Handler) http.Handler {
		return middleware.RequireServiceAccountOrAdmin(jwtManager, apiKeyStore, middleware.RequirePermission(perm, h))
	}
	renewPerm := func(h http.Handler) http.Handler {
		return middleware.RequireAdminOrMTLS(jwtManager, clientCertValidator, middleware.RequirePermission(auth.PermCertificatesRenew, h))
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", healthHandler)
	mux.Handle("GET /api/v1/ca/root", caHandler.RootCert())
	mux.Handle("GET /api/v1/ca/chain", caHandler.Chain())
	mux.Handle("GET /api/v1/ca/info", certPerm(auth.PermCertificatesRead, caHandler.Info()))
	mux.Handle("GET /api/v1/ca/provisioners", certPerm(auth.PermCertificatesRead, caHandler.Provisioners()))
	mux.Handle("GET /api/v1/ca/crl", caHandler.CRL())
	mux.Handle("GET /api/v1/crl", caHandler.CRL())
	mux.Handle("GET /api/v1/public/ca/intermediate", publicHandler.IntermediateCert())
	mux.Handle("GET /api/v1/public/certificates", publicHandler.ListCertificates())
	mux.Handle("GET /api/v1/public/certificates/{serial}", publicHandler.GetCertificate())
	mux.Handle("POST /ocsp", ocspHandler.Post())
	mux.Handle("GET /ocsp/{request}", ocspHandler.Get())
	mux.Handle("POST /api/v1/auth/login", authHandler.Login())
	mux.Handle("POST /api/v1/auth/logout", authHandler.Logout())
	mux.Handle("POST /api/v1/auth/service-accounts", adminPerm(auth.PermServiceAccounts, authHandler.CreateServiceAccount()))

	mux.Handle("POST /api/v1/certificates/issue", certPerm(auth.PermCertificatesIssue, certHandler.Issue()))
	mux.Handle("POST /api/v1/certificates/generate", certPerm(auth.PermCertificatesIssue, certHandler.Generate()))
	mux.Handle("POST /api/v1/certificates/issue-with-token", certPerm(auth.PermCertificatesIssue, certHandler.IssueWithToken()))
	mux.Handle("POST /api/v1/certificates/auto", adminPerm(auth.PermCertificatesIssue, certHandler.Auto()))
	mux.Handle("POST /api/v1/certificates/revoke", certPerm(auth.PermCertificatesRevoke, certHandler.Revoke()))
	mux.Handle("POST /api/v1/certificates/lint", certPerm(auth.PermCertificatesLint, certHandler.Lint()))
	mux.Handle("GET /api/v1/certificates/stats", certPerm(auth.PermCertificatesRead, statsHandler.CertificateStats()))
	mux.Handle("GET /api/v1/certificates", certPerm(auth.PermCertificatesRead, certHandler.List()))
	mux.Handle("GET /api/v1/certificates/{serial}", certPerm(auth.PermCertificatesRead, certHandler.GetBySerial()))
	mux.Handle("GET /api/v1/certificates/{serial}/key", adminPerm(auth.PermCertificatesRead, certHandler.GetPrivateKey()))
	mux.Handle("GET /api/v1/certificates/{serial}/bundle", adminPerm(auth.PermCertificatesRead, certHandler.DownloadBundle()))
	mux.Handle("POST /api/v1/provisioners/token", certPerm(auth.PermProvisionersToken, provisionerHandler.Token()))
	mux.Handle("GET /api/v1/k8s/status", certPerm(auth.PermEnrollmentStatus, provisionerHandler.K8sStatus()))
	mux.Handle("POST /api/v1/certificates/renew", renewPerm(renewalHandler.Renew()))
	mux.Handle("POST /api/v1/certificates/rekey", renewPerm(renewalHandler.Rekey()))
	mux.Handle("GET /api/v1/acme/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.Status()))
	mux.Handle("POST /api/v1/acme/eab-keys", certPerm(auth.PermACMEEAB, acmeHandler.CreateEABKey()))
	mux.Handle("GET /api/v1/scep/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.SCEPStatus()))
	mux.Handle("GET /api/v1/ndes/status", certPerm(auth.PermEnrollmentStatus, acmeHandler.NDESStatus()))

	mux.Handle("POST /api/v1/templates", certPerm(auth.PermTemplatesManage, templateHandler.Create()))
	mux.Handle("GET /api/v1/templates", certPerm(auth.PermTemplatesRead, templateHandler.List()))

	mux.Handle("POST /api/v1/ssh/generate/user", certPerm(auth.PermSSHSignUser, sshHandler.GenerateUser()))
	mux.Handle("POST /api/v1/ssh/generate/host", certPerm(auth.PermSSHSignHost, sshHandler.GenerateHost()))
	mux.Handle("POST /api/v1/ssh/sign-user", sshHandler.SignUser())
	mux.Handle("POST /api/v1/ssh/sign-host", certPerm(auth.PermSSHSignHost, sshHandler.SignHost()))
	mux.Handle("POST /api/v1/ssh/inspect", certPerm(auth.PermSSHInspect, sshHandler.Inspect()))
	mux.Handle("GET /api/v1/ssh/stats", certPerm(auth.PermSSHInspect, statsHandler.SSHStats()))
	mux.Handle("GET /api/v1/ssh/certificates", certPerm(auth.PermSSHInspect, sshHandler.ListCertificates()))
	mux.Handle("GET /api/v1/ssh/roots", sshHandler.Roots())
	mux.Handle("GET /api/v1/audit", certPerm(auth.PermAuditRead, auditHandler.List()))
	mux.Handle("GET /api/v1/notifications/stream", certPerm(auth.PermAuditRead, notificationHandler.Stream()))
	mux.Handle("GET /api/v1/notifications", adminPerm(auth.PermAuditRead, notificationHandler.List()))
	mux.Handle("POST /api/v1/notifications/read-all", adminPerm(auth.PermAuditRead, notificationHandler.MarkAllRead()))
	mux.Handle("POST /api/v1/notifications/archive-all", adminPerm(auth.PermAuditRead, notificationHandler.ArchiveAll()))
	mux.Handle("POST /api/v1/notifications/{id}/read", adminPerm(auth.PermAuditRead, notificationHandler.MarkRead()))
	mux.Handle("DELETE /api/v1/notifications/{id}", adminPerm(auth.PermAuditRead, notificationHandler.Delete()))
	mux.Handle("GET /api/v1/webhooks/events", adminPerm(auth.PermWebhooksManage, webhookHandler.Events()))
	mux.Handle("GET /api/v1/webhooks", adminPerm(auth.PermWebhooksManage, webhookHandler.List()))
	mux.Handle("POST /api/v1/webhooks", adminPerm(auth.PermWebhooksManage, webhookHandler.Create()))
	mux.Handle("PUT /api/v1/webhooks/{id}", adminPerm(auth.PermWebhooksManage, webhookHandler.Update()))
	mux.Handle("DELETE /api/v1/webhooks/{id}", adminPerm(auth.PermWebhooksManage, webhookHandler.Delete()))
	mux.Handle("POST /api/v1/webhooks/{id}/test", adminPerm(auth.PermWebhooksManage, webhookHandler.Test()))
	mux.Handle("GET /api/v1/settings/config", adminPerm(auth.PermSettingsManage, configHandler.Get()))
	mux.Handle("PUT /api/v1/settings/config", adminPerm(auth.PermSettingsManage, configHandler.Put()))
	mux.Handle("GET /api/v1/updater/current-changelog", adminPerm(auth.PermSettingsManage, updaterHandler.CurrentChangelog()))

	if pkiEngine.ACMEEnabled() {
		mux.Handle("/acme/", http.StripPrefix("/acme", pkiEngine.ACMEHandler()))
		log.Info("ACME enabled; directory available at /acme/directory")
	}
	if pkiEngine.SCEPEnabled() {
		mux.Handle("/scep/", http.StripPrefix("/scep", pkiEngine.SCEPHandler()))
		log.Info("SCEP enabled; endpoint available at /scep", slog.String("provisioner", pkiEngine.SCEPProvisionerName()))
	}
	if pkiEngine.NDESEnabled() {
		mux.Handle("/certsrv/", http.StripPrefix("/certsrv", pkiEngine.NDESHandler()))
		ca.LogNDESConnectors(pkiEngine.NDESRegistryRef())
	}

	notifications.RecordSystemEvent(auditStore, webhookDispatcher, auditdb.ActionSysStart, map[string]any{
		"listen_addr": listenAddr,
	})

	handler := middleware.Logger(mux)
	handler = middleware.Audit(auditStore, webhookDispatcher, handler)
	corsOpts := middleware.CORSOptions{
		AllowedOrigins:   serverCfg.WebUI.CORS.AllowedOrigins,
		AllowedMethods:   serverCfg.WebUI.CORS.AllowedMethods,
		AllowedHeaders:   serverCfg.WebUI.CORS.AllowedHeaders,
		AllowCredentials: serverCfg.WebUI.CORS.AllowCredentials,
	}
	handler = middleware.CORS(corsOpts, handler)
	handler = telemetry.HTTPMiddleware(handler)

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

	tlsConfig, err := arxserver.BuildAPITLSConfig(pkiEngine, serverCfg.Server.TLS)
	if err != nil {
		return fmt.Errorf("initialize API TLS: %w", err)
	}
	server.TLSConfig = tlsConfig

	certFile, keyFile, err := arxserver.ValidateAPITLSCredentials(serverCfg.Server)
	if err != nil {
		return err
	}

	var webUIServer *arxserver.WebUIServer
	if serverCfg.WebUI.Enabled {
		var apiUpstream *url.URL
		if serverCfg.WebUI.ProxyAPIEnabled() {
			apiUpstream, err = arxserver.BuildAPIUpstreamURL(serverCfg)
			if err != nil {
				return fmt.Errorf("resolve WebUI API upstream: %w", err)
			}
			log.Info("WebUI API reverse proxy enabled", slog.String("upstream", apiUpstream.String()))
		}
		webUIServer, err = arxserver.NewWebUIServer(serverCfg.WebUI, apiUpstream, log)
		if err != nil {
			return fmt.Errorf("initialize WebUI server: %w", err)
		}
		webUIServer.Start()
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	restartCh := make(chan struct{}, 1)
	if serverCfg.Updater.Enabled {
		updateEngine := updater.NewEngine(serverCfg.Updater, auditStore, webhookDispatcher, func() {
			select {
			case restartCh <- struct{}{}:
			default:
			}
		})
		go updateEngine.Run(serverCtx)
	}

	errCh := make(chan error, 1)
	go func() {
		if serverCfg.Server.TLS.Enabled {
			log.Info("arx server listening (TLS)", slog.String("address", listenAddr), slog.String("log_level", serverCfg.Server.LogLevel))
			if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
			return
		}
		log.Info("arx server listening", slog.String("address", listenAddr), slog.String("log_level", serverCfg.Server.LogLevel))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	restartAfterShutdown := false

	select {
	case err := <-errCh:
		serverCancel()
		return err
	case sig := <-sigCh:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))
		serverCancel()
	case <-restartCh:
		log.Info("updater: graceful restart scheduled after binary update")
		restartAfterShutdown = true
		serverCancel()
	}

	if webUIServer != nil {
		webShutdownTimeout := serverCfg.Server.WriteTimeout
		if wt, err := serverCfg.WebUI.WriteTimeoutDuration(); err == nil {
			webShutdownTimeout = wt
		}
		webCtx, webCancel := context.WithTimeout(context.Background(), webShutdownTimeout)
		if err := webUIServer.Shutdown(webCtx); err != nil {
			webCancel()
			return err
		}
		webCancel()
	}

	apiCtx, apiCancel := context.WithTimeout(context.Background(), serverCfg.Server.WriteTimeout)
	defer apiCancel()
	if err := server.Shutdown(apiCtx); err != nil {
		return err
	}
	log.Info("server stopped")

	if restartAfterShutdown {
		if err := updater.RestartExecutable(); err != nil {
			return fmt.Errorf("restart after update: %w", err)
		}
	}

	return nil
}
