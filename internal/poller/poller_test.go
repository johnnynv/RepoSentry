package poller

import (
	"context"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestGetDefaultPollerConfig(t *testing.T) {
	config := GetDefaultPollerConfig()

	if config.Interval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", config.Interval)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxWorkers != 5 {
		t.Errorf("Expected max workers 5, got %d", config.MaxWorkers)
	}

	if config.BatchSize != 10 {
		t.Errorf("Expected batch size 10, got %d", config.BatchSize)
	}

	if !config.EnableFallback {
		t.Error("Expected fallback to be enabled by default")
	}
}

func TestPollResult_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		result PollResult
		want   bool
	}{
		{
			name: "Valid result",
			result: PollResult{
				Repository: types.Repository{
					Name:     "test-repo",
					Provider: "github",
				},
			},
			want: true,
		},
		{
			name: "Missing repository name",
			result: PollResult{
				Repository: types.Repository{
					Provider: "github",
				},
			},
			want: false,
		},
		{
			name: "Missing provider",
			result: PollResult{
				Repository: types.Repository{
					Name: "test-repo",
				},
			},
			want: false,
		},
		{
			name:   "Empty result",
			result: PollResult{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBranchChange_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		change BranchChange
		want   bool
	}{
		{
			name: "Valid change",
			change: BranchChange{
				Repository:   "test-repo",
				Branch:       "main",
				NewCommitSHA: "abc123",
			},
			want: true,
		},
		{
			name: "Missing repository",
			change: BranchChange{
				Branch:       "main",
				NewCommitSHA: "abc123",
			},
			want: false,
		},
		{
			name: "Missing branch",
			change: BranchChange{
				Repository:   "test-repo",
				NewCommitSHA: "abc123",
			},
			want: false,
		},
		{
			name: "Missing commit SHA",
			change: BranchChange{
				Repository: "test-repo",
				Branch:     "main",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.change.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBranchChange_ChangeTypes(t *testing.T) {
	tests := []struct {
		name     string
		change   BranchChange
		isNew    bool
		isUpdate bool
		isDelete bool
	}{
		{
			name: "New branch",
			change: BranchChange{
				ChangeType:   ChangeTypeNew,
				OldCommitSHA: "",
				NewCommitSHA: "abc123",
			},
			isNew:    true,
			isUpdate: false,
			isDelete: false,
		},
		{
			name: "Updated branch",
			change: BranchChange{
				ChangeType:   ChangeTypeUpdated,
				OldCommitSHA: "def456",
				NewCommitSHA: "abc123",
			},
			isNew:    false,
			isUpdate: true,
			isDelete: false,
		},
		{
			name: "Deleted branch",
			change: BranchChange{
				ChangeType: ChangeTypeDeleted,
			},
			isNew:    false,
			isUpdate: false,
			isDelete: true,
		},
		{
			name: "Updated branch with same SHA (should not be updated)",
			change: BranchChange{
				ChangeType:   ChangeTypeUpdated,
				OldCommitSHA: "abc123",
				NewCommitSHA: "abc123",
			},
			isNew:    false,
			isUpdate: false, // Same SHA means no actual update
			isDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.change.IsNewBranch(); got != tt.isNew {
				t.Errorf("IsNewBranch() = %v, want %v", got, tt.isNew)
			}
			if got := tt.change.IsUpdated(); got != tt.isUpdate {
				t.Errorf("IsUpdated() = %v, want %v", got, tt.isUpdate)
			}
			if got := tt.change.IsDeleted(); got != tt.isDelete {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.isDelete)
			}
		})
	}
}

func TestEventFilter_ApplyFilter(t *testing.T) {
	changes := []BranchChange{
		{
			Repository:  "test-repo",
			Branch:      "main",
			ChangeType:  ChangeTypeNew,
			Protected:   false,
			Timestamp:   time.Now().Add(-2 * time.Hour),
		},
		{
			Repository:  "test-repo",
			Branch:      "protected-main",
			ChangeType:  ChangeTypeUpdated,
			Protected:   true,
			Timestamp:   time.Now().Add(-1 * time.Hour),
		},
		{
			Repository:  "test-repo",
			Branch:      "feature",
			ChangeType:  ChangeTypeDeleted,
			Protected:   false,
			Timestamp:   time.Now().Add(-30 * time.Minute),
		},
	}

	tests := []struct {
		name           string
		filter         *EventFilter
		expectedCount  int
		description    string
	}{
		{
			name:          "No filter",
			filter:        nil,
			expectedCount: 3,
			description:   "Should return all changes when no filter applied",
		},
		{
			name: "Exclude protected",
			filter: &EventFilter{
				ExcludeProtected: true,
			},
			expectedCount: 2,
			description:   "Should exclude protected branches",
		},
		{
			name: "Include only protected",
			filter: &EventFilter{
				IncludeProtected: true,
			},
			expectedCount: 1,
			description:   "Should include only protected branches",
		},
		{
			name: "Include only new changes",
			filter: &EventFilter{
				IncludeChangeTypes: []string{ChangeTypeNew},
			},
			expectedCount: 1,
			description:   "Should include only new changes",
		},
		{
			name: "Exclude deleted changes",
			filter: &EventFilter{
				ExcludeChangeTypes: []string{ChangeTypeDeleted},
			},
			expectedCount: 2,
			description:   "Should exclude deleted changes",
		},
		{
			name: "Min commit age filter",
			filter: &EventFilter{
				MinCommitAge: 45 * time.Minute,
			},
			expectedCount: 2,
			description:   "Should exclude changes newer than 45 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tt.filter.ApplyFilter(changes)
			if len(filtered) != tt.expectedCount {
				t.Errorf("%s: expected %d changes, got %d", 
					tt.description, tt.expectedCount, len(filtered))
			}
		})
	}
}

func TestNewEventBatch(t *testing.T) {
	events := []types.Event{
		{
			ID:         "event1",
			Repository: "test-repo",
			Branch:     "main",
		},
		{
			ID:         "event2",
			Repository: "test-repo",
			Branch:     "develop",
		},
	}

	batch := NewEventBatch("test-repo", events)

	if batch.Repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got %s", batch.Repository)
	}

	if batch.Size != 2 {
		t.Errorf("Expected batch size 2, got %d", batch.Size)
	}

	if len(batch.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(batch.Events))
	}

	if batch.ID == "" {
		t.Error("Batch ID should not be empty")
	}

	if batch.CreatedAt.IsZero() {
		t.Error("Batch CreatedAt should not be zero")
	}
}

