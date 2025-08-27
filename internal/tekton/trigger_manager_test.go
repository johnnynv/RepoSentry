package tekton

import (
	"context"
	"fmt"
	"testing"

	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Import trigger types for MockTrigger
type TriggerResult = trigger.TriggerResult
type BatchTriggerResult = trigger.BatchTriggerResult
type TriggerConfig = trigger.TriggerConfig
type TriggerMetrics = trigger.TriggerMetrics

// MockTrigger implements the trigger.Trigger interface for testing
type MockTrigger struct {
	sentEvents []types.Event
	shouldFail bool
	failError  error
}

func (mt *MockTrigger) SendEvent(ctx context.Context, event types.Event) (*TriggerResult, error) {
	if mt.shouldFail {
		return nil, mt.failError
	}
	mt.sentEvents = append(mt.sentEvents, event)
	return &TriggerResult{Success: true}, nil
}

func (mt *MockTrigger) BatchSendEvents(ctx context.Context, events []types.Event) (*BatchTriggerResult, error) {
	if mt.shouldFail {
		return nil, mt.failError
	}
	mt.sentEvents = append(mt.sentEvents, events...)
	return &BatchTriggerResult{SuccessCount: len(events)}, nil
}

func (mt *MockTrigger) ValidateConfig(config TriggerConfig) error {
	return nil
}

func (mt *MockTrigger) GetType() string {
	return "mock"
}

func (mt *MockTrigger) HealthCheck(ctx context.Context) error {
	return nil
}

func (mt *MockTrigger) GetMetrics() TriggerMetrics {
	return TriggerMetrics{}
}

func (mt *MockTrigger) Close() error {
	return nil
}

func (mt *MockTrigger) GetSentEvents() []types.Event {
	return mt.sentEvents
}

func (mt *MockTrigger) Reset() {
	mt.sentEvents = []types.Event{}
	mt.shouldFail = false
	mt.failError = nil
}

func TestNewTektonTriggerManager(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.detector == nil {
		t.Fatal("Expected detector to be set")
	}

	if manager.eventGenerator == nil {
		t.Fatal("Expected eventGenerator to be set")
	}

	if manager.trigger == nil {
		t.Fatal("Expected trigger to be set")
	}

	if manager.logger == nil {
		t.Fatal("Expected logger to be set")
	}
}

func TestTektonTriggerManager_ProcessRepositoryChange_WithTektonResources(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo:abc123:.tekton": true,
		},
		filesList: map[string][]string{
			"test-repo:abc123:.tekton": {
				".tekton/pipeline.yaml",
				".tekton/task.yaml",
			},
		},
		fileContents: map[string][]byte{
			"test-repo:abc123:.tekton/pipeline.yaml": []byte(`apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: test-task`),
			"test-repo:abc123:.tekton/task.yaml": []byte(`apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-task
spec:
  steps:
  - name: echo
    image: ubuntu
    command: ["echo"]
    args: ["hello"]`),
		},
	}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	request := &TektonProcessRequest{
		Repository: types.Repository{
			Name: "test-repo",
			URL:  "https://github.com/test/repo.git",
		},
		CommitSHA: "abc123",
		Branch:    "main",
	}

	result, err := manager.ProcessRepositoryChange(context.Background(), request)
	if err != nil {
		t.Fatalf("Expected no error processing repository change, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	// Verify detection results
	if result.Detection == nil {
		t.Fatal("Expected detection to be performed")
	}

	if result.Detection.EstimatedAction != "apply" {
		t.Errorf("Expected estimated action to be 'apply', got: %s", result.Detection.EstimatedAction)
	}

	if len(result.Detection.Resources) != 2 {
		t.Errorf("Expected 2 resources to be detected, got: %d", len(result.Detection.Resources))
	}

	// Verify event was sent
	if !result.EventSent {
		t.Error("Expected event to be sent")
	}

	if result.Status != "event_sent" {
		t.Errorf("Expected status to be 'event_sent', got: %s", result.Status)
	}

	// Verify trigger received the event
	sentEvents := trigger.GetSentEvents()
	if len(sentEvents) != 1 {
		t.Errorf("Expected 1 event to be sent to trigger, got: %d", len(sentEvents))
	}

	if len(sentEvents) > 0 {
		event := sentEvents[0]
		if event.Type != types.EventTypeTektonDetected {
			t.Errorf("Expected event type to be %s, got: %s", types.EventTypeTektonDetected, event.Type)
		}

		if event.Repository != "test-repo" {
			t.Errorf("Expected event repository to be 'test-repo', got: %s", event.Repository)
		}
	}
}

func TestTektonTriggerManager_ProcessRepositoryChange_NoTektonResources(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo-no-tekton:def456:.tekton": false,
		},
	}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	request := &TektonProcessRequest{
		Repository: types.Repository{
			Name: "test-repo-no-tekton",
			URL:  "https://github.com/test/repo-no-tekton.git",
		},
		CommitSHA: "def456",
		Branch:    "main",
	}

	result, err := manager.ProcessRepositoryChange(context.Background(), request)
	if err != nil {
		t.Fatalf("Expected no error processing repository with no Tekton resources, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	// Verify detection results
	if result.Detection == nil {
		t.Fatal("Expected detection to be performed")
	}

	if result.Detection.EstimatedAction != "skip" {
		t.Errorf("Expected estimated action to be 'skip', got: %s", result.Detection.EstimatedAction)
	}

	// Verify no event was sent for skip action
	if result.EventSent {
		t.Error("Expected no event to be sent for skip action")
	}

	if result.Status != "skipped" {
		t.Errorf("Expected status to be 'skipped', got: %s", result.Status)
	}

	// Verify no events sent to trigger
	sentEvents := trigger.GetSentEvents()
	if len(sentEvents) != 0 {
		t.Errorf("Expected 0 events to be sent to trigger for skip action, got: %d", len(sentEvents))
	}
}

func TestTektonTriggerManager_ProcessRepositoryChange_DetectionError(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{
		shouldError:  true,
		errorMessage: "git client error",
	}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	request := &TektonProcessRequest{
		Repository: types.Repository{
			Name: "test-repo-error",
			URL:  "https://github.com/test/repo-error.git",
		},
		CommitSHA: "error123",
		Branch:    "main",
	}

	result, err := manager.ProcessRepositoryChange(context.Background(), request)

	// Should return error for detection failure
	if err == nil {
		t.Fatal("Expected error for detection failure")
	}

	if result == nil {
		t.Fatal("Expected result to be returned even on error")
	}

	if result.Status != "unsupported_action" {
		t.Errorf("Expected status to be 'unsupported_action', got: %s", result.Status)
	}

	if result.Error == nil {
		t.Error("Expected error to be set in result")
	}

	// Verify no events sent to trigger
	sentEvents := trigger.GetSentEvents()
	if len(sentEvents) != 0 {
		t.Errorf("Expected 0 events to be sent to trigger on detection error, got: %d", len(sentEvents))
	}
}

func TestTektonTriggerManager_ProcessRepositoryChange_TriggerError(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo-trigger-error:trigger123:.tekton": true,
		},
		filesList: map[string][]string{
			"test-repo-trigger-error:trigger123:.tekton": {
				".tekton/pipeline.yaml",
			},
		},
		fileContents: map[string][]byte{
			"test-repo-trigger-error:trigger123:.tekton/pipeline.yaml": []byte(`apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: test-task`),
		},
	}
	trigger := &MockTrigger{
		shouldFail: true,
		failError:  fmt.Errorf("trigger send error"),
	}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	request := &TektonProcessRequest{
		Repository: types.Repository{
			Name: "test-repo-trigger-error",
			URL:  "https://github.com/test/repo-trigger-error.git",
		},
		CommitSHA: "trigger123",
		Branch:    "main",
	}

	result, err := manager.ProcessRepositoryChange(context.Background(), request)

	// Should return error for trigger failure
	if err == nil {
		t.Fatal("Expected error for trigger failure")
	}

	if result == nil {
		t.Fatal("Expected result to be returned even on error")
	}

	if result.Status != "event_send_failed" {
		t.Errorf("Expected status to be 'event_send_failed', got: %s", result.Status)
	}

	if result.Error == nil {
		t.Error("Expected error to be set in result")
	}

	// Event should not be sent due to trigger failure
	if result.EventSent {
		t.Error("Expected event not to be sent due to trigger failure")
	}
}

