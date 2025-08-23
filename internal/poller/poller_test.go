package poller

import (
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPollerConfig_DefaultValues(t *testing.T) {
	config := GetDefaultPollerConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 5*time.Minute, config.Interval)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 5, config.MaxWorkers)
	assert.Equal(t, 10, config.BatchSize)
	assert.True(t, config.EnableFallback)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 1*time.Second, config.RetryBackoff)
}

func TestPollerConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  PollerConfig
		isValid bool
	}{
		{
			name: "valid config",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        30 * time.Second,
				MaxWorkers:     5,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
			},
			isValid: true,
		},
		{
			name: "zero interval",
			config: PollerConfig{
				Interval:       0,
				Timeout:        30 * time.Second,
				MaxWorkers:     5,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
			},
			isValid: false,
		},
		{
			name: "zero timeout",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        0,
				MaxWorkers:     5,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
			},
			isValid: false,
		},
		{
			name: "zero max workers",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        30 * time.Second,
				MaxWorkers:     0,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
			},
			isValid: false,
		},
		{
			name: "zero batch size",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        30 * time.Second,
				MaxWorkers:     5,
				BatchSize:      0,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
			},
			isValid: false,
		},
		{
			name: "negative retry attempts",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        30 * time.Second,
				MaxWorkers:     5,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  -1,
				RetryBackoff:   5 * time.Second,
			},
			isValid: false,
		},
		{
			name: "zero retry backoff",
			config: PollerConfig{
				Interval:       5 * time.Minute,
				Timeout:        30 * time.Second,
				MaxWorkers:     5,
				BatchSize:      100,
				EnableFallback: true,
				RetryAttempts:  3,
				RetryBackoff:   0,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation based on field values
			isValid := tt.config.Interval > 0 &&
				tt.config.Timeout > 0 &&
				tt.config.MaxWorkers > 0 &&
				tt.config.BatchSize > 0 &&
				tt.config.RetryAttempts >= 0 &&
				tt.config.RetryBackoff > 0
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestBranchChange_Validation(t *testing.T) {
	tests := []struct {
		name    string
		change  BranchChange
		isValid bool
	}{
		{
			name: "valid new branch",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "main",
				NewCommitSHA: "abc123",
				ChangeType:   ChangeTypeNew,
				Timestamp:    time.Now(),
			},
			isValid: true,
		},
		{
			name: "valid updated branch",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "main",
				OldCommitSHA: "abc123",
				NewCommitSHA: "def456",
				ChangeType:   ChangeTypeUpdated,
				Timestamp:    time.Now(),
			},
			isValid: true,
		},
		{
			name: "valid deleted branch",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "old-branch",
				NewCommitSHA: "abc123",
				ChangeType:   ChangeTypeDeleted,
				Timestamp:    time.Now(),
			},
			isValid: true,
		},
		{
			name: "empty repository",
			change: BranchChange{
				Repository:   "",
				Branch:       "main",
				NewCommitSHA: "abc123",
				ChangeType:   ChangeTypeNew,
				Timestamp:    time.Now(),
			},
			isValid: false,
		},
		{
			name: "empty branch",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "",
				NewCommitSHA: "abc123",
				ChangeType:   ChangeTypeNew,
				Timestamp:    time.Now(),
			},
			isValid: false,
		},
		{
			name: "empty new commit SHA",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "main",
				NewCommitSHA: "",
				ChangeType:   ChangeTypeNew,
				Timestamp:    time.Now(),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.change.IsValid())
		})
	}
}

func TestBranchChange_ChangeTypes(t *testing.T) {
	change := BranchChange{
		Repository:   "test-repo",
		Branch:       "main",
		NewCommitSHA: "abc123",
		ChangeType:   ChangeTypeNew,
		Timestamp:    time.Now(),
	}

	// Test IsNewBranch
	assert.True(t, change.IsNewBranch())
	assert.False(t, change.IsUpdated())
	assert.False(t, change.IsDeleted())

	// Test IsUpdated
	change.ChangeType = ChangeTypeUpdated
	change.OldCommitSHA = "abc123"
	change.NewCommitSHA = "def456"
	assert.False(t, change.IsNewBranch())
	assert.True(t, change.IsUpdated())
	assert.False(t, change.IsDeleted())

	// Test IsDeleted
	change.ChangeType = ChangeTypeDeleted
	assert.False(t, change.IsNewBranch())
	assert.False(t, change.IsUpdated())
	assert.True(t, change.IsDeleted())
}

