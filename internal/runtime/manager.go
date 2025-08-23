package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/internal/poller"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// RuntimeManager implements the Runtime interface
type RuntimeManager struct {
	config *types.Config

	// Enterprise logging system
	loggerManager  *logger.Manager
	businessLogger logger.BusinessLogger
	logger         *logger.Entry

	startedAt time.Time
	state     RuntimeState
	mu        sync.RWMutex

	// Core components
	configManager  *config.Manager
	storage        storage.Storage
	poller         poller.Poller
	triggerManager trigger.Trigger
	// healthServer removed - functionality moved to API server

	// Component management
	components     map[string]Component
	componentOrder []string // Start order

	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWg sync.WaitGroup
}

// NewRuntimeManager creates a new RuntimeManager with enterprise logging
func NewRuntimeManager(cfg *types.Config, loggerManager *logger.Manager) (*RuntimeManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if loggerManager == nil {
		return nil, fmt.Errorf("loggerManager cannot be nil")
	}

	// Create runtime-specific logger
	runtimeLogger := loggerManager.ForModule("runtime", "manager")

	// Create business logger for operations
	businessLogger := logger.NewBusinessLogger(loggerManager)

	ctx, cancel := context.WithCancel(context.Background())

	rm := &RuntimeManager{
		config:         cfg,
		loggerManager:  loggerManager,
		businessLogger: businessLogger,
		logger:         runtimeLogger,
		state:          RuntimeStateUnknown,
		ctx:            ctx,
		cancel:         cancel,
		components:     make(map[string]Component),
		componentOrder: []string{},
	}

	// Initialize components
	if err := rm.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return rm, nil
}

// initializeComponents initializes all components in dependency order
func (rm *RuntimeManager) initializeComponents() error {
	var err error

	// Use business logger for component lifecycle tracking
	componentCtx := logger.WithContext(rm.ctx, logger.LogContext{
		Component: "runtime",
		Module:    "manager",
		Operation: "initialize_components",
	})

	rm.businessLogger.LogComponentStart(componentCtx, "runtime", "manager", map[string]interface{}{
		"component_count": 6,
		"config_loaded":   rm.config != nil,
	})

	// 1. Configuration Manager
	rm.configManager = config.NewManager(rm.loggerManager.GetRootLogger())
	// Load the current configuration into the config manager
	// Note: In production, this should use the same config that was loaded in main
	// For now, we'll create a config manager that has the config data
	rm.configManager.SetConfig(rm.config)
	configComponent := NewConfigComponent(rm.configManager, rm.loggerManager.ForComponent("config"))
	rm.addComponent("config", configComponent)

	// 2. Storage
	rm.storage, err = storage.NewSQLiteStorage(&rm.config.Storage.SQLite)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	storageComponent := NewStorageComponent(rm.storage, rm.loggerManager.ForComponent("storage"))
	rm.addComponent("storage", storageComponent)

	// 3. Git Client Factory
	gitFactory := gitclient.NewClientFactory(rm.loggerManager.ForComponent("gitclient"))
	gitComponent := NewGitClientFactoryComponent(gitFactory, rm.loggerManager.ForComponent("gitclient"))
	rm.addComponent("git_client", gitComponent)

	// 4. Trigger Manager
	triggerFactory := trigger.NewTriggerFactory()
	triggerConfig := trigger.DefaultTriggerConfig()
	triggerConfig.Tekton.EventListenerURL = rm.config.Tekton.EventListenerURL
	triggerConfig.Tekton.Namespace = "default"
	triggerConfig.Timeout = rm.config.Tekton.Timeout

	rm.triggerManager, err = triggerFactory.Create(triggerConfig, rm.loggerManager.ForComponent("trigger"))
	if err != nil {
		return fmt.Errorf("failed to create trigger manager: %w", err)
	}

	triggerComponent := NewTriggerComponent(rm.triggerManager, rm.loggerManager.ForComponent("trigger"))
	rm.addComponent("trigger", triggerComponent)

	// 5. Poller Component
	pollerConfig := pollerConfigFromConfig(rm.config)
	rm.poller = poller.NewPoller(pollerConfig, rm.storage, gitFactory, rm.triggerManager, rm.loggerManager.ForComponent("poller"))
	pollerComponent := NewPollerComponent(rm.poller, rm.config.Repositories, rm.loggerManager.ForComponent("poller"))
	rm.addComponent("poller", pollerComponent)

	// 6. API Server (includes health endpoints)
	if rm.config.App.HealthCheckPort > 0 {
		apiComponent := NewAPIComponent(rm.configManager, rm.storage, rm.config.App.HealthCheckPort, rm, rm.loggerManager.ForComponent("api"))
		rm.addComponent("api_server", apiComponent)
	}

	rm.logger.WithFields(logger.Fields{
		"operation":       "initialize_components",
		"component_count": len(rm.components),
		"component_order": rm.componentOrder,
	}).Info("Successfully initialized all runtime components")

	return nil
}

// pollerConfigFromConfig converts types.Config to poller.PollerConfig
func pollerConfigFromConfig(config *types.Config) poller.PollerConfig {
	return poller.PollerConfig{
		Interval:       config.Polling.Interval,
		Timeout:        config.Polling.Timeout,
		MaxWorkers:     config.Polling.MaxWorkers,
		BatchSize:      config.Polling.BatchSize,
		EnableFallback: config.Polling.EnableAPIFallback,
		RetryAttempts:  config.Polling.RetryAttempts,
		RetryBackoff:   config.Polling.RetryBackoff,
	}
}

