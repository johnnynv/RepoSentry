package logger

import (
	"context"
	"time"
)

// BusinessLogger defines business-specific logging operations
type BusinessLogger interface {
	// Repository operations
	LogRepositoryPollStart(ctx context.Context, repository, provider, url string)
	LogRepositoryPollSuccess(ctx context.Context, repository string, changeCount int, duration time.Duration)
	LogRepositoryPollError(ctx context.Context, repository string, err error, duration time.Duration)

	// Branch operations
	LogBranchChange(ctx context.Context, repository, branch, changeType, oldCommit, newCommit string, protected bool)
	LogBranchChangesDetected(ctx context.Context, repository string, changeCount int)

	// Event operations
	LogEventGeneration(ctx context.Context, repository string, eventCount int, duration time.Duration)
	LogEventGenerationError(ctx context.Context, repository string, err error)
	LogEventCreated(ctx context.Context, eventID, repository, branch, changeType string)

	// Trigger operations
	LogTriggerAttempt(ctx context.Context, eventID, repository string)
	LogTriggerSuccess(ctx context.Context, eventID, repository string, statusCode int, duration time.Duration)
	LogTriggerError(ctx context.Context, eventID, repository string, err error, statusCode int)

	// API operations
	LogAPIRequest(ctx context.Context, method, path, userAgent, remoteAddr string)
	LogAPIResponse(ctx context.Context, method, path string, statusCode int, duration time.Duration)
	LogAPIError(ctx context.Context, method, path string, err error, statusCode int)

	// System operations
	LogComponentStart(ctx context.Context, component, module string, config interface{})
	LogComponentStop(ctx context.Context, component, module string, duration time.Duration)
	LogComponentError(ctx context.Context, component, module string, err error)
	LogComponentHealth(ctx context.Context, component string, healthy bool, checks map[string]string)
}

// businessLoggerImpl implements BusinessLogger
type businessLoggerImpl struct {
	manager *Manager
}

// NewBusinessLogger creates a new business logger
func NewBusinessLogger(manager *Manager) BusinessLogger {
	return &businessLoggerImpl{
		manager: manager,
	}
}

// Repository operations
func (bl *businessLoggerImpl) LogRepositoryPollStart(ctx context.Context, repository, provider, url string) {
	op := bl.manager.StartOperation(ctx, "poller", "repository", "poll_start")
	op.WithRepository(repository, provider).Info("Starting repository poll", Fields{
		"url": url,
	})
}

func (bl *businessLoggerImpl) LogRepositoryPollSuccess(ctx context.Context, repository string, changeCount int, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":    "poller",
		"module":       "repository",
		"operation":    "poll_complete",
		"repository":   repository,
		"change_count": changeCount,
		"duration":     duration,
		"duration_ms":  duration.Milliseconds(),
		"success":      true,
	}).Info("Repository poll completed successfully")
}

func (bl *businessLoggerImpl) LogRepositoryPollError(ctx context.Context, repository string, err error, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "poller",
		"module":      "repository",
		"operation":   "poll_complete",
		"repository":  repository,
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
		"success":     false,
		"error":       err.Error(),
	}).Error("Repository poll failed")
}

// Branch operations
func (bl *businessLoggerImpl) LogBranchChange(ctx context.Context, repository, branch, changeType, oldCommit, newCommit string, protected bool) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "poller",
		"module":      "branch_monitor",
		"operation":   "detect_change",
		"repository":  repository,
		"branch":      branch,
		"change_type": changeType,
		"old_commit":  oldCommit,
		"new_commit":  newCommit,
		"protected":   protected,
	}).Info("Branch change detected")
}

func (bl *businessLoggerImpl) LogBranchChangesDetected(ctx context.Context, repository string, changeCount int) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":    "poller",
		"module":       "branch_monitor",
		"operation":    "detect_changes",
		"repository":   repository,
		"change_count": changeCount,
	}).Info("Branch changes detected")
}

