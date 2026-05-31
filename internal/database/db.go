package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	arxconfig "github.com/your-org/arx-ca/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const (
	maxConnectAttempts = 5
	connectRetryDelay  = 3 * time.Second
)

// Open connects to the application user store (SQLite by default, PostgreSQL when configured).
// The initial ping is retried up to maxConnectAttempts times with connectRetryDelay between attempts.
func Open(cfg arxconfig.ServerConfig) (*sql.DB, error) {
	driver, dsn, endpoint, err := resolveDriverAndDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}
	if cfg.Database.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	}
	if err := pingWithRetry(context.Background(), db, endpoint); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func pingWithRetry(ctx context.Context, db *sql.DB, endpoint string) error {
	var lastErr error
	for attempt := 1; attempt <= maxConnectAttempts; attempt++ {
		lastErr = db.PingContext(ctx)
		if lastErr == nil {
			return nil
		}
		if attempt >= maxConnectAttempts {
			break
		}
		log.Printf(
			"WARNING: Database not ready, retrying in %s... (Attempt %d/%d): %v",
			connectRetryDelay,
			attempt,
			maxConnectAttempts,
			lastErr,
		)
		if err := waitForRetry(ctx, connectRetryDelay); err != nil {
			return err
		}
	}
	return fmt.Errorf(
		"application database is unreachable at %s after %d connection attempts: %w",
		endpoint,
		maxConnectAttempts,
		lastErr,
	)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("database connection retry interrupted: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func resolveDriverAndDSN(cfg arxconfig.ServerConfig) (driver, dsn, endpoint string, err error) {
	if cfg.Database.UsesPostgreSQL() {
		dsn = cfg.Database.DSN()
		if dsn == "" {
			return "", "", "", fmt.Errorf("postgresql database host is set but connection parameters are incomplete")
		}
		return "pgx", dsn, databaseEndpoint(cfg.Database), nil
	}

	path, err := defaultSQLitePath(cfg)
	if err != nil {
		return "", "", "", err
	}
	return "sqlite", sqliteDSN(path), path, nil
}

func databaseEndpoint(db arxconfig.DatabaseConfig) string {
	port := db.Port
	if port <= 0 {
		port = arxconfig.DefaultServerConfig().Database.Port
	}
	return fmt.Sprintf("%s:%d", strings.TrimSpace(db.Host), port)
}

func sqliteDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)"
}

func defaultSQLitePath(cfg arxconfig.ServerConfig) (string, error) {
	caPath := cfg.CA.ConfigPath()
	base := filepath.Dir(filepath.Dir(caPath))
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("create application database directory: %w", err)
	}
	return filepath.Join(base, "arx-ca-users.db"), nil
}
