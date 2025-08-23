package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// SQLiteStorage implements Storage interface using SQLite
type SQLiteStorage struct {
	db               *sql.DB
	config           *types.SQLiteConfig
	migrationManager *MigrationManager
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(config *types.SQLiteConfig) (*SQLiteStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("SQLite config is required")
	}

	// Ensure directory exists (skip for in-memory database)
	if config.Path != ":memory:" {
		dir := filepath.Dir(config.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database
	db, err := sql.Open("sqlite3", config.Path+"?_journal_mode=WAL&_foreign_keys=1&_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConnections)
	db.SetMaxIdleConns(config.MaxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)

	storage := &SQLiteStorage{
		db:               db,
		config:           config,
		migrationManager: NewMigrationManager(db),
	}

	return storage, nil
}

// Initialize initializes the database and runs migrations
func (s *SQLiteStorage) Initialize(ctx context.Context) error {
	// Test connection
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := s.migrationManager.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// HealthCheck checks if the database is accessible
func (s *SQLiteStorage) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// SaveRepoState saves or updates a repository state
func (s *SQLiteStorage) SaveRepoState(ctx context.Context, state *types.RepoState) error {
	query := `
		INSERT INTO repository_states (repository, branch, commit_sha, last_checked, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository, branch) DO UPDATE SET
			commit_sha = excluded.commit_sha,
			last_checked = excluded.last_checked,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now

	result, err := s.db.ExecContext(ctx, query,
		state.Repository, state.Branch, state.CommitSHA,
		state.LastChecked, state.CreatedAt, state.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save repository state: %w", err)
	}

	// Set ID if this was an insert
	if state.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			state.ID = id
		}
	}

	return nil
}

// GetRepoState retrieves a repository state
func (s *SQLiteStorage) GetRepoState(ctx context.Context, repository, branch string) (*types.RepoState, error) {
	query := `
		SELECT id, repository, branch, commit_sha, last_checked, created_at, updated_at
		FROM repository_states
		WHERE repository = ? AND branch = ?
	`

	var sqliteState SQLiteRepoState
	err := s.db.QueryRowContext(ctx, query, repository, branch).Scan(
		&sqliteState.ID, &sqliteState.Repository, &sqliteState.Branch,
		&sqliteState.CommitSHA, &sqliteState.LastChecked,
		&sqliteState.CreatedAt, &sqliteState.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, &RepositoryNotFoundError{Repository: repository, Branch: branch}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository state: %w", err)
	}

	return sqliteState.ToRepoState(), nil
}

// GetRepoStates retrieves all states for a repository
func (s *SQLiteStorage) GetRepoStates(ctx context.Context, repository string) ([]*types.RepoState, error) {
	query := `
		SELECT id, repository, branch, commit_sha, last_checked, created_at, updated_at
		FROM repository_states
		WHERE repository = ?
		ORDER BY branch
	`

	rows, err := s.db.QueryContext(ctx, query, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to query repository states: %w", err)
	}
	defer rows.Close()

	var states []*types.RepoState
	for rows.Next() {
		var sqliteState SQLiteRepoState
		err := rows.Scan(&sqliteState.ID, &sqliteState.Repository, &sqliteState.Branch,
			&sqliteState.CommitSHA, &sqliteState.LastChecked,
			&sqliteState.CreatedAt, &sqliteState.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository state: %w", err)
		}
		states = append(states, sqliteState.ToRepoState())
	}

	return states, rows.Err()
}

// DeleteRepoState deletes a repository state
func (s *SQLiteStorage) DeleteRepoState(ctx context.Context, repository, branch string) error {
	query := "DELETE FROM repository_states WHERE repository = ? AND branch = ?"
	result, err := s.db.ExecContext(ctx, query, repository, branch)
	if err != nil {
		return fmt.Errorf("failed to delete repository state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &RepositoryNotFoundError{Repository: repository, Branch: branch}
	}

	return nil
}

// GetAllRepoStates retrieves all repository states
func (s *SQLiteStorage) GetAllRepoStates(ctx context.Context) ([]*types.RepoState, error) {
	query := `
		SELECT id, repository, branch, commit_sha, last_checked, created_at, updated_at
		FROM repository_states
		ORDER BY repository, branch
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all repository states: %w", err)
	}
	defer rows.Close()

	var states []*types.RepoState
	for rows.Next() {
		var sqliteState SQLiteRepoState
		err := rows.Scan(&sqliteState.ID, &sqliteState.Repository, &sqliteState.Branch,
			&sqliteState.CommitSHA, &sqliteState.LastChecked,
			&sqliteState.CreatedAt, &sqliteState.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository state: %w", err)
		}
		states = append(states, sqliteState.ToRepoState())
	}

	return states, rows.Err()
}

// SaveEvent saves an event
func (s *SQLiteStorage) SaveEvent(ctx context.Context, event *types.Event) error {
	var sqliteEvent SQLiteEvent
	sqliteEvent.FromEvent(event)

	// Set timestamps
	now := time.Now()
	if sqliteEvent.CreatedAt.IsZero() {
		sqliteEvent.CreatedAt = now
	}
	sqliteEvent.UpdatedAt = now

	query := `
		INSERT INTO events (id, type, repository, branch, commit_sha, prev_commit, 
			provider, timestamp, metadata, status, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		sqliteEvent.ID, sqliteEvent.Type, sqliteEvent.Repository, sqliteEvent.Branch,
		sqliteEvent.CommitSHA, sqliteEvent.PrevCommit, sqliteEvent.Provider,
		sqliteEvent.Timestamp, sqliteEvent.Metadata, sqliteEvent.Status,
		sqliteEvent.ProcessedAt, sqliteEvent.CreatedAt, sqliteEvent.UpdatedAt)

	if err != nil {
		if isUniqueConstraintError(err) {
			return &DuplicateEventError{EventID: event.ID}
		}
		return fmt.Errorf("failed to save event: %w", err)
	}

	// Update original event with timestamps
	event.CreatedAt = sqliteEvent.CreatedAt
	event.UpdatedAt = sqliteEvent.UpdatedAt

	return nil
}

// GetEvent retrieves an event by ID
func (s *SQLiteStorage) GetEvent(ctx context.Context, eventID string) (*types.Event, error) {
	query := `
		SELECT id, type, repository, branch, commit_sha, prev_commit, 
			provider, timestamp, metadata, status, processed_at, created_at, updated_at
		FROM events
		WHERE id = ?
	`

	var sqliteEvent SQLiteEvent
	err := s.db.QueryRowContext(ctx, query, eventID).Scan(
		&sqliteEvent.ID, &sqliteEvent.Type, &sqliteEvent.Repository, &sqliteEvent.Branch,
		&sqliteEvent.CommitSHA, &sqliteEvent.PrevCommit, &sqliteEvent.Provider,
		&sqliteEvent.Timestamp, &sqliteEvent.Metadata, &sqliteEvent.Status,
		&sqliteEvent.ProcessedAt, &sqliteEvent.CreatedAt, &sqliteEvent.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, &EventNotFoundError{EventID: eventID}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return sqliteEvent.ToEvent(), nil
}

// GetPendingEvents retrieves pending events
func (s *SQLiteStorage) GetPendingEvents(ctx context.Context, limit int) ([]*types.Event, error) {
	query := `
		SELECT id, type, repository, branch, commit_sha, prev_commit, 
			provider, timestamp, metadata, status, processed_at, created_at, updated_at
		FROM events
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT ?
	`

	return s.queryEvents(ctx, query, limit)
}

// GetEventsByRepository retrieves events for a repository
func (s *SQLiteStorage) GetEventsByRepository(ctx context.Context, repository string, limit int) ([]*types.Event, error) {
	query := `
		SELECT id, type, repository, branch, commit_sha, prev_commit, 
			provider, timestamp, metadata, status, processed_at, created_at, updated_at
		FROM events
		WHERE repository = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	return s.queryEvents(ctx, query, repository, limit)
}

// UpdateEventStatus updates an event's status
func (s *SQLiteStorage) UpdateEventStatus(ctx context.Context, eventID string, status types.EventStatus) error {
	var processedAt *time.Time
	if status == types.EventStatusProcessed || status == types.EventStatusFailed {
		now := time.Now()
		processedAt = &now
	}

	query := `
		UPDATE events 
		SET status = ?, processed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, string(status), processedAt, eventID)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &EventNotFoundError{EventID: eventID}
	}

	return nil
}

// DeleteOldEvents deletes events older than the specified time
func (s *SQLiteStorage) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	query := "DELETE FROM events WHERE created_at < ?"
	result, err := s.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	return result.RowsAffected()
}

// GetStats retrieves storage statistics
func (s *SQLiteStorage) GetStats(ctx context.Context) (*StorageStats, error) {
	query := `
		SELECT 
			(SELECT COUNT(DISTINCT repository) FROM repository_states) as total_repositories,
			(SELECT COUNT(*) FROM repository_states) as total_branches,
			(SELECT COUNT(*) FROM events) as total_events,
			(SELECT COUNT(*) FROM events WHERE status = 'pending') as pending_events,
			(SELECT COUNT(*) FROM events WHERE status = 'failed') as failed_events,
			(SELECT MAX(timestamp) FROM events) as last_event_time,
			(SELECT MIN(created_at) FROM events WHERE status = 'pending') as oldest_pending_event
	`

	var stats SQLiteStats
	var lastEventTimeStr, oldestPendingEventStr sql.NullString

	err := s.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalRepositories, &stats.TotalBranches, &stats.TotalEvents,
		&stats.PendingEvents, &stats.FailedEvents, &lastEventTimeStr,
		&oldestPendingEventStr)

	// Parse time strings
	if lastEventTimeStr.Valid && lastEventTimeStr.String != "" {
		if t, err := time.Parse(time.RFC3339, lastEventTimeStr.String); err == nil {
			stats.LastEventTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", lastEventTimeStr.String); err == nil {
			stats.LastEventTime = t
		}
	}

	if oldestPendingEventStr.Valid && oldestPendingEventStr.String != "" {
		if t, err := time.Parse(time.RFC3339, oldestPendingEventStr.String); err == nil {
			stats.OldestPendingEvent = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", oldestPendingEventStr.String); err == nil {
			stats.OldestPendingEvent = t
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	storageStats := stats.ToStorageStats()

	// Get database file size
	if fileInfo, err := os.Stat(s.config.Path); err == nil {
		storageStats.DatabaseSize = fileInfo.Size()
	}

	return storageStats, nil
}

// queryEvents is a helper method to query events
func (s *SQLiteStorage) queryEvents(ctx context.Context, query string, args ...interface{}) ([]*types.Event, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*types.Event
	for rows.Next() {
		var sqliteEvent SQLiteEvent
		err := rows.Scan(&sqliteEvent.ID, &sqliteEvent.Type, &sqliteEvent.Repository,
			&sqliteEvent.Branch, &sqliteEvent.CommitSHA, &sqliteEvent.PrevCommit,
			&sqliteEvent.Provider, &sqliteEvent.Timestamp, &sqliteEvent.Metadata,
			&sqliteEvent.Status, &sqliteEvent.ProcessedAt, &sqliteEvent.CreatedAt,
			&sqliteEvent.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, sqliteEvent.ToEvent())
	}

	return events, rows.Err()
}

// CreateEvent is an alias for SaveEvent to match poller interface
func (s *SQLiteStorage) CreateEvent(ctx context.Context, event types.Event) error {
	return s.SaveEvent(ctx, &event)
}

// UpsertRepoState inserts or updates repository state for poller
func (s *SQLiteStorage) UpsertRepoState(ctx context.Context, state RepositoryState) error {
	query := `
		INSERT INTO repository_states (repository, branch, commit_sha, last_checked, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository, branch) DO UPDATE SET
			commit_sha = excluded.commit_sha,
			last_checked = excluded.last_checked,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		state.Repository, state.Branch, state.CommitSHA,
		state.LastCheck, state.CreatedAt, state.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert repository state: %w", err)
	}

	return nil
}

// GetEvents retrieves events with pagination
func (s *SQLiteStorage) GetEvents(ctx context.Context, limit, offset int) ([]*types.Event, error) {
	query := `
		SELECT id, type, repository, branch, commit_sha, status, metadata, 
		       error_message, created_at, updated_at
		FROM events 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*types.Event
	for rows.Next() {
		event := &types.Event{}
		var metadata, errorMessage sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Repository,
			&event.Branch,
			&event.CommitSHA,
			&event.Status,
			&metadata,
			&errorMessage,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse metadata JSON
		if metadata.Valid && metadata.String != "" {
			var metadataMap map[string]string
			if err := json.Unmarshal([]byte(metadata.String), &metadataMap); err == nil {
				event.Metadata = metadataMap
			}
		}

		if errorMessage.Valid {
			event.ErrorMessage = errorMessage.String
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over events: %w", err)
	}

	return events, nil
}

// GetEventsSince retrieves events since a specific time
func (s *SQLiteStorage) GetEventsSince(ctx context.Context, since time.Time) ([]*types.Event, error) {
	query := `
		SELECT id, type, repository, branch, commit_sha, status, metadata, 
		       error_message, created_at, updated_at
		FROM events 
		WHERE created_at >= ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query events since %v: %w", since, err)
	}
	defer rows.Close()

	var events []*types.Event
	for rows.Next() {
		event := &types.Event{}
		var metadata, errorMessage sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Repository,
			&event.Branch,
			&event.CommitSHA,
			&event.Status,
			&metadata,
			&errorMessage,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse metadata JSON
		if metadata.Valid && metadata.String != "" {
			var metadataMap map[string]string
			if err := json.Unmarshal([]byte(metadata.String), &metadataMap); err == nil {
				event.Metadata = metadataMap
			}
		}

		if errorMessage.Valid {
			event.ErrorMessage = errorMessage.String
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over events since %v: %w", since, err)
	}

	return events, nil
}

// isUniqueConstraintError checks if error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	return err != nil && (fmt.Sprintf("%v", err) == "UNIQUE constraint failed: events.id" ||
		fmt.Sprintf("%v", err) == "database is locked")
}
