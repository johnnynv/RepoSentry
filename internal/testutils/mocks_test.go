package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestMockStorage_Initialize(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	err := mock.Initialize(ctx)
	assert.NoError(t, err)
}

func TestMockStorage_Close(t *testing.T) {
	mock := NewMockStorage()

	err := mock.Close()
	assert.NoError(t, err)
}

func TestMockStorage_HealthCheck(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	status := mock.HealthCheck(ctx)
	assert.NoError(t, status)
}

func TestMockStorage_SaveRepoState(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	state := &types.RepoState{
		Repository:  "test-repo",
		Branch:      "main",
		CommitSHA:   "abc123",
		LastChecked: time.Now(),
	}

	// Set up mock expectations
	mock.On("SaveRepoState", ctx, state).Return(nil)

	err := mock.SaveRepoState(ctx, state)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetRepoState(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	repo := "test-repo"
	branch := "main"

	expectedState := &types.RepoState{
		Repository:  repo,
		Branch:      branch,
		CommitSHA:   "abc123",
		LastChecked: time.Now(),
	}

	// Set up mock expectations
	mock.On("GetRepoState", ctx, repo, branch).Return(expectedState, nil)

	state, err := mock.GetRepoState(ctx, repo, branch)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, expectedState, state)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetRepoStates(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	repo := "test-repo"

	expectedStates := []*types.RepoState{
		{
			Repository:  repo,
			Branch:      "main",
			CommitSHA:   "abc123",
			LastChecked: time.Now(),
		},
		{
			Repository:  repo,
			Branch:      "develop",
			CommitSHA:   "def456",
			LastChecked: time.Now(),
		},
	}

	// Set up mock expectations
	mock.On("GetRepoStates", ctx, repo).Return(expectedStates, nil)

	states, err := mock.GetRepoStates(ctx, repo)
	assert.NoError(t, err)
	assert.NotNil(t, states)
	assert.Equal(t, expectedStates, states)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetAllRepoStates(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	expectedStates := []*types.RepoState{
		{
			Repository:  "repo1",
			Branch:      "main",
			CommitSHA:   "abc123",
			LastChecked: time.Now(),
		},
		{
			Repository:  "repo2",
			Branch:      "develop",
			CommitSHA:   "def456",
			LastChecked: time.Now(),
		},
	}

	// Set up mock expectations
	mock.On("GetAllRepoStates", ctx).Return(expectedStates, nil)

	states, err := mock.GetAllRepoStates(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, states)
	assert.Equal(t, expectedStates, states)
	mock.AssertExpectations(t)
}

func TestMockStorage_DeleteRepoState(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	repo := "test-repo"
	branch := "main"

	// Set up mock expectations
	mock.On("DeleteRepoState", ctx, repo, branch).Return(nil)

	err := mock.DeleteRepoState(ctx, repo, branch)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_SaveEvent(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	event := &types.Event{
		ID:         "event-1",
		Repository: "test-repo",
		Branch:     "main",
		Type:       types.EventTypeBranchUpdated,
		CommitSHA:  "abc123",
		Timestamp:  time.Now(),
		Status:     types.EventStatusPending,
	}

	// Set up mock expectations
	mock.On("SaveEvent", ctx, event).Return(nil)

	err := mock.SaveEvent(ctx, event)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_CreateEvent(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	event := types.Event{
		ID:         "event-1",
		Repository: "test-repo",
		Branch:     "main",
		Type:       types.EventTypeBranchUpdated,
		CommitSHA:  "abc123",
		Timestamp:  time.Now(),
		Status:     types.EventStatusPending,
	}

	// Set up mock expectations
	mock.On("CreateEvent", ctx, event).Return(nil)

	err := mock.CreateEvent(ctx, event)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetEvent(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	eventID := "event-1"

	expectedEvent := &types.Event{
		ID:         eventID,
		Repository: "test-repo",
		Branch:     "main",
		Type:       types.EventTypeBranchUpdated,
		CommitSHA:  "abc123",
		Timestamp:  time.Now(),
		Status:     types.EventStatusPending,
	}

	// Set up mock expectations
	mock.On("GetEvent", ctx, eventID).Return(expectedEvent, nil)

	event, err := mock.GetEvent(ctx, eventID)
	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, expectedEvent, event)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetPendingEvents(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	limit := 10

	expectedEvents := []*types.Event{
		{
			ID:         "event-1",
			Repository: "test-repo",
			Branch:     "main",
			Type:       types.EventTypeBranchUpdated,
			CommitSHA:  "abc123",
			Timestamp:  time.Now(),
			Status:     types.EventStatusPending,
		},
		{
			ID:         "event-2",
			Repository: "test-repo",
			Branch:     "develop",
			Type:       types.EventTypeBranchCreated,
			CommitSHA:  "def456",
			Timestamp:  time.Now(),
			Status:     types.EventStatusPending,
		},
	}

	// Set up mock expectations
	mock.On("GetPendingEvents", ctx, limit).Return(expectedEvents, nil)

	events, err := mock.GetPendingEvents(ctx, limit)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Equal(t, expectedEvents, events)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetEventsByRepository(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	repo := "test-repo"
	limit := 10

	expectedEvents := []*types.Event{
		{
			ID:         "event-1",
			Repository: repo,
			Branch:     "main",
			Type:       types.EventTypeBranchUpdated,
			CommitSHA:  "abc123",
			Timestamp:  time.Now(),
			Status:     types.EventStatusPending,
		},
	}

	// Set up mock expectations
	mock.On("GetEventsByRepository", ctx, repo, limit).Return(expectedEvents, nil)

	events, err := mock.GetEventsByRepository(ctx, repo, limit)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Equal(t, expectedEvents, events)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetEvents(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	limit := 10
	offset := 0

	expectedEvents := []*types.Event{
		{
			ID:         "event-1",
			Repository: "test-repo",
			Branch:     "main",
			Type:       types.EventTypeBranchUpdated,
			CommitSHA:  "abc123",
			Timestamp:  time.Now(),
			Status:     types.EventStatusPending,
		},
	}

	// Set up mock expectations
	mock.On("GetEvents", ctx, limit, offset).Return(expectedEvents, nil)

	events, err := mock.GetEvents(ctx, limit, offset)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Equal(t, expectedEvents, events)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetEventsSince(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	expectedEvents := []*types.Event{
		{
			ID:         "event-1",
			Repository: "test-repo",
			Branch:     "main",
			Type:       types.EventTypeBranchUpdated,
			CommitSHA:  "abc123",
			Timestamp:  time.Now(),
			Status:     types.EventStatusPending,
		},
	}

	// Set up mock expectations
	mock.On("GetEventsSince", ctx, since).Return(expectedEvents, nil)

	events, err := mock.GetEventsSince(ctx, since)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Equal(t, expectedEvents, events)
	mock.AssertExpectations(t)
}

func TestMockStorage_UpdateEventStatus(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	eventID := "event-1"
	status := types.EventStatusProcessed

	// Set up mock expectations
	mock.On("UpdateEventStatus", ctx, eventID, status).Return(nil)

	err := mock.UpdateEventStatus(ctx, eventID, status)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_DeleteOldEvents(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	expectedCount := int64(5)

	// Set up mock expectations
	mock.On("DeleteOldEvents", ctx, olderThan).Return(expectedCount, nil)

	count, err := mock.DeleteOldEvents(ctx, olderThan)
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mock.AssertExpectations(t)
}

func TestMockStorage_UpsertRepoState(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()
	state := storage.RepositoryState{
		Repository: "test-repo",
		Branch:     "main",
		CommitSHA:  "abc123",
		LastCheck:  time.Now(),
	}

	// Set up mock expectations
	mock.On("UpsertRepoState", ctx, state).Return(nil)

	err := mock.UpsertRepoState(ctx, state)
	assert.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestMockStorage_GetStats(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	expectedStats := &storage.StorageStats{
		TotalRepositories:  5,
		TotalBranches:      10,
		TotalEvents:        25,
		PendingEvents:      3,
		FailedEvents:       1,
		LastEventTime:      time.Now(),
		OldestPendingEvent: time.Now().Add(-1 * time.Hour),
	}

	// Set up mock expectations
	mock.On("GetStats", ctx).Return(expectedStats, nil)

	stats, err := mock.GetStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, expectedStats, stats)
	mock.AssertExpectations(t)
}

func TestMockStorage_SetupMockBehavior(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	// Test setting up specific mock behavior
	expectedState := &types.RepoState{
		Repository:  "test-repo",
		Branch:      "main",
		CommitSHA:   "abc123",
		LastChecked: time.Now(),
	}

	mock.On("GetRepoState", ctx, "test-repo", "main").Return(expectedState, nil)

	state, err := mock.GetRepoState(ctx, "test-repo", "main")
	assert.NoError(t, err)
	assert.Equal(t, "test-repo", state.Repository)
	assert.Equal(t, "main", state.Branch)
	assert.Equal(t, "abc123", state.CommitSHA)

	mock.AssertExpectations(t)
}

func TestMockStorage_ContextSupport(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	// Test that context is properly passed through
	expectedState := &types.RepoState{
		Repository:  "test-repo",
		Branch:      "main",
		CommitSHA:   "abc123",
		LastChecked: time.Now(),
	}

	mock.On("GetRepoState", MockAny, "test-repo", "main").Return(expectedState, nil)

	state, err := mock.GetRepoState(ctx, "test-repo", "main")
	assert.NoError(t, err)
	assert.NotNil(t, state)

	mock.AssertExpectations(t)
}
