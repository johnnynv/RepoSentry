package runtime

import (
	"fmt"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// DefaultRuntimeFactory implements RuntimeFactory
type DefaultRuntimeFactory struct{}

// NewDefaultRuntimeFactory creates a new DefaultRuntimeFactory
func NewDefaultRuntimeFactory() *DefaultRuntimeFactory {
	return &DefaultRuntimeFactory{}
}

// CreateRuntime implements RuntimeFactory.CreateRuntime
func (f *DefaultRuntimeFactory) CreateRuntime(config *types.Config) (Runtime, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if err := validateRuntimeConfig(config); err != nil {
		return nil, fmt.Errorf("invalid runtime configuration: %w", err)
	}

	// Create runtime manager
	runtime, err := NewRuntimeManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime manager: %w", err)
	}

	return runtime, nil
}

// validateRuntimeConfig validates the runtime-specific configuration
func validateRuntimeConfig(config *types.Config) error {
	if config.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}

	if config.App.DataDir == "" {
		return fmt.Errorf("app.data_dir is required")
	}

	// Validate health check port if specified
	if config.App.HealthCheckPort < 0 || config.App.HealthCheckPort > 65535 {
		return fmt.Errorf("app.health_check_port must be between 0 and 65535")
	}

	// Validate polling configuration
	if len(config.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be configured")
	}

	return nil
}