// Mock implementations for testing
type MockBranchMonitor struct {
	changes    []BranchChange
	err        error
	lastCheck  time.Time
}

func (m *MockBranchMonitor) CheckBranches(ctx context.Context, repo types.Repository) ([]BranchChange, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.changes, nil
}

func (m *MockBranchMonitor) GetLastCheckTime(repo types.Repository) (time.Time, bool) {
	return m.lastCheck, !m.lastCheck.IsZero()
}

func (m *MockBranchMonitor) UpdateLastCheck(repo types.Repository, checkTime time.Time) error {
	m.lastCheck = checkTime
	return nil
}

type MockEventGenerator struct {
	events []types.Event
	err    error
}

func (m *MockEventGenerator) GenerateEvents(ctx context.Context, repo types.Repository, changes []BranchChange) ([]types.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *MockEventGenerator) FilterChanges(repo types.Repository, changes []BranchChange) ([]BranchChange, error) {
	return changes, nil
}

type MockScheduler struct {
	repos   map[string]types.Repository
	running bool
}

func NewMockScheduler() *MockScheduler {
	return &MockScheduler{
		repos: make(map[string]types.Repository),
	}
}

func (m *MockScheduler) Schedule(repo types.Repository) error {
	m.repos[repo.Name] = repo
	return nil
}

func (m *MockScheduler) Unschedule(repo types.Repository) error {
	delete(m.repos, repo.Name)
	return nil
}

func (m *MockScheduler) GetNextPollTime(repo types.Repository) (time.Time, bool) {
	_, exists := m.repos[repo.Name]
	if exists {
		return time.Now().Add(5 * time.Minute), true
	}
	return time.Time{}, false
}

func (m *MockScheduler) Start(ctx context.Context) error {
	m.running = true
	return nil
}

func (m *MockScheduler) Stop(ctx context.Context) error {
	m.running = false
	return nil
}

// Integration test helpers
func createTestRepository() types.Repository {
	return types.Repository{
		Name:        "test-repo",
		URL:         "https://github.com/test/repo",
		Provider:    "github",
		Token:       "test-token",
		BranchRegex: "^(main|develop)$",
		Enabled:     true,
	}
}

func createTestBranchChange() BranchChange {
	return BranchChange{
		Repository:   "test-repo",
		Branch:       "main",
		OldCommitSHA: "old123",
		NewCommitSHA: "new456",
		ChangeType:   ChangeTypeUpdated,
		Timestamp:    time.Now(),
		Protected:    false,
	}
}
