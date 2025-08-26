package tekton

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// MockGitClient implements GitClient interface for testing
type MockGitClient struct {
	directoryExists map[string]bool
	filesList       map[string][]string
	fileContents    map[string][]byte
	shouldError     bool
	errorMessage    string
}

func NewMockGitClient() *MockGitClient {
	return &MockGitClient{
		directoryExists: make(map[string]bool),
		filesList:       make(map[string][]string),
		fileContents:    make(map[string][]byte),
		shouldError:     false,
	}
}

func (m *MockGitClient) CheckDirectoryExists(ctx context.Context, repo types.Repository, commitSHA, dirPath string) (bool, error) {
	if m.shouldError {
		return false, fmt.Errorf("%s", m.errorMessage)
	}
	key := fmt.Sprintf("%s:%s:%s", repo.Name, commitSHA, dirPath)
	return m.directoryExists[key], nil
}

func (m *MockGitClient) ListFiles(ctx context.Context, repo types.Repository, commitSHA, path string) ([]string, error) {
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	key := fmt.Sprintf("%s:%s:%s", repo.Name, commitSHA, path)
	return m.filesList[key], nil
}

func (m *MockGitClient) GetFileContent(ctx context.Context, repo types.Repository, commitSHA, filePath string) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	key := fmt.Sprintf("%s:%s:%s", repo.Name, commitSHA, filePath)
	content, exists := m.fileContents[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	return content, nil
}

// Implement other GitClient methods (not used in detector tests)
func (m *MockGitClient) GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitClient) GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *MockGitClient) CheckPermissions(ctx context.Context, repo types.Repository) error {
	return fmt.Errorf("not implemented")
}

func (m *MockGitClient) GetRateLimit(ctx context.Context) (*types.RateLimit, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitClient) GetProvider() string {
	return "mock"
}

func (m *MockGitClient) Close() error {
	return nil
}

// Helper methods for test setup
func (m *MockGitClient) SetDirectoryExists(repo, commitSHA, dirPath string, exists bool) {
	key := fmt.Sprintf("%s:%s:%s", repo, commitSHA, dirPath)
	m.directoryExists[key] = exists
}

func (m *MockGitClient) SetFilesList(repo, commitSHA, path string, files []string) {
	key := fmt.Sprintf("%s:%s:%s", repo, commitSHA, path)
	m.filesList[key] = files
}

func (m *MockGitClient) SetFileContent(repo, commitSHA, filePath string, content []byte) {
	key := fmt.Sprintf("%s:%s:%s", repo, commitSHA, filePath)
	m.fileContents[key] = content
}

func (m *MockGitClient) SetError(shouldError bool, message string) {
	m.shouldError = shouldError
	m.errorMessage = message
}

func createTestLogger() *logger.Entry {
	testLogger, _ := logger.NewLogger(logger.Config{
		Level:  "info",
		Format: "text",
	})
	return testLogger.WithField("test", "tekton-detector")
}

func TestNewTektonDetector(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()

	detector := NewTektonDetector(mockClient, testLogger)

	if detector == nil {
		t.Fatal("Expected detector to be created")
	}

	if detector.gitClient != mockClient {
		t.Error("Git client not set correctly")
	}

	config := detector.GetConfig()
	if config.ScanPath != ".tekton" {
		t.Errorf("Expected scan path '.tekton', got %s", config.ScanPath)
	}
}

