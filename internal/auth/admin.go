package auth

import "errors"

var (
	// ErrInvalidCredentials is returned when admin email or password is wrong.
	ErrInvalidCredentials = errors.New("invalid admin credentials")
)
