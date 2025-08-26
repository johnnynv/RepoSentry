package tekton

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// TektonIntegrationManager orchestrates the complete Tekton integration workflow
type TektonIntegrationManager struct {
	detector          *TektonDetector
	eventGenerator    *TektonEventGenerator
	pipelineGenerator *BootstrapPipelineGenerator
	applier           *KubernetesApplier
	logger            *logger.Entry
}

// NewTektonIntegrationManager creates a new integration manager
func NewTektonIntegrationManager(gitClient gitclient.GitClient, parentLogger *logger.Entry) *TektonIntegrationManager {
	managerLogger := parentLogger.WithFields(logger.Fields{
		"component": "tekton-integration-manager",
	})

	return &TektonIntegrationManager{
		detector:          NewTektonDetector(gitClient, managerLogger),
		eventGenerator:    NewTektonEventGenerator(managerLogger),
		pipelineGenerator: NewBootstrapPipelineGenerator(managerLogger),
		applier:           NewKubernetesApplier(managerLogger),
		logger:            managerLogger,
	}
}

// TektonIntegrationRequest represents a request to process a repository
type TektonIntegrationRequest struct {
	Repository types.Repository
	CommitSHA  string
	Branch     string
	Event      types.Event
}

// TektonIntegrationResult represents the result of processing
type TektonIntegrationResult struct {
	Request            *TektonIntegrationRequest
	Detection          *TektonDetection
	DetectionEvent     *types.TektonDetectionEvent
	StandardEvent      *types.Event
	BootstrapResources *BootstrapPipelineResources
	ExecutionStatus    string
	Namespace          string
	Errors             []string
	ProcessedAt        time.Time
	Duration           time.Duration
}

// ProcessRepositoryChange processes a repository change and executes the full Tekton workflow
func (tim *TektonIntegrationManager) ProcessRepositoryChange(ctx context.Context, request *TektonIntegrationRequest) (*TektonIntegrationResult, error) {
	startTime := time.Now()

	tim.logger.WithFields(logger.Fields{
		"operation":  "process_repository_change",
		"repository": request.Repository.Name,
		"commit":     request.CommitSHA,
		"branch":     request.Branch,
	}).Info("Starting Tekton integration workflow")

	result := &TektonIntegrationResult{
		Request:         request,
		ExecutionStatus: "started",
		Errors:          []string{},
		ProcessedAt:     startTime,
	}

	// Step 1: Detect Tekton resources
	detection, err := tim.detector.DetectTektonResources(ctx, request.Repository, request.CommitSHA, request.Branch)
	if err != nil {
		return tim.handleError(result, "detection_failed", fmt.Errorf("detection failed: %w", err))
	}
	result.Detection = detection

	tim.logger.WithFields(logger.Fields{
		"estimated_action": detection.EstimatedAction,
		"resource_count":   len(detection.Resources),
		"has_tekton_dir":   detection.HasTektonDirectory,
	}).Info("Detection completed")

	// Step 2: Generate events
	detectionEvent, err := tim.eventGenerator.GenerateDetectionEvent(detection)
	if err != nil {
		return tim.handleError(result, "event_generation_failed", fmt.Errorf("event generation failed: %w", err))
	}
	result.DetectionEvent = detectionEvent

	standardEvent, err := tim.eventGenerator.GenerateStandardEvent(detection)
	if err != nil {
		return tim.handleError(result, "standard_event_generation_failed", fmt.Errorf("standard event generation failed: %w", err))
	}
	result.StandardEvent = standardEvent

	// Step 3: Generate namespace and Bootstrap Pipeline resources
	namespace := GetGeneratedNamespace(request.Repository)
	result.Namespace = namespace

	if detection.EstimatedAction == "skip" {
		// For skip action, we don't need to proceed with Bootstrap Pipeline
		result.ExecutionStatus = "skipped"
		result.Duration = time.Since(startTime)

		tim.logger.WithFields(logger.Fields{
			"repository": request.Repository.Name,
			"reason":     "no_tekton_resources",
		}).Info("Skipping Bootstrap Pipeline execution")

		return result, nil
	}

	// Step 4: Generate Bootstrap Pipeline resources
	config := &BootstrapPipelineConfig{
		Repository: request.Repository,
		CommitSHA:  request.CommitSHA,
		Branch:     request.Branch,
		Detection:  detection,
		Namespace:  namespace,
	}

	bootstrapResources, err := tim.pipelineGenerator.GenerateBootstrapResources(config)
	if err != nil {
		return tim.handleError(result, "bootstrap_generation_failed", fmt.Errorf("bootstrap generation failed: %w", err))
	}
	result.BootstrapResources = bootstrapResources

	// Step 5: Apply resources to Kubernetes
	err = tim.applier.ApplyBootstrapResources(ctx, bootstrapResources)
	if err != nil {
		return tim.handleError(result, "kubernetes_apply_failed", fmt.Errorf("kubernetes apply failed: %w", err))
	}

	result.ExecutionStatus = "applied"
	result.Duration = time.Since(startTime)

	tim.logger.WithFields(logger.Fields{
		"repository":     request.Repository.Name,
		"namespace":      namespace,
		"action":         detection.EstimatedAction,
		"duration":       result.Duration,
		"resource_count": len(detection.Resources),
	}).Info("Tekton integration workflow completed successfully")

	return result, nil
}

