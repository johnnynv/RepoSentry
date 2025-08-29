package tekton

import (
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestNewTektonEventGenerator(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	if generator == nil {
		t.Fatal("Expected event generator to be created")
	}

	if generator.logger == nil {
		t.Error("Logger not set correctly")
	}
}

func TestGenerateDetectionEvent(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	// Create test detection data
	detection := &TektonDetection{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-owner/test-repo",
			Provider: "github",
		},
		CommitSHA:          "abc123",
		Branch:             "main",
		HasTektonDirectory: true,
		TektonFiles: []TektonFile{
			{Path: ".tekton/pipeline.yaml", IsValid: true, Size: 1024},
		},
		Resources: []TektonResource{
			{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Pipeline",
				Name:       "test-pipeline",
				FilePath:   ".tekton/pipeline.yaml",
				IsValid:    true,
			},
		},
		DetectedAt:      time.Now(),
		ScanPath:        ".tekton",
		TotalFiles:      1,
		ValidFiles:      1,
		EstimatedAction: "apply",
		Errors:          []string{},
		Warnings:        []string{},
	}

	event, err := generator.GenerateDetectionEvent(detection)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify event structure
	if event == nil {
		t.Fatal("Expected event to be generated")
	}

	if event.Source != "reposentry" {
		t.Errorf("Expected source 'reposentry', got %s", event.Source)
	}

	if event.EventType != "tekton_detected" {
		t.Errorf("Expected event type 'tekton_detected', got %s", event.EventType)
	}

	if event.Repository.Name != "test-repo" {
		t.Errorf("Expected repository name 'test-repo', got %s", event.Repository.Name)
	}

	if event.Repository.Owner != "test-owner" {
		t.Errorf("Expected repository owner 'test-owner', got %s", event.Repository.Owner)
	}

	if event.Branch.Name != "main" {
		t.Errorf("Expected branch 'main', got %s", event.Branch.Name)
	}

	if event.Commit.SHA != "abc123" {
		t.Errorf("Expected commit SHA 'abc123', got %s", event.Commit.SHA)
	}

	if event.Provider != "github" {
		t.Errorf("Expected provider 'github', got %s", event.Provider)
	}

	// Verify detection payload
	if !event.Detection.HasTektonDirectory {
		t.Error("Expected HasTektonDirectory to be true")
	}

	if event.Detection.EstimatedAction != "apply" {
		t.Errorf("Expected estimated action 'apply', got %s", event.Detection.EstimatedAction)
	}

	if event.Detection.TotalFiles != 1 {
		t.Errorf("Expected total files 1, got %d", event.Detection.TotalFiles)
	}

	if event.Detection.ValidFiles != 1 {
		t.Errorf("Expected valid files 1, got %d", event.Detection.ValidFiles)
	}

	if len(event.Detection.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(event.Detection.Resources))
	}

	resource := event.Detection.Resources[0]
	if resource.Kind != "Pipeline" {
		t.Errorf("Expected resource kind 'Pipeline', got %s", resource.Kind)
	}

	if resource.Name != "test-pipeline" {
		t.Errorf("Expected resource name 'test-pipeline', got %s", resource.Name)
	}

	// Verify resource counts
	if count, exists := event.Detection.ResourceCounts["Pipeline"]; !exists || count != 1 {
		t.Errorf("Expected Pipeline count 1, got %d", count)
	}

	// Verify headers
	if event.Headers == nil {
		t.Error("Expected headers to be set")
	}

	if event.Headers["X-RepoSentry-Source"] != "tekton-detector" {
		t.Errorf("Expected header 'X-RepoSentry-Source' to be 'tekton-detector'")
	}

	if event.Headers["X-Tekton-Directory-Found"] != "true" {
		t.Errorf("Expected header 'X-Tekton-Directory-Found' to be 'true'")
	}
}

func TestGenerateStandardEvent(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	detection := &TektonDetection{
		Repository: types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test-owner/test-repo",
			Provider: "github",
		},
		CommitSHA:          "abc123",
		Branch:             "main",
		HasTektonDirectory: true,
		Resources: []TektonResource{
			{Kind: "Pipeline", Name: "test-pipeline"},
			{Kind: "Task", Name: "test-task"},
		},
		DetectedAt:      time.Now(),
		ScanPath:        ".tekton",
		TotalFiles:      2,
		ValidFiles:      2,
		EstimatedAction: "apply",
	}

	event, err := generator.GenerateStandardEvent(detection)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if event.Type != types.EventTypeTektonDetected {
		t.Errorf("Expected event type 'tekton_detected', got %s", string(event.Type))
	}

	if event.Repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got %s", event.Repository)
	}

	if event.Branch != "main" {
		t.Errorf("Expected branch 'main', got %s", event.Branch)
	}

	if event.CommitSHA != "abc123" {
		t.Errorf("Expected commit SHA 'abc123', got %s", event.CommitSHA)
	}

	if event.Provider != "github" {
		t.Errorf("Expected provider 'github', got %s", event.Provider)
	}

	if event.Status != types.EventStatusPending {
		t.Errorf("Expected status 'pending', got %s", string(event.Status))
	}

	// Verify metadata
	if event.Metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if event.Metadata["estimated_action"] != "apply" {
		t.Errorf("Expected metadata estimated_action 'apply', got %s", event.Metadata["estimated_action"])
	}

	if event.Metadata["has_tekton_directory"] != "true" {
		t.Errorf("Expected metadata has_tekton_directory 'true', got %s", event.Metadata["has_tekton_directory"])
	}

	if event.Metadata["total_files"] != "2" {
		t.Errorf("Expected metadata total_files '2', got %s", event.Metadata["total_files"])
	}

	if event.Metadata["resources_Pipeline"] != "1" {
		t.Errorf("Expected metadata resources_Pipeline '1', got %s", event.Metadata["resources_Pipeline"])
	}

	if event.Metadata["resources_Task"] != "1" {
		t.Errorf("Expected metadata resources_Task '1', got %s", event.Metadata["resources_Task"])
	}
}

