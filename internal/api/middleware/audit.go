package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/events"
	"github.com/google/uuid"
)

const headerXForwardedFor = "X-Forwarded-For"
const headerXRealIP = "X-Real-IP"
const headerXRequestID = "X-Request-ID"

type auditContextKey struct{}

// AuditContext carries forensic business metadata handlers attach before the middleware persists the log entry.
type AuditContext struct {
	Action      string
	Provisioner string
	Fingerprint string
	Metadata    map[string]any
	ActorType   string
	ActorID     string
	ActorRoles  []string
	Skip        bool
}

// SetAction records the business action performed by the handler.
func (a *AuditContext) SetAction(action string) {
	if a == nil {
		return
	}
	a.Action = strings.TrimSpace(action)
}

// SetProvisioner records the CA provisioner involved in the operation.
func (a *AuditContext) SetProvisioner(provisioner string) {
	if a == nil {
		return
	}
	a.Provisioner = strings.TrimSpace(provisioner)
}

// SetFingerprint records a certificate SHA-256 fingerprint (hex-encoded).
func (a *AuditContext) SetFingerprint(fingerprint string) {
	if a == nil {
		return
	}
	a.Fingerprint = strings.TrimSpace(fingerprint)
}

// SetFingerprintFromPEM derives the SHA-256 fingerprint from a PEM-encoded certificate.
func (a *AuditContext) SetFingerprintFromPEM(certPEM string) {
	if a == nil {
		return
	}
	if fp, err := CertificateFingerprintSHA256(certPEM); err == nil {
		a.Fingerprint = fp
	}
}

// SetActor overrides the resolved actor when authentication context is unavailable (e.g. login).
func (a *AuditContext) SetActor(actorType, actorID string, roles ...string) {
	if a == nil {
		return
	}
	a.ActorType = strings.TrimSpace(actorType)
	a.ActorID = strings.TrimSpace(actorID)
	if len(roles) > 0 {
		a.ActorRoles = append([]string(nil), roles...)
	}
}

// PutMetadata adds a key/value pair to the extended metadata map.
func (a *AuditContext) PutMetadata(key string, value any) {
	if a == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if a.Metadata == nil {
		a.Metadata = make(map[string]any)
	}
	a.Metadata[key] = value
}

// AuditFromContext returns the AuditContext injected by Audit middleware.
func AuditFromContext(ctx context.Context) *AuditContext {
	ac, _ := ctx.Value(auditContextKey{}).(*AuditContext)
	return ac
}

type auditResponseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *auditResponseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Audit returns middleware that captures network context, injects AuditContext, and emits audit events.
func Audit(eventManager *events.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if eventManager == nil {
			next.ServeHTTP(w, r)
			return
		}

		if shouldSkipAudit(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		requestID := uuid.NewString()
		w.Header().Set(headerXRequestID, requestID)

		auditCtx := &AuditContext{Metadata: make(map[string]any)}
		ctx := context.WithValue(r.Context(), auditContextKey{}, auditCtx)
		r = r.WithContext(ctx)

		userAgent := strings.TrimSpace(r.UserAgent())
		auditCtx.PutMetadata("user_agent", userAgent)

		recorder := &auditResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		started := time.Now().UTC()

		next.ServeHTTP(recorder, r)

		if auditCtx.Skip || isReadOnlyHTTPMethod(r.Method) {
			return
		}

		action := auditCtx.Action
		if action == "" {
			action = defaultActionForRequest(r.Method)
		}

		entry := db.AuditLog{
			Timestamp:   started,
			RequestID:   requestID,
			IPAddress:   clientIP(r),
			HTTPMethod:  r.Method,
			Endpoint:    r.URL.Path,
			StatusCode:  recorder.statusCode,
			Action:      action,
			Provisioner: auditCtx.Provisioner,
			Fingerprint: auditCtx.Fingerprint,
			Metadata:    cloneMetadata(auditCtx.Metadata),
		}

		resolveActor(r.Context(), auditCtx, &entry)

		eventManager.Trigger(events.EventAuditRecorded, events.PayloadAuditRecorded(
			entry.Action,
			entry.RequestID,
			entry.IPAddress,
			entry.HTTPMethod,
			entry.Endpoint,
			entry.StatusCode,
			entry.ActorType,
			entry.ActorID,
			entry.ActorRoles,
			entry.Provisioner,
			entry.Fingerprint,
			cloneMetadata(auditCtx.Metadata),
		))
		emitCertificateEvent(eventManager, action, auditCtx)
	})
}

