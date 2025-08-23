package trigger

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// EventTransformerImpl implements EventTransformer interface
type EventTransformerImpl struct {
	logger    *logger.Entry
	urlParser *URLParser
}

// NewEventTransformer creates a new event transformer
func NewEventTransformer(parentLogger *logger.Entry) *EventTransformerImpl {
	return &EventTransformerImpl{
		logger: parentLogger.WithFields(logger.Fields{
			"component": "trigger",
			"module":    "transformer",
		}),
		urlParser: NewURLParser(parentLogger),
	}
}

// TransformToCloudEvents transforms event to CloudEvents 1.0 standard format
func (t *EventTransformerImpl) TransformToCloudEvents(event types.Event) (CloudEventsPayload, error) {
	t.logger.WithFields(logger.Fields{
		"operation":  "transform_to_cloudevents",
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Debug("Transforming event to CloudEvents format")

	// Get repository URL from event metadata (set by poller)
	repositoryURL, exists := event.Metadata["repository_url"]
	if !exists || repositoryURL == "" {
		return CloudEventsPayload{}, fmt.Errorf("repository URL not found in event metadata")
	}

	repoInfo, err := t.urlParser.ParseRepositoryURL(repositoryURL)
	if err != nil {
		return CloudEventsPayload{}, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Create repository data
	repository := CloudEventsRepository{
		Provider:     repoInfo.Provider,
		Organization: repoInfo.Namespace,   // Use Namespace as Organization
		Name:         repoInfo.ProjectName, // Use ProjectName as Name
		FullName:     repoInfo.FullName,
		URL:          repositoryURL,
		ID:           fmt.Sprintf("%s-%s", strings.ToLower(repoInfo.Provider), strings.ReplaceAll(strings.ToLower(repoInfo.FullName), "/", "-")),
	}

	// Create branch data
	branch := CloudEventsBranch{
		Name: event.Branch,
		Ref:  fmt.Sprintf("refs/heads/%s", event.Branch),
	}

	// Create commit data
	var shortSHA string
	if len(event.CommitSHA) >= 8 {
		shortSHA = event.CommitSHA[:8]
	} else {
		shortSHA = event.CommitSHA
	}

	// Get commit message from metadata if available
	commitMessage := "No commit message available"
	if msg, exists := event.Metadata["commit_message"]; exists && msg != "" {
		commitMessage = msg
	}

	commit := CloudEventsCommit{
		SHA:      event.CommitSHA,
		ShortSHA: shortSHA,
		Message:  commitMessage,
	}

	// Create previous commit data if available
	var previousCommit *CloudEventsCommit
	if event.PrevCommit != "" {
		var prevShortSHA string
		if len(event.PrevCommit) >= 8 {
			prevShortSHA = event.PrevCommit[:8]
		} else {
			prevShortSHA = event.PrevCommit
		}
		previousCommit = &CloudEventsCommit{
			SHA:      event.PrevCommit,
			ShortSHA: prevShortSHA,
		}
	}

	// Create event data - convert EventType to string
	var eventTypeStr string
	switch event.Type {
	case types.EventTypeBranchUpdated:
		eventTypeStr = "branch_updated"
	case types.EventTypeBranchCreated:
		eventTypeStr = "branch_created"
	case types.EventTypeBranchDeleted:
		eventTypeStr = "branch_deleted"
	default:
		eventTypeStr = "unknown"
	}

	eventData := CloudEventsEvent{
		Type:          eventTypeStr,
		TriggerSource: "reposentry",
		TriggerID:     fmt.Sprintf("reposentry-%s-%d", strings.ToLower(repoInfo.Provider), event.Timestamp.Unix()),
		DetectionTime: event.Timestamp.Format(time.RFC3339),
	}

	// Create CloudEvents payload
	payload := CloudEventsPayload{
		SpecVersion:     "1.0",
		Type:            fmt.Sprintf("dev.reposentry.repository.%s", eventTypeStr),
		Source:          fmt.Sprintf("reposentry/%s", strings.ToLower(repoInfo.Provider)),
		ID:              fmt.Sprintf("event_%s_%s", commit.ShortSHA, event.Timestamp.Format("20060102_150405")),
		Time:            event.Timestamp.Format(time.RFC3339),
		DataContentType: "application/json",
		Data: CloudEventsData{
			Repository:     repository,
			Branch:         branch,
			Commit:         commit,
			Event:          eventData,
			PreviousCommit: previousCommit,
		},
	}

	t.logger.WithFields(logger.Fields{
		"operation":          "transform_to_cloudevents",
		"event_id":           event.ID,
		"cloudevents_id":     payload.ID,
		"cloudevents_type":   payload.Type,
		"cloudevents_source": payload.Source,
		"repository":         payload.Data.Repository.FullName,
		"provider":           payload.Data.Repository.Provider,
		"organization":       payload.Data.Repository.Organization,
	}).Info("Successfully transformed event to CloudEvents format")

	return payload, nil
}

// TransformToGitHub transforms event to GitHub webhook format
func (t *EventTransformerImpl) TransformToGitHub(event types.Event) (GitHubPayload, error) {
	t.logger.WithFields(logger.Fields{
		"operation":  "transform_to_github",
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Debug("Transforming event to GitHub format")

	// Extract repository information
	repo, err := t.extractRepositoryInfo(event)
	if err != nil {
		return GitHubPayload{}, fmt.Errorf("failed to extract repository info: %w", err)
	}

	// Create GitHub-style payload
	payload := GitHubPayload{
		Repository: GitHubRepository{
			Name:     event.Repository,
			FullName: repo.FullName,
			CloneURL: repo.CloneURL,
			HTMLURL:  repo.HTMLURL,
			Private:  repo.Private,
		},
		After:    event.CommitSHA,
		ShortSHA: t.getShortSHA(event.CommitSHA),
		Ref:      t.getBranchRef(event.Branch),
		Before:   event.PrevCommit,
	}

	// Add commit information if available
	if commit := t.createCommitFromEvent(event); commit != nil {
		payload.HeadCommit = commit
		payload.Commits = []GitHubCommit{*commit}
	}

	t.logger.WithFields(logger.Fields{
		"operation":  "transform_to_github",
		"event_id":   event.ID,
		"repository": payload.Repository.Name,
		"commit_sha": payload.After,
		"short_sha":  payload.ShortSHA,
		"ref":        payload.Ref,
	}).Info("Successfully transformed event to GitHub format")

	return payload, nil
}

// TransformToTekton transforms event to Tekton EventListener format
func (t *EventTransformerImpl) TransformToTekton(event types.Event) (TektonPayload, error) {
	t.logger.WithFields(logger.Fields{
		"operation":  "transform_to_tekton",
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Debug("Transforming event to Tekton format")

	// First transform to GitHub format
	githubPayload, err := t.TransformToGitHub(event)
	if err != nil {
		return TektonPayload{}, fmt.Errorf("failed to create base GitHub payload: %w", err)
	}

	// Create Tekton-specific metadata
	metadata := make(map[string]interface{})
	metadata["event_type"] = string(event.Type)
	metadata["provider"] = event.Provider
	metadata["protected"] = t.getBranchProtection(event)
	metadata["reposentry_event_id"] = event.ID
	metadata["reposentry_timestamp"] = event.Timestamp.Format(time.RFC3339)

	// Enhanced metadata for better identification
	metadata["branch"] = event.Branch
	metadata["repository_name"] = event.Repository
	metadata["commit_sha"] = event.CommitSHA
	metadata["short_sha"] = ""
	if len(event.CommitSHA) >= 8 {
		metadata["short_sha"] = event.CommitSHA[:8]
	}

	// Extract organization from repository name (e.g., "johnnynv/TaaP_POC" -> "johnnynv")
	repoParts := strings.Split(event.Repository, "/")
	if len(repoParts) >= 1 {
		metadata["organization"] = repoParts[0]
	}
	if len(repoParts) >= 2 {
		metadata["project_name"] = repoParts[len(repoParts)-1] // Last part is project name
	}

	// Generate unique identifier for this repository + provider combination
	metadata["repository_id"] = fmt.Sprintf("%s-%s", event.Provider, strings.ReplaceAll(event.Repository, "/", "-"))

	// Add trigger context for debugging
	metadata["trigger_source"] = "reposentry"
	metadata["trigger_id"] = fmt.Sprintf("reposentry-%s-%d", event.Provider, event.Timestamp.Unix())

	// Add custom metadata from event
	for key, value := range event.Metadata {
		metadata["custom_"+key] = value
	}

	// Create Tekton payload
	payload := TektonPayload{
		GitHubPayload: githubPayload,
		Metadata:      metadata,
		Source:        "reposentry",
		EventID:       event.ID,
	}

	t.logger.WithFields(logger.Fields{
		"operation":      "transform_to_tekton",
		"event_id":       event.ID,
		"repository":     payload.Repository.Name,
		"source":         payload.Source,
		"metadata_count": len(payload.Metadata),
	}).Info("Successfully transformed event to Tekton format")

	return payload, nil
}

// TransformToGeneric transforms event to generic webhook format
func (t *EventTransformerImpl) TransformToGeneric(event types.Event) (GenericPayload, error) {
	t.logger.WithFields(logger.Fields{
		"operation":  "transform_to_generic",
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Debug("Transforming event to generic format")

	// Extract repository information
	repo, err := t.extractRepositoryInfo(event)
	if err != nil {
		return GenericPayload{}, fmt.Errorf("failed to extract repository info: %w", err)
	}

	// Create repository map
	repositoryMap := map[string]interface{}{
		"name":      event.Repository,
		"provider":  event.Provider,
		"branch":    event.Branch,
		"clone_url": repo.CloneURL,
		"html_url":  repo.HTMLURL,
	}

	// Create metadata map
	metadata := map[string]interface{}{
		"event_type":   string(event.Type),
		"commit_sha":   event.CommitSHA,
		"prev_commit":  event.PrevCommit,
		"protected":    t.getBranchProtection(event),
		"processed_at": event.ProcessedAt,
		"created_at":   event.CreatedAt,
		"updated_at":   event.UpdatedAt,
	}

	// Add custom metadata
	for key, value := range event.Metadata {
		metadata[key] = value
	}

	payload := GenericPayload{
		Event:      event,
		Repository: repositoryMap,
		Metadata:   metadata,
		Source:     "reposentry",
		Timestamp:  time.Now(),
	}

	t.logger.WithFields(logger.Fields{
		"operation":      "transform_to_generic",
		"event_id":       event.ID,
		"repository":     payload.Repository["name"],
		"metadata_count": len(payload.Metadata),
	}).Info("Successfully transformed event to generic format")

	return payload, nil
}

// extractRepositoryInfo extracts repository information from event metadata
func (t *EventTransformerImpl) extractRepositoryInfo(event types.Event) (repositoryInfo, error) {
	// Try to get repository URL from metadata first
	repoURL, hasURL := event.Metadata["repository_url"]

	var repoInfo *RepositoryInfo
	var err error

	if hasURL && repoURL != "" {
		// Parse the repository URL using our intelligent parser
		repoInfo, err = t.urlParser.ParseRepositoryURL(repoURL)
		if err != nil {
			t.logger.WithFields(logger.Fields{
				"operation": "extract_repository_info",
				"event_id":  event.ID,
				"repo_url":  repoURL,
				"error":     err.Error(),
			}).Warn("Failed to parse repository URL from metadata, using fallback")
		}
	}

	// Fallback: construct from available information
	if repoInfo == nil {
		t.logger.WithFields(logger.Fields{
			"operation":  "extract_repository_info",
			"event_id":   event.ID,
			"repository": event.Repository,
			"provider":   event.Provider,
		}).Debug("Using fallback repository info construction")

		// Use the URLParser to build URLs from components
		// For fallback, we need to determine instance (default to public instances)
		instance := t.getDefaultInstance(event.Provider)
		repoInfo = t.urlParser.BuildRepoURLs(instance, event.Repository, event.Provider)
	}

	// Convert to legacy repositoryInfo format for compatibility
	info := repositoryInfo{
		FullName: repoInfo.FullName,
		CloneURL: repoInfo.CloneURL,
		HTMLURL:  repoInfo.HTMLURL,
		Private:  false, // Default to public (can be enhanced later)
	}

	t.logger.WithFields(logger.Fields{
		"operation":     "extract_repository_info",
		"event_id":      event.ID,
		"provider":      repoInfo.Provider,
		"instance":      repoInfo.Instance,
		"full_name":     repoInfo.FullName,
		"is_enterprise": repoInfo.IsEnterprise,
	}).Debug("Successfully extracted repository information")

	return info, nil
}

// getDefaultInstance returns the default instance for a provider
func (t *EventTransformerImpl) getDefaultInstance(provider string) string {
	switch provider {
	case "github":
		return "github.com"
	case "gitlab":
		return "gitlab.com"
	default:
		return "gitlab.com" // Default fallback
	}
}

// repositoryInfo holds extracted repository information
type repositoryInfo struct {
	FullName string
	CloneURL string
	HTMLURL  string
	Private  bool
}

// getShortSHA returns the short version of commit SHA
func (t *EventTransformerImpl) getShortSHA(commitSHA string) string {
	if len(commitSHA) >= 8 {
		return commitSHA[:8]
	}
	return commitSHA
}

// getBranchRef converts branch name to Git ref format
func (t *EventTransformerImpl) getBranchRef(branch string) string {
	if strings.HasPrefix(branch, "refs/") {
		return branch
	}
	return "refs/heads/" + branch
}

// getBranchProtection extracts branch protection status from metadata
func (t *EventTransformerImpl) getBranchProtection(event types.Event) bool {
	if protectedStr, ok := event.Metadata["protected"]; ok {
		if protected, err := strconv.ParseBool(protectedStr); err == nil {
			return protected
		}
	}
	return false
}

// createCommitFromEvent creates a GitHubCommit from event information
func (t *EventTransformerImpl) createCommitFromEvent(event types.Event) *GitHubCommit {
	if event.CommitSHA == "" {
		return nil
	}

	commit := &GitHubCommit{
		ID:        event.CommitSHA,
		Timestamp: event.Timestamp,
	}

	// Extract commit message from metadata if available
	if message, ok := event.Metadata["commit_message"]; ok {
		commit.Message = message
	}

	// Extract author information from metadata if available
	if authorName, ok := event.Metadata["author_name"]; ok {
		commit.Author.Name = authorName
	}
	if authorEmail, ok := event.Metadata["author_email"]; ok {
		commit.Author.Email = authorEmail
	}

	// Extract commit URL from metadata if available
	if commitURL, ok := event.Metadata["commit_url"]; ok {
		commit.URL = commitURL
	}

	return commit
}