func TestDetectTektonResources_NoTektonDirectory(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	commitSHA := "abc123"
	branch := "main"

	// Setup: .tekton directory does not exist
	mockClient.SetDirectoryExists("test-repo", commitSHA, ".tekton", false)

	ctx := context.Background()
	detection, err := detector.DetectTektonResources(ctx, repo, commitSHA, branch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if detection.HasTektonDirectory {
		t.Error("Expected HasTektonDirectory to be false")
	}

	if detection.EstimatedAction != "skip" {
		t.Errorf("Expected estimated action 'skip', got %s", detection.EstimatedAction)
	}

	if len(detection.TektonFiles) != 0 {
		t.Errorf("Expected 0 tekton files, got %d", len(detection.TektonFiles))
	}
}

func TestDetectTektonResources_WithValidTektonFiles(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	commitSHA := "abc123"
	branch := "main"

	// Setup: .tekton directory exists with files
	mockClient.SetDirectoryExists("test-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("test-repo", commitSHA, ".tekton", []string{
		".tekton/pipeline.yaml",
		".tekton/task.yml",
		".tekton/README.md", // Should be ignored
	})

	// Setup file contents
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
    image: ubuntu
    script: echo "Hello World"
`

	mockClient.SetFileContent("test-repo", commitSHA, ".tekton/pipeline.yaml", []byte(pipelineYAML))
	mockClient.SetFileContent("test-repo", commitSHA, ".tekton/task.yml", []byte(taskYAML))

	ctx := context.Background()
	detection, err := detector.DetectTektonResources(ctx, repo, commitSHA, branch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !detection.HasTektonDirectory {
		t.Error("Expected HasTektonDirectory to be true")
	}

	if detection.TotalFiles != 3 {
		t.Errorf("Expected total files 3, got %d", detection.TotalFiles)
	}

	if detection.ValidFiles != 2 {
		t.Errorf("Expected valid files 2, got %d", detection.ValidFiles)
	}

	if len(detection.TektonFiles) != 2 {
		t.Errorf("Expected 2 tekton files, got %d", len(detection.TektonFiles))
	}

	if len(detection.Resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(detection.Resources))
	}

	// Check resources
	foundPipeline := false
	foundTask := false
	for _, resource := range detection.Resources {
		if resource.Kind == "Pipeline" && resource.Name == "test-pipeline" {
			foundPipeline = true
		}
		if resource.Kind == "Task" && resource.Name == "hello-task" {
			foundTask = true
		}
	}

	if !foundPipeline {
		t.Error("Pipeline resource not found")
	}
	if !foundTask {
		t.Error("Task resource not found")
	}

	if detection.EstimatedAction != "apply" {
		t.Errorf("Expected estimated action 'apply', got %s", detection.EstimatedAction)
	}
}

func TestDetectTektonResources_WithPipelineRun(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	commitSHA := "abc123"
	branch := "main"

	// Setup: .tekton directory exists with PipelineRun
	mockClient.SetDirectoryExists("test-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("test-repo", commitSHA, ".tekton", []string{
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

	mockClient.SetFileContent("test-repo", commitSHA, ".tekton/pipelinerun.yaml", []byte(pipelineRunYAML))

	ctx := context.Background()
	detection, err := detector.DetectTektonResources(ctx, repo, commitSHA, branch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if detection.EstimatedAction != "trigger" {
		t.Errorf("Expected estimated action 'trigger', got %s", detection.EstimatedAction)
	}

	if len(detection.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(detection.Resources))
	}

	resource := detection.Resources[0]
	if resource.Kind != "PipelineRun" {
		t.Errorf("Expected kind 'PipelineRun', got %s", resource.Kind)
	}
}

func TestDetectTektonResources_InvalidYAML(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	commitSHA := "abc123"
	branch := "main"

	// Setup: .tekton directory exists with invalid YAML
	mockClient.SetDirectoryExists("test-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("test-repo", commitSHA, ".tekton", []string{
		".tekton/invalid.yaml",
	})

	invalidYAML := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  invalid: yaml: content: [
`

	mockClient.SetFileContent("test-repo", commitSHA, ".tekton/invalid.yaml", []byte(invalidYAML))

	ctx := context.Background()
	detection, err := detector.DetectTektonResources(ctx, repo, commitSHA, branch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(detection.TektonFiles) != 1 {
		t.Errorf("Expected 1 tekton file, got %d", len(detection.TektonFiles))
	}

	tektonFile := detection.TektonFiles[0]
	if tektonFile.IsValid {
		t.Error("Expected file to be invalid")
	}

	if tektonFile.ErrorMessage == "" {
		t.Error("Expected error message for invalid file")
	}
}

func TestDetectTektonResources_GitClientError(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	commitSHA := "abc123"
	branch := "main"

	// Setup: Git client returns error
	mockClient.SetError(true, "API rate limit exceeded")

	ctx := context.Background()
	detection, err := detector.DetectTektonResources(ctx, repo, commitSHA, branch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(detection.Errors) == 0 {
		t.Error("Expected errors to be recorded")
	}

	if !strings.Contains(detection.Errors[0], "API rate limit exceeded") {
		t.Errorf("Expected error message about rate limit, got: %s", detection.Errors[0])
	}
}

func TestIsTektonResource(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		expected   bool
	}{
		{"Valid Pipeline v1beta1", "tekton.dev/v1beta1", "Pipeline", true},
		{"Valid Task v1", "tekton.dev/v1", "Task", true},
		{"Valid EventListener", "triggers.tekton.dev/v1beta1", "EventListener", true},
		{"Invalid apiVersion", "apps/v1", "Deployment", false},
		{"Invalid kind", "tekton.dev/v1beta1", "Deployment", false},
		{"Empty apiVersion", "", "Pipeline", false},
		{"Empty kind", "tekton.dev/v1beta1", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isTektonResource(tt.apiVersion, tt.kind)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateResource(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	tests := []struct {
		name        string
		resource    TektonResource
		expectError bool
	}{
		{
			name: "Valid Pipeline",
			resource: TektonResource{
				Kind: "Pipeline",
				Name: "test-pipeline",
				Spec: map[string]interface{}{
					"tasks": []interface{}{
						map[string]interface{}{"name": "hello"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid name format",
			resource: TektonResource{
				Kind: "Pipeline",
				Name: "Test_Pipeline", // Invalid characters
				Spec: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "Empty name",
			resource: TektonResource{
				Kind: "Pipeline",
				Name: "",
				Spec: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "Pipeline without tasks",
			resource: TektonResource{
				Kind: "Pipeline",
				Name: "test-pipeline",
				Spec: map[string]interface{}{
					"tasks": []interface{}{}, // Empty tasks
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.ValidateResource(&tt.resource)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDetectorConfig(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)

	// Test default config
	config := detector.GetConfig()
	if config.ScanPath != ".tekton" {
		t.Errorf("Expected default scan path '.tekton', got %s", config.ScanPath)
	}

	// Test setting custom config
	customConfig := DetectorConfig{
		ScanPath:       ".ci",
		FileExtensions: []string{".yaml"},
		MaxFileSize:    500 * 1024, // 500KB
		Timeout:        10 * time.Second,
	}

	detector.SetConfig(customConfig)
	newConfig := detector.GetConfig()

	if newConfig.ScanPath != ".ci" {
		t.Errorf("Expected scan path '.ci', got %s", newConfig.ScanPath)
	}

	if newConfig.MaxFileSize != 500*1024 {
		t.Errorf("Expected max file size 500KB, got %d", newConfig.MaxFileSize)
	}
}
