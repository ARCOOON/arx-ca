package config

import "context"

// StorageBackendType identifies where certificate metadata is persisted.
type StorageBackendType string

const (
	StorageBackendLocal      StorageBackendType = "local"
	StorageBackendPostgreSQL StorageBackendType = "postgresql"
)

// StorageBackend abstracts the certificate database. Local storage uses Badger/bbolt via step-ca.
type StorageBackend interface {
	Type() StorageBackendType
	Healthy(ctx context.Context) error
}

// LocalStorageBackend uses embedded Badger/bbolt under the PKI directory.
type LocalStorageBackend struct{}

func (LocalStorageBackend) Type() StorageBackendType { return StorageBackendLocal }

func (LocalStorageBackend) Healthy(context.Context) error { return nil }

// PostgreSQLStorageBackend is a placeholder for production PostgreSQL wiring through step-ca db config.
type PostgreSQLStorageBackend struct {
	DSN string
}

func (b PostgreSQLStorageBackend) Type() StorageBackendType { return StorageBackendPostgreSQL }

func (b PostgreSQLStorageBackend) Healthy(context.Context) error { return nil }

// NewStorageBackend returns the configured StorageBackend implementation.
func NewStorageBackend(cfg Config) StorageBackend {
	switch cfg.Store.Backend {
	case StorageBackendPostgreSQL:
		return PostgreSQLStorageBackend{DSN: cfg.Store.DSN}
	default:
		return LocalStorageBackend{}
	}
}
