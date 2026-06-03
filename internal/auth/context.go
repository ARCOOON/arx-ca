package auth

import (
	"context"
	"crypto/x509"
)

type contextKey int

const (
	contextKeyAdminUsername contextKey = iota + 1
	contextKeyServiceAccount
	contextKeyRoles
	contextKeyMTLSAuthenticated
	contextKeyMTLSCommonName
	contextKeyMTLSCertificate
)

// WithAdminUsername stores the authenticated admin username in the request context.
func WithAdminUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, contextKeyAdminUsername, username)
}

// AdminUsernameFromContext returns the admin username set by RequireAdmin middleware.
func AdminUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(contextKeyAdminUsername).(string)
	return username, ok
}

// WithServiceAccount stores the authenticated service account in the request context.
func WithServiceAccount(ctx context.Context, account *ServiceAccount) context.Context {
	return context.WithValue(ctx, contextKeyServiceAccount, account)
}

// ServiceAccountFromContext returns the service account set by RequireServiceAccount middleware.
func ServiceAccountFromContext(ctx context.Context) (*ServiceAccount, bool) {
	account, ok := ctx.Value(contextKeyServiceAccount).(*ServiceAccount)
	return account, ok
}

// WithRoles stores RBAC roles in the request context.
func WithRoles(ctx context.Context, roles []Role) context.Context {
	return context.WithValue(ctx, contextKeyRoles, NormalizeRoles(roles))
}

// RolesFromContext returns roles set by authentication middleware.
func RolesFromContext(ctx context.Context) ([]Role, bool) {
	roles, ok := ctx.Value(contextKeyRoles).([]Role)
	if !ok || len(roles) == 0 {
		return nil, false
	}
	out := make([]Role, len(roles))
	copy(out, roles)
	return out, true
}

// WithMTLSAuthentication marks the request context as authenticated via mutual TLS.
func WithMTLSAuthentication(ctx context.Context, cert *x509.Certificate) context.Context {
	cn := ""
	if cert != nil {
		cn = cert.Subject.CommonName
	}
	ctx = context.WithValue(ctx, contextKeyMTLSAuthenticated, true)
	ctx = context.WithValue(ctx, contextKeyMTLSCommonName, cn)
	ctx = context.WithValue(ctx, contextKeyMTLSCertificate, cert)
	return ctx
}

// MTLSAuthenticatedFromContext reports whether the client authenticated with a valid mTLS certificate.
func MTLSAuthenticatedFromContext(ctx context.Context) bool {
	authenticated, ok := ctx.Value(contextKeyMTLSAuthenticated).(bool)
	return ok && authenticated
}

// MTLSCommonNameFromContext returns the common name from the client mTLS certificate.
func MTLSCommonNameFromContext(ctx context.Context) (string, bool) {
	cn, ok := ctx.Value(contextKeyMTLSCommonName).(string)
	if !ok || cn == "" {
		return "", false
	}
	return cn, true
}

// MTLSCertificateFromContext returns the validated client certificate presented over mTLS.
func MTLSCertificateFromContext(ctx context.Context) (*x509.Certificate, bool) {
	cert, ok := ctx.Value(contextKeyMTLSCertificate).(*x509.Certificate)
	return cert, ok && cert != nil
}
