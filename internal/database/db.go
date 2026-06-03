package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/logging"

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
	maxOpen := cfg.Database.MaxOpenConns
	maxIdle := cfg.Database.MaxIdleConns
	if cfg.Database.UsesSQLite() {
		// WAL mode allows concurrent readers; use a small pool so ACME and API
		// handlers do not serialize on a single connection.
		if maxOpen <= 0 || maxOpen > 4 {
			maxOpen = 4
		}
		if maxIdle <= 0 {
			maxIdle = 2
		}
	}
	if maxOpen > 0 {
		db.SetMaxOpenConns(maxOpen)
	}
	if maxIdle > 0 {
		db.SetMaxIdleConns(maxIdle)
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
		logging.Logger().Warn("database not ready, retrying",
			slog.Duration("delay", connectRetryDelay),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", maxConnectAttempts),
			slog.Any("error", lastErr),
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
	switch cfg.Database.EffectiveDriver() {
	case "postgres":
		dsn = cfg.Database.DSN()
		if dsn == "" {
			return "", "", "", fmt.Errorf("postgresql driver selected but connection parameters are incomplete")
		}
		return "pgx", dsn, databaseEndpoint(cfg.Database), nil
	case "sqlite":
		path, err := prepareSQLiteDatabase(cfg.Database)
		if err != nil {
			return "", "", "", err
		}
		return "sqlite", sqliteDSN(path), path, nil
	default:
		return "", "", "", fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
}

func prepareSQLiteDatabase(db arxconfig.DatabaseConfig) (string, error) {
	path, err := db.ResolvedSQLitePath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create sqlite database directory %s: %w", dir, err)
	}
	return path, nil
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
		if strings.Contains(path, "_pragma=") {
			return path
		}
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + sqlitePragmasQuery()
	}
	return "file:" + filepath.ToSlash(path) + "?" + sqlitePragmasQuery()
}

func sqlitePragmasQuery() string {
	return "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
}
