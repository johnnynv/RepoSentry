package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Name        string
	Description string
	Up          string
	Down        string
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db *sql.DB
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db}
}

// GetMigrations returns all available migrations
func (m *MigrationManager) GetMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Name:        "initial_schema",
			Description: "Create initial tables for repository states and events",
			Up: `
				-- Create repository_states table
				CREATE TABLE IF NOT EXISTS repository_states (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					repository TEXT NOT NULL,
					branch TEXT NOT NULL,
					commit_sha TEXT NOT NULL,
					last_checked DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(repository, branch)
				);

				-- Create events table
				CREATE TABLE IF NOT EXISTS events (
					id TEXT PRIMARY KEY,
					type TEXT NOT NULL,
					repository TEXT NOT NULL,
					branch TEXT NOT NULL,
					commit_sha TEXT NOT NULL,
					prev_commit TEXT,
					provider TEXT NOT NULL,
					timestamp DATETIME NOT NULL,
					metadata TEXT,
					status TEXT NOT NULL DEFAULT 'pending',
					processed_at DATETIME,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
			`,
			Down: `
				DROP TABLE IF EXISTS events;
				DROP TABLE IF EXISTS repository_states;
			`,
		},
		{
			Version:     2,
			Name:        "add_indexes",
			Description: "Add performance indexes",
			Up: `
				-- Indexes for repository_states
				CREATE INDEX IF NOT EXISTS idx_repository_states_repository ON repository_states(repository);
				CREATE INDEX IF NOT EXISTS idx_repository_states_last_checked ON repository_states(last_checked);

				-- Indexes for events
				CREATE INDEX IF NOT EXISTS idx_events_repository ON events(repository);
				CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
				CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
				CREATE INDEX IF NOT EXISTS idx_events_processed_at ON events(processed_at);
				CREATE INDEX IF NOT EXISTS idx_events_repository_status ON events(repository, status);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_events_repository_status;
				DROP INDEX IF EXISTS idx_events_processed_at;
				DROP INDEX IF EXISTS idx_events_timestamp;
				DROP INDEX IF EXISTS idx_events_status;
				DROP INDEX IF EXISTS idx_events_repository;
				DROP INDEX IF EXISTS idx_repository_states_last_checked;
				DROP INDEX IF EXISTS idx_repository_states_repository;
			`,
		},

		// Migration 3: Add error_message column to events table
		{
			Version:     3,
			Name:        "add_error_message_column",
			Description: "Add error_message column to events table for better error tracking",
			Up: `
				ALTER TABLE events ADD COLUMN error_message TEXT;
				CREATE INDEX IF NOT EXISTS idx_events_status_error ON events(status) WHERE error_message IS NOT NULL;
			`,
			Down: `
				DROP INDEX IF EXISTS idx_events_status_error;
				-- SQLite doesn't support DROP COLUMN, so we'd need to recreate the table
				-- For now, just leave the column but don't use it
			`,
		},

	}
}

// Migrate runs all pending migrations
func (m *MigrationManager) Migrate(ctx context.Context) error {
	// Ensure schema_migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	migrations := m.GetMigrations()
	
	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", 
				migration.Version, migration.Name, err)
		}
	}

	return nil
}

// Rollback rolls back to a specific version
func (m *MigrationManager) Rollback(ctx context.Context, targetVersion int) error {
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d is not less than current version %d", 
			targetVersion, currentVersion)
	}

	migrations := m.GetMigrations()
	
	// Apply rollbacks in reverse order
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := migrations[i]
		if migration.Version <= targetVersion {
			break
		}
		if migration.Version > currentVersion {
			continue
		}

		if err := m.rollbackMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to rollback migration %d (%s): %w", 
				migration.Version, migration.Name, err)
		}
	}

	return nil
}

// GetCurrentVersion returns the current schema version
func (m *MigrationManager) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"
	
	err := m.db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		return 0, err
	}
	
	return version, nil
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *MigrationManager) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// applyMigration applies a single migration
func (m *MigrationManager) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	statements := m.splitSQL(migration.Up)
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %s: %w", stmt, err)
		}
	}

	// Record migration
	_, err = tx.ExecContext(ctx, 
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (m *MigrationManager) rollbackMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute rollback SQL
	statements := m.splitSQL(migration.Down)
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute rollback statement: %s: %w", stmt, err)
		}
	}

	// Remove migration record
	_, err = tx.ExecContext(ctx, 
		"DELETE FROM schema_migrations WHERE version = ?", migration.Version)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// splitSQL splits SQL script into individual statements
func (m *MigrationManager) splitSQL(sql string) []string {
	// Remove leading/trailing whitespace
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []string{}
	}
	
	// Split by semicolon
	statements := strings.Split(sql, ";")
	var result []string
	
	for _, stmt := range statements {
		// Clean up the statement
		stmt = strings.TrimSpace(stmt)
		
		// Skip empty statements and comments
		if stmt == "" {
			continue
		}
		
		// Remove SQL comments (-- style)
		lines := strings.Split(stmt, "\n")
		var cleanLines []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "--") {
				cleanLines = append(cleanLines, line)
			}
		}
		
		if len(cleanLines) > 0 {
			cleanStmt := strings.Join(cleanLines, " ")
			cleanStmt = strings.TrimSpace(cleanStmt)
			if cleanStmt != "" {
				result = append(result, cleanStmt)
			}
		}
	}
	
	return result
}

// GetAppliedMigrations returns list of applied migrations
func (m *MigrationManager) GetAppliedMigrations(ctx context.Context) ([]AppliedMigration, error) {
	query := `
		SELECT version, name, applied_at 
		FROM schema_migrations 
		ORDER BY version
	`
	
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []AppliedMigration
	for rows.Next() {
		var migration AppliedMigration
		err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

// AppliedMigration represents an applied migration
type AppliedMigration struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
}
