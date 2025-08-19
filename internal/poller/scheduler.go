package poller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// SchedulerImpl implements the Scheduler interface
type SchedulerImpl struct {
	repositories map[string]*ScheduledRepository
	config       PollerConfig
	logger       *logger.Entry
	mu           sync.RWMutex
	stopChan     chan struct{}
	running      bool
	ticker       *time.Ticker
}

// ScheduledRepository represents a repository with scheduling information
type ScheduledRepository struct {
	Repository   types.Repository `json:"repository"`
	NextPollTime time.Time        `json:"next_poll_time"`
	LastPollTime time.Time        `json:"last_poll_time,omitempty"`
	PollCount    int64            `json:"poll_count"`
	Enabled      bool             `json:"enabled"`
}

// NewScheduler creates a new scheduler
func NewScheduler(config PollerConfig) *SchedulerImpl {
	return &SchedulerImpl{
		repositories: make(map[string]*ScheduledRepository),
		config:       config,
		logger: logger.GetDefaultLogger().WithFields(logger.Fields{
			"component": "poller",
			"module":    "scheduler",
		}),
		stopChan: make(chan struct{}),
	}
}

// Schedule schedules a repository for polling
func (s *SchedulerImpl) Schedule(repo types.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !repo.Enabled {
		s.logger.WithFields(logger.Fields{
			"operation":  "schedule",
			"repository": repo.Name,
		}).Debug("Repository is disabled, not scheduling")
		return nil
	}

	nextPollTime := time.Now().Add(s.config.Interval)
	
	scheduledRepo := &ScheduledRepository{
		Repository:   repo,
		NextPollTime: nextPollTime,
		PollCount:    0,
		Enabled:      true,
	}

	s.repositories[repo.Name] = scheduledRepo

	s.logger.WithFields(logger.Fields{
		"operation":      "schedule",
		"repository":     repo.Name,
		"provider":       repo.Provider,
		"next_poll_time": nextPollTime.Format(time.RFC3339),
		"interval":       s.config.Interval.String(),
	}).Info("Scheduled repository for polling")

	return nil
}

// Unschedule removes a repository from polling
func (s *SchedulerImpl) Unschedule(repo types.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.repositories[repo.Name]; !exists {
		s.logger.WithFields(logger.Fields{
			"operation":  "unschedule",
			"repository": repo.Name,
		}).Debug("Repository not found in schedule")
		return nil
	}

	delete(s.repositories, repo.Name)

	s.logger.WithFields(logger.Fields{
		"operation":  "unschedule",
		"repository": repo.Name,
	}).Info("Unscheduled repository from polling")

	return nil
}

// GetNextPollTime returns the next scheduled poll time for a repository
func (s *SchedulerImpl) GetNextPollTime(repo types.Repository) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheduledRepo, exists := s.repositories[repo.Name]
	if !exists {
		return time.Time{}, false
	}

	return scheduledRepo.NextPollTime, true
}

// Start begins the scheduler
func (s *SchedulerImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.WithFields(logger.Fields{
		"operation": "start",
		"interval":  s.config.Interval.String(),
	}).Info("Starting scheduler")

	// Create ticker for polling interval
	s.ticker = time.NewTicker(s.config.Interval)

	go s.run(ctx)

	return nil
}

// Stop gracefully stops the scheduler
func (s *SchedulerImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	s.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping scheduler")

	// Stop ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Signal stop
	close(s.stopChan)

	return nil
}

// run is the main scheduler loop
func (s *SchedulerImpl) run(ctx context.Context) {
	s.logger.Info("Scheduler started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("Scheduler stopped")
			return
		case <-s.ticker.C:
			s.processPendingPolls(ctx)
		}
	}
}

// processPendingPolls checks for repositories that need to be polled
func (s *SchedulerImpl) processPendingPolls(ctx context.Context) {
	s.mu.RLock()
	now := time.Now()
	var readyRepos []*ScheduledRepository

	for _, scheduledRepo := range s.repositories {
		if scheduledRepo.Enabled && now.After(scheduledRepo.NextPollTime) {
			readyRepos = append(readyRepos, scheduledRepo)
		}
	}
	s.mu.RUnlock()

	if len(readyRepos) == 0 {
		s.logger.Debug("No repositories ready for polling")
		return
	}

	s.logger.WithFields(logger.Fields{
		"operation":   "process_pending_polls",
		"ready_count": len(readyRepos),
	}).Info("Processing pending polls")

	// Update poll times for ready repositories
	s.mu.Lock()
	for _, scheduledRepo := range readyRepos {
		scheduledRepo.LastPollTime = now
		scheduledRepo.NextPollTime = now.Add(s.config.Interval)
		scheduledRepo.PollCount++
	}
	s.mu.Unlock()

	// Log each repository being polled
	for _, scheduledRepo := range readyRepos {
		s.logger.WithFields(logger.Fields{
			"operation":      "process_pending_polls",
			"repository":     scheduledRepo.Repository.Name,
			"provider":       scheduledRepo.Repository.Provider,
			"poll_count":     scheduledRepo.PollCount,
			"next_poll_time": scheduledRepo.NextPollTime.Format(time.RFC3339),
		}).Info("Repository ready for polling")
	}
}

