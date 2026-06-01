package database

import (
	"database/sql"
	"testing"

	"github.com/your-org/arx-ca/internal/auth"
	"github.com/your-org/arx-ca/internal/config"

	_ "modernc.org/sqlite"
)

func TestSeedInitialAdminCreatesUserOnce(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	cfg := config.Bootstrap{
		AdminEmail:        "admin@arx.local",
		AdminPasswordHash: "$2a$10$dSttx8r7tN32Mbo/C3zOteNowfq2vyhloZndZ2OGBgFEcMl1QYj0a",
	}
	if err := SeedInitialAdmin(db, cfg); err != nil {
		t.Fatalf("SeedInitialAdmin first call: %v", err)
	}

	var email, role string
	if err := db.QueryRow(`SELECT email, role FROM users LIMIT 1`).Scan(&email, &role); err != nil {
		t.Fatalf("select seeded user: %v", err)
	}
	if email != cfg.AdminEmail {
		t.Fatalf("email = %q, want %q", email, cfg.AdminEmail)
	}
	if role != string(auth.RoleSuperAdmin) {
		t.Fatalf("role = %q, want %q", role, auth.RoleSuperAdmin)
	}

	if err := SeedInitialAdmin(db, config.Bootstrap{
		AdminEmail:        "other@arx.local",
		AdminPasswordHash: "$2a$10$otherhashotherhashotherhashotherhashotherhashotherhash",
	}); err != nil {
		t.Fatalf("SeedInitialAdmin second call: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1 (idempotent seed)", count)
	}
}
