package tekton

import (
	"context"
	"strings"
	"testing"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// MockClientFactory implements GitClientFactory interface for testing
type MockClientFactory struct {
	gitClient gitclient.GitClient
}

func NewMockClientFactory(gitClient gitclient.GitClient) *MockClientFactory {
	return &MockClientFactory{
		gitClient: gitClient,
	}
}

func (mcf *MockClientFactory) CreateClient(repo types.Repository, config gitclient.ClientConfig) (gitclient.GitClient, error) {
	return mcf.gitClient, nil
}

// MockPoller simulates the Poller behavior for testing integration
type MockPoller struct {
	tektonManager *TektonTriggerManager
	repositories  []types.Repository
	logger        *MockLogger
}

// MockLogger for testing Poller integration
type MockLogger struct {
	messages []string
}

func (ml *MockLogger) WithField(key string, value interface{}) *MockLogger {
	return ml
}

func (ml *MockLogger) WithFields(fields map[string]interface{}) *MockLogger {
	return ml
}

func (ml *MockLogger) Info(msg string) {
	ml.messages = append(ml.messages, "INFO: "+msg)
}

func (ml *MockLogger) Error(msg string) {
	ml.messages = append(ml.messages, "ERROR: "+msg)
}

func (ml *MockLogger) Warn(msg string) {
	ml.messages = append(ml.messages, "WARN: "+msg)
}

func NewMockPoller(tektonManager *TektonTriggerManager) *MockPoller {
	return &MockPoller{
		tektonManager: tektonManager,
		repositories: []types.Repository{
			{
				Name:     "test-repo",
				URL:      "https://github.com/test-org/test-repo",
				Provider: "github",
			},
		},
		logger: &MockLogger{messages: make([]string, 0)},
	}
}

// SimulateRepositoryChange simulates the Poller processing a repository change
func (mp *MockPoller) SimulateRepositoryChange(ctx context.Context, repo types.Repository, commitSHA, branch string) error {
	if mp.tektonManager == nil {
		mp.logger.Info("No Tekton manager configured, skipping Tekton processing")
		return nil
	}

	mp.logger.Info("Processing repository change with Tekton")

	// Create Tekton process request (similar to Poller implementation)
	request := &TektonProcessRequest{
		Repository: repo,
		CommitSHA:  commitSHA,
		Branch:     branch,
	}

	result, err := mp.tektonManager.ProcessRepositoryChange(ctx, request)
	if err != nil {
		mp.logger.Error("Tekton processing failed: " + err.Error())
		return err
	}

	mp.logger.Info("Tekton processing completed successfully")
	if result.EventSent {
		mp.logger.Info("CloudEvent sent to Bootstrap Pipeline")
	}

	return nil
}

func TestPollerTektonIntegration_WithTektonManager(t *testing.T) {
	// Setup test environment
	mockGitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo:abc123:.tekton": true,
		},
		filesList: map[string][]string{
			"test-repo:abc123:.tekton": {".tekton/pipeline.yaml", ".tekton/task.yaml"},
		},
		fileContents: map[string][]byte{
			"test-repo:abc123:.tekton/pipeline.yaml": []byte(`
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: test-task
    taskRef:
      name: test-task
`),
			"test-repo:abc123:.tekton/task.yaml": []byte(`
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-task
spec:
  steps:
  - name: test-step
    image: alpine
    script: echo "test"
`),
		},
	}

	mockTrigger := &MockTrigger{}
	testLogger := createTestLogger()

	// Create TektonTriggerManager
	clientFactory := NewMockClientFactory(mockGitClient)
	tektonManager := NewTektonTriggerManager(clientFactory, mockTrigger, testLogger)

	// Create mock poller with Tekton integration
	mockPoller := NewMockPoller(tektonManager)

	// Test repository with Tekton resources
	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test-org/test-repo",
		Provider: "github",
	}

	ctx := context.Background()
	err := mockPoller.SimulateRepositoryChange(ctx, repo, "abc123", "main")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify Tekton processing occurred
	if len(mockTrigger.sentEvents) == 0 {
		t.Error("Expected CloudEvent to be sent to trigger")
	}

	// Verify event details
	if len(mockTrigger.sentEvents) > 0 {
		event := mockTrigger.sentEvents[0]
		if event.Type != types.EventTypeTektonDetected {
			t.Errorf("Expected event type %s, got %s", types.EventTypeTektonDetected, event.Type)
		}
		if event.Repository != "test-repo" {
			t.Errorf("Expected repository test-repo, got %s", event.Repository)
		}
		if event.CommitSHA != "abc123" {
			t.Errorf("Expected commit abc123, got %s", event.CommitSHA)
		}
	}
}

func TestPollerTektonIntegration_WithoutTektonManager(t *testing.T) {
	// Create mock poller without Tekton integration
	mockPoller := NewMockPoller(nil)

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test-org/test-repo",
		Provider: "github",
	}

	ctx := context.Background()
	err := mockPoller.SimulateRepositoryChange(ctx, repo, "abc123", "main")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no Tekton processing occurred
	found := false
	for _, msg := range mockPoller.logger.messages {
		if msg == "INFO: No Tekton manager configured, skipping Tekton processing" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected message about skipping Tekton processing")
	}
}

