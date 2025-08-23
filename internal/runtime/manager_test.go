package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestRuntimeManager_NewRuntimeManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *types.Config
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 8080,
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: "/tmp/test-reposentry-test.db",
					},
				},
				Tekton: types.TektonConfig{
					EventListenerURL: "http://localhost:8080/webhook",
					Timeout:          10 * time.Second,
				},
				Repositories: []types.Repository{
					{
						Name: "test-repo",
						URL:  "https://github.com/owner/repo",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil configuration",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger manager
			loggerManager, _ := logger.NewManager(logger.Config{
				Level:  "info",
				Format: "json",
			})

			rm, err := NewRuntimeManager(tt.config, loggerManager)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if rm == nil {
				t.Error("Expected runtime manager but got nil")
				return
			}

			// Verify initial state
			if rm.state != RuntimeStateUnknown {
				t.Errorf("Expected initial state %s, got %s", RuntimeStateUnknown, rm.state)
			}

			// Verify components are initialized
			expectedComponents := []string{"config", "storage", "git_client", "trigger"}
			if tt.config.App.HealthCheckPort > 0 {
				expectedComponents = append(expectedComponents, "api_server")
			}

			for _, compName := range expectedComponents {
				if _, exists := rm.components[compName]; !exists {
					t.Errorf("Expected component %s to be initialized", compName)
				}
			}

			// Clean up
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = rm.Stop(ctx)
		})
	}
}

func TestRuntimeManager_StartStop(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:            "test-reposentry",
			DataDir:         "/tmp/test",
			HealthCheckPort: 0, // Disable health server for simpler test
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-startstop.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test Start
	err = rm.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start runtime: %v", err)
	}

	// Verify state
	if rm.state != RuntimeStateRunning {
		t.Errorf("Expected state %s after start, got %s", RuntimeStateRunning, rm.state)
	}

	// Verify components are running
	for name, component := range rm.components {
		status := component.GetStatus()
		if status.State != ComponentStateRunning {
			t.Errorf("Component %s expected state %s, got %s", name, ComponentStateRunning, status.State)
		}
	}

	// Test Stop
	err = rm.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop runtime: %v", err)
	}

	// Verify state
	if rm.state != RuntimeStateStopped {
		t.Errorf("Expected state %s after stop, got %s", RuntimeStateStopped, rm.state)
	}
}

func TestRuntimeManager_Health(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-health.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start runtime
	err = rm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rm.Stop(ctx)

	// Test health check
	health, err := rm.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	if health == nil {
		t.Fatal("Expected health status but got nil")
	}

	if health.Status != HealthStateHealthy {
		t.Errorf("Expected health status %s, got %s", HealthStateHealthy, health.Status)
	}

	// Verify all components are reported as healthy
	expectedComponents := []string{"config", "storage", "git_client", "trigger"}
	for _, compName := range expectedComponents {
		if status, exists := health.Components[compName]; !exists {
			t.Errorf("Component %s not found in health status", compName)
		} else if status != HealthStateHealthy {
			t.Errorf("Component %s expected health %s, got %s", compName, HealthStateHealthy, status)
		}
	}

	// Verify health checks are present
	if len(health.Checks) == 0 {
		t.Error("Expected health checks but got none")
	}
}

func TestRuntimeManager_GetStatus(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-status.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test status before start
	status := rm.GetStatus()
	if status.State != RuntimeStateUnknown {
		t.Errorf("Expected initial state %s, got %s", RuntimeStateUnknown, status.State)
	}

	// Start runtime
	err = rm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rm.Stop(ctx)

	// Test status after start
	status = rm.GetStatus()
	if status.State != RuntimeStateRunning {
		t.Errorf("Expected state %s after start, got %s", RuntimeStateRunning, status.State)
	}

	if status.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}

	if status.Version == "" {
		t.Error("Expected version to be set")
	}

	// Verify component statuses
	expectedComponents := []string{"config", "storage", "git_client", "trigger"}
	for _, compName := range expectedComponents {
		if compStatus, exists := status.Components[compName]; !exists {
			t.Errorf("Component %s not found in status", compName)
		} else if compStatus.State != ComponentStateRunning {
			t.Errorf("Component %s expected state %s, got %s", compName, ComponentStateRunning, compStatus.State)
		}
	}
}

func TestRuntimeManager_Reload(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-reload.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start runtime
	err = rm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rm.Stop(ctx)

	// Test reload
	err = rm.Reload(ctx)
	if err != nil {
		t.Errorf("Failed to reload runtime: %v", err)
	}

	// Verify runtime is still running
	if rm.state != RuntimeStateRunning {
		t.Errorf("Expected state %s after reload, got %s", RuntimeStateRunning, rm.state)
	}
}

func TestRuntimeManager_DoubleStart(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-double.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First start should succeed
	err = rm.Start(ctx)
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	defer rm.Stop(ctx)

	// Second start should fail
	err = rm.Start(ctx)
	if err == nil {
		t.Error("Expected error on double start but got none")
	}
}

func TestRuntimeManager_StopWithoutStart(t *testing.T) {
	config := &types.Config{
		App: types.AppConfig{
			Name:    "test-reposentry",
			DataDir: "/tmp/test",
		},
		Storage: types.StorageConfig{
			SQLite: types.SQLiteConfig{
				Path: "/tmp/test-reposentry-stop.db",
			},
		},
		Polling: types.PollingConfig{
			Interval: 1 * time.Minute, // Set a valid polling interval
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "http://localhost:8080/webhook",
			Timeout:          10 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name: "test-repo",
				URL:  "https://github.com/owner/repo",
			},
		},
	}

	// Create test logger manager
	loggerManager, _ := logger.NewManager(logger.Config{
		Level:  "info",
		Format: "json",
	})
	rm, err := NewRuntimeManager(config, loggerManager)
	if err != nil {
		t.Fatalf("Failed to create runtime manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop without start should not error
	err = rm.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without start failed: %v", err)
	}
}

// cleanupTestDBs removes test database files
func cleanupTestDBs() {
	testDBs := []string{
		"/tmp/test-reposentry-test.db",
		"/tmp/test-reposentry-startstop.db",
		"/tmp/test-reposentry-health.db",
		"/tmp/test-reposentry-status.db",
		"/tmp/test-reposentry-reload.db",
		"/tmp/test-reposentry-double.db",
		"/tmp/test-reposentry-stop.db",
	}

	for _, dbPath := range testDBs {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Clean up test databases
	cleanupTestDBs()

	os.Exit(code)
}
