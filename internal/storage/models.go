package storage

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// SQLiteRepoState represents repository state in SQLite
type SQLiteRepoState struct {
	ID          int64     `db:"id"`
	Repository  string    `db:"repository"`
	Branch      string    `db:"branch"`
	CommitSHA   string    `db:"commit_sha"`
	LastChecked time.Time `db:"last_checked"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// ToRepoState converts SQLiteRepoState to types.RepoState
func (s *SQLiteRepoState) ToRepoState() *types.RepoState {
	return &types.RepoState{
		ID:          s.ID,
		Repository:  s.Repository,
		Branch:      s.Branch,
		CommitSHA:   s.CommitSHA,
		LastChecked: s.LastChecked,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// FromRepoState converts types.RepoState to SQLiteRepoState
func (s *SQLiteRepoState) FromRepoState(state *types.RepoState) {
	s.ID = state.ID
	s.Repository = state.Repository
	s.Branch = state.Branch
	s.CommitSHA = state.CommitSHA
	s.LastChecked = state.LastChecked
	s.CreatedAt = state.CreatedAt
	s.UpdatedAt = state.UpdatedAt
}

// RepositoryState represents repository state for the poller
type RepositoryState struct {
	ID         int64     `json:"id"`
	Repository string    `json:"repository"`
	Branch     string    `json:"branch"`
	CommitSHA  string    `json:"commit_sha"`
	Protected  bool      `json:"protected"`
	LastCheck  time.Time `json:"last_check"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SQLiteEvent represents event in SQLite
type SQLiteEvent struct {
	ID          string       `db:"id"`
	Type        string       `db:"type"`
	Repository  string       `db:"repository"`
	Branch      string       `db:"branch"`
	CommitSHA   string       `db:"commit_sha"`
	PrevCommit  string       `db:"prev_commit"`
	Provider    string       `db:"provider"`
	Timestamp   time.Time    `db:"timestamp"`
	Metadata    MetadataJSON `db:"metadata"`
	Status      string       `db:"status"`
	ProcessedAt *time.Time   `db:"processed_at"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
}

// ToEvent converts SQLiteEvent to types.Event
func (e *SQLiteEvent) ToEvent() *types.Event {
	return &types.Event{
		ID:          e.ID,
		Type:        types.EventType(e.Type),
		Repository:  e.Repository,
		Branch:      e.Branch,
		CommitSHA:   e.CommitSHA,
		PrevCommit:  e.PrevCommit,
		Provider:    e.Provider,
		Timestamp:   e.Timestamp,
		Metadata:    map[string]string(e.Metadata),
		Status:      types.EventStatus(e.Status),
		ProcessedAt: e.ProcessedAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// FromEvent converts types.Event to SQLiteEvent
func (e *SQLiteEvent) FromEvent(event *types.Event) {
	e.ID = event.ID
	e.Type = string(event.Type)
	e.Repository = event.Repository
	e.Branch = event.Branch
	e.CommitSHA = event.CommitSHA
	e.PrevCommit = event.PrevCommit
	e.Provider = event.Provider
	e.Timestamp = event.Timestamp
	e.Metadata = MetadataJSON(event.Metadata)
	e.Status = string(event.Status)
	e.ProcessedAt = event.ProcessedAt
	e.CreatedAt = event.CreatedAt
	e.UpdatedAt = event.UpdatedAt
}

// MetadataJSON handles JSON serialization for metadata
type MetadataJSON map[string]string

// Value implements driver.Valuer interface for database storage
func (m MetadataJSON) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return string(data), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (m *MetadataJSON) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]string)
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into MetadataJSON", value)
	}

	if len(data) == 0 {
		*m = make(map[string]string)
		return nil
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	*m = MetadataJSON(result)
	return nil
}

// SQLiteStats represents database statistics
type SQLiteStats struct {
	TotalRepositories  int64     `db:"total_repositories"`
	TotalBranches      int64     `db:"total_branches"`
	TotalEvents        int64     `db:"total_events"`
	PendingEvents      int64     `db:"pending_events"`
	FailedEvents       int64     `db:"failed_events"`
	LastEventTime      time.Time `db:"last_event_time"`
	OldestPendingEvent time.Time `db:"oldest_pending_event"`
}

// ToStorageStats converts SQLiteStats to StorageStats
func (s *SQLiteStats) ToStorageStats() *StorageStats {
	return &StorageStats{
		TotalRepositories:  s.TotalRepositories,
		TotalBranches:      s.TotalBranches,
		TotalEvents:        s.TotalEvents,
		PendingEvents:      s.PendingEvents,
		FailedEvents:       s.FailedEvents,
		LastEventTime:      s.LastEventTime,
		OldestPendingEvent: s.OldestPendingEvent,
	}
}
