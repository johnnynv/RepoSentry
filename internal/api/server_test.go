package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/testutils"
	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// Use existing mock from mocks_test.go

func TestServerSuite(t *testing.T) {
	t.Run("TestServer_Configuration", func(t *testing.T) {
		// Test server configuration
		port := 8080
		configManager := &config.Manager{}
		storage := testutils.NewMockStorage()
		loggerEntry := logger.GetDefaultLogger().WithField("test", "api").WithField("test", "server")

		server := NewServer(port, configManager, storage, loggerEntry)

		if server.port != port {
			t.Errorf("Expected port %d, got %d", port, server.port)
		}

		if server.configManager != configManager {
			t.Error("Expected configManager to be set")
		}

		if server.storage != storage {
			t.Error("Expected storage to be set")
		}

		if server.logger == nil {
			t.Error("Expected logger to be set")
		}
	})

	t.Run("TestServer_Creation", func(t *testing.T) {
		// Test server creation with different ports
		ports := []int{8080, 9090, 0}

		for _, port := range ports {
			server := NewServer(port, &config.Manager{}, testutils.NewMockStorage(), logger.GetDefaultLogger().WithField("test", "api"))
			if server == nil {
				t.Errorf("Failed to create server with port %d", port)
			}
		}
	})
}

func TestServer_HealthHandlers(t *testing.T) {
	// Create test server
	server := NewServer(8080, &config.Manager{}, testutils.NewMockStorage(), logger.GetDefaultLogger().WithField("test", "api"))

	t.Run("TestHealthHandler_WithoutRuntime", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Check response contains expected fields
		response := w.Body.String()
		if !contains(response, "healthy") {
			t.Errorf("Expected response to contain 'healthy', got: %s", response)
		}
	})

	t.Run("TestHealthHandler_WithRuntime", func(t *testing.T) {
		// Set mock runtime
		mockRuntime := NewMockRuntimeProvider()
		server.SetRuntime(mockRuntime)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("TestLivenessHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/live", nil)
		w := httptest.NewRecorder()

		server.handleLiveness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		response := w.Body.String()
		if !contains(response, "alive") {
			t.Errorf("Expected response to contain 'alive', got: %s", response)
		}
	})

	t.Run("TestReadinessHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/ready", nil)
		w := httptest.NewRecorder()

		server.handleReadiness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		response := w.Body.String()
		if !contains(response, "ready") {
			t.Errorf("Expected response to contain 'ready', got: %s", response)
		}
	})
}

func TestServer_StatusHandlers(t *testing.T) {
	server := NewServer(8080, &config.Manager{}, testutils.NewMockStorage(), logger.GetDefaultLogger().WithField("test", "api"))

	t.Run("TestStatusHandler_WithoutRuntime", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()

		server.handleStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		response := w.Body.String()
		if !contains(response, "running") {
			t.Errorf("Expected response to contain 'running', got: %s", response)
		}
	})

	t.Run("TestStatusHandler_WithRuntime", func(t *testing.T) {
		// Set mock runtime with status
		mockRuntime := NewMockRuntimeProvider()
		server.SetRuntime(mockRuntime)

		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()

		server.handleStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestServer_StartStop(t *testing.T) {
	server := NewServer(0, &config.Manager{}, testutils.NewMockStorage(), logger.GetDefaultLogger().WithField("test", "api")) // Port 0 for testing

	t.Run("TestServerStart", func(t *testing.T) {
		ctx := context.Background()
		err := server.Start(ctx)
		if err != nil {
			t.Errorf("Failed to start server: %v", err)
		}

		// Give server time to start
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("TestServerStop", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Stop(ctx)
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	})
}

func TestServer_SetRuntime(t *testing.T) {
	server := NewServer(8080, &config.Manager{}, testutils.NewMockStorage(), logger.GetDefaultLogger().WithField("test", "api"))

	// Initially runtime should be nil
	if server.runtime != nil {
		t.Error("Expected runtime to be nil initially")
	}

	// Set runtime
	mockRuntime := &MockRuntimeProvider{}
	server.SetRuntime(mockRuntime)

	if server.runtime != mockRuntime {
		t.Error("Expected runtime to be set")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:], substr)))
}
