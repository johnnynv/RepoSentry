package tekton

import (
	"context"
	"strings"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestNewTektonIntegrationManager(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()

	manager := NewTektonIntegrationManager(mockClient, testLogger)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.detector == nil {
		t.Error("Detector not initialized")
	}

	if manager.eventGenerator == nil {
		t.Error("Event generator not initialized")
	}

	if manager.pipelineGenerator == nil {
		t.Error("Pipeline generator not initialized")
	}

	if manager.applier == nil {
		t.Error("Applier not initialized")
	}
}

func TestProcessRepositoryChange_Apply(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	// Setup mock data
	repo := types.Repository{
		Name:     "integration-test-repo",
		URL:      "https://github.com/test-org/integration-repo",
		Provider: "github",
	}
	commitSHA := "integration-commit-123"
	branch := "main"

	// Setup .tekton directory with Pipeline
	mockClient.SetDirectoryExists("integration-test-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("integration-test-repo", commitSHA, ".tekton", []string{
		".tekton/pipeline.yaml",
		".tekton/task.yaml",
	})

	pipelineYAML := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: hello
    taskRef:
      name: hello-task
`

	taskYAML := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: hello-task
spec:
  steps:
  - name: hello
    image: alpine
    script: echo "Hello World"
`

	mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/pipeline.yaml", []byte(pipelineYAML))
	mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/task.yaml", []byte(taskYAML))

	// Create request
	request := &TektonIntegrationRequest{
		Repository: repo,
		CommitSHA:  commitSHA,
		Branch:     branch,
		Event:      types.Event{Type: types.EventTypeBranchUpdated},
	}

	// Process repository change
	ctx := context.Background()
	result, err := manager.ProcessRepositoryChange(ctx, request)

	// Verify results
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.ExecutionStatus != "applied" {
		t.Errorf("Expected execution status 'applied', got %s", result.ExecutionStatus)
	}

	if result.Detection == nil {
		t.Error("Detection result not set")
	}

	if result.Detection.EstimatedAction != "apply" {
		t.Errorf("Expected estimated action 'apply', got %s", result.Detection.EstimatedAction)
	}

	if result.DetectionEvent == nil {
		t.Error("Detection event not generated")
	}

	if result.StandardEvent == nil {
		t.Error("Standard event not generated")
	}

	if result.BootstrapResources == nil {
		t.Error("Bootstrap resources not generated")
	}

	if result.Namespace == "" {
		t.Error("Namespace not set")
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected errors: %v", result.Errors)
	}

	if result.Duration == 0 {
		t.Error("Duration not set")
	}

	// Verify namespace format
	expectedPrefix := "reposentry-user-repo-"
	if !strings.HasPrefix(result.Namespace, expectedPrefix) {
		t.Errorf("Expected namespace to start with %s, got %s", expectedPrefix, result.Namespace)
	}
}

func TestProcessRepositoryChange_Skip(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	repo := types.Repository{
		Name:     "empty-repo",
		URL:      "https://github.com/test-org/empty-repo",
		Provider: "github",
	}
	commitSHA := "empty-commit-123"
	branch := "main"

	// Setup repository without .tekton directory
	mockClient.SetDirectoryExists("empty-repo", commitSHA, ".tekton", false)

	request := &TektonIntegrationRequest{
		Repository: repo,
		CommitSHA:  commitSHA,
		Branch:     branch,
		Event:      types.Event{Type: types.EventTypeBranchUpdated},
	}

	ctx := context.Background()
	result, err := manager.ProcessRepositoryChange(ctx, request)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExecutionStatus != "skipped" {
		t.Errorf("Expected execution status 'skipped', got %s", result.ExecutionStatus)
	}

	if result.Detection.EstimatedAction != "skip" {
		t.Errorf("Expected estimated action 'skip', got %s", result.Detection.EstimatedAction)
	}

	if result.BootstrapResources != nil {
		t.Error("Bootstrap resources should not be generated for skip action")
	}
}

func TestProcessRepositoryChange_Trigger(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	repo := types.Repository{
		Name:     "trigger-repo",
		URL:      "https://github.com/test-org/trigger-repo",
		Provider: "github",
	}
	commitSHA := "trigger-commit-123"
	branch := "main"

	// Setup .tekton directory with PipelineRun
	mockClient.SetDirectoryExists("trigger-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("trigger-repo", commitSHA, ".tekton", []string{
		".tekton/pipelinerun.yaml",
	})

	pipelineRunYAML := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: test-pipeline-run
spec:
  pipelineRef:
    name: test-pipeline