// GetIntegrationStatus gets the status of a Tekton integration
func (tim *TektonIntegrationManager) GetIntegrationStatus(ctx context.Context, repository types.Repository, commitSHA string) (*TektonIntegrationStatus, error) {
	namespace := GetGeneratedNamespace(repository)

	tim.logger.WithFields(logger.Fields{
		"operation":  "get_integration_status",
		"repository": repository.Name,
		"namespace":  namespace,
		"commit":     commitSHA,
	}).Debug("Getting integration status")

	status, err := tim.applier.GetNamespaceStatus(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace status: %w", err)
	}

	integrationStatus := &TektonIntegrationStatus{
		Repository:      repository,
		CommitSHA:       commitSHA,
		Namespace:       namespace,
		NamespaceStatus: status,
		CheckedAt:       time.Now(),
	}

	return integrationStatus, nil
}

// CleanupRepository cleans up resources for a repository
func (tim *TektonIntegrationManager) CleanupRepository(ctx context.Context, repository types.Repository) error {
	namespace := GetGeneratedNamespace(repository)

	tim.logger.WithFields(logger.Fields{
		"operation":  "cleanup_repository",
		"repository": repository.Name,
		"namespace":  namespace,
	}).Info("Starting repository cleanup")

	err := tim.applier.CleanupNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to cleanup namespace: %w", err)
	}

	tim.logger.WithFields(logger.Fields{
		"repository": repository.Name,
		"namespace":  namespace,
	}).Info("Repository cleanup completed")

	return nil
}

// handleError handles errors and updates result
func (tim *TektonIntegrationManager) handleError(result *TektonIntegrationResult, status string, err error) (*TektonIntegrationResult, error) {
	result.ExecutionStatus = status
	result.Errors = append(result.Errors, err.Error())
	result.Duration = time.Since(result.ProcessedAt)

	tim.logger.WithError(err).WithFields(logger.Fields{
		"repository": result.Request.Repository.Name,
		"status":     status,
	}).Error("Tekton integration workflow failed")

	return result, err
}

// TektonIntegrationStatus represents the status of a Tekton integration
type TektonIntegrationStatus struct {
	Repository      types.Repository `json:"repository"`
	CommitSHA       string           `json:"commit_sha"`
	Namespace       string           `json:"namespace"`
	NamespaceStatus *NamespaceStatus `json:"namespace_status"`
	CheckedAt       time.Time        `json:"checked_at"`
}

// NamespaceStatus represents the status of a namespace
type NamespaceStatus struct {
	Exists        bool                    `json:"exists"`
	Phase         string                  `json:"phase"`
	PipelineRuns  []PipelineRunStatus     `json:"pipeline_runs"`
	TaskRuns      []TaskRunStatus         `json:"task_runs"`
	Resources     map[string]int          `json:"resources"` // count by resource type
	LastActivity  *time.Time              `json:"last_activity,omitempty"`
	ResourceUsage *NamespaceResourceUsage `json:"resource_usage,omitempty"`
}

// PipelineRunStatus represents the status of a PipelineRun
type PipelineRunStatus struct {
	Name           string         `json:"name"`
	Status         string         `json:"status"` // "Running", "Succeeded", "Failed", etc.
	StartTime      *time.Time     `json:"start_time,omitempty"`
	CompletionTime *time.Time     `json:"completion_time,omitempty"`
	Duration       *time.Duration `json:"duration,omitempty"`
	Message        string         `json:"message,omitempty"`
}

// TaskRunStatus represents the status of a TaskRun
type TaskRunStatus struct {
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	StartTime      *time.Time     `json:"start_time,omitempty"`
	CompletionTime *time.Time     `json:"completion_time,omitempty"`
	Duration       *time.Duration `json:"duration,omitempty"`
	Message        string         `json:"message,omitempty"`
	PipelineRun    string         `json:"pipeline_run,omitempty"` // Associated PipelineRun
}

// NamespaceResourceUsage represents resource usage in a namespace
type NamespaceResourceUsage struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   int    `json:"pods"`
	PVCs   int    `json:"pvcs"`
}

// ValidateIntegrationRequest validates a Tekton integration request
func (tim *TektonIntegrationManager) ValidateIntegrationRequest(request *TektonIntegrationRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if request.Repository.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if request.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	if request.CommitSHA == "" {
		return fmt.Errorf("commit SHA is required")
	}

	if request.Branch == "" {
		return fmt.Errorf("branch is required")
	}

	// Validate repository URL format
	if !isValidRepositoryURL(request.Repository.URL) {
		return fmt.Errorf("invalid repository URL format: %s", request.Repository.URL)
	}

	return nil
}

// isValidRepositoryURL validates repository URL format
func isValidRepositoryURL(url string) bool {
	// Basic validation - should be HTTP/HTTPS and contain proper Git hosting patterns
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// Check for common Git hosting patterns
	gitHosts := []string{"github.com", "gitlab.com", "gitlab-"}
	for _, host := range gitHosts {
		if strings.Contains(url, host) {
			return true
		}
	}

	return false
}

// GetMetrics returns integration metrics
func (tim *TektonIntegrationManager) GetMetrics() *TektonIntegrationMetrics {
	// TODO: Implement metrics collection
	return &TektonIntegrationMetrics{
		TotalProcessed:        0,
		SuccessfulRuns:        0,
		FailedRuns:            0,
		SkippedRuns:           0,
		AverageProcessingTime: 0,
		ActiveNamespaces:      0,
	}
}

// TektonIntegrationMetrics represents integration metrics
type TektonIntegrationMetrics struct {
	TotalProcessed        int           `json:"total_processed"`
	SuccessfulRuns        int           `json:"successful_runs"`
	FailedRuns            int           `json:"failed_runs"`
	SkippedRuns           int           `json:"skipped_runs"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	ActiveNamespaces      int           `json:"active_namespaces"`
}
