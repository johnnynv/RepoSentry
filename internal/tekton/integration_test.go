package tekton

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// TestTektonIntegrationWorkflow tests the complete workflow from detection to event generation
func TestTektonIntegrationWorkflow(t *testing.T) {
	// Setup test environment
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	
	// Create components
	detector := NewTektonDetector(mockClient, testLogger)
	eventGenerator := NewTektonEventGenerator(testLogger)

	repo := types.Repository{
		Name:     "integration-test-repo",
		URL:      "https://github.com/test-org/integration-repo",
		Provider: "github",
	}
	commitSHA := "integration-test-commit-abc123"
	branch := "main"

	t.Run("Complete Workflow - Pipeline Detection", func(t *testing.T) {
		// Setup: Repository has .tekton directory with Pipeline and Task
		mockClient.SetDirectoryExists("integration-test-repo", commitSHA, ".tekton", true)
		mockClient.SetFilesList("integration-test-repo", commitSHA, ".tekton", []string{
			".tekton/ci-pipeline.yaml",
			".tekton/build-task.yaml",
			".tekton/test-task.yml",
		})

		// Setup file contents
		pipelineYAML := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: ci-pipeline
  namespace: default
spec:
  params:
  - name: repo-url
    type: string
  tasks:
  - name: build
    taskRef:
      name: build-task
  - name: test
    taskRef:
      name: test-task
    runAfter:
    - build
`

		buildTaskYAML := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: build-task
spec:
  params:
  - name: repo-url
    type: string
  steps:
  - name: build
    image: golang:1.19
    script: |
      echo "Building application..."
      go build -o app ./cmd/main.go
`

		testTaskYAML := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-task
spec:
  steps:
  - name: test
    image: golang:1.19
    script: |
      echo "Running tests..."
      go test ./...
`

		mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/ci-pipeline.yaml", []byte(pipelineYAML))
		mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/build-task.yaml", []byte(buildTaskYAML))
		mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/test-task.yml", []byte(testTaskYAML))

		// Step 1: Detect Tekton resources
		detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// Verify detection results
		if !detection.HasTektonDirectory {
			t.Error("Expected .tekton directory to be detected")
		}

		if detection.TotalFiles != 3 {
			t.Errorf("Expected 3 files, got %d", detection.TotalFiles)
		}

		if detection.ValidFiles != 3 {
			t.Errorf("Expected 3 valid files, got %d", detection.ValidFiles)
		}

		if len(detection.Resources) != 3 {
			t.Errorf("Expected 3 resources, got %d", len(detection.Resources))
		}

		if detection.EstimatedAction != "apply" {
			t.Errorf("Expected estimated action 'apply', got %s", detection.EstimatedAction)
		}

		// Verify specific resources
		foundPipeline := false
		foundBuildTask := false
		foundTestTask := false

		for _, resource := range detection.Resources {
			switch resource.Name {
			case "ci-pipeline":
				if resource.Kind != "Pipeline" {
					t.Errorf("Expected Pipeline, got %s", resource.Kind)
				}
				foundPipeline = true
			case "build-task":
				if resource.Kind != "Task" {
					t.Errorf("Expected Task, got %s", resource.Kind)
				}
				foundBuildTask = true
			case "test-task":
				if resource.Kind != "Task" {
					t.Errorf("Expected Task, got %s", resource.Kind)
				}
				foundTestTask = true
			}
		}

		if !foundPipeline || !foundBuildTask || !foundTestTask {
			t.Error("Not all expected resources were found")
		}

		// Step 2: Generate TektonDetectionEvent
		detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
		if err != nil {
			t.Fatalf("Event generation failed: %v", err)
		}

		// Verify detection event
		if detectionEvent.EventType != "tekton_detected" {
			t.Errorf("Expected event type 'tekton_detected', got %s", detectionEvent.EventType)
		}

		if detectionEvent.Repository.Name != "integration-test-repo" {
			t.Errorf("Expected repository name 'integration-test-repo', got %s", detectionEvent.Repository.Name)
		}

		if detectionEvent.Repository.Owner != "test-org" {
			t.Errorf("Expected repository owner 'test-org', got %s", detectionEvent.Repository.Owner)
		}

		if detectionEvent.Detection.EstimatedAction != "apply" {
			t.Errorf("Expected estimated action 'apply', got %s", detectionEvent.Detection.EstimatedAction)
		}

		if len(detectionEvent.Detection.Resources) != 3 {
			t.Errorf("Expected 3 resources in event, got %d", len(detectionEvent.Detection.Resources))
		}

		// Verify resource counts
		expectedCounts := map[string]int{"Pipeline": 1, "Task": 2}
		for resourceType, expectedCount := range expectedCounts {
			if count, exists := detectionEvent.Detection.ResourceCounts[resourceType]; !exists || count != expectedCount {
				t.Errorf("Expected %s count %d, got %d", resourceType, expectedCount, count)
			}
		}

		// Step 3: Generate Standard Event for storage
		standardEvent, err := eventGenerator.GenerateStandardEvent(detection)
		if err != nil {
			t.Fatalf("Standard event generation failed: %v", err)
		}

		// Verify standard event
		if standardEvent.Type != types.EventTypeTektonDetected {
			t.Errorf("Expected event type 'tekton_detected', got %s", string(standardEvent.Type))
		}

		if standardEvent.Metadata["estimated_action"] != "apply" {
			t.Errorf("Expected metadata estimated_action 'apply', got %s", standardEvent.Metadata["estimated_action"])
		}

		if standardEvent.Metadata["resources_Pipeline"] != "1" {
			t.Errorf("Expected Pipeline count '1' in metadata, got %s", standardEvent.Metadata["resources_Pipeline"])
		}

		if standardEvent.Metadata["resources_Task"] != "2" {
			t.Errorf("Expected Task count '2' in metadata, got %s", standardEvent.Metadata["resources_Task"])
		}

		t.Logf("Integration test completed successfully - Detection Event ID: %s, Standard Event ID: %s", 
			detectionEvent.EventID, standardEvent.ID)
	})

	t.Run("Complete Workflow - PipelineRun Trigger", func(t *testing.T) {
		// Setup: Repository has .tekton directory with PipelineRun (triggerable)
		mockClient.SetDirectoryExists("integration-test-repo", commitSHA, ".tekton", true)
		mockClient.SetFilesList("integration-test-repo", commitSHA, ".tekton", []string{
			".tekton/pr-trigger.yaml",
		})

		pipelineRunYAML := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: pr-ci-pipeline-run
spec:
  pipelineRef:
    name: ci-pipeline
  params:
  - name: repo-url
    value: "https://github.com/test-org/integration-repo"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
`

		mockClient.SetFileContent("integration-test-repo", commitSHA, ".tekton/pr-trigger.yaml", []byte(pipelineRunYAML))

		// Detection
		detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// Should trigger since we have PipelineRun
		if detection.EstimatedAction != "trigger" {
			t.Errorf("Expected estimated action 'trigger', got %s", detection.EstimatedAction)
		}

		// Generate events
		detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
		if err != nil {
			t.Fatalf("Event generation failed: %v", err)
		}

		// Verify action reasons include trigger explanation
		found := false
		for _, reason := range detectionEvent.Detection.ActionReasons {
			if contains(reason, "runnable resource") && contains(reason, "PipelineRun") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected action reasons to mention runnable PipelineRun resource")
		}

		t.Logf("PipelineRun trigger workflow completed - Action: %s", detection.EstimatedAction)
	})

	t.Run("Complete Workflow - No Tekton Directory", func(t *testing.T) {
		// Setup: Repository has no .tekton directory
		mockClient.SetDirectoryExists("integration-test-repo", commitSHA, ".tekton", false)

		// Detection
		detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// Should skip
		if detection.EstimatedAction != "skip" {
			t.Errorf("Expected estimated action 'skip', got %s", detection.EstimatedAction)
		}

		if detection.HasTektonDirectory {
			t.Error("Expected HasTektonDirectory to be false")
		}

		// Generate events
		detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
		if err != nil {
			t.Fatalf("Event generation failed: %v", err)
		}

		// Verify skip reasons
		found := false
		for _, reason := range detectionEvent.Detection.ActionReasons {
			if contains(reason, "No .tekton directory found") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected action reasons to mention missing .tekton directory")
		}

		t.Logf("No Tekton directory workflow completed - Action: %s", detection.EstimatedAction)
	})

	t.Run("Complete Workflow - Error Handling", func(t *testing.T) {
		// Setup: Git client returns error
		mockClient.SetError(true, "Network timeout while accessing remote repository")

		// Detection should handle errors gracefully
		detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
		if err != nil {
			t.Fatalf("Detection should not return error, but handle it gracefully: %v", err)
		}

		// Should have errors recorded
		if len(detection.Errors) == 0 {
			t.Error("Expected errors to be recorded in detection")
		}

		if !contains(detection.Errors[0], "Network timeout") {
			t.Errorf("Expected error about network timeout, got: %s", detection.Errors[0])
		}

		// Generate events with errors
		detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
		if err != nil {
			t.Fatalf("Event generation failed: %v", err)
		}

		// Verify error propagation
		if len(detectionEvent.Detection.Errors) == 0 {
			t.Error("Expected errors to be propagated to event")
		}

		// Verify action reasons include error information
		found := false
		for _, reason := range detectionEvent.Detection.ActionReasons {
			if contains(reason, "Processing errors") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected action reasons to mention processing errors")
		}

		t.Logf("Error handling workflow completed - Errors: %d", len(detection.Errors))

		// Reset error state for other tests
		mockClient.SetError(false, "")
	})
}

