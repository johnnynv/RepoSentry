package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestBaseComponent_StateManagement(t *testing.T) {
	logger := logger.GetDefaultLogger().WithField("test", "base_component")
	
	comp := &BaseComponent{
		name:   "test_component",
		logger: logger,
		state:  ComponentStateUnknown,
	}

	// Test initial state
	if comp.GetName() != "test_component" {
		t.Errorf("Expected name 'test_component', got '%s'", comp.GetName())
	}

	status := comp.GetStatus()
	if status.Name != "test_component" {
		t.Errorf("Expected status name 'test_component', got '%s'", status.Name)
	}
	if status.State != ComponentStateUnknown {
		t.Errorf("Expected state %s, got %s", ComponentStateUnknown, status.State)
	}
	if status.Health != HealthStateUnknown {
		t.Errorf("Expected health %s, got %s", HealthStateUnknown, status.Health)
	}

	// Test state transitions
	comp.setState(ComponentStateRunning)
	status = comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s, got %s", ComponentStateRunning, status.State)
	}
	if status.Health != HealthStateHealthy {
		t.Errorf("Expected health %s for running state, got %s", HealthStateHealthy, status.Health)
	}

	// Test error state
	testErr := fmt.Errorf("test error")
	comp.setError(testErr)
	status = comp.GetStatus()
	if status.State != ComponentStateError {
		t.Errorf("Expected state %s, got %s", ComponentStateError, status.State)
	}
	if status.Health != HealthStateUnhealthy {
		t.Errorf("Expected health %s for error state, got %s", HealthStateUnhealthy, status.Health)
	}
	if status.LastError != "test error" {
		t.Errorf("Expected last error 'test error', got '%s'", status.LastError)
	}
}

func TestConfigComponent_Lifecycle(t *testing.T) {
	logger := logger.GetDefaultLogger()
	manager := config.NewManager(logger)
	
	comp := NewConfigComponent(*manager, logger.WithField("test", "config"))

	if comp.GetName() != "config" {
		t.Errorf("Expected name 'config', got '%s'", comp.GetName())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err := comp.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start config component: %v", err)
	}

	status := comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s after start, got %s", ComponentStateRunning, status.State)
	}

	// Test health check
	err = comp.Health(ctx)
	if err != nil {
		t.Errorf("Config component health check failed: %v", err)
	}

	// Test stop
	err = comp.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop config component: %v", err)
	}

	status = comp.GetStatus()
	if status.State != ComponentStateStopped {
		t.Errorf("Expected state %s after stop, got %s", ComponentStateStopped, status.State)
	}
}

func TestStorageComponent_Lifecycle(t *testing.T) {
	// Create in-memory SQLite storage for testing
	storageConfig := &types.SQLiteConfig{
		Path: ":memory:",
	}
	
	storage, err := storage.NewSQLiteStorage(storageConfig)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	logger := logger.GetDefaultLogger()
	comp := NewStorageComponent(storage, logger.WithField("test", "storage"))

	if comp.GetName() != "storage" {
		t.Errorf("Expected name 'storage', got '%s'", comp.GetName())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err = comp.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start storage component: %v", err)
	}

	status := comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s after start, got %s", ComponentStateRunning, status.State)
	}

	// Test health check
	err = comp.Health(ctx)
	if err != nil {
		t.Errorf("Storage component health check failed: %v", err)
	}

	// Test stop
	err = comp.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop storage component: %v", err)
	}

	status = comp.GetStatus()
	if status.State != ComponentStateStopped {
		t.Errorf("Expected state %s after stop, got %s", ComponentStateStopped, status.State)
	}
}

