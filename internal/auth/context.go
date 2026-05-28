package auth

import "context"

type contextKey int

const (
	contextKeyAdminUsername contextKey = iota + 1
	contextKeyServiceAccount
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
