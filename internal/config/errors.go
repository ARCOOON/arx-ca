package config

import "errors"

// ErrIntegrationDisabled is returned by placeholder cloud/Vault clients when the integration is not active.
var ErrIntegrationDisabled = errors.New("integration is disabled; using local cryptography and storage")
