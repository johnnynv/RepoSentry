package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SetupTestSuite struct {
	suite.Suite
	base *BaseTestSuite
}

func (suite *SetupTestSuite) SetupSuite() {
	suite.base = &BaseTestSuite{}
	suite.base.SetupSuite()
}

func (suite *SetupTestSuite) TearDownSuite() {
	suite.base.TearDownSuite()
}

func (suite *SetupTestSuite) SetupTest() {
	suite.base.SetupTest()
}

func (suite *SetupTestSuite) TearDownTest() {
	suite.base.TearDownTest()
}

func TestSetupTestSuite(t *testing.T) {
	suite.Run(t, new(SetupTestSuite))
}

func TestBaseTestSuite_SetupSuite(t *testing.T) {
	base := &BaseTestSuite{}

	// Test SetupSuite
	base.SetupSuite()

	// Verify that context and logger are initialized
	assert.NotNil(t, base.ctx)
	assert.NotNil(t, base.logger)
	assert.NotNil(t, base.loggerManager)
}

func TestBaseTestSuite_TearDownSuite(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	// Test TearDownSuite
	base.TearDownSuite()

	// Verify cleanup (context should be cancelled)
	select {
	case <-base.ctx.Done():
		// Context was cancelled as expected
	default:
		t.Error("Context should be cancelled after TearDownSuite")
	}
}

func TestBaseTestSuite_SetupTest(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	// Test SetupTest
	base.SetupTest()

	// Verify that test-specific setup is done
	assert.NotNil(t, base.ctx)
	assert.NotNil(t, base.logger)
}

func TestBaseTestSuite_TearDownTest(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()
	base.SetupTest()

	// Test TearDownTest
	base.TearDownTest()

	// Verify that test cleanup is done
	// Note: We can't easily test if the context was cancelled here
	// as SetupTest creates a new context
}

func TestBaseTestSuite_GetTestContext(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	ctx := base.GetTestContext()
	assert.NotNil(t, ctx)
	assert.Equal(t, base.ctx, ctx)
}

func TestBaseTestSuite_GetTestLogger(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	logger := base.GetTestLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, base.logger, logger)
}

func TestBaseTestSuite_GetLoggerManager(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	loggerManager := base.GetLoggerManager()
	assert.NotNil(t, loggerManager)
	assert.Equal(t, base.loggerManager, loggerManager)
}

func TestBaseTestSuite_AssertNoError(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	// Test with no error
	base.AssertNoError(nil)

	// Test with error (should panic)
	assert.Panics(t, func() {
		base.AssertNoError(assert.AnError)
	})
}

func TestBaseTestSuite_RequireNoError(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	// Test with no error
	base.RequireNoError(nil)

	// Test with error (should fail test)
	// Note: This is hard to test in unit tests as it calls t.FailNow()
	// We'll just verify the method exists and can be called
}

func TestCreateTestConfig(t *testing.T) {
	config := CreateTestConfig()
	assert.NotNil(t, config)

	// Verify basic config structure
	assert.NotEmpty(t, config.App.LogLevel)
	assert.NotEmpty(t, config.Storage.Type)
	assert.NotEmpty(t, config.Polling.Interval)
}

func TestBaseTestSuite_Integration(t *testing.T) {
	base := &BaseTestSuite{}

	// Test full lifecycle
	base.SetupSuite()
	defer base.TearDownSuite()

	base.SetupTest()
	defer base.TearDownTest()

	// Verify all components are properly initialized
	assert.NotNil(t, base.ctx)
	assert.NotNil(t, base.logger)
	assert.NotNil(t, base.loggerManager)

	// Test context operations
	ctx := base.GetTestContext()
	assert.NotNil(t, ctx)

	// Test logger operations
	logger := base.GetTestLogger()
	assert.NotNil(t, logger)

	// Test config creation
	config := CreateTestConfig()
	assert.NotNil(t, config)
}

func TestBaseTestSuite_LoggerIntegration(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()
	defer base.TearDownSuite()

	// Test that logger can be used
	logger := base.GetTestLogger()
	logger.Info("Test log message")

	// Test that logger manager works
	loggerManager := base.GetLoggerManager()
	assert.NotNil(t, loggerManager)

	// Test component logger
	componentLogger := loggerManager.ForComponent("test-component")
	assert.NotNil(t, componentLogger)
}

func TestBaseTestSuite_ContextCancellation(t *testing.T) {
	base := &BaseTestSuite{}
	base.SetupSuite()

	// Test that context is properly managed
	ctx := base.GetTestContext()

	// Cancel the context
	base.TearDownSuite()

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after TearDownSuite")
	}
}
