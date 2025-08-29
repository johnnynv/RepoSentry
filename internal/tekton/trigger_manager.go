package tekton

import (
	"context"
	"fmt"
	"time"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// TektonTriggerManager provides a streamlined Tekton integration workflow
// This manager only handles detection and triggering, with Bootstrap Pipeline pre-deployed
type TektonTriggerManager struct {
	clientFactory  gitclient.GitClientFactory
	eventGenerator *TektonEventGenerator
	trigger        trigger.Trigger
	logger         *logger.Entry
}

// NewTektonTriggerManager creates a new Tekton trigger manager
func NewTektonTriggerManager(clientFactory gitclient.GitClientFactory, tektonTrigger trigger.Trigger, parentLogger *logger.Entry) *TektonTriggerManager {
	managerLogger := parentLogger.WithFields(logger.Fields{
		"component": "tekton-trigger-manager",
	})

	return &TektonTriggerManager{
		clientFactory:  clientFactory,
		eventGenerator: NewTektonEventGenerator(managerLogger),
		trigger:        tektonTrigger,
		logger:         managerLogger,
	}
}

// TektonProcessRequest represents a simplified request to process a repository
type TektonProcessRequest struct {
	Repository types.Repository
	CommitSHA  string
	Branch     string
}

// TektonProcessResult represents the simplified result of processing
type TektonProcessResult struct {
	Request     *TektonProcessRequest
	Detection   *TektonDetection
	EventSent   bool
	ProcessedAt time.Time
	Duration    time.Duration
	Status      string
	Error       error
}

// ProcessRepositoryChange processes a repository change using the simplified workflow
func (ttm *TektonTriggerManager) ProcessRepositoryChange(ctx context.Context, request *TektonProcessRequest) (*TektonProcessResult, error) {
	startTime := time.Now()

	ttm.logger.WithFields(logger.Fields{
		"operation":  "process_repository_change",
		"repository": request.Repository.Name,
		"commit":     request.CommitSHA,
		"branch":     request.Branch,
	}).Info("Starting Tekton trigger workflow")

	result := &TektonProcessResult{
		Request:     request,
		ProcessedAt: startTime,
		Status:      "started",
		EventSent:   false,
	}

	// Create Git client for the specific repository
	clientConfig := gitclient.GetDefaultConfig()
	clientConfig.Token = request.Repository.Token

	// Set provider-specific configuration
	if request.Repository.Provider == "gitlab" && request.Repository.APIBaseURL != "" {
		clientConfig.BaseURL = request.Repository.APIBaseURL
	} else if request.Repository.Provider == "github" && request.Repository.APIBaseURL != "" {
		clientConfig.BaseURL = request.Repository.APIBaseURL
	}

	gitClient, err := ttm.clientFactory.CreateClient(request.Repository, clientConfig)
	if err != nil {
		result.Status = "client_creation_failed"
		result.Error = fmt.Errorf("failed to create Git client: %w", err)
		result.Duration = time.Since(startTime)

		ttm.logger.WithError(err).WithFields(logger.Fields{
			"repository": request.Repository.Name,
			"provider":   request.Repository.Provider,
		}).Error("Failed to create Git client for Tekton detection")

		return result, result.Error
	}
	defer gitClient.Close()

	// Create TektonDetector with the repository-specific GitClient
	detector := NewTektonDetector(gitClient, ttm.logger)

	// Step 1: Detect Tekton resources in remote repository
	detection, err := detector.DetectTektonResources(ctx, request.Repository, request.CommitSHA, request.Branch)
	if err != nil {
		result.Status = "detection_failed"
		result.Error = fmt.Errorf("detection failed: %w", err)
		result.Duration = time.Since(startTime)

		ttm.logger.WithError(err).WithFields(logger.Fields{
			"repository": request.Repository.Name,
		}).Error("Tekton resource detection failed")

		return result, result.Error
	}
	result.Detection = detection

	ttm.logger.WithFields(logger.Fields{
		"estimated_action": detection.EstimatedAction,
		"resource_count":   len(detection.Resources),
		"has_tekton_dir":   detection.HasTektonDirectory,
	}).Info("Detection completed")

	// Step 2: Handle based on estimated action
	switch detection.EstimatedAction {
	case "skip":
		result.Status = "skipped"
		result.Duration = time.Since(startTime)

		ttm.logger.WithFields(logger.Fields{
			"repository": request.Repository.Name,
			"reason":     "no_tekton_resources",
		}).Info("Skipping Bootstrap Pipeline execution - no Tekton resources found")

		return result, nil

	case "apply", "trigger", "validate":
		// Step 3: Send CloudEvent to pre-deployed Bootstrap Pipeline
		eventSent, err := ttm.SendBootstrapEvent(ctx, request, detection)
		if err != nil {
			result.Status = "event_send_failed"
			result.Error = fmt.Errorf("failed to send bootstrap event: %w", err)
			result.Duration = time.Since(startTime)

			ttm.logger.WithError(err).WithFields(logger.Fields{
				"repository": request.Repository.Name,
				"action":     detection.EstimatedAction,
			}).Error("Failed to send event to Bootstrap Pipeline")

			return result, result.Error
		}

		result.EventSent = eventSent
		result.Status = "event_sent"
		result.Duration = time.Since(startTime)

		ttm.logger.WithFields(logger.Fields{
			"repository":     request.Repository.Name,
			"action":         detection.EstimatedAction,
			"duration":       result.Duration,
			"resource_count": len(detection.Resources),
			"event_sent":     eventSent,
		}).Info("Simplified Tekton integration workflow completed successfully")

		return result, nil

	default:
		result.Status = "unsupported_action"
		result.Error = fmt.Errorf("unsupported estimated action: %s", detection.EstimatedAction)
		result.Duration = time.Since(startTime)

		ttm.logger.WithFields(logger.Fields{
			"repository": request.Repository.Name,
			"action":     detection.EstimatedAction,
		}).Error("Unsupported estimated action")

		return result, result.Error
	}
}

// SendBootstrapEvent sends a CloudEvent to the pre-deployed Bootstrap Pipeline
func (ttm *TektonTriggerManager) SendBootstrapEvent(ctx context.Context, request *TektonProcessRequest, detection *TektonDetection) (bool, error) {
	ttm.logger.WithFields(logger.Fields{
		"operation":        "send_bootstrap_event",
		"repository":       request.Repository.Name,
		"estimated_action": detection.EstimatedAction,
	}).Info("Sending CloudEvent to Bootstrap Pipeline")

	// Generate detection event for logging purposes
	_, err := ttm.eventGenerator.GenerateDetectionEvent(detection)
	if err != nil {
		return false, fmt.Errorf("failed to generate detection event: %w", err)
	}

	// Create CloudEvent with repository and detection information
	cloudEvent := &types.Event{
		ID:         fmt.Sprintf("reposentry-tekton-%d", time.Now().Unix()),
		Type:       types.EventTypeTektonDetected,
		Repository: request.Repository.Name,
		Branch:     request.Branch,
		CommitSHA:  request.CommitSHA,
		Provider:   "reposentry",
		Timestamp:  time.Now().UTC(),
		Status:     types.EventStatusPending,
		Metadata: map[string]string{
			"source":           "reposentry.tekton.detection",
			"repository_url":   request.Repository.URL,
			"estimated_action": detection.EstimatedAction,
			"scan_path":        detection.ScanPath,
			"resource_count":   fmt.Sprintf("%d", len(detection.Resources)),
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Send event using the trigger (which will route to Bootstrap Pipeline)
	_, err = ttm.trigger.SendEvent(ctx, *cloudEvent)
	if err != nil {
		return false, fmt.Errorf("failed to send CloudEvent to trigger: %w", err)
	}

	ttm.logger.WithFields(logger.Fields{
		"event_id":         cloudEvent.ID,
		"event_type":       cloudEvent.Type,
		"repository":       request.Repository.Name,
		"estimated_action": detection.EstimatedAction,
	}).Info("CloudEvent sent successfully to Bootstrap Pipeline")

	return true, nil
}

// GetDetectionStatus provides a simple status check for repository detection
func (ttm *TektonTriggerManager) GetDetectionStatus(ctx context.Context, repository types.Repository, commitSHA string) (*TektonDetection, error) {
	ttm.logger.WithFields(logger.Fields{
		"operation":  "get_detection_status",
		"repository": repository.Name,
		"commit":     commitSHA,
	}).Debug("Getting Tekton detection status")

	// Create Git client for the specific repository
	clientConfig := gitclient.GetDefaultConfig()
	clientConfig.Token = repository.Token

	// Set provider-specific configuration
	if repository.Provider == "gitlab" && repository.APIBaseURL != "" {
		clientConfig.BaseURL = repository.APIBaseURL
	} else if repository.Provider == "github" && repository.APIBaseURL != "" {
		clientConfig.BaseURL = repository.APIBaseURL
	}

	gitClient, err := ttm.clientFactory.CreateClient(repository, clientConfig)
	if err != nil {
		ttm.logger.WithError(err).WithFields(logger.Fields{
			"repository": repository.Name,
			"provider":   repository.Provider,
		}).Error("Failed to create Git client for detection status")
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}
	defer gitClient.Close()

	// Create TektonDetector with the repository-specific GitClient
	detector := NewTektonDetector(gitClient, ttm.logger)

	// Perform detection only (no triggering)
	detection, err := detector.DetectTektonResources(ctx, repository, commitSHA, "main")
	if err != nil {
		ttm.logger.WithError(err).WithFields(logger.Fields{
			"repository": repository.Name,
		}).Error("Failed to get detection status")
		return nil, fmt.Errorf("failed to detect Tekton resources: %w", err)
	}

	ttm.logger.WithFields(logger.Fields{
		"repository":       repository.Name,
		"estimated_action": detection.EstimatedAction,
		"resource_count":   len(detection.Resources),
	}).Debug("Detection status retrieved")

	return detection, nil
}

// IsEnabled returns whether the Tekton trigger manager is enabled
func (ttm *TektonTriggerManager) IsEnabled() bool {
	return ttm.trigger != nil && ttm.clientFactory != nil
}

// GetSupportedActions returns the list of supported actions
func (ttm *TektonTriggerManager) GetSupportedActions() []string {
	return []string{"apply", "trigger", "validate", "skip"}
}
