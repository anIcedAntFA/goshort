package storage_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/storage"
	_ "modernc.org/sqlite"
)

// TestSQLiteStorage_GooseMigrationApplied verifies that goose migrations are applied
// and tracked in the goose_db_version table after NewSQLiteStorage.
func TestSQLiteStorage_GooseMigrationApplied(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := storage.NewSQLiteStorage(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Open the same file-based DB to verify goose applied the migration.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close() //nolint:errcheck

	var version int64
	err = db.QueryRowContext(
		context.Background(),
		"SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1",
	).Scan(&version)
	if err != nil {
		t.Fatalf("query goose_db_version: %v (table likely missing — goose not yet wired)", err)
	}
	if version != 1 {
		t.Errorf("applied migration version = %d, want 1", version)
	}
}

// TestSQLiteStorage_MigrationIdempotent verifies that opening the same database
// twice (running migrations twice) does not fail.
func TestSQLiteStorage_MigrationIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s1, err := storage.NewSQLiteStorage(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("first NewSQLiteStorage: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close s1: %v", err)
	}

	s2, err := storage.NewSQLiteStorage(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("second NewSQLiteStorage (idempotent): %v", err)
	}
	if err := s2.Close(); err != nil {
		t.Fatalf("Close s2: %v", err)
	}
}
