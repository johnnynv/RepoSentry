package poller

import (
	"context"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Poller defines the interface for repository polling operations
type Poller interface {
	// Start begins the polling process
	Start(ctx context.Context) error

	// Stop gracefully stops the polling process
	Stop(ctx context.Context) error

	// PollRepository polls a specific repository once
	PollRepository(ctx context.Context, repo types.Repository) (*PollResult, error)

	// GetStatus returns the current status of the poller
	GetStatus() PollerStatus

	// GetMetrics returns polling metrics
	GetMetrics() PollerMetrics

	// GetScheduler returns the scheduler instance
	GetScheduler() Scheduler
}

// BranchMonitor defines the interface for monitoring repository branches
type BranchMonitor interface {
	// CheckBranches checks for changes in repository branches
	CheckBranches(ctx context.Context, repo types.Repository) ([]BranchChange, error)

	// GetLastCheckTime returns the last time the repository was checked
	GetLastCheckTime(repo types.Repository) (time.Time, bool)

	// UpdateLastCheck updates the last check time for a repository
	UpdateLastCheck(repo types.Repository, checkTime time.Time) error
}

// EventGenerator defines the interface for generating events from repository changes
type EventGenerator interface {
	// GenerateEvents creates events from branch changes
	GenerateEvents(ctx context.Context, repo types.Repository, changes []BranchChange) ([]types.Event, error)

	// FilterChanges applies repository-specific filtering to changes
	FilterChanges(repo types.Repository, changes []BranchChange) ([]BranchChange, error)
}

// Scheduler defines the interface for managing polling schedules
type Scheduler interface {
	// Schedule schedules a repository for polling
	Schedule(repo types.Repository) error

	// Unschedule removes a repository from polling
	Unschedule(repo types.Repository) error

	// GetNextPollTime returns the next scheduled poll time for a repository
	GetNextPollTime(repo types.Repository) (time.Time, bool)

	// Start begins the scheduler
	Start(ctx context.Context) error

	// Stop gracefully stops the scheduler
	Stop(ctx context.Context) error

	// GetSchedulerStatus returns the current status of the scheduler
	GetSchedulerStatus() SchedulerStatus

	// GetScheduledRepositories returns all currently scheduled repositories
	GetScheduledRepositories() []ScheduledRepository
}

// PollResult represents the result of polling a repository
type PollResult struct {
	Repository   types.Repository `json:"repository"`
	Success      bool             `json:"success"`
	Error        error            `json:"error,omitempty"`
	BranchCount  int              `json:"branch_count"`
	Changes      []BranchChange   `json:"changes"`
	Events       []types.Event    `json:"events"`
	Duration     time.Duration    `json:"duration"`
	Timestamp    time.Time        `json:"timestamp"`
	UsedFallback bool             `json:"used_fallback"`
}

// BranchChange represents a change detected in a repository branch
type BranchChange struct {
	Repository   string    `json:"repository"`
	Branch       string    `json:"branch"`
	OldCommitSHA string    `json:"old_commit_sha,omitempty"`
	NewCommitSHA string    `json:"new_commit_sha"`
	ChangeType   string    `json:"change_type"` // new, updated, deleted
	Timestamp    time.Time `json:"timestamp"`
	Protected    bool      `json:"protected"`
}

// PollerStatus represents the current status of the poller
type PollerStatus struct {
	Running            bool               `json:"running"`
	StartTime          time.Time          `json:"start_time,omitempty"`
	LastPollTime       time.Time          `json:"last_poll_time,omitempty"`
	ActiveRepositories int                `json:"active_repositories"`
	WorkerCount        int                `json:"worker_count"`
	QueueSize          int                `json:"queue_size"`
	Repositories       []RepositoryStatus `json:"repositories"`
}

// RepositoryStatus represents the status of a specific repository
type RepositoryStatus struct {
	Name         string    `json:"name"`
	Provider     string    `json:"provider"`
	Enabled      bool      `json:"enabled"`
	LastPollTime time.Time `json:"last_poll_time,omitempty"`
	NextPollTime time.Time `json:"next_poll_time,omitempty"`
	LastSuccess  bool      `json:"last_success"`
	LastError    string    `json:"last_error,omitempty"`
	PollCount    int64     `json:"poll_count"`
	ChangeCount  int64     `json:"change_count"`
	EventCount   int64     `json:"event_count"`
}

// PollerMetrics represents polling performance metrics
type PollerMetrics struct {
	TotalPolls          int64         `json:"total_polls"`
	SuccessfulPolls     int64         `json:"successful_polls"`
	FailedPolls         int64         `json:"failed_polls"`
	TotalChanges        int64         `json:"total_changes"`
	TotalEvents         int64         `json:"total_events"`
	AveragePollDuration time.Duration `json:"average_poll_duration"`
	LastResetTime       time.Time     `json:"last_reset_time"`
	Uptime              time.Duration `json:"uptime"`
	APICallCount        int64         `json:"api_call_count"`
	FallbackCount       int64         `json:"fallback_count"`
}

// PollerConfig represents configuration for the poller
type PollerConfig struct {
	Interval       time.Duration `yaml:"interval" json:"interval"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	MaxWorkers     int           `yaml:"max_workers" json:"max_workers"`
	BatchSize      int           `yaml:"batch_size" json:"batch_size"`
	EnableFallback bool          `yaml:"enable_fallback" json:"enable_fallback"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryBackoff   time.Duration `yaml:"retry_backoff" json:"retry_backoff"`
}

// GetDefaultPollerConfig returns default poller configuration
func GetDefaultPollerConfig() PollerConfig {
	return PollerConfig{
		Interval:       5 * time.Minute,
		Timeout:        30 * time.Second,
		MaxWorkers:     5,
		BatchSize:      10,
		EnableFallback: true,
		RetryAttempts:  3,
		RetryBackoff:   1 * time.Second,
	}
}

// ChangeType constants
const (
	ChangeTypeNew     = "new"
	ChangeTypeUpdated = "updated"
	ChangeTypeDeleted = "deleted"
)

// Validation functions
func (pr *PollResult) IsValid() bool {
	return pr.Repository.Name != "" && pr.Repository.Provider != ""
}

func (bc *BranchChange) IsValid() bool {
	return bc.Repository != "" && bc.Branch != "" && bc.NewCommitSHA != ""
}

func (bc *BranchChange) IsNewBranch() bool {
	return bc.ChangeType == ChangeTypeNew && bc.OldCommitSHA == ""
}

func (bc *BranchChange) IsUpdated() bool {
	return bc.ChangeType == ChangeTypeUpdated && bc.OldCommitSHA != bc.NewCommitSHA
}

func (bc *BranchChange) IsDeleted() bool {
	return bc.ChangeType == ChangeTypeDeleted
}
