package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/johnnynv/RepoSentry/internal/api"
	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// APIComponent wraps the API server
type APIComponent struct {
	BaseComponent
	server *api.Server
}

// NewAPIComponent creates a new API component
func NewAPIComponent(configManager *config.Manager, storage storage.Storage, port int, runtime Runtime, parentLogger *logger.Entry) *APIComponent {
	server := api.NewServer(port, configManager, storage)
	adapter := newRuntimeAPIAdapter(runtime)
	server.SetRuntime(adapter) // Set runtime adapter for health checks
	
	return &APIComponent{
		BaseComponent: BaseComponent{
			name:   "api_server",
			logger: parentLogger.WithField("component", "api_server"),
		},
		server: server,
	}
}

// Start implements Component.Start
func (c *APIComponent) Start(ctx context.Context) error {
	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"module":    "api_server",
	}).Info("Starting API server component")

	startTime := time.Now()
	
	if err := c.server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	c.state = ComponentStateRunning
	c.startedAt = time.Now()
	duration := time.Since(startTime)

	c.logger.WithFields(logger.Fields{
		"operation": "start",
		"module":    "api_server",
		"duration":  duration,
	}).Info("API server component started successfully")

	return nil
}

// Stop implements Component.Stop
func (c *APIComponent) Stop(ctx context.Context) error {
	c.logger.WithFields(logger.Fields{
		"operation": "stop",
		"module":    "api_server",
	}).Info("Stopping API server component")

	if err := c.server.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop API server: %w", err)
	}

	c.state = ComponentStateStopped

	c.logger.WithFields(logger.Fields{
		"operation": "stop",
		"module":    "api_server",
	}).Info("API server component stopped successfully")

	return nil
}

// Health implements Component.Health
func (c *APIComponent) Health(ctx context.Context) error {
	return c.server.Health(ctx)
}

// GetServer returns the underlying API server
func (c *APIComponent) GetServer() *api.Server {
	return c.server
}
