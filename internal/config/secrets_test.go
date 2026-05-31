package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSecretUsesValueWhenNoFile(t *testing.T) {
	if got := ResolveSecret("inline-secret", ""); got != "inline-secret" {
		t.Fatalf("ResolveSecret() = %q, want inline-secret", got)
	}
}

func TestResolveSecretReadsFileWhenSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte("  file-secret  \n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	if got := ResolveSecret("inline-secret", path); got != "file-secret" {
		t.Fatalf("ResolveSecret() = %q, want file-secret", got)
	}
}

func TestResolveSecretFallsBackToValueOnReadError(t *testing.T) {
	if got := ResolveSecret("fallback", filepath.Join(t.TempDir(), "missing.txt")); got != "fallback" {
		t.Fatalf("ResolveSecret() = %q, want fallback", got)
	}
}

func TestDatabaseDSNUsesPasswordFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db-pass.txt")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	dsn := DatabaseConfig{
		Host:         "db.example.com",
		Port:         5432,
		User:         "arx",
		Password:     "ignored",
		PasswordFile: path,
		DBName:       "arx_ca",
		SSLMode:      "require",
	}.DSN()
	if !strings.Contains(dsn, "from-file") {
		t.Fatalf("expected DSN to use password file contents, got %s", dsn)
	}
}