func TestGitClientFactoryComponent_Lifecycle(t *testing.T) {
	factory := gitclient.NewClientFactory()
	logger := logger.GetDefaultLogger()
	comp := NewGitClientFactoryComponent(factory, logger.WithField("test", "git_client"))

	if comp.GetName() != "git_client" {
		t.Errorf("Expected name 'git_client', got '%s'", comp.GetName())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err := comp.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start git client component: %v", err)
	}

	status := comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s after start, got %s", ComponentStateRunning, status.State)
	}

	// Test health check
	err = comp.Health(ctx)
	if err != nil {
		t.Errorf("Git client component health check failed: %v", err)
	}

	// Test stop
	err = comp.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop git client component: %v", err)
	}

	status = comp.GetStatus()
	if status.State != ComponentStateStopped {
		t.Errorf("Expected state %s after stop, got %s", ComponentStateStopped, status.State)
	}
}

func TestTriggerFactoryComponent_Lifecycle(t *testing.T) {
	factory := trigger.NewTriggerFactory()
	logger := logger.GetDefaultLogger()
	comp := NewTriggerFactoryComponent(factory, logger.WithField("test", "trigger"))

	if comp.GetName() != "trigger" {
		t.Errorf("Expected name 'trigger', got '%s'", comp.GetName())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err := comp.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start trigger component: %v", err)
	}

	status := comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s after start, got %s", ComponentStateRunning, status.State)
	}

	// Test health check
	err = comp.Health(ctx)
	if err != nil {
		t.Errorf("Trigger component health check failed: %v", err)
	}

	// Test stop
	err = comp.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop trigger component: %v", err)
	}

	status = comp.GetStatus()
	if status.State != ComponentStateStopped {
		t.Errorf("Expected state %s after stop, got %s", ComponentStateStopped, status.State)
	}
}

func TestHealthServerComponent_Lifecycle(t *testing.T) {
	// Create a mock runtime for the health server
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: ":memory:",
			},
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	runtime, err := NewRuntimeManager(config)
	if err != nil {
		t.Fatalf("Failed to create runtime for health server test: %v", err)
	}

	// Use a port that's likely to be available
	// Health server functionality now integrated into API server
	// Testing API component instead
	logger := logger.GetDefaultLogger()
	apiComp := NewAPIComponent(runtime.configManager, runtime.storage, 8899, runtime, logger.WithField("test", "api_server"))

	if comp.GetName() != "health_server" {
		t.Errorf("Expected name 'health_server', got '%s'", comp.GetName())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test start
	err = comp.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start health server component: %v", err)
	}

	status := comp.GetStatus()
	if status.State != ComponentStateRunning {
		t.Errorf("Expected state %s after start, got %s", ComponentStateRunning, status.State)
	}

	// Test health check
	err = comp.Health(ctx)
	if err != nil {
		t.Errorf("Health server component health check failed: %v", err)
	}

	// Test stop
	err = comp.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop health server component: %v", err)
	}

	status = comp.GetStatus()
	if status.State != ComponentStateStopped {
		t.Errorf("Expected state %s after stop, got %s", ComponentStateStopped, status.State)
	}
}

func TestComponent_StartedAtAndUptime(t *testing.T) {
	logger := logger.GetDefaultLogger()
	manager := config.NewManager(logger)
	comp := NewConfigComponent(*manager, logger.WithField("test", "config"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Before start, StartedAt should be zero and uptime should be zero
	status := comp.GetStatus()
	if !status.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be zero before start")
	}
	if status.Uptime != 0 {
		t.Error("Expected Uptime to be zero before start")
	}

	// Start component
	err := comp.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start component: %v", err)
	}
	defer comp.Stop(ctx)

	// After start, StartedAt should be set and uptime should be positive
	status = comp.GetStatus()
	if status.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set after start")
	}
	if status.Uptime <= 0 {
		t.Error("Expected positive uptime after start")
	}

	// Wait a bit and check that uptime increases
	time.Sleep(10 * time.Millisecond)
	newStatus := comp.GetStatus()
	if newStatus.Uptime <= status.Uptime {
		t.Error("Expected uptime to increase over time")
	}
}