func TestGenerateEventID(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	detection1 := &TektonDetection{
		Repository: types.Repository{Name: "repo1"},
		CommitSHA:  "abc123",
		Branch:     "main",
		DetectedAt: time.Unix(1234567890, 0),
	}

	detection2 := &TektonDetection{
		Repository: types.Repository{Name: "repo1"},
		CommitSHA:  "abc123",
		Branch:     "main",
		DetectedAt: time.Unix(1234567890, 0), // Same timestamp
	}

	detection3 := &TektonDetection{
		Repository: types.Repository{Name: "repo1"},
		CommitSHA:  "def456", // Different commit
		Branch:     "main",
		DetectedAt: time.Unix(1234567890, 0),
	}

	id1 := generator.generateEventID(detection1)
	id2 := generator.generateEventID(detection2)
	id3 := generator.generateEventID(detection3)

	// Same data should generate same ID
	if id1 != id2 {
		t.Errorf("Expected same IDs for identical data, got %s and %s", id1, id2)
	}

	// Different data should generate different IDs
	if id1 == id3 {
		t.Errorf("Expected different IDs for different data, got %s for both", id1)
	}

	// Check ID format
	if !strings.HasPrefix(id1, "tekton-detection-") {
		t.Errorf("Expected ID to start with 'tekton-detection-', got %s", id1)
	}
}

func TestCountResourcesByType(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	resources := []TektonResource{
		{Kind: "Pipeline"},
		{Kind: "Pipeline"},
		{Kind: "Task"},
		{Kind: "PipelineRun"},
		{Kind: "Task"},
	}

	counts := generator.countResourcesByType(resources)

	expected := map[string]int{
		"Pipeline":    2,
		"Task":        2,
		"PipelineRun": 1,
	}

	for kind, expectedCount := range expected {
		if count, exists := counts[kind]; !exists || count != expectedCount {
			t.Errorf("Expected count for %s: %d, got %d", kind, expectedCount, count)
		}
	}

	if len(counts) != len(expected) {
		t.Errorf("Expected %d resource types, got %d", len(expected), len(counts))
	}
}

func TestGenerateActionReasons(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewTektonEventGenerator(testLogger)

	tests := []struct {
		name                string
		detection           *TektonDetection
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name: "Skip - No directory",
			detection: &TektonDetection{
				HasTektonDirectory: false,
				EstimatedAction:    "skip",
			},
			expectedContains: []string{"No .tekton directory found"},
		},
		{
			name: "Skip - No resources",
			detection: &TektonDetection{
				HasTektonDirectory: true,
				Resources:          []TektonResource{},
				EstimatedAction:    "skip",
			},
			expectedContains: []string{"No valid Tekton resources found"},
		},
		{
			name: "Trigger - With PipelineRun",
			detection: &TektonDetection{
				HasTektonDirectory: true,
				Resources: []TektonResource{
					{Kind: "PipelineRun", Name: "test-run"},
				},
				EstimatedAction: "trigger",
			},
			expectedContains: []string{"Found runnable resource: PipelineRun/test-run"},
		},
		{
			name: "Apply - With definitions",
			detection: &TektonDetection{
				HasTektonDirectory: true,
				Resources: []TektonResource{
					{Kind: "Pipeline", Name: "test-pipeline"},
					{Kind: "Task", Name: "test-task"},
				},
				EstimatedAction: "apply",
			},
			expectedContains: []string{
				"Found definition resource: Pipeline/test-pipeline",
				"Found definition resource: Task/test-task",
			},
		},
		{
			name: "With errors and warnings",
			detection: &TektonDetection{
				EstimatedAction: "validate",
				Errors:          []string{"error1", "error2"},
				Warnings:        []string{"warning1"},
			},
			expectedContains: []string{
				"Processing errors: 2",
				"Processing warnings: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reasons := generator.generateActionReasons(tt.detection)

			for _, expectedContain := range tt.expectedContains {
				found := false
				for _, reason := range reasons {
					if strings.Contains(reason, expectedContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected reason to contain '%s', but not found in: %v", expectedContain, reasons)
				}
			}

			for _, expectedNotContain := range tt.expectedNotContains {
				for _, reason := range reasons {
					if strings.Contains(reason, expectedNotContain) {
						t.Errorf("Expected reason not to contain '%s', but found in: %s", expectedNotContain, reason)
					}
				}
			}
		})
	}
}

func TestExtractOwnerFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub URL",
			url:      "https://github.com/owner/repo",
			expected: "owner",
		},
		{
			name:     "GitLab URL",
			url:      "https://gitlab.com/group/project",
			expected: "group",
		},
		{
			name:     "Private GitLab URL",
			url:      "https://gitlab-master.nvidia.com/group/project",
			expected: "group",
		},
		{
			name:     "URL with .git suffix",
			url:      "https://github.com/owner/repo.git",
			expected: "owner",
		},
		{
			name:     "URL without protocol",
			url:      "github.com/owner/repo",
			expected: "owner",
		},
		{
			name:     "Invalid URL",
			url:      "invalid-url",
			expected: "",
		},
		{
			name:     "URL with only domain",
			url:      "https://github.com/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOwnerFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