func TestPollerTektonIntegration_NoTektonResources(t *testing.T) {
	// Setup test environment with no Tekton resources
	mockGitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo-no-tekton:def456:.tekton": false,
		},
		filesList:    map[string][]string{},
		fileContents: map[string][]byte{},
	}

	mockTrigger := &MockTrigger{}
	testLogger := createTestLogger()

	// Create TektonTriggerManager
	clientFactory := NewMockClientFactory(mockGitClient)
	tektonManager := NewTektonTriggerManager(clientFactory, mockTrigger, testLogger)

	// Create mock poller with Tekton integration
	mockPoller := NewMockPoller(tektonManager)

	// Test repository without Tekton resources
	repo := types.Repository{
		Name:     "test-repo-no-tekton",
		URL:      "https://github.com/test-org/test-repo-no-tekton",
		Provider: "github",
	}

	ctx := context.Background()
	err := mockPoller.SimulateRepositoryChange(ctx, repo, "def456", "main")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no CloudEvent was sent for repository without Tekton resources
	if len(mockTrigger.sentEvents) > 0 {
		t.Error("Expected no CloudEvent to be sent for repository without Tekton resources")
	}
}

func TestPollerTektonIntegration_ProcessingError(t *testing.T) {
	// Setup test environment with GitClient error
	mockGitClient := &MockGitClient{
		shouldError:  true,
		errorMessage: "GitClient error for testing",
	}

	mockTrigger := &MockTrigger{}
	testLogger := createTestLogger()

	// Create TektonTriggerManager
	clientFactory := NewMockClientFactory(mockGitClient)
	tektonManager := NewTektonTriggerManager(clientFactory, mockTrigger, testLogger)

	// Create mock poller with Tekton integration
	mockPoller := NewMockPoller(tektonManager)

	repo := types.Repository{
		Name:     "test-repo-error",
		URL:      "https://github.com/test-org/test-repo-error",
		Provider: "github",
	}

	ctx := context.Background()
	err := mockPoller.SimulateRepositoryChange(ctx, repo, "error123", "main")

	// Processing should fail when Tekton detection encounters errors
	if err == nil {
		t.Fatal("Expected error from Tekton processing, got none")
	}

	// Verify the error is about unsupported action
	expectedErrorPrefix := "unsupported estimated action"
	if !strings.Contains(err.Error(), expectedErrorPrefix) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorPrefix, err.Error())
	}

	// Verify no CloudEvent was sent due to error
	if len(mockTrigger.sentEvents) > 0 {
		t.Error("Expected no CloudEvent to be sent when processing encounters errors")
	}
}

func TestPollerTektonIntegration_ConcurrentProcessing(t *testing.T) {
	// Test concurrent processing of multiple repositories
	mockGitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo-1:commit1:.tekton": true,
			"test-repo-2:commit2:.tekton": true,
		},
		filesList: map[string][]string{
			"test-repo-1:commit1:.tekton": {".tekton/pipeline.yaml"},
			"test-repo-2:commit2:.tekton": {".tekton/task.yaml"},
		},
		fileContents: map[string][]byte{
			"test-repo-1:commit1:.tekton/pipeline.yaml": []byte(`
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline-1
spec:
  tasks: []
`),
			"test-repo-2:commit2:.tekton/task.yaml": []byte(`
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-task-2
spec:
  steps: []
`),
		},
	}

	mockTrigger := &MockTrigger{}
	testLogger := createTestLogger()

	clientFactory := NewMockClientFactory(mockGitClient)
	tektonManager := NewTektonTriggerManager(clientFactory, mockTrigger, testLogger)
	mockPoller := NewMockPoller(tektonManager)

	repos := []types.Repository{
		{Name: "test-repo-1", URL: "https://github.com/test-org/test-repo-1", Provider: "github"},
		{Name: "test-repo-2", URL: "https://github.com/test-org/test-repo-2", Provider: "github"},
	}

	commits := []string{"commit1", "commit2"}

	// Process repositories concurrently
	ctx := context.Background()
	errChan := make(chan error, len(repos))

	for i, repo := range repos {
		go func(r types.Repository, commit string) {
			err := mockPoller.SimulateRepositoryChange(ctx, r, commit, "main")
			errChan <- err
		}(repo, commits[i])
	}

	// Wait for all processing to complete
	for i := 0; i < len(repos); i++ {
		err := <-errChan
		if err != nil {
			t.Errorf("Expected no error from concurrent processing, got: %v", err)
		}
	}

	// Verify both repositories generated CloudEvents
	if len(mockTrigger.sentEvents) != 2 {
		t.Errorf("Expected 2 CloudEvents for concurrent processing, got %d", len(mockTrigger.sentEvents))
	}

	// Verify event repositories
	repositories := make(map[string]bool)
	for _, event := range mockTrigger.sentEvents {
		repositories[event.Repository] = true
	}

	if !repositories["test-repo-1"] || !repositories["test-repo-2"] {
		t.Error("Expected events for both test-repo-1 and test-repo-2")
	}
}
