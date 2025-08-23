package poller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// EventGeneratorImpl implements the EventGenerator interface
type EventGeneratorImpl struct {
	logger *logger.Entry
}

// NewEventGenerator creates a new event generator
func NewEventGenerator(parentLogger *logger.Entry) *EventGeneratorImpl {
	return &EventGeneratorImpl{
		logger: parentLogger.WithFields(logger.Fields{
			"component": "poller",
			"module":    "event_generator",
		}),
	}
}

// GenerateEvents creates events from branch changes
func (eg *EventGeneratorImpl) GenerateEvents(ctx context.Context, repo types.Repository, changes []BranchChange) ([]types.Event, error) {
	eg.logger.WithFields(logger.Fields{
		"operation":    "generate_events",
		"repository":   repo.Name,
		"change_count": len(changes),
	}).Info("Starting event generation")

	if len(changes) == 0 {
		eg.logger.WithFields(logger.Fields{
			"operation":  "generate_events",
			"repository": repo.Name,
		}).Debug("No changes detected, skipping event generation")
		return []types.Event{}, nil
	}

	// Filter changes based on repository configuration
	filteredChanges, err := eg.FilterChanges(repo, changes)
	if err != nil {
		eg.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "generate_events",
			"repository": repo.Name,
		}).Error("Failed to filter changes")
		return nil, fmt.Errorf("failed to filter changes: %w", err)
	}

	if len(filteredChanges) == 0 {
		eg.logger.WithFields(logger.Fields{
			"operation":        "generate_events",
			"repository":       repo.Name,
			"original_changes": len(changes),
		}).Info("All changes filtered out, no events to generate")
		return []types.Event{}, nil
	}

	var events []types.Event
	timestamp := time.Now()

	for _, change := range filteredChanges {
		event, err := eg.createEventFromChange(repo, change, timestamp)
		if err != nil {
			eg.logger.WithError(err).WithFields(logger.Fields{
				"operation":   "generate_events",
				"repository":  repo.Name,
				"branch":      change.Branch,
				"change_type": change.ChangeType,
			}).Error("Failed to create event from change")
			continue // Skip this change, continue with others
		}

		events = append(events, event)

		eg.logger.WithFields(logger.Fields{
			"operation":   "generate_events",
			"repository":  repo.Name,
			"branch":      change.Branch,
			"change_type": change.ChangeType,
			"event_id":    event.ID,
		}).Info("Generated event from branch change")
	}

	eg.logger.WithFields(logger.Fields{
		"operation":    "generate_events",
		"repository":   repo.Name,
		"event_count":  len(events),
		"change_count": len(filteredChanges),
	}).Info("Completed event generation")

	return events, nil
}

// FilterChanges applies repository-specific filtering to changes
func (eg *EventGeneratorImpl) FilterChanges(repo types.Repository, changes []BranchChange) ([]BranchChange, error) {
	eg.logger.WithFields(logger.Fields{
		"operation":    "filter_changes",
		"repository":   repo.Name,
		"input_count":  len(changes),
		"branch_regex": repo.BranchRegex,
	}).Debug("Applying change filters")

	if repo.BranchRegex == "" {
		// No regex filter, return all changes
		eg.logger.WithFields(logger.Fields{
			"operation":    "filter_changes",
			"repository":   repo.Name,
			"output_count": len(changes),
		}).Debug("No branch regex specified, returning all changes")
		return changes, nil
	}

	regex, err := regexp.Compile(repo.BranchRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid branch regex '%s': %w", repo.BranchRegex, err)
	}

	var filtered []BranchChange
	for _, change := range changes {
		if regex.MatchString(change.Branch) {
			filtered = append(filtered, change)

			eg.logger.WithFields(logger.Fields{
				"operation":   "filter_changes",
				"repository":  repo.Name,
				"branch":      change.Branch,
				"change_type": change.ChangeType,
			}).Debug("Change passed regex filter")
		} else {
			eg.logger.WithFields(logger.Fields{
				"operation":    "filter_changes",
				"repository":   repo.Name,
				"branch":       change.Branch,
				"change_type":  change.ChangeType,
				"branch_regex": repo.BranchRegex,
			}).Debug("Change filtered out by regex")
		}
	}

	eg.logger.WithFields(logger.Fields{
		"operation":    "filter_changes",
		"repository":   repo.Name,
		"input_count":  len(changes),
		"output_count": len(filtered),
		"filtered_out": len(changes) - len(filtered),
	}).Debug("Completed change filtering")

	return filtered, nil
}

