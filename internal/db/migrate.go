package db

import (
	"database/sql"
	"fmt"
)

func migrateNotificationsArchive(db *sql.DB) error {
	if !tableColumnExists(db, "notifications", "is_archived") {
		if _, err := db.Exec(`ALTER TABLE notifications ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add is_archived column: %w", err)
		}
	}

	ddl, err := migrationFS.ReadFile("migrations/004_notifications_archive.sql")
	if err != nil {
		return fmt.Errorf("read notifications_archive migration: %w", err)
	}
	if _, err := db.Exec(string(ddl)); err != nil {
		return fmt.Errorf("apply notifications_archive migration: %w", err)
	}
	return nil
}

func tableColumnExists(db *sql.DB, tableName, columnName string) bool {
	if isPostgreSQL(db) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = CURRENT_SCHEMA()
					AND table_name = $1
					AND column_name = $2
			)`,
			tableName,
			columnName,
		).Scan(&exists)
		return err == nil && exists
	}

	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue any
			pk           int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}
