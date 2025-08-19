package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestNewServer(t *testing.T) {
	// Create test dependencies
	loggerInstance := logger.GetDefaultLogger()
	configManager := config.NewManager(loggerInstance)
	
	// Create in-memory storage for testing
	storageInstance, err := storage.NewSQLiteStorage(&types.SQLiteConfig{
		Path:              ":memory:",
		MaxConnections:    1,
		ConnectionTimeout: time.Duration(30) * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageInstance.Close()

	// Test server creation
	server := NewServer(8899, configManager, storageInstance)
	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.port != 8899 {
		t.Errorf("Expected port 8899, got %d", server.port)
	}

	if server.configManager != configManager {
		t.Error("Expected configManager to be set")
	}

	if server.storage != storageInstance {
		t.Error("Expected storage to be set")
	}
}

func TestServer_SetRuntime(t *testing.T) {
	// Create test dependencies
	loggerInstance := logger.GetDefaultLogger()
	configManager := config.NewManager(loggerInstance)
	
	storageInstance, err := storage.NewSQLiteStorage(&types.SQLiteConfig{
		Path:              ":memory:",
		MaxConnections:    1,
		ConnectionTimeout: time.Duration(30) * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageInstance.Close()

	server := NewServer(8898, configManager, storageInstance)

	// Create mock runtime provider
	mockRuntime := &mockRuntimeProvider{}
	server.SetRuntime(mockRuntime)

	if server.runtime != mockRuntime {
		t.Error("Expected runtime to be set")
	}
}

func TestServer_Health(t *testing.T) {
	// Create test dependencies
	loggerInstance := logger.GetDefaultLogger()
	configManager := config.NewManager(loggerInstance)
	
	storageInstance, err := storage.NewSQLiteStorage(&types.SQLiteConfig{
		Path:              ":memory:",
		MaxConnections:    1,
		ConnectionTimeout: time.Duration(30) * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageInstance.Close()

	server := NewServer(8897, configManager, storageInstance)

	// Test health check
	ctx := context.Background()
	err = server.Health(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Mock runtime provider for testing
type mockRuntimeProvider struct{}

func (m *mockRuntimeProvider) Health(ctx context.Context) RuntimeHealthStatus {
	return RuntimeHealthStatus{
		Healthy:    true,
		Components: make(map[string]ComponentHealth),
		Checks:     []HealthCheck{},
	}
}

func (m *mockRuntimeProvider) GetStatus() *RuntimeStatus {
	return &RuntimeStatus{
		State:      "running",
		StartedAt:  time.Now(),
		Uptime:     time.Minute,
		Version:    "test",
		Components: make(map[string]ComponentStatus),
	}
}

func TestServer_StartStop(t *testing.T) {
	// Create test dependencies
	loggerInstance := logger.GetDefaultLogger()
	configManager := config.NewManager(loggerInstance)
	
	storageInstance, err := storage.NewSQLiteStorage(&types.SQLiteConfig{
		Path:              ":memory:",
		MaxConnections:    1,
		ConnectionTimeout: time.Duration(30) * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageInstance.Close()

	server := NewServer(8896, configManager, storageInstance)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is actually listening
	resp, err := http.Get("http://localhost:8896/health")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test stop
	err = server.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Give server time to stop
	time.Sleep(100 * time.Millisecond)

	// Test that server is no longer listening
	_, err = http.Get("http://localhost:8896/health")
	if err == nil {
		t.Error("Expected connection to fail after server stop")
	}
}
