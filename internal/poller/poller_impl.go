package poller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/internal/tekton"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// PollerImpl implements the Poller interface
type PollerImpl struct {
	config         PollerConfig
	storage        storage.Storage
	branchMonitor  BranchMonitor
	eventGenerator EventGenerator
	scheduler      Scheduler
	clientFactory  *gitclient.ClientFactory
	trigger        trigger.Trigger
	tektonManager  *tekton.TektonTriggerManager // Added Tekton integration
	logger         *logger.Entry

	// Runtime state
	mu        sync.RWMutex
	running   bool
	startTime time.Time
	stopChan  chan struct{}
	workQueue chan types.Repository
	workers   []*worker
	metrics   PollerMetrics
}

// worker represents a polling worker
type worker struct {
	id     int
	poller *PollerImpl
	logger *logger.Entry
}

// NewPoller creates a new poller instance
func NewPoller(config PollerConfig, storage storage.Storage, clientFactory *gitclient.ClientFactory, trigger trigger.Trigger, tektonManager *tekton.TektonTriggerManager, parentLogger *logger.Entry) *PollerImpl {
	branchMonitor := NewBranchMonitor(storage, clientFactory, parentLogger)
	eventGenerator := NewEventGenerator(parentLogger)
	scheduler := NewScheduler(config, parentLogger)

	poller := &PollerImpl{
		config:         config,
		storage:        storage,
		branchMonitor:  branchMonitor,
		eventGenerator: eventGenerator,
		scheduler:      scheduler,
		clientFactory:  clientFactory,
		trigger:        trigger,
		tektonManager:  tektonManager,
		logger: parentLogger.WithFields(logger.Fields{
			"component": "poller",
			"module":    "poller_impl",
		}),

		stopChan:  make(chan struct{}),
		workQueue: make(chan types.Repository, config.BatchSize*2), // Buffer for work queue
		metrics: PollerMetrics{
			LastResetTime: time.Now(),
		},
	}

	return poller
}

// Start begins the polling process
func (p *PollerImpl) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("poller is already running")
	}

	p.logger.WithFields(logger.Fields{
		"operation":   "start",
		"max_workers": p.config.MaxWorkers,
		"batch_size":  p.config.BatchSize,
		"interval":    p.config.Interval.String(),
		"timeout":     p.config.Timeout.String(),
	}).Info("Starting poller")

	p.running = true
	p.startTime = time.Now()

	// Start scheduler
	if err := p.scheduler.Start(ctx); err != nil {
		p.running = false
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Start workers
	p.workers = make([]*worker, p.config.MaxWorkers)
	for i := 0; i < p.config.MaxWorkers; i++ {
		p.workers[i] = &worker{
			id:     i + 1,
			poller: p,
			logger: p.logger.WithField("worker_id", i+1),
		}
		go p.workers[i].run(ctx)
	}

	// Start main polling loop
	go p.run(ctx)

	p.logger.Info("Poller started successfully")
	return nil
}

// Stop gracefully stops the polling process
func (p *PollerImpl) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping poller")

	p.running = false

	// Stop scheduler
	if err := p.scheduler.Stop(ctx); err != nil {
		p.logger.WithError(err).Error("Failed to stop scheduler")
	}

	// Signal stop to all workers
	close(p.stopChan)

	// Close work queue
	close(p.workQueue)

	p.logger.Info("Poller stopped")
	return nil
}

