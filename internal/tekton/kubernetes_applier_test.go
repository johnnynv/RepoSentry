package tekton

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewKubernetesApplier(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	if applier == nil {
		t.Fatal("Expected applier to be created")
	}

	if applier.config == nil {
		t.Error("Config not initialized")
	}

	if applier.logger == nil {
		t.Error("Logger not initialized")
	}

	// Verify default config
	if applier.config.DryRun != false {
		t.Error("Expected DryRun to be false by default")
	}

	if applier.config.Timeout != 10*time.Minute {
		t.Errorf("Expected timeout 10m, got %v", applier.config.Timeout)
	}

	if applier.config.RetryAttempts != 3 {
		t.Errorf("Expected retry attempts 3, got %d", applier.config.RetryAttempts)
	}

	if applier.config.ValidateResources != true {
		t.Error("Expected ValidateResources to be true by default")
	}
}

func TestApplyBootstrapResources_Success(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Create test Bootstrap Pipeline resources
	resources := &BootstrapPipelineResources{
		Namespace:         "test-namespace",
		ServiceAccount:    "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: test-sa",
		RoleBinding:       "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: test-rb",
		ResourceQuota:     "apiVersion: v1\nkind: ResourceQuota\nmetadata:\n  name: test-quota",
		NetworkPolicy:     "apiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: test-np",
		BootstrapTasks:    []string{"apiVersion: tekton.dev/v1beta1\nkind: Task\nmetadata:\n  name: test-task"},
		BootstrapPipeline: "apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: test-pipeline",
		PipelineRun:       "apiVersion: tekton.dev/v1beta1\nkind: PipelineRun\nmetadata:\n  name: test-pr",
	}

	ctx := context.Background()
	err := applier.ApplyBootstrapResources(ctx, resources)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestApplyBootstrapResources_MissingRequired(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Create resources with missing required field
	resources := &BootstrapPipelineResources{
		Namespace:      "test-namespace",
		ServiceAccount: "", // Missing required resource
		RoleBinding:    "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: test-rb",
	}

	ctx := context.Background()
	err := applier.ApplyBootstrapResources(ctx, resources)

	if err == nil {
		t.Error("Expected error for missing required resource")
	}

	if !strings.Contains(err.Error(), "ServiceAccount is empty") {
		t.Errorf("Expected error about ServiceAccount, got: %v", err)
	}
}

func TestApplyBootstrapResources_InvalidYAML(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Create resources with invalid YAML
	resources := &BootstrapPipelineResources{
		Namespace:         "test-namespace",
		ServiceAccount:    "invalid yaml without apiVersion",
		RoleBinding:       "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: test-rb",
		ResourceQuota:     "apiVersion: v1\nkind: ResourceQuota\nmetadata:\n  name: test-quota",
		BootstrapTasks:    []string{"apiVersion: tekton.dev/v1beta1\nkind: Task\nmetadata:\n  name: test-task"},
		BootstrapPipeline: "apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: test-pipeline",
	}

	ctx := context.Background()
	err := applier.ApplyBootstrapResources(ctx, resources)

	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("Expected error about missing YAML field, got: %v", err)
	}
}

func TestApplyBootstrapResources_DryRun(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Enable dry run
	applier.config.DryRun = true

	resources := &BootstrapPipelineResources{
		Namespace:         "test-namespace",
		ServiceAccount:    "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: test-sa",
		RoleBinding:       "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: test-rb",
		ResourceQuota:     "apiVersion: v1\nkind: ResourceQuota\nmetadata:\n  name: test-quota",
		BootstrapTasks:    []string{"apiVersion: tekton.dev/v1beta1\nkind: Task\nmetadata:\n  name: test-task"},
		BootstrapPipeline: "apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: test-pipeline",
	}

	ctx := context.Background()
	err := applier.ApplyBootstrapResources(ctx, resources)

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Reset dry run
	applier.config.DryRun = false
}

