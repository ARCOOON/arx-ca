package handlers

import (
	"context"
	"fmt"

	"github.com/ARCOOON/arx-ca/internal/auth"
	"github.com/ARCOOON/arx-ca/internal/repository"
)

// ResolveRequestorID returns the application user ID or service account ID for the authenticated caller.
func ResolveRequestorID(ctx context.Context, users *repository.UserStore) (string, error) {
	if account, ok := auth.ServiceAccountFromContext(ctx); ok && account != nil {
		if id := account.ID; id != "" {
			return id, nil
		}
		return account.Name, nil
	}

	email, ok := auth.AdminUsernameFromContext(ctx)
	if !ok || email == "" {
		return "", fmt.Errorf("authenticated requestor not found")
	}

	if users != nil {
		user, err := users.GetByEmail(ctx, email)
		if err == nil && user != nil && user.ID != "" {
			return user.ID, nil
		}
	}

	return email, nil
}