// PollRepository polls a specific repository once
func (p *PollerImpl) PollRepository(ctx context.Context, repo types.Repository) (*PollResult, error) {
	startTime := time.Now()

	p.logger.WithFields(logger.Fields{
		"operation":  "poll_repository",
		"repository": repo.Name,
		"provider":   repo.Provider,
	}).Info("Starting repository poll")

	result := &PollResult{
		Repository: repo,
		Timestamp:  startTime,
	}

	// Check for branch changes
	changes, err := p.branchMonitor.CheckBranches(ctx, repo)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Duration = time.Since(startTime)

		p.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "poll_repository",
			"repository": repo.Name,
			"duration":   result.Duration.String(),
		}).Error("Failed to check branches")

		p.updateMetrics(result)
		return result, err
	}

	result.Changes = changes
	result.BranchCount = len(changes)

	// Generate events from changes
	if len(changes) > 0 {
		events, err := p.eventGenerator.GenerateEvents(ctx, repo, changes)
		if err != nil {
			p.logger.WithError(err).WithFields(logger.Fields{
				"operation":    "poll_repository",
				"repository":   repo.Name,
				"change_count": len(changes),
			}).Error("Failed to generate events")
			// Don't fail the entire poll if event generation fails
		} else {
			result.Events = events

			// Store events in storage
			for _, event := range events {
				if err := p.storage.CreateEvent(ctx, event); err != nil {
					p.logger.WithError(err).WithFields(logger.Fields{
						"operation":  "poll_repository",
						"repository": repo.Name,
						"event_id":   event.ID,
					}).Error("Failed to store event")
				}
			}

			// Process with Tekton if available
			if p.tektonManager != nil {
				for _, event := range events {
					go func(e types.Event) {
						tektonCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()

						p.logger.WithFields(logger.Fields{
							"operation":  "tekton_process",
							"event_id":   e.ID,
							"repository": e.Repository,
							"branch":     e.Branch,
						}).Info("Processing repository change with Tekton")

						// Create Tekton process request
						request := &tekton.TektonProcessRequest{
							Repository: types.Repository{
								Name:     e.Repository,
								URL:      repo.URL,      // Use the original repo URL
								Provider: repo.Provider, // Use the original repo provider
							},
							CommitSHA: e.CommitSHA,
							Branch:    e.Branch,
						}

						tektonResult, err := p.tektonManager.ProcessRepositoryChange(tektonCtx, request)
						if err != nil {
							p.logger.WithError(err).WithFields(logger.Fields{
								"operation":  "tekton_process",
								"event_id":   e.ID,
								"repository": e.Repository,
							}).Error("Tekton processing failed")
						} else {
							p.logger.WithFields(logger.Fields{
								"operation":       "tekton_process",
								"event_id":        e.ID,
								"repository":      e.Repository,
								"detection":       tektonResult.Detection.EstimatedAction,
								"event_sent":      tektonResult.EventSent,
								"resources_found": len(tektonResult.Detection.Resources),
								"has_tekton_dir":  tektonResult.Detection.HasTektonDirectory,
							}).Info("Tekton processing completed")
						}
					}(event)
				}
			}

			// Fallback to regular trigger if no Tekton manager
			if p.tektonManager == nil && p.trigger != nil {
				for _, event := range events {
					go func(e types.Event) {
						triggerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()

						p.logger.WithFields(logger.Fields{
							"operation":  "auto_trigger",
							"event_id":   e.ID,
							"repository": e.Repository,
							"branch":     e.Branch,
						}).Info("Automatically triggering pipeline for event (fallback mode)")

						result, err := p.trigger.SendEvent(triggerCtx, e)
						if err != nil {
							p.logger.WithError(err).WithFields(logger.Fields{
								"operation":  "auto_trigger",
								"event_id":   e.ID,
								"repository": e.Repository,
							}).Error("Failed to trigger pipeline")
						} else if result.Success {
							p.logger.WithFields(logger.Fields{
								"operation":   "auto_trigger",
								"event_id":    e.ID,
								"repository":  e.Repository,
								"status_code": result.StatusCode,
								"duration":    result.Duration,
							}).Info("Successfully triggered pipeline")
						} else {
							p.logger.WithFields(logger.Fields{
								"operation":   "auto_trigger",
								"event_id":    e.ID,
								"repository":  e.Repository,
								"status_code": result.StatusCode,
								"error":       result.Error,
							}).Error("Pipeline trigger failed")
						}
					}(event)
				}
			}
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	p.logger.WithFields(logger.Fields{
		"operation":    "poll_repository",
		"repository":   repo.Name,
		"success":      result.Success,
		"change_count": len(result.Changes),
		"event_count":  len(result.Events),
		"duration":     result.Duration.String(),
	}).Info("Completed repository poll")

	p.updateMetrics(result)
	return result, nil
}

// GetStatus returns the current status of the poller
func (p *PollerImpl) GetStatus() PollerStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	schedulerStatus := p.scheduler.GetSchedulerStatus()

	var repositories []RepositoryStatus
	for _, scheduledRepo := range p.scheduler.GetScheduledRepositories() {
		repoStatus := RepositoryStatus{
			Name:         scheduledRepo.Repository.Name,
			Provider:     scheduledRepo.Repository.Provider,
			Enabled:      scheduledRepo.Enabled,
			LastPollTime: scheduledRepo.LastPollTime,
			NextPollTime: scheduledRepo.NextPollTime,
			PollCount:    scheduledRepo.PollCount,
			LastSuccess:  true, // TODO: Track success/failure per repository
		}
		repositories = append(repositories, repoStatus)
	}

	return PollerStatus{
		Running:            p.running,
		StartTime:          p.startTime,
		LastPollTime:       time.Now(), // TODO: Track actual last poll time
		ActiveRepositories: schedulerStatus.EnabledRepositories,
		WorkerCount:        len(p.workers),
		QueueSize:          len(p.workQueue),
		Repositories:       repositories,
	}
}

// GetMetrics returns polling metrics
func (p *PollerImpl) GetMetrics() PollerMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics := p.metrics
	if p.running {
		metrics.Uptime = time.Since(p.startTime)
	}

	return metrics
}

