package storage

import (
	"context"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Storage defines the interface for data persistence operations
type Storage interface {
	// Lifecycle operations
	Initialize(ctx context.Context) error
	Close() error
	HealthCheck(ctx context.Context) error

	// Repository state operations
	SaveRepoState(ctx context.Context, state *types.RepoState) error
	GetRepoState(ctx context.Context, repository, branch string) (*types.RepoState, error)
	GetRepoStates(ctx context.Context, repository string) ([]*types.RepoState, error)
	DeleteRepoState(ctx context.Context, repository, branch string) error
	GetAllRepoStates(ctx context.Context) ([]*types.RepoState, error)

	// Event operations
	SaveEvent(ctx context.Context, event *types.Event) error
	CreateEvent(ctx context.Context, event types.Event) error // Alias for SaveEvent
	GetEvent(ctx context.Context, eventID string) (*types.Event, error)
	GetPendingEvents(ctx context.Context, limit int) ([]*types.Event, error)
	GetEventsByRepository(ctx context.Context, repository string, limit int) ([]*types.Event, error)
	GetEvents(ctx context.Context, limit, offset int) ([]*types.Event, error)
	GetEventsSince(ctx context.Context, since time.Time) ([]*types.Event, error)
	UpdateEventStatus(ctx context.Context, eventID string, status types.EventStatus) error
	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)

	// Enhanced repository state operations for poller
	UpsertRepoState(ctx context.Context, state RepositoryState) error

	// Statistics operations
	GetStats(ctx context.Context) (*StorageStats, error)
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalRepositories  int64     `json:"total_repositories"`
	TotalBranches      int64     `json:"total_branches"`
	TotalEvents        int64     `json:"total_events"`
	PendingEvents      int64     `json:"pending_events"`
	FailedEvents       int64     `json:"failed_events"`
	LastEventTime      time.Time `json:"last_event_time,omitempty"`
	OldestPendingEvent time.Time `json:"oldest_pending_event,omitempty"`
	DatabaseSize       int64     `json:"database_size_bytes,omitempty"`
}

// StorageConfig represents storage configuration interface
type StorageConfig interface {
	GetType() string
	Validate() error
}

// Factory creates storage instances based on configuration
type Factory struct{}

// NewFactory creates a new storage factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a storage instance based on configuration
func (f *Factory) Create(config *types.StorageConfig) (Storage, error) {
	switch config.Type {
	case "sqlite":
		return NewSQLiteStorage(&config.SQLite)
	default:
		return nil, &UnsupportedStorageTypeError{Type: config.Type}
	}
}

// Storage errors
type UnsupportedStorageTypeError struct {
	Type string
}

func (e *UnsupportedStorageTypeError) Error() string {
	return "unsupported storage type: " + e.Type
}

type RepositoryNotFoundError struct {
	Repository string
	Branch     string
}

func (e *RepositoryNotFoundError) Error() string {
	if e.Branch != "" {
		return "repository state not found: " + e.Repository + "/" + e.Branch
	}
	return "repository not found: " + e.Repository
}

type EventNotFoundError struct {
	EventID string
}

func (e *EventNotFoundError) Error() string {
	return "event not found: " + e.EventID
}

type DuplicateEventError struct {
	EventID string
}

func (e *DuplicateEventError) Error() string {
	return "event already exists: " + e.EventID
}
