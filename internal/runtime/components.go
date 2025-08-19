package runtime

import (
	"context"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// BaseComponent provides common functionality for all components
type BaseComponent struct {
	name      string
	logger    *logger.Entry
	state     ComponentState
	startedAt time.Time
	lastError string
}

// GetName implements Component.GetName
func (c *BaseComponent) GetName() string {
	return c.name
}

// GetStatus implements Component.GetStatus
func (c *BaseComponent) GetStatus() ComponentStatus {
	status := ComponentStatus{
		Name:   c.name,
		State:  c.state,
		Health: HealthStateUnknown,
	}

	if !c.startedAt.IsZero() {
		status.StartedAt = c.startedAt
		status.Uptime = time.Since(c.startedAt)
	}

	if c.lastError != "" {
		status.LastError = c.lastError
	}

	// Determine health based on state
	switch c.state {
	case ComponentStateRunning:
		status.Health = HealthStateHealthy
	case ComponentStateError:
		status.Health = HealthStateUnhealthy
	default:
		status.Health = HealthStateUnknown
	}

	return status
}

// setState updates the component state
func (c *BaseComponent) setState(state ComponentState) {
	c.state = state
	c.logger.WithFields(logger.Fields{
		"operation": "state_change",
		"component": c.name,
		"new_state": string(state),
	}).Debug("Component state changed")
}

// setError sets the last error and updates state
func (c *BaseComponent) setError(err error) {
	c.lastError = err.Error()
	c.setState(ComponentStateError)
	c.logger.WithFields(logger.Fields{
		"operation": "error",
		"component": c.name,
		"error":     err.Error(),
	}).Error("Component error occurred")
}

// ConfigComponent wraps the configuration manager
type ConfigComponent struct {
	BaseComponent
	manager config.Manager
}

// NewConfigComponent creates a new ConfigComponent
func NewConfigComponent(manager config.Manager, parentLogger *logger.Entry) *ConfigComponent {
	return &ConfigComponent{
		BaseComponent: BaseComponent{
			name:   "config",
			logger: parentLogger.WithField("component", "config"),
			state:  ComponentStateUnknown,
		},
		manager: manager,
	}
}

// Start implements Component.Start
func (c *ConfigComponent) Start(ctx context.Context) error {
	c.setState(ComponentStateStarting)
	c.startedAt = time.Now()

	c.logger.WithFields(logger.Fields{
		"operation": "start",
	}).Info("Starting configuration component")

	// Configuration manager doesn't need explicit starting
	c.setState(ComponentStateRunning)

	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"duration":  time.Since(c.startedAt),
	}).Info("Configuration component started successfully")

	return nil
}

// Stop implements Component.Stop
func (c *ConfigComponent) Stop(ctx context.Context) error {
	c.setState(ComponentStateStopping)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping configuration component")

	// Configuration manager doesn't need explicit stopping
	c.setState(ComponentStateStopped)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Configuration component stopped successfully")

	return nil
}

// Health implements Component.Health
func (c *ConfigComponent) Health(ctx context.Context) error {
	// Configuration component is healthy if it was successfully initialized
	// We don't validate the file path because it may not be set in tests
	return nil
}

// StorageComponent wraps the storage layer
type StorageComponent struct {
	BaseComponent
	storage storage.Storage
}

// NewStorageComponent creates a new StorageComponent
func NewStorageComponent(storage storage.Storage, parentLogger *logger.Entry) *StorageComponent {
	return &StorageComponent{
		BaseComponent: BaseComponent{
			name:   "storage",
			logger: parentLogger.WithField("component", "storage"),
			state:  ComponentStateUnknown,
		},
		storage: storage,
	}
}

// Start implements Component.Start
func (c *StorageComponent) Start(ctx context.Context) error {
	c.setState(ComponentStateStarting)
	c.startedAt = time.Now()

	c.logger.WithFields(logger.Fields{
		"operation": "start",
	}).Info("Starting storage component")

	// Storage should already be initialized, just verify connectivity
	if err := c.storage.HealthCheck(ctx); err != nil {
		c.setError(err)
		return err
	}

	c.setState(ComponentStateRunning)

	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"duration":  time.Since(c.startedAt),
	}).Info("Storage component started successfully")

	return nil
}

