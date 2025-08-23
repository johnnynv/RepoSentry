package testutils

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// MockAny can be used in mock expectations for any argument
var MockAny = mock.Anything

// MockStorage is a mock implementation of storage.Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStorage) SaveRepoState(ctx context.Context, state *types.RepoState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockStorage) GetRepoState(ctx context.Context, repository, branch string) (*types.RepoState, error) {
	args := m.Called(ctx, repository, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.RepoState), args.Error(1)
}

func (m *MockStorage) GetRepoStates(ctx context.Context, repository string) ([]*types.RepoState, error) {
	args := m.Called(ctx, repository)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.RepoState), args.Error(1)
}

func (m *MockStorage) GetAllRepoStates(ctx context.Context) ([]*types.RepoState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.RepoState), args.Error(1)
}

func (m *MockStorage) DeleteRepoState(ctx context.Context, repository, branch string) error {
	args := m.Called(ctx, repository, branch)
	return args.Error(0)
}

func (m *MockStorage) SaveEvent(ctx context.Context, event *types.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockStorage) CreateEvent(ctx context.Context, event types.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockStorage) GetEvent(ctx context.Context, eventID string) (*types.Event, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Event), args.Error(1)
}

func (m *MockStorage) GetPendingEvents(ctx context.Context, limit int) ([]*types.Event, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Event), args.Error(1)
}

func (m *MockStorage) GetEventsByRepository(ctx context.Context, repository string, limit int) ([]*types.Event, error) {
	args := m.Called(ctx, repository, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Event), args.Error(1)
}

func (m *MockStorage) GetEvents(ctx context.Context, limit, offset int) ([]*types.Event, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Event), args.Error(1)
}

func (m *MockStorage) GetEventsSince(ctx context.Context, since time.Time) ([]*types.Event, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Event), args.Error(1)
}

func (m *MockStorage) UpdateEventStatus(ctx context.Context, eventID string, status types.EventStatus) error {
	args := m.Called(ctx, eventID, status)
	return args.Error(0)
}

func (m *MockStorage) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) UpsertRepoState(ctx context.Context, state storage.RepositoryState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockStorage) GetStats(ctx context.Context) (*storage.StorageStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.StorageStats), args.Error(1)
}

// MockRuntimeProvider removed - use api.MockRuntimeProvider for API tests

// NewMockStorage creates a new mock storage with common expectations
func NewMockStorage() *MockStorage {
	mock := &MockStorage{}

	// Set up common successful operations - make them optional
	mock.On("Initialize", MockAny).Return(nil).Maybe()
	mock.On("Close").Return(nil).Maybe()
	mock.On("HealthCheck", MockAny).Return(nil).Maybe()

	return mock
}

// NewMockRuntimeProvider removed - use api.NewMockRuntimeProvider for API tests