`

	mockClient.SetFileContent("trigger-repo", commitSHA, ".tekton/pipelinerun.yaml", []byte(pipelineRunYAML))

	request := &TektonIntegrationRequest{
		Repository: repo,
		CommitSHA:  commitSHA,
		Branch:     branch,
		Event:      types.Event{Type: types.EventTypeBranchUpdated},
	}

	ctx := context.Background()
	result, err := manager.ProcessRepositoryChange(ctx, request)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExecutionStatus != "applied" {
		t.Errorf("Expected execution status 'applied', got %s", result.ExecutionStatus)
	}

	if result.Detection.EstimatedAction != "trigger" {
		t.Errorf("Expected estimated action 'trigger', got %s", result.Detection.EstimatedAction)
	}

	// Verify trigger-specific bootstrap pipeline
	if result.BootstrapResources == nil {
		t.Fatal("Bootstrap resources not generated")
	}

	if !strings.Contains(result.BootstrapResources.BootstrapPipeline, "reposentry-bootstrap-trigger") {
		t.Error("Bootstrap pipeline should be trigger mode")
	}
}

func TestValidateIntegrationRequest(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	tests := []struct {
		name      string
		request   *TektonIntegrationRequest
		expectErr bool
	}{
		{
			name:      "Nil request",
			request:   nil,
			expectErr: true,
		},
		{
			name: "Empty repository name",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "", URL: "https://github.com/test/repo"},
			},
			expectErr: true,
		},
		{
			name: "Empty repository URL",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "test-repo", URL: ""},
			},
			expectErr: true,
		},
		{
			name: "Empty commit SHA",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "test-repo", URL: "https://github.com/test/repo"},
				CommitSHA:  "",
			},
			expectErr: true,
		},
		{
			name: "Empty branch",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "test-repo", URL: "https://github.com/test/repo"},
				CommitSHA:  "abc123",
				Branch:     "",
			},
			expectErr: true,
		},
		{
			name: "Invalid repository URL",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "test-repo", URL: "not-a-valid-url"},
				CommitSHA:  "abc123",
				Branch:     "main",
			},
			expectErr: true,
		},
		{
			name: "Valid request",
			request: &TektonIntegrationRequest{
				Repository: types.Repository{Name: "test-repo", URL: "https://github.com/test/repo"},
				CommitSHA:  "abc123",
				Branch:     "main",
				Event:      types.Event{Type: types.EventTypeBranchUpdated},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateIntegrationRequest(tt.request)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetIntegrationStatus(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	repo := types.Repository{
		Name:     "status-test-repo",
		URL:      "https://github.com/test-org/status-repo",
		Provider: "github",
	}
	commitSHA := "status-commit-123"

	ctx := context.Background()
	status, err := manager.GetIntegrationStatus(ctx, repo, commitSHA)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status == nil {
		t.Fatal("Expected status to be returned")
	}

	if status.Repository.Name != repo.Name {
		t.Errorf("Expected repository name %s, got %s", repo.Name, status.Repository.Name)
	}

	if status.CommitSHA != commitSHA {
		t.Errorf("Expected commit SHA %s, got %s", commitSHA, status.CommitSHA)
	}

	if status.Namespace == "" {
		t.Error("Namespace not set")
	}

	if status.NamespaceStatus == nil {
		t.Error("Namespace status not set")
	}

	if status.CheckedAt.IsZero() {
		t.Error("CheckedAt not set")
	}
}

func TestCleanupRepository(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	repo := types.Repository{
		Name:     "cleanup-test-repo",
		URL:      "https://github.com/test-org/cleanup-repo",
		Provider: "github",
	}

	ctx := context.Background()
	err := manager.CleanupRepository(ctx, repo)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestIsValidRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Valid GitHub HTTPS URL",
			url:      "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "Valid GitLab HTTPS URL",
			url:      "https://gitlab.com/user/repo",
			expected: true,
		},
		{
			name:     "Valid Private GitLab URL",
			url:      "https://gitlab-master.nvidia.com/group/project",
			expected: true,
		},
		{
			name:     "Valid GitHub HTTP URL",
			url:      "http://github.com/user/repo",
			expected: true,
		},
		{
			name:     "Invalid URL - no scheme",
			url:      "github.com/user/repo",
			expected: false,
		},
		{
			name:     "Invalid URL - unsupported host",
			url:      "https://bitbucket.org/user/repo",
			expected: false,
		},
		{
			name:     "Invalid URL - empty",
			url:      "",
			expected: false,
		},
		{
			name:     "Invalid URL - malformed",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidRepositoryURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestProcessRepositoryChange_ErrorHandling(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	repo := types.Repository{
		Name:     "error-test-repo",
		URL:      "https://github.com/test-org/error-repo",
		Provider: "github",
	}
	commitSHA := "error-commit-123"
	branch := "main"

	// Setup git client to return error when checking directory existence
	mockClient.SetError(true, "Network timeout")

	request := &TektonIntegrationRequest{
		Repository: repo,
		CommitSHA:  commitSHA,
		Branch:     branch,
		Event:      types.Event{Type: types.EventTypeBranchUpdated},
	}

	ctx := context.Background()
	result, err := manager.ProcessRepositoryChange(ctx, request)

	// Should return error
	if err == nil {
		t.Error("Expected error but got none")
	}

	if result == nil {
		t.Fatal("Expected result even with error")
	}

	if result.ExecutionStatus != "bootstrap_generation_failed" {
		t.Errorf("Expected execution status 'bootstrap_generation_failed', got %s", result.ExecutionStatus)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors to be recorded")
	}

	if !strings.Contains(result.Errors[0], "unsupported estimated action") {
		t.Errorf("Expected error about unsupported action, got: %s", result.Errors[0])
	}
}
