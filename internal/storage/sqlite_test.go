package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestSQLiteStorage_Initialize(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	err := storage.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test health check
	err = storage.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestSQLiteStorage_RepoState(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test saving a repository state
	state := &types.RepoState{
		Repository:  "test/repo",
		Branch:      "main",
		CommitSHA:   "abc123",
		LastChecked: time.Now(),
	}

	err := storage.SaveRepoState(ctx, state)
	if err != nil {
		t.Fatalf("Failed to save repository state: %v", err)
	}

	// Verify state was saved and ID was set
	if state.ID == 0 {
		t.Error("Expected ID to be set after saving")
	}

	// Test retrieving the state
	retrieved, err := storage.GetRepoState(ctx, "test/repo", "main")
	if err != nil {
		t.Fatalf("Failed to get repository state: %v", err)
	}

	if retrieved.Repository != state.Repository {
		t.Errorf("Expected repository %s, got %s", state.Repository, retrieved.Repository)
	}
	if retrieved.Branch != state.Branch {
		t.Errorf("Expected branch %s, got %s", state.Branch, retrieved.Branch)
	}
	if retrieved.CommitSHA != state.CommitSHA {
		t.Errorf("Expected commit SHA %s, got %s", state.CommitSHA, retrieved.CommitSHA)
	}

	// Test updating the state
	state.CommitSHA = "def456"
	err = storage.SaveRepoState(ctx, state)
	if err != nil {
		t.Fatalf("Failed to update repository state: %v", err)
	}

	// Verify update
	updated, err := storage.GetRepoState(ctx, "test/repo", "main")
	if err != nil {
		t.Fatalf("Failed to get updated repository state: %v", err)
	}

	if updated.CommitSHA != "def456" {
		t.Errorf("Expected updated commit SHA def456, got %s", updated.CommitSHA)
	}

	// Test getting non-existent state
	_, err = storage.GetRepoState(ctx, "nonexistent", "main")
	if err == nil {
		t.Error("Expected error for non-existent repository state")
	}

	if _, ok := err.(*RepositoryNotFoundError); !ok {
		t.Errorf("Expected RepositoryNotFoundError, got %T", err)
	}
}

func TestSQLiteStorage_GetRepoStates(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Save multiple states for the same repository
	states := []*types.RepoState{
		{Repository: "test/repo", Branch: "main", CommitSHA: "abc123", LastChecked: time.Now()},
		{Repository: "test/repo", Branch: "develop", CommitSHA: "def456", LastChecked: time.Now()},
		{Repository: "other/repo", Branch: "main", CommitSHA: "ghi789", LastChecked: time.Now()},
	}

	for _, state := range states {
		if err := storage.SaveRepoState(ctx, state); err != nil {
			t.Fatalf("Failed to save repository state: %v", err)
		}
	}

	// Get states for specific repository
	repoStates, err := storage.GetRepoStates(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository states: %v", err)
	}

	if len(repoStates) != 2 {
		t.Errorf("Expected 2 states for test/repo, got %d", len(repoStates))
	}

	// Get all states
	allStates, err := storage.GetAllRepoStates(ctx)
	if err != nil {
		t.Fatalf("Failed to get all repository states: %v", err)
	}

	if len(allStates) != 3 {
		t.Errorf("Expected 3 total states, got %d", len(allStates))
	}
}

func TestSQLiteStorage_Events(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test saving an event
	event := &types.Event{
		ID:         "event-123",
		Type:       types.EventTypeBranchUpdated,
		Repository: "test/repo",
		Branch:     "main",
		CommitSHA:  "abc123",
		PrevCommit: "xyz789",
		Provider:   "github",
		Timestamp:  time.Now(),
		Metadata:   map[string]string{"author": "test-user"},
		Status:     types.EventStatusPending,
	}

	err := storage.SaveEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Test retrieving the event
	retrieved, err := storage.GetEvent(ctx, "event-123")
	if err != nil {
		t.Fatalf("Failed to get event: %v", err)
	}

	if retrieved.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, retrieved.ID)
	}
	if retrieved.Type != event.Type {
		t.Errorf("Expected event type %s, got %s", event.Type, retrieved.Type)
	}
	if retrieved.Status != event.Status {
		t.Errorf("Expected status %s, got %s", event.Status, retrieved.Status)
	}
	if retrieved.Metadata["author"] != "test-user" {
		t.Errorf("Expected metadata author test-user, got %s", retrieved.Metadata["author"])
	}

	// Test duplicate event
	err = storage.SaveEvent(ctx, event)
	if err == nil {
		t.Error("Expected error for duplicate event")
	}
	if _, ok := err.(*DuplicateEventError); !ok {
		t.Errorf("Expected DuplicateEventError, got %T", err)
	}

	// Test updating event status
	err = storage.UpdateEventStatus(ctx, "event-123", types.EventStatusProcessed)
	if err != nil {
		t.Fatalf("Failed to update event status: %v", err)
	}

	// Verify status update
	updated, err := storage.GetEvent(ctx, "event-123")
	if err != nil {
		t.Fatalf("Failed to get updated event: %v", err)
	}

	if updated.Status != types.EventStatusProcessed {
		t.Errorf("Expected status processed, got %s", updated.Status)
	}
	if updated.ProcessedAt == nil {
		t.Error("Expected ProcessedAt to be set")
	}

	// Test getting non-existent event
	_, err = storage.GetEvent(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent event")
	}
	if _, ok := err.(*EventNotFoundError); !ok {
		t.Errorf("Expected EventNotFoundError, got %T", err)
	}
}