// GetScheduler returns the scheduler instance
func (p *PollerImpl) GetScheduler() Scheduler {
	return p.scheduler
}

// AddRepository adds a repository to the polling schedule
func (p *PollerImpl) AddRepository(repo types.Repository) error {
	if !repo.Enabled {
		p.logger.WithFields(logger.Fields{
			"operation":  "add_repository",
			"repository": repo.Name,
		}).Debug("Repository is disabled, not adding to schedule")
		return nil
	}

	if err := p.scheduler.Schedule(repo); err != nil {
		return fmt.Errorf("failed to schedule repository: %w", err)
	}

	p.logger.WithFields(logger.Fields{
		"operation":  "add_repository",
		"repository": repo.Name,
		"provider":   repo.Provider,
	}).Info("Added repository to polling schedule")

	return nil
}

// RemoveRepository removes a repository from the polling schedule
func (p *PollerImpl) RemoveRepository(repo types.Repository) error {
	if err := p.scheduler.Unschedule(repo); err != nil {
		return fmt.Errorf("failed to unschedule repository: %w", err)
	}

	p.logger.WithFields(logger.Fields{
		"operation":  "remove_repository",
		"repository": repo.Name,
	}).Info("Removed repository from polling schedule")

	return nil
}

// run is the main polling loop
func (p *PollerImpl) run(ctx context.Context) {
	p.logger.Info("Poller main loop started")

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Poller stopped due to context cancellation")
			return
		case <-p.stopChan:
			p.logger.Info("Poller stopped")
			return
		case <-ticker.C:
			p.processScheduledPolls(ctx)
		}
	}
}

// processScheduledPolls processes repositories that are ready for polling
func (p *PollerImpl) processScheduledPolls(ctx context.Context) {
	scheduledRepos := p.scheduler.GetScheduledRepositories()
	now := time.Now()

	var readyRepos []types.Repository
	for _, scheduledRepo := range scheduledRepos {
		if scheduledRepo.Enabled && now.After(scheduledRepo.NextPollTime) {
			readyRepos = append(readyRepos, scheduledRepo.Repository)
		}
	}

	if len(readyRepos) == 0 {
		return
	}

	p.logger.WithFields(logger.Fields{
		"operation":   "process_scheduled_polls",
		"ready_count": len(readyRepos),
	}).Debug("Processing scheduled polls")

	// Queue repositories for polling
	for _, repo := range readyRepos {
		select {
		case p.workQueue <- repo:
			// Successfully queued
		case <-ctx.Done():
			return
		default:
			p.logger.WithFields(logger.Fields{
				"operation":  "process_scheduled_polls",
				"repository": repo.Name,
			}).Warn("Work queue is full, skipping repository")
		}
	}
}

// updateMetrics updates polling metrics
func (p *PollerImpl) updateMetrics(result *PollResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalPolls++
	if result.Success {
		p.metrics.SuccessfulPolls++
	} else {
		p.metrics.FailedPolls++
	}

	p.metrics.TotalChanges += int64(len(result.Changes))
	p.metrics.TotalEvents += int64(len(result.Events))
	p.metrics.APICallCount++ // Assume each poll makes at least one API call

	if result.UsedFallback {
		p.metrics.FallbackCount++
	}

	// Update average duration
	if p.metrics.TotalPolls > 0 {
		totalDuration := p.metrics.AveragePollDuration * time.Duration(p.metrics.TotalPolls-1)
		totalDuration += result.Duration
		p.metrics.AveragePollDuration = totalDuration / time.Duration(p.metrics.TotalPolls)
	} else {
		p.metrics.AveragePollDuration = result.Duration
	}
}

// worker.run is the worker loop
func (w *worker) run(ctx context.Context) {
	w.logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker stopped due to context cancellation")
			return
		case <-w.poller.stopChan:
			w.logger.Info("Worker stopped")
			return
		case repo, ok := <-w.poller.workQueue:
			if !ok {
				w.logger.Info("Work queue closed, worker stopping")
				return
			}
			w.processRepository(ctx, repo)
		}
	}
}

// processRepository processes a single repository
func (w *worker) processRepository(ctx context.Context, repo types.Repository) {
	w.logger.WithFields(logger.Fields{
		"operation":  "process_repository",
		"repository": repo.Name,
		"worker_id":  w.id,
	}).Debug("Processing repository")

	pollCtx, cancel := context.WithTimeout(ctx, w.poller.config.Timeout)
	defer cancel()

	_, err := w.poller.PollRepository(pollCtx, repo)
	if err != nil {
		w.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "process_repository",
			"repository": repo.Name,
			"worker_id":  w.id,
		}).Error("Failed to poll repository")
	}
}
