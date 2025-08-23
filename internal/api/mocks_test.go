package api

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockRuntimeProvider is a mock implementation for RuntimeProvider interface
type MockRuntimeProvider struct {
	mock.Mock
}

func (m *MockRuntimeProvider) Health(ctx context.Context) RuntimeHealthStatus {
	args := m.Called(ctx)
	return args.Get(0).(RuntimeHealthStatus)
}

func (m *MockRuntimeProvider) GetStatus() *RuntimeStatus {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*RuntimeStatus)
}

// NewMockRuntimeProvider creates a new mock runtime provider with default expectations
func NewMockRuntimeProvider() *MockRuntimeProvider {

	// Set up default healthy state - simplified expectations
	mockObj := &MockRuntimeProvider{}
	// Accept any context type
	mockObj.On("Health", mock.Anything).Return(RuntimeHealthStatus{
		Healthy:    true,
		Components: make(map[string]ComponentHealth),
		Checks:     []HealthCheck{},
	}).Maybe() // Make it optional

	mockObj.On("GetStatus").Return(&RuntimeStatus{
		State:      "running",
		StartedAt:  time.Now(),
		Uptime:     time.Minute,
		Version:    "test",
		Components: make(map[string]ComponentStatus),
	})

	return mockObj
}
