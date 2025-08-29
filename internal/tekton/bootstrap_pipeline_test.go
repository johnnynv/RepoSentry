package tekton

import (
	"strings"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestBootstrapPipelineGenerator_GeneratePipelineRun(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(parentLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "test-repo",
			URL:  "https://github.com/test/repo.git",
		},
		CommitSHA: "abc123",
		Branch:    "main",
		Detection: &TektonDetection{
			EstimatedAction: "apply",
			ScanPath:        ".tekton",
		},
		Namespace:      "reposentry-user-repo-test",
		WorkspaceSize:  "1Gi",
		ServiceAccount: "reposentry-bootstrap-sa",
	}

	resources, err := generator.GeneratePipelineRun(config)
	if err != nil {
		t.Fatalf("Expected no error generating PipelineRun, got: %v", err)
	}

	if resources == nil {
		t.Fatal("Expected resources to be generated")
	}

	if resources.PipelineRun == "" {
		t.Fatal("Expected PipelineRun to be generated")
	}

	// Verify PipelineRun contains expected content
	expectedContent := []string{
		"generateName: reposentry-bootstrap-run-",
		"namespace: reposentry-system",
		"pipelineRef:",
		"name: reposentry-bootstrap-pipeline",
		"repo-url",
		"https://github.com/test/repo.git",
		"repo-branch",
		"main",
		"commit-sha",
		"abc123",
		"target-namespace",
		"reposentry-user-repo-test",
		"tekton-path",
		".tekton",
		"source-workspace",
		"tekton-workspace",
		"serviceAccountName: reposentry-bootstrap-sa",
	}

	for _, content := range expectedContent {
		if !strings.Contains(resources.PipelineRun, content) {
			t.Errorf("Expected PipelineRun to contain: %s", content)
		}
	}

	// Verify metadata
	if resources.TargetNamespace != "reposentry-user-repo-test" {
		t.Errorf("Expected TargetNamespace to be 'reposentry-user-repo-test', got: %s", resources.TargetNamespace)
	}

	if resources.GeneratedAt == "" {
		t.Error("Expected GeneratedAt to be set")
	}
}

func TestBootstrapPipelineGenerator_SetDefaults(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(parentLogger)

	config := &BootstrapPipelineConfig{}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Expected no error setting defaults, got: %v", err)
	}

	// Verify defaults are set
	if config.CloneImage == "" {
		t.Error("Expected CloneImage to be set")
	}

	if config.KubectlImage == "" {
		t.Error("Expected KubectlImage to be set")
	}

	if config.TektonImage == "" {
		t.Error("Expected TektonImage to be set")
	}

	if config.WorkspaceSize == "" {
		t.Error("Expected WorkspaceSize to be set")
	}

	if config.ServiceAccount == "" {
		t.Error("Expected ServiceAccount to be set")
	}

	if config.ResourceLimits == nil {
		t.Error("Expected ResourceLimits to be set")
	}

	if config.SecurityContext == nil {
		t.Error("Expected SecurityContext to be set")
	}
}

func TestBootstrapPipelineGenerator_GeneratePipelineRun_WithCustomConfig(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(parentLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "custom-repo",
			URL:  "https://gitlab.com/custom/repo.git",
		},
		CommitSHA: "def456",
		Branch:    "develop",
		Detection: &TektonDetection{
			EstimatedAction: "trigger",
			ScanPath:        ".tekton/pipelines",
		},
		Namespace:      "custom-namespace",
		WorkspaceSize:  "2Gi",
		ServiceAccount: "custom-sa",
		CloneImage:     "custom/clone:latest",
		KubectlImage:   "custom/kubectl:latest",
		TektonImage:    "custom/tekton:latest",
	}

	resources, err := generator.GeneratePipelineRun(config)
	if err != nil {
		t.Fatalf("Expected no error generating PipelineRun with custom config, got: %v", err)
	}

	if resources == nil {
		t.Fatal("Expected resources to be generated")
	}

	// Verify custom values are used
	if !strings.Contains(resources.PipelineRun, "https://gitlab.com/custom/repo.git") {
		t.Error("Expected custom repository URL to be used")
	}

	if !strings.Contains(resources.PipelineRun, "develop") {
		t.Error("Expected custom branch to be used")
	}

	if !strings.Contains(resources.PipelineRun, "def456") {
		t.Error("Expected custom commit SHA to be used")
	}

	if !strings.Contains(resources.PipelineRun, "custom-namespace") {
		t.Error("Expected custom namespace to be used")
	}

	if !strings.Contains(resources.PipelineRun, ".tekton/pipelines") {
		t.Error("Expected custom tekton path to be used")
	}

	if !strings.Contains(resources.PipelineRun, "2Gi") {
		t.Error("Expected custom workspace size to be used")
	}

	if !strings.Contains(resources.PipelineRun, "custom-sa") {
		t.Error("Expected custom service account to be used")
	}
}