func TestGetNamespaceStatus(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	status, err := applier.GetNamespaceStatus(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status == nil {
		t.Fatal("Expected status to be returned")
	}

	// Verify status structure
	if status.Phase == "" {
		t.Error("Phase not set")
	}

	if status.Resources == nil {
		t.Error("Resources map not initialized")
	}

	// Check if namespace exists determines the content
	if status.Exists {
		if len(status.PipelineRuns) == 0 {
			t.Error("Expected simulated PipelineRuns when namespace exists")
		}

		if len(status.TaskRuns) == 0 {
			t.Error("Expected simulated TaskRuns when namespace exists")
		}

		if status.ResourceUsage == nil {
			t.Error("ResourceUsage not set when namespace exists")
		}

		if status.LastActivity == nil {
			t.Error("LastActivity not set when namespace exists")
		}
	} else {
		// Namespace doesn't exist, should have minimal data
		if status.Phase != "NotFound" {
			t.Error("Expected Phase to be NotFound when namespace doesn't exist")
		}
	}
}

func TestCleanupNamespace(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	err := applier.CleanupNamespace(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCleanupNamespace_DryRun(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Enable dry run
	applier.config.DryRun = true

	ctx := context.Background()
	err := applier.CleanupNamespace(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Reset dry run
	applier.config.DryRun = false
}

func TestSetAndGetConfig(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Test getting default config
	config := applier.GetConfig()
	if config == nil {
		t.Fatal("Expected config to be returned")
	}

	// Test setting new config
	newConfig := &KubernetesApplierConfig{
		DryRun:            true,
		Timeout:           5 * time.Minute,
		RetryAttempts:     5,
		RetryDelay:        10 * time.Second,
		ValidateResources: false,
		SkipTLSVerify:     true,
	}

	applier.SetConfig(newConfig)

	// Verify config was set
	updatedConfig := applier.GetConfig()
	if updatedConfig.DryRun != true {
		t.Error("DryRun not updated")
	}

	if updatedConfig.Timeout != 5*time.Minute {
		t.Error("Timeout not updated")
	}

	if updatedConfig.RetryAttempts != 5 {
		t.Error("RetryAttempts not updated")
	}

	if updatedConfig.ValidateResources != false {
		t.Error("ValidateResources not updated")
	}

	if updatedConfig.SkipTLSVerify != true {
		t.Error("SkipTLSVerify not updated")
	}
}

func TestGetDefaultApplierConfig(t *testing.T) {
	config := getDefaultApplierConfig()

	if config == nil {
		t.Fatal("Expected config to be returned")
	}

	// Verify default values
	if config.DryRun != false {
		t.Error("Expected DryRun to be false")
	}

	if config.Timeout != 10*time.Minute {
		t.Error("Expected timeout to be 10 minutes")
	}

	if config.RetryAttempts != 3 {
		t.Error("Expected retry attempts to be 3")
	}

	if config.RetryDelay != 5*time.Second {
		t.Error("Expected retry delay to be 5 seconds")
	}

	if config.ValidateResources != true {
		t.Error("Expected ValidateResources to be true")
	}

	if config.SkipTLSVerify != false {
		t.Error("Expected SkipTLSVerify to be false")
	}
}

func TestGetLastActivity(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	// Test with empty runs
	lastActivity := applier.getLastActivity([]PipelineRunStatus{}, []TaskRunStatus{})
	if lastActivity != nil {
		t.Error("Expected nil for empty runs")
	}

	// Test with PipelineRuns
	now := time.Now()
	pipelineRuns := []PipelineRunStatus{
		{
			Name:           "pr1",
			StartTime:      &now,
			CompletionTime: nil,
		},
		{
			Name:           "pr2",
			StartTime:      &now,
			CompletionTime: &now,
		},
	}

	lastActivity = applier.getLastActivity(pipelineRuns, []TaskRunStatus{})
	if lastActivity == nil {
		t.Error("Expected last activity to be set")
	}

	// Test with TaskRuns
	future := now.Add(1 * time.Hour)
	taskRuns := []TaskRunStatus{
		{
			Name:           "tr1",
			StartTime:      &future, // Later than pipeline runs
			CompletionTime: &future,
		},
	}

	lastActivity = applier.getLastActivity(pipelineRuns, taskRuns)
	if lastActivity == nil {
		t.Error("Expected last activity to be set")
	}

	if !lastActivity.Equal(future) {
		t.Error("Expected last activity to be from TaskRun")
	}
}

func TestApplyYAMLContent_InvalidYAML(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Missing apiVersion",
			yamlContent: "kind: ServiceAccount\nmetadata:\n  name: test",
			expectError: true,
			errorMsg:    "missing apiVersion",
		},
		{
			name:        "Missing kind",
			yamlContent: "apiVersion: v1\nmetadata:\n  name: test",
			expectError: true,
			errorMsg:    "missing kind",
		},
		{
			name:        "Valid YAML",
			yamlContent: "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: test",
			expectError: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applier.applyYAMLContent(ctx, tt.yamlContent, "test-namespace")

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	err := applier.createNamespace(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCheckNamespaceExists(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	exists, err := applier.checkNamespaceExists(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// In our simulation, namespace doesn't exist
	if exists {
		t.Error("Expected namespace to not exist")
	}
}

func TestGetPipelineRuns(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	pipelineRuns, err := applier.getPipelineRuns(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(pipelineRuns) == 0 {
		t.Error("Expected simulated PipelineRuns")
	}

	// Verify PipelineRun structure
	pr := pipelineRuns[0]
	if pr.Name == "" {
		t.Error("PipelineRun name not set")
	}

	if pr.Status == "" {
		t.Error("PipelineRun status not set")
	}

	if pr.StartTime == nil {
		t.Error("PipelineRun start time not set")
	}
}

func TestGetTaskRuns(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	taskRuns, err := applier.getTaskRuns(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(taskRuns) == 0 {
		t.Error("Expected simulated TaskRuns")
	}

	// Verify TaskRun structure
	tr := taskRuns[0]
	if tr.Name == "" {
		t.Error("TaskRun name not set")
	}

	if tr.Status == "" {
		t.Error("TaskRun status not set")
	}

	if tr.PipelineRun == "" {
		t.Error("TaskRun PipelineRun not set")
	}
}

func TestGetResourceUsage(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	ctx := context.Background()
	usage, err := applier.getResourceUsage(ctx, "test-namespace")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if usage == nil {
		t.Fatal("Expected resource usage to be returned")
	}

	// Verify usage structure
	if usage.CPU == "" {
		t.Error("CPU usage not set")
	}

	if usage.Memory == "" {
		t.Error("Memory usage not set")
	}

	if usage.Pods == 0 {
		t.Error("Pods count not set")
	}

	if usage.PVCs == 0 {
		t.Error("PVCs count not set")
	}
}
