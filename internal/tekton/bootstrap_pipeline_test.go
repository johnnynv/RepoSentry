package tekton

import (
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestNewBootstrapPipelineGenerator(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	if generator == nil {
		t.Fatal("Expected generator to be created")
	}

	if generator.logger == nil {
		t.Error("Logger not set correctly")
	}
}

func TestGenerateBootstrapResources_Apply(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	// Create test configuration
	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-org/test-repo",
			Provider: "github",
		},
		CommitSHA: "abc123def456",
		Branch:    "main",
		Detection: &TektonDetection{
			HasTektonDirectory: true,
			ScanPath:          ".tekton",
			EstimatedAction:   "apply",
			Resources: []TektonResource{
				{Kind: "Pipeline", Name: "test-pipeline"},
				{Kind: "Task", Name: "test-task"},
			},
			DetectedAt: time.Now(),
		},
		Namespace: "reposentry-user-repo-12345678",
	}

	// Generate resources
	resources, err := generator.GenerateBootstrapResources(config)
	if err != nil {
		t.Fatalf("Failed to generate resources: %v", err)
	}

	// Verify basic structure
	if resources == nil {
		t.Fatal("Expected resources to be generated")
	}

	if resources.Namespace != config.Namespace {
		t.Errorf("Expected namespace %s, got %s", config.Namespace, resources.Namespace)
	}

	if resources.Config != config {
		t.Error("Config not preserved in resources")
	}

	// Verify Bootstrap Pipeline
	if resources.BootstrapPipeline == "" {
		t.Error("Bootstrap Pipeline not generated")
	}

	if !strings.Contains(resources.BootstrapPipeline, "reposentry-bootstrap-apply") {
		t.Error("Bootstrap Pipeline should contain apply pipeline name")
	}

	if !strings.Contains(resources.BootstrapPipeline, config.Namespace) {
		t.Error("Bootstrap Pipeline should contain target namespace")
	}

	// Verify tasks
	if len(resources.BootstrapTasks) == 0 {
		t.Error("Bootstrap tasks not generated")
	}

	expectedTasks := []string{"clone", "validate", "apply-resources"}
	for _, expectedTask := range expectedTasks {
		found := false
		for _, task := range resources.BootstrapTasks {
			if strings.Contains(task, expectedTask) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected task containing '%s' not found", expectedTask)
		}
	}

	// Verify supporting resources
	if resources.ServiceAccount == "" {
		t.Error("ServiceAccount not generated")
	}

	if resources.RoleBinding == "" {
		t.Error("RoleBinding not generated")
	}

	if resources.ResourceQuota == "" {
		t.Error("ResourceQuota not generated")
	}

	if resources.NetworkPolicy == "" {
		t.Error("NetworkPolicy not generated")
	}

	// Verify PipelineRun
	if resources.PipelineRun == "" {
		t.Error("PipelineRun not generated")
	}

	if !strings.Contains(resources.PipelineRun, "reposentry-bootstrap-apply-") {
		t.Error("PipelineRun should reference apply pipeline")
	}
}

func TestGenerateBootstrapResources_Trigger(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-org/test-repo",
			Provider: "github",
		},
		CommitSHA: "abc123def456",
		Branch:    "main",
		Detection: &TektonDetection{
			HasTektonDirectory: true,
			ScanPath:          ".tekton",
			EstimatedAction:   "trigger",
			Resources: []TektonResource{
				{Kind: "PipelineRun", Name: "test-pipeline-run"},
			},
			DetectedAt: time.Now(),
		},
		Namespace: "reposentry-user-repo-12345678",
	}

	resources, err := generator.GenerateBootstrapResources(config)
	if err != nil {
		t.Fatalf("Failed to generate resources: %v", err)
	}

	// Verify trigger-specific pipeline
	if !strings.Contains(resources.BootstrapPipeline, "reposentry-bootstrap-trigger") {
		t.Error("Bootstrap Pipeline should contain trigger pipeline name")
	}

	// Verify trigger tasks
	foundTriggerTask := false
	for _, task := range resources.BootstrapTasks {
		if strings.Contains(task, "trigger-runs") {
			foundTriggerTask = true
			break
		}
	}
	if !foundTriggerTask {
		t.Error("Expected trigger-runs task not found")
	}

	// Verify PipelineRun
	if !strings.Contains(resources.PipelineRun, "reposentry-bootstrap-trigger-") {
		t.Error("PipelineRun should reference trigger pipeline")
	}
}

func TestGenerateBootstrapResources_Validate(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-org/test-repo",
			Provider: "github",
		},
		CommitSHA: "abc123def456",
		Branch:    "main",
		Detection: &TektonDetection{
			HasTektonDirectory: true,
			ScanPath:          ".tekton",
			EstimatedAction:   "validate",
			Resources: []TektonResource{
				{Kind: "Pipeline", Name: "test-pipeline"},
			},
			DetectedAt: time.Now(),
		},
		Namespace: "reposentry-user-repo-12345678",
	}

	resources, err := generator.GenerateBootstrapResources(config)
	if err != nil {
		t.Fatalf("Failed to generate resources: %v", err)
	}

	// Verify validate-specific pipeline
	if !strings.Contains(resources.BootstrapPipeline, "reposentry-bootstrap-validate") {
		t.Error("Bootstrap Pipeline should contain validate pipeline name")
	}

	// Should not have PipelineRun for validate-only mode
	if resources.PipelineRun != "" {
		t.Error("PipelineRun should not be generated for validate mode")
	}
}