// createEventFromChange creates a single event from a branch change
func (eg *EventGeneratorImpl) createEventFromChange(repo types.Repository, change BranchChange, timestamp time.Time) (types.Event, error) {
	// Generate unique event ID based on repository, branch, commit, and timestamp
	eventID := eg.generateEventID(repo.Name, change.Branch, change.NewCommitSHA, timestamp)

	// Create event metadata
	metadata := map[string]interface{}{
		"repository":     repo.Name,
		"provider":       repo.Provider,
		"branch":         change.Branch,
		"change_type":    change.ChangeType,
		"old_commit_sha": change.OldCommitSHA,
		"new_commit_sha": change.NewCommitSHA,
		"protected":      change.Protected,
		"source":         "reposentry-poller",
		"poller_version": "1.0.0",
	}

	// Add repository URL if available
	if repo.URL != "" {
		metadata["repository_url"] = repo.URL
	}

	// Convert metadata to string map
	metadataStr := make(map[string]string)
	for key, value := range metadata {
		metadataStr[key] = fmt.Sprintf("%v", value)
	}

	event := types.Event{
		ID:         eventID,
		Type:       eg.getEventType(change.ChangeType),
		Repository: repo.Name,
		Branch:     change.Branch,
		CommitSHA:  change.NewCommitSHA,
		PrevCommit: change.OldCommitSHA,
		Provider:   repo.Provider,
		Timestamp:  timestamp,
		Status:     types.EventStatusPending,
		Metadata:   metadataStr,
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp,
	}

	return event, nil
}

// generateEventID creates a unique event ID
func (eg *EventGeneratorImpl) generateEventID(repository, branch, commitSHA string, timestamp time.Time) string {
	// Create a unique identifier based on multiple factors to ensure idempotency
	source := fmt.Sprintf("%s:%s:%s:%d", repository, branch, commitSHA, timestamp.Unix())
	hash := sha256.Sum256([]byte(source))
	return fmt.Sprintf("event_%x", hash[:8]) // Use first 8 bytes for shorter ID
}

// getEventType maps change type to event type
func (eg *EventGeneratorImpl) getEventType(changeType string) types.EventType {
	switch changeType {
	case ChangeTypeNew:
		return types.EventTypeBranchCreated
	case ChangeTypeUpdated:
		return types.EventTypeBranchUpdated
	case ChangeTypeDeleted:
		return types.EventTypeBranchDeleted
	default:
		return types.EventTypeBranchUpdated
	}
}

// EventFilter provides additional filtering capabilities
type EventFilter struct {
	IncludeProtected   bool          `yaml:"include_protected" json:"include_protected"`
	ExcludeProtected   bool          `yaml:"exclude_protected" json:"exclude_protected"`
	IncludeChangeTypes []string      `yaml:"include_change_types" json:"include_change_types"`
	ExcludeChangeTypes []string      `yaml:"exclude_change_types" json:"exclude_change_types"`
	MinCommitAge       time.Duration `yaml:"min_commit_age" json:"min_commit_age"`
}

// ApplyFilter applies an event filter to changes
func (ef *EventFilter) ApplyFilter(changes []BranchChange) []BranchChange {
	if ef == nil {
		return changes
	}

	var filtered []BranchChange

	for _, change := range changes {
		// Check protected branch filter
		if ef.ExcludeProtected && change.Protected {
			continue
		}
		if ef.IncludeProtected && !change.Protected {
			continue
		}

		// Check change type inclusion
		if len(ef.IncludeChangeTypes) > 0 {
			found := false
			for _, includeType := range ef.IncludeChangeTypes {
				if change.ChangeType == includeType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check change type exclusion
		if len(ef.ExcludeChangeTypes) > 0 {
			excluded := false
			for _, excludeType := range ef.ExcludeChangeTypes {
				if change.ChangeType == excludeType {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// Check minimum commit age (for throttling rapid changes)
		if ef.MinCommitAge > 0 {
			timeSinceChange := time.Since(change.Timestamp)
			if timeSinceChange < ef.MinCommitAge {
				continue
			}
		}

		filtered = append(filtered, change)
	}

	return filtered
}

// EventStatistics provides statistics about event generation
type EventStatistics struct {
	TotalChanges      int64 `json:"total_changes"`
	FilteredChanges   int64 `json:"filtered_changes"`
	GeneratedEvents   int64 `json:"generated_events"`
	FailedEvents      int64 `json:"failed_events"`
	NewBranches       int64 `json:"new_branches"`
	UpdatedBranches   int64 `json:"updated_branches"`
	DeletedBranches   int64 `json:"deleted_branches"`
	ProtectedBranches int64 `json:"protected_branches"`
}

// EventBatch represents a batch of events for processing
type EventBatch struct {
	ID         string        `json:"id"`
	Repository string        `json:"repository"`
	Events     []types.Event `json:"events"`
	CreatedAt  time.Time     `json:"created_at"`
	Size       int           `json:"size"`
}

// NewEventBatch creates a new event batch
func NewEventBatch(repository string, events []types.Event) *EventBatch {
	return &EventBatch{
		ID:         fmt.Sprintf("batch_%s_%d", repository, time.Now().Unix()),
		Repository: repository,
		Events:     events,
		CreatedAt:  time.Now(),
		Size:       len(events),
	}
}