// GetScheduledRepositories returns all currently scheduled repositories
func (s *SchedulerImpl) GetScheduledRepositories() []ScheduledRepository {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var repos []ScheduledRepository
	for _, scheduledRepo := range s.repositories {
		repos = append(repos, *scheduledRepo)
	}

	return repos
}

// GetSchedulerStatus returns the current status of the scheduler
func (s *SchedulerImpl) GetSchedulerStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var enabledCount, disabledCount int
	var nextPollTime time.Time
	var earliestNext time.Time

	for _, scheduledRepo := range s.repositories {
		if scheduledRepo.Enabled {
			enabledCount++
			if earliestNext.IsZero() || scheduledRepo.NextPollTime.Before(earliestNext) {
				earliestNext = scheduledRepo.NextPollTime
			}
		} else {
			disabledCount++
		}
	}

	if !earliestNext.IsZero() {
		nextPollTime = earliestNext
	}

	return SchedulerStatus{
		Running:                s.running,
		TotalRepositories:      len(s.repositories),
		EnabledRepositories:    enabledCount,
		DisabledRepositories:   disabledCount,
		NextScheduledPollTime:  nextPollTime,
		PollingInterval:        s.config.Interval,
	}
}

// SchedulerStatus represents the current status of the scheduler
type SchedulerStatus struct {
	Running                bool          `json:"running"`
	TotalRepositories      int           `json:"total_repositories"`
	EnabledRepositories    int           `json:"enabled_repositories"`
	DisabledRepositories   int           `json:"disabled_repositories"`
	NextScheduledPollTime  time.Time     `json:"next_scheduled_poll_time,omitempty"`
	PollingInterval        time.Duration `json:"polling_interval"`
}

// UpdateRepositorySchedule updates the schedule for a specific repository
func (s *SchedulerImpl) UpdateRepositorySchedule(repo types.Repository, nextPollTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledRepo, exists := s.repositories[repo.Name]
	if !exists {
		return fmt.Errorf("repository %s is not scheduled", repo.Name)
	}

	oldNextPollTime := scheduledRepo.NextPollTime
	scheduledRepo.NextPollTime = nextPollTime

	s.logger.WithFields(logger.Fields{
		"operation":          "update_schedule",
		"repository":         repo.Name,
		"old_next_poll_time": oldNextPollTime.Format(time.RFC3339),
		"new_next_poll_time": nextPollTime.Format(time.RFC3339),
	}).Info("Updated repository schedule")

	return nil
}

// EnableRepository enables polling for a repository
func (s *SchedulerImpl) EnableRepository(repoName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledRepo, exists := s.repositories[repoName]
	if !exists {
		return fmt.Errorf("repository %s is not scheduled", repoName)
	}

	if scheduledRepo.Enabled {
		return nil // Already enabled
	}

	scheduledRepo.Enabled = true
	scheduledRepo.NextPollTime = time.Now().Add(s.config.Interval)

	s.logger.WithFields(logger.Fields{
		"operation":      "enable_repository",
		"repository":     repoName,
		"next_poll_time": scheduledRepo.NextPollTime.Format(time.RFC3339),
	}).Info("Enabled repository for polling")

	return nil
}

// DisableRepository disables polling for a repository
func (s *SchedulerImpl) DisableRepository(repoName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledRepo, exists := s.repositories[repoName]
	if !exists {
		return fmt.Errorf("repository %s is not scheduled", repoName)
	}

	if !scheduledRepo.Enabled {
		return nil // Already disabled
	}

	scheduledRepo.Enabled = false

	s.logger.WithFields(logger.Fields{
		"operation":  "disable_repository",
		"repository": repoName,
	}).Info("Disabled repository from polling")

	return nil
}

// GetRepositoryStats returns statistics for a specific repository
func (s *SchedulerImpl) GetRepositoryStats(repoName string) (*RepositoryScheduleStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheduledRepo, exists := s.repositories[repoName]
	if !exists {
		return nil, fmt.Errorf("repository %s is not scheduled", repoName)
	}

	var nextPollIn time.Duration
	if scheduledRepo.Enabled && !scheduledRepo.NextPollTime.IsZero() {
		nextPollIn = time.Until(scheduledRepo.NextPollTime)
		if nextPollIn < 0 {
			nextPollIn = 0
		}
	}

	return &RepositoryScheduleStats{
		Repository:   scheduledRepo.Repository.Name,
		Provider:     scheduledRepo.Repository.Provider,
		Enabled:      scheduledRepo.Enabled,
		PollCount:    scheduledRepo.PollCount,
		LastPollTime: scheduledRepo.LastPollTime,
		NextPollTime: scheduledRepo.NextPollTime,
		NextPollIn:   nextPollIn,
		Interval:     s.config.Interval,
	}, nil
}

// RepositoryScheduleStats represents statistics for a scheduled repository
type RepositoryScheduleStats struct {
	Repository   string        `json:"repository"`
	Provider     string        `json:"provider"`
	Enabled      bool          `json:"enabled"`
	PollCount    int64         `json:"poll_count"`
	LastPollTime time.Time     `json:"last_poll_time,omitempty"`
	NextPollTime time.Time     `json:"next_poll_time,omitempty"`
	NextPollIn   time.Duration `json:"next_poll_in"`
	Interval     time.Duration `json:"interval"`
}