func TestSQLiteStorage_GetPendingEvents(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Save multiple events with different statuses
	events := []*types.Event{
		{ID: "event-1", Type: types.EventTypeBranchUpdated, Repository: "repo1", Branch: "main", 
		 CommitSHA: "abc", Provider: "github", Timestamp: time.Now(), Status: types.EventStatusPending},
		{ID: "event-2", Type: types.EventTypeBranchUpdated, Repository: "repo2", Branch: "main", 
		 CommitSHA: "def", Provider: "github", Timestamp: time.Now(), Status: types.EventStatusProcessed},
		{ID: "event-3", Type: types.EventTypeBranchUpdated, Repository: "repo3", Branch: "main", 
		 CommitSHA: "ghi", Provider: "github", Timestamp: time.Now(), Status: types.EventStatusPending},
	}

	for _, event := range events {
		if err := storage.SaveEvent(ctx, event); err != nil {
			t.Fatalf("Failed to save event: %v", err)
		}
	}

	// Get pending events
	pending, err := storage.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending events: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending events, got %d", len(pending))
	}

	// Verify all returned events are pending
	for _, event := range pending {
		if event.Status != types.EventStatusPending {
			t.Errorf("Expected pending status, got %s", event.Status)
		}
	}
}

func TestSQLiteStorage_GetStats(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add some test data
	state := &types.RepoState{
		Repository: "test/repo", Branch: "main", CommitSHA: "abc123", LastChecked: time.Now(),
	}
	if err := storage.SaveRepoState(ctx, state); err != nil {
		t.Fatalf("Failed to save repository state: %v", err)
	}

	event := &types.Event{
		ID: "event-1", Type: types.EventTypeBranchUpdated, Repository: "test/repo", Branch: "main",
		CommitSHA: "abc123", Provider: "github", Timestamp: time.Now(), Status: types.EventStatusPending,
	}
	if err := storage.SaveEvent(ctx, event); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Get stats
	stats, err := storage.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalRepositories != 1 {
		t.Errorf("Expected 1 repository, got %d", stats.TotalRepositories)
	}
	if stats.TotalBranches != 1 {
		t.Errorf("Expected 1 branch, got %d", stats.TotalBranches)
	}
	if stats.TotalEvents != 1 {
		t.Errorf("Expected 1 event, got %d", stats.TotalEvents)
	}
	if stats.PendingEvents != 1 {
		t.Errorf("Expected 1 pending event, got %d", stats.PendingEvents)
	}
}

func TestSQLiteStorage_DeleteRepoState(t *testing.T) {
	storage, cleanup := createTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Save a state
	state := &types.RepoState{
		Repository: "test/repo", Branch: "main", CommitSHA: "abc123", LastChecked: time.Now(),
	}
	if err := storage.SaveRepoState(ctx, state); err != nil {
		t.Fatalf("Failed to save repository state: %v", err)
	}

	// Delete the state
	err := storage.DeleteRepoState(ctx, "test/repo", "main")
	if err != nil {
		t.Fatalf("Failed to delete repository state: %v", err)
	}

	// Verify it's gone
	_, err = storage.GetRepoState(ctx, "test/repo", "main")
	if err == nil {
		t.Error("Expected error after deleting repository state")
	}

	// Test deleting non-existent state
	err = storage.DeleteRepoState(ctx, "nonexistent", "main")
	if err == nil {
		t.Error("Expected error for deleting non-existent state")
	}
	if _, ok := err.(*RepositoryNotFoundError); !ok {
		t.Errorf("Expected RepositoryNotFoundError, got %T", err)
	}
}

// createTestStorage creates a test storage instance with a temporary database
func createTestStorage(t *testing.T) (*SQLiteStorage, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := &types.SQLiteConfig{
		Path:              dbPath,
		MaxConnections:    5,
		ConnectionTimeout: 10 * time.Second,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.RemoveAll(tempDir)
	}

	return storage, cleanup
}
