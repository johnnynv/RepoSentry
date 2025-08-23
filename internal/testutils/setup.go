package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// BaseTestSuite provides common test utilities
type BaseTestSuite struct {
	suite.Suite
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *logger.Logger
	loggerManager *logger.Manager
}

// SetupSuite runs before all tests in the suite
func (s *BaseTestSuite) SetupSuite() {
	// Create test context with timeout
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// Create test logger (silent for tests)
	var err error
	s.loggerManager, err = logger.NewManager(logger.Config{
		Level:  "error", // Suppress logs in tests
		Format: "json",
		Output: "stderr",
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create logger manager: %v", err))
	}

	s.logger = s.loggerManager.GetRootLogger()
}

// TearDownSuite runs after all tests in the suite
func (s *BaseTestSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

// SetupTest runs before each test
func (s *BaseTestSuite) SetupTest() {
	// Reset context for each test
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 10*time.Second)
}

// TearDownTest runs after each test
func (s *BaseTestSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
}

// GetTestContext returns test context
func (s *BaseTestSuite) GetTestContext() context.Context {
	return s.ctx
}

// GetTestLogger returns test logger
func (s *BaseTestSuite) GetTestLogger() *logger.Logger {
	return s.logger
}

// GetLoggerManager returns logger manager
func (s *BaseTestSuite) GetLoggerManager() *logger.Manager {
	return s.loggerManager
}

// AssertNoError is a convenience method
func (s *BaseTestSuite) AssertNoError(err error, msgAndArgs ...interface{}) {
	assert.NoError(s.T(), err, msgAndArgs...)
}

// RequireNoError is a convenience method that fails fast
func (s *BaseTestSuite) RequireNoError(err error, msgAndArgs ...interface{}) {
	if err != nil {
		panic(fmt.Sprintf("error occurred: %v", err))
	}
}

// CreateTestConfig creates a minimal test configuration
func CreateTestConfig() *types.Config {
	return &types.Config{
		App: types.AppConfig{
			Name:     "test-reposentry",
			LogLevel: "error",
			DataDir:  "/tmp/reposentry-test",
		},
		Polling: types.PollingConfig{
			Interval:   2 * time.Minute,
			Timeout:    30 * time.Second,
			MaxWorkers: 2,
		},
		Storage: types.StorageConfig{
			Type: "sqlite",
			SQLite: types.SQLiteConfig{
				Path:              ":memory:",
				MaxConnections:    5,
				ConnectionTimeout: 30 * time.Second,
			},
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          5 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name:     "test-repo",
				URL:      "https://github.com/test/repo",
				Provider: "github",
				Token:    "test-token",
				Enabled:  true,
			},
		},
	}
}
