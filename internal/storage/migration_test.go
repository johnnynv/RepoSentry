package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrationManager_Basic(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	manager := NewMigrationManager(db)
	ctx := context.Background()

	// Test ensuring migrations table
	err = manager.ensureMigrationsTable(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure migrations table: %v", err)
	}

	// Check current version (should be 0)
	version, err := manager.getCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}
	if version != 0 {
		t.Errorf("Expected version 0, got %d", version)
	}

	// Test SQL splitting
	testSQL := `
		-- Create test table
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		
		-- Insert test data
		INSERT INTO test_table (name) VALUES ('test');
	`
	
	statements := manager.splitSQL(testSQL)
	t.Logf("Split SQL into %d statements", len(statements))
	for i, stmt := range statements {
		t.Logf("Statement %d: %s", i, stmt)
	}

	// Should have 2 statements (CREATE and INSERT)
	if len(statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(statements))
	}
}

func TestMigrationManager_ApplyMigrations(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_apply_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	manager := NewMigrationManager(db)
	ctx := context.Background()

	// Apply all migrations
	err = manager.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Check that tables exist
	tables := []string{"repository_states", "events", "schema_migrations"}
	for _, table := range tables {
		var exists int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if exists != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}

	// Check migration records
	applied, err := manager.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}

	expectedMigrations := 3 // We have 3 migrations (including error_message column)
	if len(applied) != expectedMigrations {
		t.Errorf("Expected %d applied migrations, got %d", expectedMigrations, len(applied))
	}

	// Verify we can insert and retrieve data
	testRepoState := `
		INSERT INTO repository_states (repository, branch, commit_sha, last_checked, created_at, updated_at)
		VALUES ('test/repo', 'main', 'abc123', ?, ?, ?)
	`
	now := time.Now()
	_, err = db.ExecContext(ctx, testRepoState, now, now, now)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM repository_states").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count repository states: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 repository state, got %d", count)
	}
}