// Event operations
func (bl *businessLoggerImpl) LogEventGeneration(ctx context.Context, repository string, eventCount int, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "poller",
		"module":      "event_generator",
		"operation":   "generate_events",
		"repository":  repository,
		"event_count": eventCount,
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
		"success":     true,
	}).Info("Events generated successfully")
}

func (bl *businessLoggerImpl) LogEventGenerationError(ctx context.Context, repository string, err error) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":  "poller",
		"module":     "event_generator",
		"operation":  "generate_events",
		"repository": repository,
		"success":    false,
		"error":      err.Error(),
	}).Error("Event generation failed")
}

func (bl *businessLoggerImpl) LogEventCreated(ctx context.Context, eventID, repository, branch, changeType string) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "poller",
		"module":      "event_generator",
		"operation":   "create_event",
		"event_id":    eventID,
		"repository":  repository,
		"branch":      branch,
		"change_type": changeType,
	}).Info("Event created")
}

// Trigger operations
func (bl *businessLoggerImpl) LogTriggerAttempt(ctx context.Context, eventID, repository string) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":  "trigger",
		"module":     "tekton",
		"operation":  "send_event",
		"event_id":   eventID,
		"repository": repository,
	}).Info("Attempting to trigger event")
}

func (bl *businessLoggerImpl) LogTriggerSuccess(ctx context.Context, eventID, repository string, statusCode int, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "trigger",
		"module":      "tekton",
		"operation":   "send_event",
		"event_id":    eventID,
		"repository":  repository,
		"status_code": statusCode,
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
		"success":     true,
	}).Info("Event triggered successfully")
}

func (bl *businessLoggerImpl) LogTriggerError(ctx context.Context, eventID, repository string, err error, statusCode int) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "trigger",
		"module":      "tekton",
		"operation":   "send_event",
		"event_id":    eventID,
		"repository":  repository,
		"status_code": statusCode,
		"success":     false,
		"error":       err.Error(),
	}).Error("Event trigger failed")
}

// API operations
func (bl *businessLoggerImpl) LogAPIRequest(ctx context.Context, method, path, userAgent, remoteAddr string) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "api",
		"module":      "server",
		"operation":   "handle_request",
		"method":      method,
		"path":        path,
		"user_agent":  userAgent,
		"remote_addr": remoteAddr,
	}).Debug("API request received")
}

func (bl *businessLoggerImpl) LogAPIResponse(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "api",
		"module":      "server",
		"operation":   "handle_request",
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
	}).Info("API request completed")
}

func (bl *businessLoggerImpl) LogAPIError(ctx context.Context, method, path string, err error, statusCode int) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component":   "api",
		"module":      "server",
		"operation":   "handle_request",
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"error":       err.Error(),
	}).Error("API request failed")
}

// System operations
func (bl *businessLoggerImpl) LogComponentStart(ctx context.Context, component, module string, config interface{}) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component": component,
		"module":    module,
		"operation": "start",
		"config":    config,
	}).Info("Component starting")
}

func (bl *businessLoggerImpl) LogComponentStop(ctx context.Context, component, module string, duration time.Duration) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component": component,
		"module":    module,
		"operation": "stop",
		"duration":  duration,
		"uptime":    duration,
	}).Info("Component stopped")
}

func (bl *businessLoggerImpl) LogComponentError(ctx context.Context, component, module string, err error) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component": component,
		"module":    module,
		"operation": "error",
		"error":     err.Error(),
	}).Error("Component error occurred")
}

func (bl *businessLoggerImpl) LogComponentHealth(ctx context.Context, component string, healthy bool, checks map[string]string) {
	logger := bl.manager.WithGoContext(ctx)
	logger.WithFields(Fields{
		"component": component,
		"operation": "health_check",
		"healthy":   healthy,
		"checks":    checks,
	}).Info("Component health check")
}
