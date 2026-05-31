package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	arxconfig "github.com/your-org/arx-ca/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open connects to the application user store (SQLite by default, PostgreSQL when configured).
func Open(cfg arxconfig.ServerConfig) (*sql.DB, error) {
	driver, dsn, err := resolveDriverAndDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping application database: %w", err)
	}
	return db, nil
}

func resolveDriverAndDSN(cfg arxconfig.ServerConfig) (driver, dsn string, err error) {
	dsn = strings.TrimSpace(cfg.DBDataSource)
	dbType := strings.ToLower(strings.TrimSpace(cfg.DBType))

	if dsn != "" {
		if dbType == "postgresql" || dbType == "postgres" || strings.HasPrefix(strings.ToLower(dsn), "postgres") {
			return "pgx", dsn, nil
		}
		if strings.HasPrefix(dsn, "file:") || strings.HasSuffix(strings.ToLower(dsn), ".db") {
			return "sqlite", sqliteDSN(dsn), nil
		}
		return "", "", fmt.Errorf("unsupported db_data_source for application database")
	}

	path, err := defaultSQLitePath(cfg)
	if err != nil {
		return "", "", err
	}
	return "sqlite", sqliteDSN(path), nil
}

func sqliteDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)"
}

func defaultSQLitePath(cfg arxconfig.ServerConfig) (string, error) {
	caPath := strings.TrimSpace(cfg.CAConfigPath)
	if caPath == "" {
		caPath = arxconfig.DefaultServerConfig().CAConfigPath
	}
	base := filepath.Dir(filepath.Dir(caPath))
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("create application database directory: %w", err)
	}
	return filepath.Join(base, "arx-ca-users.db"), nil
}
