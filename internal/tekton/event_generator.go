package tekton

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// TektonEventGenerator generates events from Tekton detection results
type TektonEventGenerator struct {
	logger *logger.Entry
}

// NewTektonEventGenerator creates a new Tekton event generator
func NewTektonEventGenerator(parentLogger *logger.Entry) *TektonEventGenerator {
	eventLogger := parentLogger.WithFields(logger.Fields{
		"component": "tekton-event-generator",
	})

	return &TektonEventGenerator{
		logger: eventLogger,
	}
}

// GenerateDetectionEvent creates a TektonDetectionEvent from detection results
func (teg *TektonEventGenerator) GenerateDetectionEvent(detection *TektonDetection) (*types.TektonDetectionEvent, error) {
	teg.logger.WithFields(logger.Fields{
		"operation":  "generate_detection_event",
		"repository": detection.Repository.Name,
		"commit":     detection.CommitSHA,
		"branch":     detection.Branch,
		"has_tekton": detection.HasTektonDirectory,
		"action":     detection.EstimatedAction,
	}).Info("Generating Tekton detection event")

	// Generate unique event ID
	eventID := teg.generateEventID(detection)

	// Convert detection to event payload
	detectionPayload := teg.convertDetectionToPayload(detection)

	// Create the event
	event := &types.TektonDetectionEvent{
		Source:    "reposentry",
		EventType: "tekton_detected",
		EventID:   eventID,
		Timestamp: time.Now(),

		Repository: types.TektonRepository{
			Name:     detection.Repository.Name,
			URL:      detection.Repository.URL,
			CloneURL: detection.Repository.URL, // In our case, same as URL
			Owner:    extractOwnerFromURL(detection.Repository.URL),
		},

		Branch: types.TektonBranch{
			Name:      detection.Branch,
			Protected: false, // TODO: Could be enhanced to detect protected branches
		},

		Commit: types.TektonCommit{
			SHA:       detection.CommitSHA,
			Timestamp: detection.DetectedAt,
		},

		Provider:  detection.Repository.Provider,
		Detection: detectionPayload,
		Headers:   teg.generateHeaders(detection),
	}

	teg.logger.WithFields(logger.Fields{
		"event_id":         event.EventID,
		"estimated_action": event.Detection.EstimatedAction,
		"resource_count":   len(event.Detection.Resources),
		"total_files":      event.Detection.TotalFiles,
	}).Info("Successfully generated Tekton detection event")

	return event, nil
}

