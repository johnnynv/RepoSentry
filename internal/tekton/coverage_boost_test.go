package tekton

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Additional tests to boost coverage to 80%

func TestGenerateBootstrapResourcesIntegration(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "integration-test-repo",
			URL:  "https://github.com/test-org/integration-test-repo",
		},
		Namespace: "integration-test-namespace",
		Detection: &TektonDetection{
			EstimatedAction: "apply",
		},
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	resources, err := generator.GenerateBootstrapResources(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify resources structure
	if resources.BootstrapPipeline == "" {
		t.Error("Bootstrap pipeline not generated")
	}

	if len(resources.BootstrapTasks) == 0 {
		t.Error("Bootstrap tasks not generated")
	}

	if resources.ServiceAccount == "" {
		t.Error("ServiceAccount not generated")
	}

	if resources.Namespace != config.Namespace {
		t.Error("Namespace mismatch")
	}
}

func TestGenerateApplyResources(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "apply-test-repo",
			URL:  "https://github.com/test-org/apply-test-repo",
		},
		Namespace: "apply-test-namespace",
		Detection: &TektonDetection{
			EstimatedAction: "apply",
		},
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	resources := &BootstrapPipelineResources{
		Namespace: config.Namespace,
	}

	result, err := generator.generateApplyResources(config, resources)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected result to be returned")
	}

	if result.BootstrapPipeline == "" {
		t.Error("Bootstrap pipeline not generated")
	}

	if len(result.BootstrapTasks) == 0 {
		t.Error("Bootstrap tasks not generated")
	}
}

func TestGenerateTriggerResources(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "trigger-test-repo",
			URL:  "https://github.com/test-org/trigger-test-repo",
		},
		Namespace: "trigger-test-namespace",
		Detection: &TektonDetection{
			EstimatedAction: "trigger",
		},
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	resources := &BootstrapPipelineResources{
		Namespace: config.Namespace,
	}

	result, err := generator.generateTriggerResources(config, resources)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.BootstrapPipeline == "" {
		t.Error("Bootstrap pipeline not generated")
	}

	if !strings.Contains(result.BootstrapPipeline, "reposentry-bootstrap-trigger") {
		t.Error("Trigger pipeline name not found")
	}
}

func TestGenerateValidateResources(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "validate-test-repo",
			URL:  "https://github.com/test-org/validate-test-repo",
		},
		Namespace: "validate-test-namespace",
		Detection: &TektonDetection{
			EstimatedAction: "validate",
		},
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	resources := &BootstrapPipelineResources{
		Namespace: config.Namespace,
	}

	result, err := generator.generateValidateResources(config, resources)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(result.BootstrapPipeline, "reposentry-bootstrap-validate") {
		t.Error("Validate pipeline name not found")
	}
}

func TestGenerateSkipResources(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "skip-test-repo",
			URL:  "https://github.com/test-org/skip-test-repo",
		},
		Namespace: "skip-test-namespace",
		Detection: &TektonDetection{
			EstimatedAction: "skip",
		},
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	resources := &BootstrapPipelineResources{
		Namespace: config.Namespace,
	}

	result, err := generator.generateSkipResources(config, resources)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Skip mode should have minimal resources
	if result.BootstrapPipeline == "" {
		t.Error("Skip mode should generate a notification pipeline")
	}

	if !strings.Contains(result.BootstrapPipeline, "reposentry-bootstrap-skip") {
		t.Error("Skip pipeline should have correct name")
	}

	// Skip mode should not generate PipelineRun
	if result.PipelineRun != "" {
		t.Error("Skip mode should not generate PipelineRun")
	}
}

func TestTektonIntegrationManager_GetMetrics(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be returned")
	}

	// Verify metrics structure (should be initialized with zeros)
	if metrics.TotalProcessed != 0 {
		t.Error("Expected TotalProcessed to be 0")
	}

	if metrics.SuccessfulRuns != 0 {
		t.Error("Expected SuccessfulRuns to be 0")
	}

	if metrics.FailedRuns != 0 {
		t.Error("Expected FailedRuns to be 0")
	}
}

func TestIsValidRepositoryURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub with port",
			url:      "https://github.com:443/user/repo",
			expected: true,
		},
		{
			name:     "GitLab with path",
			url:      "https://gitlab.com/group/subgroup/project",
			expected: true,
		},
		{
			name:     "Private GitLab with special naming",
			url:      "https://gitlab-dev.company.internal/team/project",
			expected: true,
		},
		{
			name:     "Git SSH URL (unsupported)",
			url:      "git@github.com:user/repo.git",
			expected: false,
		},
		{
			name:     "Local file path (unsupported)",
			url:      "/local/path/to/repo",
			expected: false,
		},
		{
			name:     "FTP URL (unsupported)",
			url:      "ftp://server.com/repo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidRepositoryURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for URL: %s", tt.expected, result, tt.url)
			}
		})
	}
}

func TestKubernetesApplier_EdgeCases(t *testing.T) {
	testLogger := createTestLogger()
	applier := NewKubernetesApplier(testLogger)

	t.Run("ApplyYAMLContent with empty content", func(t *testing.T) {
		ctx := context.Background()
		err := applier.applyYAMLContent(ctx, "", "test-namespace")

		if err == nil {
			t.Error("Expected error for empty YAML content")
		}
	})

	t.Run("GetLastActivity with mixed times", func(t *testing.T) {
		now := time.Now()
		earlier := now.Add(-1 * time.Hour)
		later := now.Add(1 * time.Hour)

		pipelineRuns := []PipelineRunStatus{
			{StartTime: &earlier, CompletionTime: nil},
			{StartTime: &now, CompletionTime: &later},
		}

		taskRuns := []TaskRunStatus{
			{StartTime: &now, CompletionTime: &earlier}, // Older completion
		}

		lastActivity := applier.getLastActivity(pipelineRuns, taskRuns)
		if lastActivity == nil {
			t.Fatal("Expected last activity to be set")
		}

		// Should be the later completion time
		if !lastActivity.Equal(later) {
			t.Error("Expected last activity to be the latest completion time")
		}
	})
}

func TestGenerateRunIDConsistency(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "runid-test-repo",
			URL:  "https://github.com/test-org/runid-test-repo",
		},
		CommitSHA: "abc123def456",
		Detection: &TektonDetection{
			DetectedAt: time.Now(),
		},
	}

	runID := generator.generateRunID(config)

	// Run ID should be non-empty
	if runID == "" {
		t.Error("Run ID should not be empty")
	}

	// Run ID should be consistent for same inputs
	runID2 := generator.generateRunID(config)
	if runID != runID2 {
		t.Error("Run ID should be consistent for same inputs")
	}

	// Run ID should be different for different inputs
	config2 := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "different-repo",
			URL:  "https://github.com/test-org/different-repo",
		},
		CommitSHA: "different-commit",
		Detection: &TektonDetection{
			DetectedAt: time.Now(),
		},
	}

	runID3 := generator.generateRunID(config2)
	if runID == runID3 {
		t.Error("Run ID should be different for different inputs")
	}
}

func TestTemplateExecution_Errors(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	// Test with invalid template data (missing required fields)
	config := &BootstrapPipelineConfig{
		Repository: types.Repository{}, // Empty repository
		Namespace:  "",                 // Empty namespace
	}

	// This should still work but generate default values
	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("setDefaults should not fail: %v", err)
	}

	// Test ServiceAccount generation with minimal config
	_, err = generator.generateServiceAccount(config)
	if err != nil {
		t.Fatalf("generateServiceAccount should not fail with minimal config: %v", err)
	}
}