// TestIntegrationPerformance tests the performance of the integrated workflow
func TestIntegrationPerformance(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)
	eventGenerator := NewTektonEventGenerator(testLogger)

	repo := types.Repository{
		Name:     "performance-test-repo",
		URL:      "https://github.com/test-org/performance-repo",
		Provider: "github",
	}
	commitSHA := "performance-test-commit"
	branch := "main"

	// Setup large number of files
	var files []string
	for i := 0; i < 50; i++ {
		files = append(files, fmt.Sprintf(".tekton/task-%d.yaml", i))
	}

	mockClient.SetDirectoryExists("performance-test-repo", commitSHA, ".tekton", true)
	mockClient.SetFilesList("performance-test-repo", commitSHA, ".tekton", files)

	// Setup file contents for all files
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: task-%d
spec:
  steps:
  - name: step
    image: alpine
    script: echo "Task %d"
`

	for i, file := range files {
		content := fmt.Sprintf(taskTemplate, i, i)
		mockClient.SetFileContent("performance-test-repo", commitSHA, file, []byte(content))
	}

	// Measure performance
	start := time.Now()

	detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
	if err != nil {
		t.Fatalf("Event generation failed: %v", err)
	}

	duration := time.Since(start)

	// Verify results
	if len(detection.Resources) != 50 {
		t.Errorf("Expected 50 resources, got %d", len(detection.Resources))
	}

	if len(detectionEvent.Detection.Resources) != 50 {
		t.Errorf("Expected 50 resources in event, got %d", len(detectionEvent.Detection.Resources))
	}

	// Performance assertion (should complete within reasonable time)
	if duration > 5*time.Second {
		t.Errorf("Integration workflow took too long: %v", duration)
	}

	t.Logf("Performance test completed - Duration: %v, Resources: %d", duration, len(detection.Resources))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) <= len(s) && (substr == "" || s[len(s)-len(substr):] == substr || 
		s[:len(substr)] == substr || (len(substr) < len(s) && 
		func() bool {
			for i := 1; i < len(s)-len(substr)+1; i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}

// TestIntegrationConfigurability tests different detector configurations
func TestIntegrationConfigurability(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	detector := NewTektonDetector(mockClient, testLogger)
	eventGenerator := NewTektonEventGenerator(testLogger)

	// Test custom scan path
	customConfig := DetectorConfig{
		ScanPath:       ".ci",
		FileExtensions: []string{".yaml"},
		MaxFileSize:    500 * 1024,
		Timeout:        10 * time.Second,
	}
	detector.SetConfig(customConfig)

	repo := types.Repository{
		Name:     "config-test-repo",
		URL:      "https://github.com/test-org/config-repo",
		Provider: "github",
	}
	commitSHA := "config-test-commit"
	branch := "main"

	// Setup with custom scan path
	mockClient.SetDirectoryExists("config-test-repo", commitSHA, ".ci", true)
	mockClient.SetFilesList("config-test-repo", commitSHA, ".ci", []string{
		".ci/pipeline.yaml",
	})

	pipelineYAML := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: custom-pipeline
spec:
  tasks:
  - name: hello
    taskRef:
      name: hello-task
`

	mockClient.SetFileContent("config-test-repo", commitSHA, ".ci/pipeline.yaml", []byte(pipelineYAML))

	// Detection with custom config
	detection, err := detector.DetectTektonResources(context.Background(), repo, commitSHA, branch)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if detection.ScanPath != ".ci" {
		t.Errorf("Expected scan path '.ci', got %s", detection.ScanPath)
	}

	if len(detection.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(detection.Resources))
	}

	// Generate event
	detectionEvent, err := eventGenerator.GenerateDetectionEvent(detection)
	if err != nil {
		t.Fatalf("Event generation failed: %v", err)
	}

	if detectionEvent.Detection.ScanPath != ".ci" {
		t.Errorf("Expected scan path '.ci' in event, got %s", detectionEvent.Detection.ScanPath)
	}

	t.Logf("Configuration test completed - Custom scan path: %s", detection.ScanPath)
}