// addComponent adds a component to the runtime in startup order
func (rm *RuntimeManager) addComponent(name string, component Component) {
	rm.components[name] = component
	rm.componentOrder = append(rm.componentOrder, name)
}

// Start implements Runtime.Start
func (rm *RuntimeManager) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.state == RuntimeStateRunning {
		return fmt.Errorf("runtime is already running")
	}

	rm.logger.WithFields(logger.Fields{
		"operation":  "start",
		"components": len(rm.components),
	}).Info("Starting RepoSentry runtime")

	rm.state = RuntimeStateStarting
	rm.startedAt = time.Now()

	// Start components in order
	for _, name := range rm.componentOrder {
		component := rm.components[name]

		rm.logger.WithFields(logger.Fields{
			"operation": "start_component",
			"component": name,
		}).Info("Starting component")

		if err := component.Start(ctx); err != nil {
			rm.state = RuntimeStateError
			rm.logger.WithFields(logger.Fields{
				"operation": "start_component",
				"component": name,
				"error":     err.Error(),
			}).Error("Failed to start component")

			// Try to stop already started components
			rm.stopComponents(ctx)
			return fmt.Errorf("failed to start component %s: %w", name, err)
		}

		rm.logger.WithFields(logger.Fields{
			"operation": "start_component",
			"component": name,
		}).Info("Successfully started component")
	}

	rm.state = RuntimeStateRunning

	rm.logger.WithFields(logger.Fields{
		"operation":  "start",
		"duration":   time.Since(rm.startedAt),
		"components": len(rm.components),
	}).Info("Successfully started RepoSentry runtime")

	return nil
}

// Stop implements Runtime.Stop
func (rm *RuntimeManager) Stop(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.state == RuntimeStateStopped {
		return nil
	}

	rm.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping RepoSentry runtime")

	rm.state = RuntimeStateStopping

	// Stop components in reverse order
	rm.stopComponents(ctx)

	// Cancel context
	rm.cancel()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		rm.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		rm.logger.Info("All components stopped gracefully")
	case <-ctx.Done():
		rm.logger.Warn("Forced shutdown due to context cancellation")
	}

	rm.state = RuntimeStateStopped

	rm.logger.WithFields(logger.Fields{
		"operation": "stop",
		"uptime":    time.Since(rm.startedAt),
	}).Info("Successfully stopped RepoSentry runtime")

	return nil
}

// stopComponents stops all components in reverse order
func (rm *RuntimeManager) stopComponents(ctx context.Context) {
	// Stop in reverse order
	for i := len(rm.componentOrder) - 1; i >= 0; i-- {
		name := rm.componentOrder[i]
		component := rm.components[name]

		rm.logger.WithFields(logger.Fields{
			"operation": "stop_component",
			"component": name,
		}).Info("Stopping component")

		if err := component.Stop(ctx); err != nil {
			rm.logger.WithFields(logger.Fields{
				"operation": "stop_component",
				"component": name,
				"error":     err.Error(),
			}).Error("Failed to stop component")
		} else {
			rm.logger.WithFields(logger.Fields{
				"operation": "stop_component",
				"component": name,
			}).Info("Successfully stopped component")
		}
	}
}

// Health implements Runtime.Health
func (rm *RuntimeManager) Health(ctx context.Context) (*HealthStatus, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	healthStatus := &HealthStatus{
		Status:     HealthStateHealthy,
		Timestamp:  time.Now(),
		Components: make(map[string]HealthState),
		Checks:     []HealthCheck{},
	}

	// Check each component
	for name, component := range rm.components {
		start := time.Now()
		err := component.Health(ctx)
		duration := time.Since(start)

		check := HealthCheck{
			Name:     name,
			Duration: duration,
		}

		if err != nil {
			check.Status = HealthStateUnhealthy
			check.Error = err.Error()
			healthStatus.Status = HealthStateUnhealthy
			healthStatus.Components[name] = HealthStateUnhealthy
		} else {
			check.Status = HealthStateHealthy
			healthStatus.Components[name] = HealthStateHealthy
		}

		healthStatus.Checks = append(healthStatus.Checks, check)
	}

	return healthStatus, nil
}

// GetStatus implements Runtime.GetStatus
func (rm *RuntimeManager) GetStatus() *RuntimeStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	status := &RuntimeStatus{
		State:      rm.state,
		StartedAt:  rm.startedAt,
		Uptime:     time.Since(rm.startedAt),
		Version:    "dev", // TODO: inject from build
		Components: make(map[string]ComponentStatus),
	}

	for name, component := range rm.components {
		status.Components[name] = component.GetStatus()
	}

	return status
}

// Reload implements Runtime.Reload
func (rm *RuntimeManager) Reload(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.logger.WithFields(logger.Fields{
		"operation": "reload",
	}).Info("Reloading RepoSentry runtime configuration")

	// Reload configuration
	if err := rm.configManager.Reload(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// TODO: Implement selective component restart based on config changes
	rm.logger.WithFields(logger.Fields{
		"operation": "reload",
	}).Info("Configuration reloaded successfully")

	return nil
}

// GetConfig returns the current configuration
func (rm *RuntimeManager) GetConfig() *types.Config {
	return rm.config
}

// GetLogger returns the runtime logger
func (rm *RuntimeManager) GetLogger() *logger.Entry {
	return rm.logger
}
