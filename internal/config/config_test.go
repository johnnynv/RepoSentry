package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/johnnynv/RepoSentry/internal/testutils"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// ConfigTestSuite provides a test suite for config package
type ConfigTestSuite struct {
	testutils.BaseTestSuite
	manager *Manager
}

// SetupTest runs before each test
func (s *ConfigTestSuite) SetupTest() {
	s.BaseTestSuite.SetupTest()
	s.manager = NewManager(s.GetTestLogger())
}

// TestConfigManager_Load tests configuration loading
func (s *ConfigTestSuite) TestConfigManager_Load() {
	err := s.manager.Load("../../test/fixtures/test-config.yaml")
	s.RequireNoError(err)

	config := s.manager.Get()
	require.NotNil(s.T(), config)

	assert.Equal(s.T(), "reposentry-test", config.App.Name)
	assert.Len(s.T(), config.Repositories, 2)
}

// TestConfigManager_LoadWithDefaults tests loading with defaults
func (s *ConfigTestSuite) TestConfigManager_LoadWithDefaults() {
	err := s.manager.LoadWithDefaults("nonexistent.yaml")

	// Should fail validation due to missing required fields
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation")
}

// TestConfigManager_GetRepositories tests repository retrieval
func (s *ConfigTestSuite) TestConfigManager_GetRepositories() {
	err := s.manager.Load("../../test/fixtures/test-config.yaml")
	s.RequireNoError(err)

	repos := s.manager.GetRepositories()
	assert.Len(s.T(), repos, 2)

	// All test repositories should be enabled
	for _, repo := range repos {
		assert.True(s.T(), repo.Enabled, "Repository %s should be enabled", repo.Name)
	}
}

// TestConfigManager_GetRepository tests single repository lookup
func (s *ConfigTestSuite) TestConfigManager_GetRepository() {
	err := s.manager.Load("../../test/fixtures/test-config.yaml")
	s.RequireNoError(err)

	// Test existing repository
	repo, found := s.manager.GetRepository("test-github-repo")
	assert.True(s.T(), found)
	require.NotNil(s.T(), repo)
	assert.Equal(s.T(), "github", repo.Provider)

	// Test non-existent repository
	_, found = s.manager.GetRepository("nonexistent")
	assert.False(s.T(), found)
}

// TestConfigManager_CheckPermissions tests permission checking
func (s *ConfigTestSuite) TestConfigManager_CheckPermissions() {
	// Create a test config with environment variable references
	testConfig := &types.Config{
		App: types.AppConfig{
			Name:     "test-app",
			LogLevel: "info",
			DataDir:  "/tmp/test",
		},
		Repositories: []types.Repository{
			{
				Name:     "test-github-repo",
				URL:      "https://github.com/test/repo",
				Provider: "github",
				Token:    "${GITHUB_TOKEN}",
				Enabled:  true,
			},
			{
				Name:     "test-gitlab-repo",
				URL:      "https://gitlab.com/test/repo",
				Provider: "gitlab",
				Token:    "${GITLAB_TOKEN}",
				Enabled:  true,
			},
		},
	}

	s.manager.SetConfig(testConfig)

	// Store and clear environment variables
	originalGithubToken := os.Getenv("GITHUB_TOKEN")
	originalGitlabToken := os.Getenv("GITLAB_TOKEN")

	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITLAB_TOKEN")

	defer func() {
		if originalGithubToken != "" {
			os.Setenv("GITHUB_TOKEN", originalGithubToken)
		}
		if originalGitlabToken != "" {
			os.Setenv("GITLAB_TOKEN", originalGitlabToken)
		}
	}()

	// Test without environment variables (should fail)
	err := s.manager.CheckPermissions()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "missing required tokens")

	// Set test environment variables
	os.Setenv("GITHUB_TOKEN", "test-github-token")
	os.Setenv("GITLAB_TOKEN", "test-gitlab-token")

	// Test with environment variables (should pass)
	err = s.manager.CheckPermissions()
	assert.NoError(s.T(), err)
}

// TestConfigManager_Validation tests configuration validation
func (s *ConfigTestSuite) TestConfigManager_Validation() {
	// Test valid config
	err := s.manager.Validate("../../test/fixtures/test-config.yaml")
	assert.NoError(s.T(), err)

	// Test invalid config file
	err = s.manager.Validate("nonexistent.yaml")
	assert.Error(s.T(), err)
}

// TestConfigManager_ThreadSafety tests concurrent access
func (s *ConfigTestSuite) TestConfigManager_ThreadSafety() {
	err := s.manager.Load("../../test/fixtures/test-config.yaml")
	s.RequireNoError(err)

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			config := s.manager.Get()
			assert.NotNil(s.T(), config)

			repos := s.manager.GetRepositories()
			assert.Len(s.T(), repos, 2)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Run the test suite
func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