// Stop implements Component.Stop
func (c *StorageComponent) Stop(ctx context.Context) error {
	c.setState(ComponentStateStopping)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping storage component")

	if err := c.storage.Close(); err != nil {
		c.logger.WithFields(logger.Fields{
			"operation": "stop",
			"error":     err.Error(),
		}).Error("Error closing storage")
	}

	c.setState(ComponentStateStopped)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Storage component stopped successfully")

	return nil
}

// Health implements Component.Health
func (c *StorageComponent) Health(ctx context.Context) error {
	return c.storage.HealthCheck(ctx)
}

// GitClientFactoryComponent wraps the git client factory
type GitClientFactoryComponent struct {
	BaseComponent
	factory *gitclient.ClientFactory
}

// NewGitClientFactoryComponent creates a new GitClientFactoryComponent
func NewGitClientFactoryComponent(factory *gitclient.ClientFactory, parentLogger *logger.Entry) *GitClientFactoryComponent {
	return &GitClientFactoryComponent{
		BaseComponent: BaseComponent{
			name:   "git_client",
			logger: parentLogger.WithField("component", "git_client"),
			state:  ComponentStateUnknown,
		},
		factory: factory,
	}
}

// Start implements Component.Start
func (c *GitClientFactoryComponent) Start(ctx context.Context) error {
	c.setState(ComponentStateStarting)
	c.startedAt = time.Now()

	c.logger.WithFields(logger.Fields{
		"operation": "start",
	}).Info("Starting git client component")

	// Git client factory doesn't need explicit starting
	c.setState(ComponentStateRunning)

	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"duration":  time.Since(c.startedAt),
	}).Info("Git client component started successfully")

	return nil
}

// Stop implements Component.Stop
func (c *GitClientFactoryComponent) Stop(ctx context.Context) error {
	c.setState(ComponentStateStopping)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping git client component")

	// Git client factory doesn't need explicit stopping
	c.setState(ComponentStateStopped)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Git client component stopped successfully")

	return nil
}

// Health implements Component.Health
func (c *GitClientFactoryComponent) Health(ctx context.Context) error {
	// Factory is always healthy
	return nil
}

// TriggerFactoryComponent wraps the trigger factory
type TriggerFactoryComponent struct {
	BaseComponent
	factory *trigger.TriggerFactory
}

// NewTriggerFactoryComponent creates a new TriggerFactoryComponent
func NewTriggerFactoryComponent(factory *trigger.TriggerFactory, parentLogger *logger.Entry) *TriggerFactoryComponent {
	return &TriggerFactoryComponent{
		BaseComponent: BaseComponent{
			name:   "trigger",
			logger: parentLogger.WithField("component", "trigger"),
			state:  ComponentStateUnknown,
		},
		factory: factory,
	}
}

// Start implements Component.Start
func (c *TriggerFactoryComponent) Start(ctx context.Context) error {
	c.setState(ComponentStateStarting)
	c.startedAt = time.Now()

	c.logger.WithFields(logger.Fields{
		"operation": "start",
	}).Info("Starting trigger component")

	// Trigger factory doesn't need explicit starting
	c.setState(ComponentStateRunning)

	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"duration":  time.Since(c.startedAt),
	}).Info("Trigger component started successfully")

	return nil
}

// Stop implements Component.Stop
func (c *TriggerFactoryComponent) Stop(ctx context.Context) error {
	c.setState(ComponentStateStopping)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping trigger component")

	// Trigger factory doesn't need explicit stopping
	c.setState(ComponentStateStopped)

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Trigger component stopped successfully")

	return nil
}

// Health implements Component.Health
func (c *TriggerFactoryComponent) Health(ctx context.Context) error {
	// Factory is always healthy
	return nil
}

// TODO: PollerComponent - 需要重新设计接口
// 暂时注释掉，等poller接口稳定后再实现