func TestGenerateBootstrapResources_Skip(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-org/test-repo",
			Provider: "github",
		},
		CommitSHA: "abc123def456",
		Branch:    "main",
		Detection: &TektonDetection{
			HasTektonDirectory: false,
			ScanPath:          ".tekton",
			EstimatedAction:   "skip",
			Resources:         []TektonResource{},
			DetectedAt:        time.Now(),
		},
		Namespace: "reposentry-user-repo-12345678",
	}

	resources, err := generator.GenerateBootstrapResources(config)
	if err != nil {
		t.Fatalf("Failed to generate resources: %v", err)
	}

	// Verify skip-specific pipeline
	if !strings.Contains(resources.BootstrapPipeline, "reposentry-bootstrap-skip") {
		t.Error("Bootstrap Pipeline should contain skip pipeline name")
	}

	// Should only have notify task
	if len(resources.BootstrapTasks) != 1 {
		t.Errorf("Expected 1 task for skip mode, got %d", len(resources.BootstrapTasks))
	}

	if !strings.Contains(resources.BootstrapTasks[0], "notify") {
		t.Error("Skip mode should only have notify task")
	}
}

func TestGenerateBootstrapResources_UnsupportedAction(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-org/test-repo",
			Provider: "github",
		},
		CommitSHA: "abc123def456",
		Branch:    "main",
		Detection: &TektonDetection{
			EstimatedAction: "unsupported-action",
			DetectedAt:      time.Now(),
		},
		Namespace: "reposentry-user-repo-12345678",
	}

	_, err := generator.GenerateBootstrapResources(config)
	if err == nil {
		t.Error("Expected error for unsupported action")
	}

	if !strings.Contains(err.Error(), "unsupported estimated action") {
		t.Errorf("Expected error about unsupported action, got: %v", err)
	}
}

func TestSetDefaults(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "test-repo",
			URL:  "https://github.com/test-org/test-repo",
		},
		Detection: &TektonDetection{
			EstimatedAction: "apply",
		},
		Namespace: "test-namespace",
	}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	// Verify defaults
	if config.CloneImage == "" {
		t.Error("CloneImage not set")
	}

	if config.KubectlImage == "" {
		t.Error("KubectlImage not set")
	}

	if config.WorkspaceSize != "1Gi" {
		t.Errorf("Expected WorkspaceSize '1Gi', got %s", config.WorkspaceSize)
	}

	if config.ServiceAccount != "reposentry-bootstrap-sa" {
		t.Errorf("Expected ServiceAccount 'reposentry-bootstrap-sa', got %s", config.ServiceAccount)
	}

	if config.ResourceLimits == nil {
		t.Error("ResourceLimits not set")
	}

	if config.SecurityContext == nil {
		t.Error("SecurityContext not set")
	}
}

func TestGetGeneratedNamespace(t *testing.T) {
	tests := []struct {
		name     string
		repo     types.Repository
		expected string
	}{
		{
			name: "GitHub repository",
			repo: types.Repository{
				Name: "test-repo",
				URL:  "https://github.com/test-org/test-repo",
			},
			expected: "reposentry-user-repo-", // Should start with this prefix
		},
		{
			name: "GitLab repository",
			repo: types.Repository{
				Name: "test-project",
				URL:  "https://gitlab.com/test-group/test-project",
			},
			expected: "reposentry-user-repo-", // Should start with this prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGeneratedNamespace(tt.repo)

			if !strings.HasPrefix(result, tt.expected) {
				t.Errorf("Expected namespace to start with %s, got %s", tt.expected, result)
			}

			if len(result) != len("reposentry-user-repo-") + 16 { // 16 hex characters
				t.Errorf("Expected namespace length %d, got %d", len("reposentry-user-repo-") + 16, len(result))
			}

			// Should be valid Kubernetes namespace name
			if !isValidNamespaceName(result) {
				t.Errorf("Generated namespace name is invalid: %s", result)
			}
		})
	}
}

func TestGenerateRunID(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config1 := &BootstrapPipelineConfig{
		Repository: types.Repository{Name: "repo1"},
		CommitSHA:  "abc123",
		Branch:     "main",
		Detection:  &TektonDetection{DetectedAt: time.Unix(1234567890, 0)},
	}

	config2 := &BootstrapPipelineConfig{
		Repository: types.Repository{Name: "repo1"},
		CommitSHA:  "abc123",
		Branch:     "main",
		Detection:  &TektonDetection{DetectedAt: time.Unix(1234567890, 0)},
	}

	config3 := &BootstrapPipelineConfig{
		Repository: types.Repository{Name: "repo2"}, // Different repo
		CommitSHA:  "abc123",
		Branch:     "main",
		Detection:  &TektonDetection{DetectedAt: time.Unix(1234567890, 0)},
	}

	id1 := generator.generateRunID(config1)
	id2 := generator.generateRunID(config2)
	id3 := generator.generateRunID(config3)

	// Same config should generate same ID
	if id1 != id2 {
		t.Errorf("Expected same IDs for identical configs, got %s and %s", id1, id2)
	}

	// Different config should generate different ID
	if id1 == id3 {
		t.Errorf("Expected different IDs for different configs, got %s for both", id1)
	}

	// Check ID format (should be 12 hex characters)
	if len(id1) != 12 {
		t.Errorf("Expected ID length 12, got %d", len(id1))
	}
}

// Helper function to validate Kubernetes namespace names
func isValidNamespaceName(name string) bool {
	// Basic validation - should be lowercase, alphanumeric with hyphens
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	for i, r := range name {
		if i == 0 || i == len(name)-1 {
			// First and last character must be alphanumeric
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return false
			}
		} else {
			// Middle characters can be alphanumeric or hyphen
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}

	return true
}
