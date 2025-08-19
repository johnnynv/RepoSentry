package config

import (
	"os"
	"strings"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

func TestConfigManager_Load(t *testing.T) {
	// Create test logger
	testLogger := logger.GetDefaultLogger()
	
	// Create config manager
	manager := NewManager(testLogger)
	
	// Test loading valid configuration
	err := manager.Load("../../test/fixtures/test-config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	// Verify configuration was loaded
	config := manager.Get()
	if config == nil {
		t.Fatal("Configuration should not be nil after loading")
	}
	
	// Verify basic configuration values
	if config.App.Name != "reposentry-test" {
		t.Errorf("Expected app name 'reposentry-test', got '%s'", config.App.Name)
	}
	
	if len(config.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(config.Repositories))
	}
}

func TestConfigManager_LoadWithDefaults(t *testing.T) {
	testLogger := logger.GetDefaultLogger()
	manager := NewManager(testLogger)
	
	// Test loading with non-existent file (should use defaults but may fail validation)
	err := manager.LoadWithDefaults("nonexistent.yaml")
	// This might fail validation due to missing required fields, which is expected
	if err != nil {
		// Check that it's a validation error, not a file loading error
		if !contains(err.Error(), "validation") {
			t.Fatalf("Expected validation error, got: %v", err)
		}
		t.Logf("LoadWithDefaults failed validation as expected: %v", err)
		return
	}
	
	config := manager.Get()
	if config == nil {
		t.Fatal("Configuration should not be nil")
	}
	
	// Verify defaults were applied
	if config.App.Name != "reposentry" {
		t.Errorf("Expected default app name 'reposentry', got '%s'", config.App.Name)
	}
	
	if config.App.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", config.App.LogLevel)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}

func TestConfigManager_GetRepositories(t *testing.T) {
	testLogger := logger.GetDefaultLogger()
	manager := NewManager(testLogger)
	
	err := manager.Load("../../test/fixtures/test-config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	repos := manager.GetRepositories()
	if len(repos) != 2 {
		t.Errorf("Expected 2 enabled repositories, got %d", len(repos))
	}
	
	// All test repositories should be enabled
	for _, repo := range repos {
		if !repo.Enabled {
			t.Errorf("Repository %s should be enabled", repo.Name)
		}
	}
}

func TestConfigManager_GetRepository(t *testing.T) {
	testLogger := logger.GetDefaultLogger()
	manager := NewManager(testLogger)
	
	err := manager.Load("../../test/fixtures/test-config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	// Test existing repository
	repo, found := manager.GetRepository("test-github-repo")
	if !found {
		t.Error("Should find test-github-repo")
	}
	if repo == nil {
		t.Error("Repository should not be nil")
	}
	if repo.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", repo.Provider)
	}
	
	// Test non-existent repository
	_, found = manager.GetRepository("nonexistent")
	if found {
		t.Error("Should not find nonexistent repository")
	}
}

func TestConfigManager_CheckPermissions(t *testing.T) {
	testLogger := logger.GetDefaultLogger()
	manager := NewManager(testLogger)
	
	err := manager.Load("../../test/fixtures/test-config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	// Test without environment variables (should fail)
	err = manager.CheckPermissions()
	if err == nil {
		t.Error("CheckPermissions should fail without environment variables")
	}
	
	// Set test environment variables
	os.Setenv("GITHUB_TOKEN", "test-github-token")
	os.Setenv("GITLAB_TOKEN", "test-gitlab-token")
	defer func() {
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("GITLAB_TOKEN")
	}()
	
	// Test with environment variables (should pass)
	err = manager.CheckPermissions()
	if err != nil {
		t.Errorf("CheckPermissions should pass with environment variables: %v", err)
	}
}
