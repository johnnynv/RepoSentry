package poller

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// BranchMonitorImpl implements the BranchMonitor interface
type BranchMonitorImpl struct {
	storage       storage.Storage
	clientFactory *gitclient.ClientFactory
	logger        *logger.Entry
}

// NewBranchMonitor creates a new branch monitor
func NewBranchMonitor(storage storage.Storage, clientFactory *gitclient.ClientFactory, parentLogger *logger.Entry) *BranchMonitorImpl {
	return &BranchMonitorImpl{
		storage:       storage,
		clientFactory: clientFactory,
		logger: parentLogger.WithFields(logger.Fields{
			"component": "poller",
			"module":    "branch_monitor",
		}),
	}
}

// CheckBranches checks for changes in repository branches
func (bm *BranchMonitorImpl) CheckBranches(ctx context.Context, repo types.Repository) ([]BranchChange, error) {
	startTime := time.Now()

	bm.logger.WithFields(logger.Fields{
		"operation":  "check_branches",
		"repository": repo.Name,
		"provider":   repo.Provider,
	}).Info("Starting branch check")

	// Create Git client
	clientConfig := gitclient.GetDefaultConfig()
	clientConfig.Token = repo.Token

	// Set provider-specific configuration
	if repo.Provider == "gitlab" && repo.APIBaseURL != "" {
		clientConfig.BaseURL = repo.APIBaseURL
	} else if repo.Provider == "github" && repo.APIBaseURL != "" {
		clientConfig.BaseURL = repo.APIBaseURL
	}

	client, err := bm.clientFactory.CreateClient(repo, clientConfig)
	if err != nil {
		bm.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "check_branches",
			"repository": repo.Name,
		}).Error("Failed to create Git client")
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}
	defer client.Close()

	// Get current branches from Git provider
	currentBranches, err := client.GetBranches(ctx, repo)
	if err != nil {
		bm.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "check_branches",
			"repository": repo.Name,
		}).Error("Failed to get current branches")
		return nil, fmt.Errorf("failed to get current branches: %w", err)
	}

	bm.logger.WithFields(logger.Fields{
		"operation":    "check_branches",
		"repository":   repo.Name,
		"branch_count": len(currentBranches),
	}).Debug("Retrieved current branches")

	// Filter branches based on repository regex
	filteredBranches, err := bm.filterBranches(repo, currentBranches)
	if err != nil {
		bm.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "check_branches",
			"repository": repo.Name,
		}).Error("Failed to filter branches")
		return nil, fmt.Errorf("failed to filter branches: %w", err)
	}

	bm.logger.WithFields(logger.Fields{
		"operation":      "check_branches",
		"repository":     repo.Name,
		"filtered_count": len(filteredBranches),
		"original_count": len(currentBranches),
		"branch_regex":   repo.BranchRegex,
	}).Debug("Filtered branches by regex")

	// Get stored branch states
	storedStates, err := bm.storage.GetRepoStates(ctx, repo.Name)
	if err != nil {
		bm.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "check_branches",
			"repository": repo.Name,
		}).Error("Failed to get stored branch states")
		return nil, fmt.Errorf("failed to get stored branch states: %w", err)
	}

	// Convert stored states to map for easy lookup
	storedBranchMap := make(map[string]string) // branch -> commit_sha
	for _, state := range storedStates {
		storedBranchMap[state.Branch] = state.CommitSHA
	}

	bm.logger.WithFields(logger.Fields{
		"operation":    "check_branches",
		"repository":   repo.Name,
		"stored_count": len(storedStates),
	}).Debug("Retrieved stored branch states")

	// Detect changes
	var changes []BranchChange
	checkTime := time.Now()

	// Check for new and updated branches
	for _, branch := range filteredBranches {
		oldCommitSHA, exists := storedBranchMap[branch.Name]

		if !exists {
			// New branch
			change := BranchChange{
				Repository:   repo.Name,
				Branch:       branch.Name,
				OldCommitSHA: "",
				NewCommitSHA: branch.CommitSHA,
				ChangeType:   ChangeTypeNew,
				Timestamp:    checkTime,
				Protected:    branch.Protected,
			}
			changes = append(changes, change)

			bm.logger.WithFields(logger.Fields{
				"operation":   "check_branches",
				"repository":  repo.Name,
				"branch":      branch.Name,
				"commit_sha":  branch.CommitSHA,
				"change_type": ChangeTypeNew,
			}).Info("Detected new branch")

		} else if oldCommitSHA != branch.CommitSHA {
			// Updated branch
			change := BranchChange{
				Repository:   repo.Name,
				Branch:       branch.Name,
				OldCommitSHA: oldCommitSHA,
				NewCommitSHA: branch.CommitSHA,
				ChangeType:   ChangeTypeUpdated,
				Timestamp:    checkTime,
				Protected:    branch.Protected,
			}
			changes = append(changes, change)

			bm.logger.WithFields(logger.Fields{
				"operation":   "check_branches",
				"repository":  repo.Name,
				"branch":      branch.Name,
				"old_commit":  oldCommitSHA,
				"new_commit":  branch.CommitSHA,
				"change_type": ChangeTypeUpdated,
			}).Info("Detected branch update")
		}

		// Update stored state
		repoState := storage.RepositoryState{
			Repository: repo.Name,
			Branch:     branch.Name,
			CommitSHA:  branch.CommitSHA,
			Protected:  branch.Protected,
			LastCheck:  checkTime,
		}

		if err := bm.storage.UpsertRepoState(ctx, repoState); err != nil {
			bm.logger.WithError(err).WithFields(logger.Fields{
				"operation":  "check_branches",
				"repository": repo.Name,
				"branch":     branch.Name,
			}).Error("Failed to update repository state")
		}
	}

	// Check for deleted branches
	currentBranchMap := make(map[string]bool)
	for _, branch := range filteredBranches {
		currentBranchMap[branch.Name] = true
	}

	for branchName, oldCommitSHA := range storedBranchMap {
		if !currentBranchMap[branchName] {
			// Branch was deleted
			change := BranchChange{
				Repository:   repo.Name,
				Branch:       branchName,
				OldCommitSHA: oldCommitSHA,
				NewCommitSHA: "",
				ChangeType:   ChangeTypeDeleted,
				Timestamp:    checkTime,
				Protected:    false, // Unknown, but assuming false
			}
			changes = append(changes, change)

			bm.logger.WithFields(logger.Fields{
				"operation":   "check_branches",
				"repository":  repo.Name,
				"branch":      branchName,
				"old_commit":  oldCommitSHA,
				"change_type": ChangeTypeDeleted,
			}).Info("Detected deleted branch")

			// Remove from storage
			if err := bm.storage.DeleteRepoState(ctx, repo.Name, branchName); err != nil {
				bm.logger.WithError(err).WithFields(logger.Fields{
					"operation":  "check_branches",
					"repository": repo.Name,
					"branch":     branchName,
				}).Error("Failed to delete repository state")
			}
		}
	}

	duration := time.Since(startTime)

	bm.logger.WithFields(logger.Fields{
		"operation":    "check_branches",
		"repository":   repo.Name,
		"change_count": len(changes),
		"duration":     duration.String(),
	}).Info("Completed branch check")

	return changes, nil
}

