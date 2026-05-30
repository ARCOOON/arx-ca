package ca

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
)

func needsBadgerTruncate(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Value log truncate required")
}

func healBadgerDB(dataSource string) error {
	path := strings.TrimSpace(dataSource)
	if path == "" {
		return fmt.Errorf("badger data source is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	opts := badger.DefaultOptions(absPath)
	opts.Truncate = true

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("open badger with truncate: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close badger: %w", err)
	}

	log.Printf("ca: healed badger database at %s (value log truncated)", absPath)
	return nil
}