func TestTektonTriggerManager_GetDetectionStatus(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{
		directoryExists: map[string]bool{
			"test-repo-status:status123:.tekton": true,
		},
		filesList: map[string][]string{
			"test-repo-status:status123:.tekton": {
				".tekton/task.yaml",
			},
		},
		fileContents: map[string][]byte{
			"test-repo-status:status123:.tekton/task.yaml": []byte(`apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-task
spec:
  steps:
  - name: echo
    image: ubuntu`),
		},
	}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	repository := types.Repository{
		Name: "test-repo-status",
		URL:  "https://github.com/test/repo-status.git",
	}

	detection, err := manager.GetDetectionStatus(context.Background(), repository, "status123")
	if err != nil {
		t.Fatalf("Expected no error getting detection status, got: %v", err)
	}

	if detection == nil {
		t.Fatal("Expected detection to be returned")
	}

	if detection.EstimatedAction != "apply" {
		t.Errorf("Expected estimated action to be 'apply', got: %s", detection.EstimatedAction)
	}

	if len(detection.Resources) != 1 {
		t.Errorf("Expected 1 resource to be detected, got: %d", len(detection.Resources))
	}

	// Verify no events sent to trigger (status check only)
	sentEvents := trigger.GetSentEvents()
	if len(sentEvents) != 0 {
		t.Errorf("Expected 0 events to be sent to trigger for status check, got: %d", len(sentEvents))
	}
}

func TestTektonTriggerManager_IsEnabled(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	if !manager.IsEnabled() {
		t.Error("Expected manager to be enabled when trigger and detector are set")
	}

	// Test with nil trigger
	managerWithNilTrigger := &TektonTriggerManager{
		detector:       NewTektonDetector(gitClient, parentLogger),
		eventGenerator: NewTektonEventGenerator(parentLogger),
		trigger:        nil,
		logger:         parentLogger,
	}

	if managerWithNilTrigger.IsEnabled() {
		t.Error("Expected manager to be disabled when trigger is nil")
	}
}

func TestTektonTriggerManager_GetSupportedActions(t *testing.T) {
	parentLogger := createTestLogger()
	gitClient := &MockGitClient{}
	trigger := &MockTrigger{}

	manager := NewTektonTriggerManager(gitClient, trigger, parentLogger)

	actions := manager.GetSupportedActions()

	expectedActions := []string{"apply", "trigger", "validate", "skip"}
	if len(actions) != len(expectedActions) {
		t.Errorf("Expected %d supported actions, got: %d", len(expectedActions), len(actions))
	}

	for i, expected := range expectedActions {
		if i >= len(actions) || actions[i] != expected {
			t.Errorf("Expected action %d to be %s, got: %s", i, expected, actions[i])
		}
	}
}
