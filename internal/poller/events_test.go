package poller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestEventGenerator_GenerateEvents(t *testing.T) {
	generator := NewEventGenerator()
	ctx := context.Background()

	repo := types.Repository{
		Name:        "test-repo",
		Provider:    "github",
		URL:         "https://github.com/test/repo",
		BranchRegex: "^(main|develop)$",
	}

	tests := []struct {
		name          string
		changes       []BranchChange
		expectedCount int
		expectError   bool
	}{
		{
			name:          "No changes",
			changes:       []BranchChange{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "Single change",
			changes: []BranchChange{
				{
					Repository:   "test-repo",
					Branch:       "main",
					OldCommitSHA: "old123",
					NewCommitSHA: "new456",
					ChangeType:   ChangeTypeUpdated,
					Timestamp:    time.Now(),
					Protected:    false,
				},
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "Multiple changes",
			changes: []BranchChange{
				{
					Repository:   "test-repo",
					Branch:       "main",
					NewCommitSHA: "new123",
					ChangeType:   ChangeTypeNew,
					Timestamp:    time.Now(),
					Protected:    false,
				},
				{
					Repository:   "test-repo",
					Branch:       "develop",
					OldCommitSHA: "old456",
					NewCommitSHA: "new789",
					ChangeType:   ChangeTypeUpdated,
					Timestamp:    time.Now(),
					Protected:    true,
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := generator.GenerateEvents(ctx, repo, tt.changes)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(events) != tt.expectedCount {
				t.Errorf("Expected %d events, got %d", tt.expectedCount, len(events))
			}

			// Verify event properties
			for i, event := range events {
				if event.ID == "" {
					t.Errorf("Event %d: ID should not be empty", i)
				}
				if event.Repository != repo.Name {
					t.Errorf("Event %d: Expected repository %s, got %s", i, repo.Name, event.Repository)
				}
				if event.Type == "" {
					t.Errorf("Event %d: Type should not be empty", i)
				}
				if event.Status != "pending" {
					t.Errorf("Event %d: Expected status 'pending', got %s", i, event.Status)
				}

				// Verify metadata
				metadata := event.Metadata
				if len(metadata) == 0 {
					t.Errorf("Event %d: Metadata should not be empty", i)
				}

				if metadata["repository"] != repo.Name {
					t.Errorf("Event %d: Metadata repository mismatch", i)
				}
				if metadata["provider"] != repo.Provider {
					t.Errorf("Event %d: Metadata provider mismatch", i)
				}
			}
		})
	}
}

func TestEventGenerator_FilterChanges(t *testing.T) {
	generator := NewEventGenerator()

	tests := []struct {
		name          string
		repo          types.Repository
		changes       []BranchChange
		expectedCount int
		expectError   bool
	}{
		{
			name: "No regex filter",
			repo: types.Repository{
				Name:        "test-repo",
				BranchRegex: "",
			},
			changes: []BranchChange{
				{Branch: "main"},
				{Branch: "develop"},
				{Branch: "feature/test"},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "Regex filter matches main and develop",
			repo: types.Repository{
				Name:        "test-repo",
				BranchRegex: "^(main|develop)$",
			},
			changes: []BranchChange{
				{Branch: "main"},
				{Branch: "develop"},
				{Branch: "feature/test"},
				{Branch: "main-backup"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "Regex filter matches feature branches",
			repo: types.Repository{
				Name:        "test-repo",
				BranchRegex: "^feature/.*",
			},
			changes: []BranchChange{
				{Branch: "main"},
				{Branch: "feature/auth"},
				{Branch: "feature/ui"},
				{Branch: "hotfix/bug"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "Invalid regex",
			repo: types.Repository{
				Name:        "test-repo",
				BranchRegex: "[invalid regex",
			},
			changes: []BranchChange{
				{Branch: "main"},
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := generator.FilterChanges(tt.repo, tt.changes)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d filtered changes, got %d", tt.expectedCount, len(filtered))
			}
		})
	}
}

func TestEventGenerator_getEventType(t *testing.T) {
	generator := NewEventGenerator()

	tests := []struct {
		changeType   string
		expectedType types.EventType
	}{
		{ChangeTypeNew, types.EventTypeBranchCreated},
		{ChangeTypeUpdated, types.EventTypeBranchUpdated},
		{ChangeTypeDeleted, types.EventTypeBranchDeleted},
		{"unknown", types.EventTypeBranchUpdated},
	}

	for _, tt := range tests {
		t.Run(tt.changeType, func(t *testing.T) {
			eventType := generator.getEventType(tt.changeType)
			if eventType != tt.expectedType {
				t.Errorf("Expected event type %s, got %s", tt.expectedType, eventType)
			}
		})
	}
}

func TestEventGenerator_generateEventID(t *testing.T) {
	generator := NewEventGenerator()
	timestamp := time.Now()

	// Test that same input generates same ID
	id1 := generator.generateEventID("repo1", "main", "abc123", timestamp)
	id2 := generator.generateEventID("repo1", "main", "abc123", timestamp)

	if id1 != id2 {
		t.Error("Same input should generate same event ID")
	}

	// Test that different input generates different ID
	id3 := generator.generateEventID("repo2", "main", "abc123", timestamp)
	if id1 == id3 {
		t.Error("Different repository should generate different event ID")
	}

	id4 := generator.generateEventID("repo1", "develop", "abc123", timestamp)
	if id1 == id4 {
		t.Error("Different branch should generate different event ID")
	}

	id5 := generator.generateEventID("repo1", "main", "def456", timestamp)
	if id1 == id5 {
		t.Error("Different commit should generate different event ID")
	}

	id6 := generator.generateEventID("repo1", "main", "abc123", timestamp.Add(time.Second))
	if id1 == id6 {
		t.Error("Different timestamp should generate different event ID")
	}

	// Test ID format
	if !strings.HasPrefix(id1, "event_") {
		t.Errorf("Event ID should start with 'event_', got %s", id1)
	}
}

func TestEventGenerator_createEventFromChange(t *testing.T) {
	generator := NewEventGenerator()
	timestamp := time.Now()

	repo := types.Repository{
		Name:     "test-repo",
		Provider: "github",
		URL:      "https://github.com/test/repo",
	}

	change := BranchChange{
		Repository:   "test-repo",
		Branch:       "main",
		OldCommitSHA: "old123",
		NewCommitSHA: "new456",
		ChangeType:   ChangeTypeUpdated,
		Timestamp:    timestamp,
		Protected:    true,
	}

	event, err := generator.createEventFromChange(repo, change, timestamp)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Verify basic event properties
	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}
	if event.Type != types.EventTypeBranchUpdated {
		t.Errorf("Expected event type '%s', got %s", types.EventTypeBranchUpdated, event.Type)
	}
	if event.Repository != repo.Name {
		t.Errorf("Expected repository %s, got %s", repo.Name, event.Repository)
	}
	if event.Branch != change.Branch {
		t.Errorf("Expected branch %s, got %s", change.Branch, event.Branch)
	}
	if event.CommitSHA != change.NewCommitSHA {
		t.Errorf("Expected commit SHA %s, got %s", change.NewCommitSHA, event.CommitSHA)
	}
	if event.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", event.Status)
	}

	// Verify metadata
	metadata := event.Metadata
	if len(metadata) == 0 {
		t.Fatal("Metadata should not be empty")
	}

	expectedMetadata := map[string]string{
		"repository":     repo.Name,
		"provider":       repo.Provider,
		"branch":         change.Branch,
		"change_type":    change.ChangeType,
		"old_commit_sha": change.OldCommitSHA,
		"new_commit_sha": change.NewCommitSHA,
		"protected":      "true",
		"repository_url": repo.URL,
		"source":         "reposentry-poller",
		"poller_version": "1.0.0",
	}

	for key, expectedValue := range expectedMetadata {
		if metadata[key] != expectedValue {
			t.Errorf("Metadata[%s]: expected %v, got %v", key, expectedValue, metadata[key])
		}
	}
}

// Helper function to check if strings package is imported

func TestEventGenerator_Integration(t *testing.T) {
	generator := NewEventGenerator()
	ctx := context.Background()

	repo := types.Repository{
		Name:        "integration-test-repo",
		Provider:    "github",
		URL:         "https://github.com/test/integration",
		BranchRegex: "^(main|develop|feature/.*)$",
	}

	changes := []BranchChange{
		{
			Repository:   repo.Name,
			Branch:       "main",
			NewCommitSHA: "commit1",
			ChangeType:   ChangeTypeNew,
			Timestamp:    time.Now(),
			Protected:    true,
		},
		{
			Repository:   repo.Name,
			Branch:       "develop",
			OldCommitSHA: "old_commit",
			NewCommitSHA: "new_commit",
			ChangeType:   ChangeTypeUpdated,
			Timestamp:    time.Now(),
			Protected:    false,
		},
		{
			Repository:   repo.Name,
			Branch:       "feature/auth",
			NewCommitSHA: "feature_commit",
			ChangeType:   ChangeTypeNew,
			Timestamp:    time.Now(),
			Protected:    false,
		},
		{
			Repository:   repo.Name,
			Branch:       "hotfix/urgent",
			OldCommitSHA: "hotfix_old",
			ChangeType:   ChangeTypeDeleted,
			Timestamp:    time.Now(),
			Protected:    false,
		},
	}

	// Generate events
	events, err := generator.GenerateEvents(ctx, repo, changes)
	if err != nil {
		t.Fatalf("Failed to generate events: %v", err)
	}

	// Should filter out hotfix branch (doesn't match regex)
	expectedEventCount := 3
	if len(events) != expectedEventCount {
		t.Errorf("Expected %d events, got %d", expectedEventCount, len(events))
	}

	// Verify event types
	eventTypes := make(map[types.EventType]int)
	for _, event := range events {
		eventTypes[event.Type]++
	}

	if eventTypes[types.EventTypeBranchCreated] != 2 {
		t.Errorf("Expected 2 branch.created events, got %d", eventTypes[types.EventTypeBranchCreated])
	}
	if eventTypes[types.EventTypeBranchUpdated] != 1 {
		t.Errorf("Expected 1 branch.updated event, got %d", eventTypes[types.EventTypeBranchUpdated])
	}

	// Verify all events have unique IDs
	eventIDs := make(map[string]bool)
	for _, event := range events {
		if eventIDs[event.ID] {
			t.Errorf("Duplicate event ID found: %s", event.ID)
		}
		eventIDs[event.ID] = true
	}
}