// GenerateStandardEvent creates a standard Event from Tekton detection for storage
func (teg *TektonEventGenerator) GenerateStandardEvent(detection *TektonDetection) (*types.Event, error) {
	eventID := teg.generateEventID(detection)

	// Create metadata with detection summary
	metadata := map[string]string{
		"scan_path":        detection.ScanPath,
		"estimated_action": detection.EstimatedAction,
		"total_files":      fmt.Sprintf("%d", detection.TotalFiles),
		"valid_files":      fmt.Sprintf("%d", detection.ValidFiles),
		"resource_count":   fmt.Sprintf("%d", len(detection.Resources)),
	}

	if detection.HasTektonDirectory {
		metadata["has_tekton_directory"] = "true"
	} else {
		metadata["has_tekton_directory"] = "false"
	}

	// Add resource type counts
	resourceCounts := teg.countResourcesByType(detection.Resources)
	for resourceType, count := range resourceCounts {
		metadata[fmt.Sprintf("resources_%s", resourceType)] = fmt.Sprintf("%d", count)
	}

	event := &types.Event{
		ID:         eventID,
		Type:       types.EventTypeTektonDetected,
		Repository: detection.Repository.Name,
		Branch:     detection.Branch,
		CommitSHA:  detection.CommitSHA,
		Provider:   detection.Repository.Provider,
		Timestamp:  detection.DetectedAt,
		Metadata:   metadata,
		Status:     types.EventStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return event, nil
}

// convertDetectionToPayload converts TektonDetection to TektonDetectionPayload
func (teg *TektonEventGenerator) convertDetectionToPayload(detection *TektonDetection) types.TektonDetectionPayload {
	// Convert resources to summaries
	resourceSummaries := make([]types.TektonResourceSummary, 0, len(detection.Resources))
	for _, resource := range detection.Resources {
		summary := types.TektonResourceSummary{
			APIVersion:    resource.APIVersion,
			Kind:          resource.Kind,
			Name:          resource.Name,
			Namespace:     resource.Namespace,
			FilePath:      resource.FilePath,
			ResourceIndex: resource.ResourceIndex,
			IsValid:       resource.IsValid,
			Errors:        resource.Errors,
			Dependencies:  resource.Dependencies,
		}
		resourceSummaries = append(resourceSummaries, summary)
	}

	// Count resources by type
	resourceCounts := teg.countResourcesByType(detection.Resources)

	// Generate action reasons
	actionReasons := teg.generateActionReasons(detection)

	return types.TektonDetectionPayload{
		HasTektonDirectory: detection.HasTektonDirectory,
		ScanPath:           detection.ScanPath,
		DetectedAt:         detection.DetectedAt,
		TotalFiles:         detection.TotalFiles,
		ValidFiles:         detection.ValidFiles,
		Resources:          resourceSummaries,
		ResourceCounts:     resourceCounts,
		EstimatedAction:    detection.EstimatedAction,
		ActionReasons:      actionReasons,
		Errors:             detection.Errors,
		Warnings:           detection.Warnings,
	}
}

// generateEventID creates a unique event ID based on detection data
func (teg *TektonEventGenerator) generateEventID(detection *TektonDetection) string {
	// Create a unique identifier based on repo, commit, and timestamp
	data := fmt.Sprintf("%s:%s:%s:%d",
		detection.Repository.Name,
		detection.CommitSHA,
		detection.Branch,
		detection.DetectedAt.Unix())

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("tekton-detection-%x", hash[:8]) // Use first 8 bytes of hash
}

// generateHeaders creates event headers with additional metadata
func (teg *TektonEventGenerator) generateHeaders(detection *TektonDetection) map[string]string {
	headers := map[string]string{
		"X-RepoSentry-Source":       "tekton-detector",
		"X-RepoSentry-Version":      "1.0", // TODO: Get from build info
		"X-Repository-Provider":     detection.Repository.Provider,
		"X-Tekton-Scan-Path":        detection.ScanPath,
		"X-Tekton-Estimated-Action": detection.EstimatedAction,
	}

	if detection.HasTektonDirectory {
		headers["X-Tekton-Directory-Found"] = "true"
	} else {
		headers["X-Tekton-Directory-Found"] = "false"
	}

	return headers
}

// countResourcesByType counts resources by their Kind
func (teg *TektonEventGenerator) countResourcesByType(resources []TektonResource) map[string]int {
	counts := make(map[string]int)
	for _, resource := range resources {
		counts[resource.Kind]++
	}
	return counts
}

// generateActionReasons creates explanations for the estimated action
func (teg *TektonEventGenerator) generateActionReasons(detection *TektonDetection) []string {
	var reasons []string

	switch detection.EstimatedAction {
	case "skip":
		if !detection.HasTektonDirectory {
			reasons = append(reasons, "No .tekton directory found")
		} else if len(detection.Resources) == 0 {
			reasons = append(reasons, "No valid Tekton resources found")
		}

	case "trigger":
		hasRunnableResources := false
		for _, resource := range detection.Resources {
			if resource.Kind == "PipelineRun" || resource.Kind == "TaskRun" {
				hasRunnableResources = true
				reasons = append(reasons, fmt.Sprintf("Found runnable resource: %s/%s", resource.Kind, resource.Name))
			}
		}
		if !hasRunnableResources {
			reasons = append(reasons, "Contains runnable Tekton resources")
		}

	case "apply":
		hasDefinitions := false
		for _, resource := range detection.Resources {
			if resource.Kind == "Pipeline" || resource.Kind == "Task" {
				hasDefinitions = true
				reasons = append(reasons, fmt.Sprintf("Found definition resource: %s/%s", resource.Kind, resource.Name))
			}
		}
		if !hasDefinitions {
			reasons = append(reasons, "Contains Tekton resource definitions")
		}

	case "validate":
		reasons = append(reasons, "Contains Tekton resources requiring validation")
	}

	// Add error/warning reasons
	if len(detection.Errors) > 0 {
		reasons = append(reasons, fmt.Sprintf("Processing errors: %d", len(detection.Errors)))
	}
	if len(detection.Warnings) > 0 {
		reasons = append(reasons, fmt.Sprintf("Processing warnings: %d", len(detection.Warnings)))
	}

	return reasons
}

// extractOwnerFromURL extracts the owner/organization from a repository URL
func extractOwnerFromURL(repoURL string) string {
	// Simple extraction for GitHub/GitLab URLs
	// Examples:
	// https://github.com/owner/repo -> owner
	// https://gitlab.com/group/project -> group

	// Remove protocol
	url := repoURL
	if idx := strings.Index(url, "://"); idx != -1 {
		url = url[idx+3:]
	}

	// Split by / and get the second part
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}
