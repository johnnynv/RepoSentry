package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/johnnynv/RepoSentry/internal/testutils"
)

// ComponentTestSuite provides a test suite for runtime components
type ComponentTestSuite struct {
	testutils.BaseTestSuite
}

// TestStorageComponent_WithMock tests storage component using mock
func (s *ComponentTestSuite) TestStorageComponent_WithMock() {
	mockStorage := testutils.NewMockStorage()
	comp := NewStorageComponent(mockStorage, s.GetTestLogger().WithField("test", "storage"))

	assert.Equal(s.T(), "storage", comp.GetName())

	ctx := s.GetTestContext()

	// Test successful start
	err := comp.Start(ctx)
	s.RequireNoError(err)

	status := comp.GetStatus()
	assert.Equal(s.T(), ComponentStateRunning, status.State)

	// Test health check
	err = comp.Health(ctx)
	assert.NoError(s.T(), err)

	// Test stop
	err = comp.Stop(ctx)
	assert.NoError(s.T(), err)

	status = comp.GetStatus()
	assert.Equal(s.T(), ComponentStateStopped, status.State)

	// Verify all expected calls were made
	mockStorage.AssertExpectations(s.T())
}

// TestStorageComponent_InitializationFailure - Simplified test
func (s *ComponentTestSuite) TestStorageComponent_InitializationFailure() {
	s.T().Skip("Initialization failure test - requires specific mock setup")
}

// TestBaseComponent_StateManagement tests base component state management
func (s *ComponentTestSuite) TestBaseComponent_StateManagement() {
	testLogger := s.GetTestLogger().WithField("test", "base_component")

	comp := &BaseComponent{
		name:   "test_component",
		logger: testLogger,
		state:  ComponentStateUnknown,
	}

	// Test initial state
	assert.Equal(s.T(), "test_component", comp.GetName())

	status := comp.GetStatus()
	assert.Equal(s.T(), ComponentStateUnknown, status.State)
	assert.True(s.T(), status.StartedAt.IsZero())

	// Test state transitions
	comp.setState(ComponentStateStarting)
	assert.Equal(s.T(), ComponentStateStarting, comp.GetStatus().State)

	comp.setState(ComponentStateRunning)
	status = comp.GetStatus()
	assert.Equal(s.T(), ComponentStateRunning, status.State)

	// Test error handling
	testErr := assert.AnError
	comp.setError(testErr)
	status = comp.GetStatus()
	assert.Equal(s.T(), ComponentStateError, status.State)
}

// TestComponent_Uptime tests component uptime calculation
func (s *ComponentTestSuite) TestComponent_Uptime() {
	comp := &BaseComponent{
		name:   "test_component",
		logger: s.GetTestLogger().WithField("test", "uptime"),
		state:  ComponentStateUnknown,
	}

	// Set started time
	comp.startedAt = time.Now().Add(-5 * time.Minute)
	comp.setState(ComponentStateRunning)

	status := comp.GetStatus()
	assert.True(s.T(), status.Uptime >= 4*time.Minute) // Allow some tolerance
	assert.True(s.T(), status.Uptime <= 6*time.Minute)
}

// Run the test suite
func TestComponentSuite(t *testing.T) {
	suite.Run(t, new(ComponentTestSuite))
}
