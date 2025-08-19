package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/internal/api"
)

// Mock runtime for testing
type mockRuntime struct {
	healthStatus *HealthStatus
	runtimeStatus *RuntimeStatus
}

func (m *mockRuntime) Start(ctx context.Context) error {
	return nil
}

func (m *mockRuntime) Stop(ctx context.Context) error {
	return nil
}

func (m *mockRuntime) Health(ctx context.Context) (*HealthStatus, error) {
	return m.healthStatus, nil
}

func (m *mockRuntime) GetStatus() *RuntimeStatus {
	return m.runtimeStatus
}

func (m *mockRuntime) Reload(ctx context.Context) error {
	return nil
}

func TestNewRuntimeAPIAdapter(t *testing.T) {
	mockRuntime := &mockRuntime{}
	adapter := newRuntimeAPIAdapter(mockRuntime)

	if adapter == nil {
		t.Fatal("Expected adapter to be created, got nil")
	}

	// Verify adapter implements the interface
	var _ api.RuntimeProvider = adapter
}

func TestRuntimeAPIAdapter_Health(t *testing.T) {
	mockRuntime := &mockRuntime{
		healthStatus: &HealthStatus{
			Status:    HealthStateHealthy,
			Timestamp: time.Now(),
			Components: map[string]HealthState{
				"config":  HealthStateHealthy,
				"storage": HealthStateHealthy,
			},
			Checks: []HealthCheck{
				{
					Name:     "config",
					Status:   HealthStateHealthy,
					Duration: time.Millisecond * 10,
					Message:  "Config is healthy",
				},
				{
					Name:     "storage", 
					Status:   HealthStateHealthy,
					Duration: time.Millisecond * 20,
					Message:  "Storage is healthy",
				},
			},
		},
	}

	adapter := newRuntimeAPIAdapter(mockRuntime)
	ctx := context.Background()

	health := adapter.Health(ctx)

	if !health.Healthy {
		t.Error("Expected health to be healthy")
	}

	if len(health.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(health.Components))
	}

	if health.Components["config"].Status != "healthy" {
		t.Error("Expected config component to be healthy")
	}

	if health.Components["storage"].Status != "healthy" {
		t.Error("Expected storage component to be healthy")
	}

	if len(health.Checks) != 2 {
		t.Errorf("Expected 2 health checks, got %d", len(health.Checks))
	}

	// Verify health check conversion
	configCheck := health.Checks[0]
	if configCheck.Name != "config" {
		t.Errorf("Expected check name 'config', got %s", configCheck.Name)
	}

	if configCheck.Status != "healthy" {
		t.Errorf("Expected check status 'healthy', got %s", configCheck.Status)
	}

	if configCheck.Duration != time.Millisecond*10 {
		t.Errorf("Expected duration 10ms, got %v", configCheck.Duration)
	}
}

func TestRuntimeAPIAdapter_Health_WithError(t *testing.T) {
	mockRuntime := &mockRuntime{
		healthStatus: nil, // Simulate error case
	}

	adapter := newRuntimeAPIAdapter(mockRuntime)
	ctx := context.Background()

	health := adapter.Health(ctx)

	if health.Healthy {
		t.Error("Expected health to be unhealthy when error occurs")
	}

	if len(health.Components) != 0 {
		t.Errorf("Expected 0 components when error occurs, got %d", len(health.Components))
	}

	if len(health.Checks) != 0 {
		t.Errorf("Expected 0 checks when error occurs, got %d", len(health.Checks))
	}
}

func TestRuntimeAPIAdapter_GetStatus(t *testing.T) {
	now := time.Now()
	mockRuntime := &mockRuntime{
		runtimeStatus: &RuntimeStatus{
			State:     RuntimeStateRunning,
			StartedAt: now,
			Uptime:    time.Hour,
			Version:   "test-v1.0.0",
			Components: map[string]ComponentStatus{
				"config": {
					Name:      "config",
					State:     ComponentStateRunning,
					StartedAt: now,
					Uptime:    time.Hour,
					Health:    HealthStateHealthy,
				},
				"storage": {
					Name:      "storage",
					State:     ComponentStateRunning,
					StartedAt: now,
					Uptime:    time.Hour,
					Health:    HealthStateHealthy,
				},
			},
		},
	}

	adapter := newRuntimeAPIAdapter(mockRuntime)
	status := adapter.GetStatus()

	if status.State != "running" {
		t.Errorf("Expected state 'running', got %s", status.State)
	}

	if status.StartedAt != now {
		t.Error("Expected StartedAt to be preserved")
	}

	if status.Uptime != time.Hour {
		t.Errorf("Expected uptime 1h, got %v", status.Uptime)
	}

	if status.Version != "test-v1.0.0" {
		t.Errorf("Expected version 'test-v1.0.0', got %s", status.Version)
	}

	if len(status.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(status.Components))
	}

	configComp := status.Components["config"]
	if configComp.Name != "config" {
		t.Errorf("Expected component name 'config', got %s", configComp.Name)
	}

	if configComp.State != "running" {
		t.Errorf("Expected component state 'running', got %s", configComp.State)
	}

	if configComp.Health != "healthy" {
		t.Errorf("Expected component health 'healthy', got %s", configComp.Health)
	}
}

func TestRuntimeAPIAdapter_GetStatus_WithNil(t *testing.T) {
	mockRuntime := &mockRuntime{
		runtimeStatus: nil, // Simulate nil status
	}

	adapter := newRuntimeAPIAdapter(mockRuntime)
	status := adapter.GetStatus()

	if status.State != "unknown" {
		t.Errorf("Expected state 'unknown' when nil status, got %s", status.State)
	}

	if len(status.Components) != 0 {
		t.Errorf("Expected 0 components when nil status, got %d", len(status.Components))
	}
}