// GetLastCheckTime returns the last time the repository was checked
func (bm *BranchMonitorImpl) GetLastCheckTime(repo types.Repository) (time.Time, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	states, err := bm.storage.GetRepoStates(ctx, repo.Name)
	if err != nil || len(states) == 0 {
		return time.Time{}, false
	}

	// Return the most recent check time
	var lastCheck time.Time
	for _, state := range states {
		if state.LastChecked.After(lastCheck) {
			lastCheck = state.LastChecked
		}
	}

	return lastCheck, !lastCheck.IsZero()
}

// UpdateLastCheck updates the last check time for a repository
func (bm *BranchMonitorImpl) UpdateLastCheck(repo types.Repository, checkTime time.Time) error {
	bm.logger.WithFields(logger.Fields{
		"operation":  "update_last_check",
		"repository": repo.Name,
		"check_time": checkTime.Format(time.RFC3339),
	}).Debug("Updating last check time")

	// This is handled in CheckBranches when updating individual branch states
	// This method can be used for bulk updates or metadata tracking
	return nil
}

// filterBranches applies the repository's branch regex filter
func (bm *BranchMonitorImpl) filterBranches(repo types.Repository, branches []types.Branch) ([]types.Branch, error) {
	if repo.BranchRegex == "" {
		// No filter specified, return all branches
		return branches, nil
	}

	regex, err := regexp.Compile(repo.BranchRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid branch regex '%s': %w", repo.BranchRegex, err)
	}

	var filtered []types.Branch
	for _, branch := range branches {
		if regex.MatchString(branch.Name) {
			filtered = append(filtered, branch)
		}
	}

	return filtered, nil
}