func emitCertificateEvent(eventManager *events.Manager, action string, auditCtx *AuditContext) {
	if eventManager == nil || auditCtx == nil {
		return
	}

	var eventName string
	switch action {
	case db.ActionCertIssueNative:
		eventName = events.EventCertIssuedNative
	case db.ActionCertIssueCSR:
		eventName = events.EventCertIssuedCSR
	case "CERT_AUTO":
		eventName = events.EventCertIssuedAuto
	case db.ActionCertRevoke:
		eventName = events.EventCertRevoked
	case db.ActionCertRenew:
		eventName = events.EventCertRenewed
	case "CERT_REKEY":
		eventName = events.EventCertRekeyed
	default:
		return
	}

	serial := stringMetadata(auditCtx.Metadata, "serial")
	alias := stringMetadata(auditCtx.Metadata, "alias")
	customID := stringMetadata(auditCtx.Metadata, "custom_id")
	eventManager.Trigger(eventName, events.PayloadCertIssued(serial, alias, customID, auditCtx.Provisioner, auditCtx.Fingerprint))
}

func stringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func shouldSkipAudit(path string) bool {
	path = strings.TrimSpace(path)
	switch path {
	case "/api/v1/health", "/api/v1/notifications/stream":
		return true
	default:
		return false
	}
}

func isReadOnlyHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodOptions:
		return true
	default:
		return false
	}
}

func defaultActionForRequest(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch {
	case method == http.MethodPost:
		return "HTTP_WRITE"
	case method == http.MethodPut, method == http.MethodPatch:
		return "HTTP_UPDATE"
	case method == http.MethodDelete:
		return "HTTP_DELETE"
	default:
		return "HTTP_" + method
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get(headerXForwardedFor); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get(headerXRealIP)); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func resolveActor(ctx context.Context, auditCtx *AuditContext, entry *db.AuditLog) {
	if entry == nil {
		return
	}

	if auditCtx != nil && auditCtx.ActorID != "" {
		entry.ActorType = auditCtx.ActorType
		if entry.ActorType == "" {
			entry.ActorType = "User"
		}
		entry.ActorID = auditCtx.ActorID
		if len(auditCtx.ActorRoles) > 0 {
			entry.ActorRoles = append([]string(nil), auditCtx.ActorRoles...)
		}
		return
	}

	if username, ok := auth.AdminUsernameFromContext(ctx); ok && username != "" {
		entry.ActorType = "User"
		entry.ActorID = username
		entry.ActorRoles = roleNamesFromContext(ctx)
		return
	}

	if account, ok := auth.ServiceAccountFromContext(ctx); ok && account != nil {
		entry.ActorType = "ServiceAccount"
		entry.ActorID = account.ID
		if entry.ActorID == "" {
			entry.ActorID = account.Name
		}
		entry.ActorRoles = roleNamesFromContext(ctx)
		return
	}

	if auth.MTLSAuthenticatedFromContext(ctx) {
		entry.ActorType = "System"
		if cn, ok := auth.MTLSCommonNameFromContext(ctx); ok && cn != "" {
			entry.ActorID = cn
		} else {
			entry.ActorID = "mTLS"
		}
		return
	}

	entry.ActorType = "System"
	entry.ActorID = "anonymous"
}

func roleNamesFromContext(ctx context.Context) []string {
	roles, ok := auth.RolesFromContext(ctx)
	if !ok || len(roles) == 0 {
		return nil
	}
	out := make([]string, len(roles))
	for i, role := range roles {
		out[i] = string(role)
	}
	return out
}

func cloneMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	raw, err := json.Marshal(src)
	if err != nil {
		dup := make(map[string]any, len(src))
		for k, v := range src {
			dup[k] = v
		}
		return dup
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

// CertificateFingerprintSHA256 returns the lowercase hex SHA-256 fingerprint of a PEM-encoded X.509 certificate.
func CertificateFingerprintSHA256(certPEM string) (string, error) {
	certPEM = strings.TrimSpace(certPEM)
	if certPEM == "" {
		return "", nil
	}
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}