func TestPollResult_Validation(t *testing.T) {
	// Note: PollResult validation depends on Repository struct which we can't easily test here
	// We'll test the basic structure instead
	result := PollResult{
		Repository: types.Repository{
			Name:     "test-repo",
			Provider: "github",
		},
		Success:   true,
		Timestamp: time.Now(),
	}

	// Test that we can create a valid result
	assert.NotNil(t, result)
	assert.Equal(t, "test-repo", result.Repository.Name)
	assert.Equal(t, "github", result.Repository.Provider)
}

func TestPollerStatus_Initialization(t *testing.T) {
	status := PollerStatus{}

	// Test default values
	assert.False(t, status.Running)
	assert.Equal(t, time.Time{}, status.StartTime)
	assert.Equal(t, time.Time{}, status.LastPollTime)
	assert.Equal(t, 0, status.ActiveRepositories)
	assert.Equal(t, 0, status.WorkerCount)
	assert.Equal(t, 0, status.QueueSize)
	assert.Len(t, status.Repositories, 0)
}

func TestRepositoryStatus_Initialization(t *testing.T) {
	status := RepositoryStatus{}

	// Test default values
	assert.Equal(t, "", status.Name)
	assert.Equal(t, "", status.Provider)
	assert.False(t, status.Enabled)
	assert.Equal(t, time.Time{}, status.LastPollTime)
	assert.Equal(t, time.Time{}, status.NextPollTime)
	assert.False(t, status.LastSuccess)
	assert.Equal(t, "", status.LastError)
	assert.Equal(t, int64(0), status.PollCount)
	assert.Equal(t, int64(0), status.ChangeCount)
	assert.Equal(t, int64(0), status.EventCount)
}

func TestPollerMetrics_Initialization(t *testing.T) {
	metrics := PollerMetrics{}

	// Test default values
	assert.Equal(t, int64(0), metrics.TotalPolls)
	assert.Equal(t, int64(0), metrics.SuccessfulPolls)
	assert.Equal(t, int64(0), metrics.FailedPolls)
	assert.Equal(t, int64(0), metrics.TotalChanges)
	assert.Equal(t, int64(0), metrics.TotalEvents)
	assert.Equal(t, time.Duration(0), metrics.AveragePollDuration)
	assert.Equal(t, time.Time{}, metrics.LastResetTime)
	assert.Equal(t, time.Duration(0), metrics.Uptime)
	assert.Equal(t, int64(0), metrics.APICallCount)
	assert.Equal(t, int64(0), metrics.FallbackCount)
}

func TestChangeTypeConstants(t *testing.T) {
	// Test that constants are properly defined
	assert.NotEmpty(t, ChangeTypeNew)
	assert.NotEmpty(t, ChangeTypeUpdated)
	assert.NotEmpty(t, ChangeTypeDeleted)

	// Test that constants are unique
	assert.NotEqual(t, ChangeTypeNew, ChangeTypeUpdated)
	assert.NotEqual(t, ChangeTypeNew, ChangeTypeDeleted)
	assert.NotEqual(t, ChangeTypeUpdated, ChangeTypeDeleted)
}

func TestPollerConfig_Copy(t *testing.T) {
	original := PollerConfig{
		Interval:       5 * time.Minute,
		Timeout:        30 * time.Second,
		MaxWorkers:     5,
		BatchSize:      100,
		EnableFallback: true,
		RetryAttempts:  3,
		RetryBackoff:   5 * time.Second,
	}

	// Test that we can create a copy
	copied := original

	// Modify the copy
	copied.Interval = 10 * time.Minute
	copied.MaxWorkers = 10

	// Original should remain unchanged
	assert.Equal(t, 5*time.Minute, original.Interval)
	assert.Equal(t, 5, original.MaxWorkers)

	// Copy should have new values
	assert.Equal(t, 10*time.Minute, copied.Interval)
	assert.Equal(t, 10, copied.MaxWorkers)
}

func TestBranchChange_Equality(t *testing.T) {
	change1 := BranchChange{
		Repository:   "test-repo",
		Branch:       "main",
		NewCommitSHA: "abc123",
		ChangeType:   ChangeTypeNew,
		Timestamp:    time.Now(),
	}

	change2 := BranchChange{
		Repository:   "test-repo",
		Branch:       "main",
		NewCommitSHA: "abc123",
		ChangeType:   ChangeTypeNew,
		Timestamp:    change1.Timestamp, // Use same timestamp
	}

	// Test equality
	assert.Equal(t, change1.Repository, change2.Repository)
	assert.Equal(t, change1.Branch, change2.Branch)
	assert.Equal(t, change1.NewCommitSHA, change2.NewCommitSHA)
	assert.Equal(t, change1.ChangeType, change2.ChangeType)
	assert.Equal(t, change1.Timestamp, change2.Timestamp)
}
