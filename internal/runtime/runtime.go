package runtime

import (
	"context"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Runtime represents the main application runtime that orchestrates all components
type Runtime interface {
	// Start initializes and starts all components
	Start(ctx context.Context) error

	// Stop gracefully shuts down all components
	Stop(ctx context.Context) error

	// Health returns the health status of all components
	Health(ctx context.Context) (*HealthStatus, error)

	// GetStatus returns the current runtime status
	GetStatus() *RuntimeStatus

	// Reload reloads configuration and restarts affected components
	Reload(ctx context.Context) error
}

// RuntimeStatus represents the current status of the runtime
type RuntimeStatus struct {
	State      RuntimeState               `json:"state"`
	StartedAt  time.Time                  `json:"started_at"`
	Uptime     time.Duration              `json:"uptime"`
	Version    string                     `json:"version"`
	Components map[string]ComponentStatus `json:"components"`
}

// ComponentStatus represents the status of an individual component
type ComponentStatus struct {
	Name      string         `json:"name"`
	State     ComponentState `json:"state"`
	StartedAt time.Time      `json:"started_at,omitempty"`
	Uptime    time.Duration  `json:"uptime,omitempty"`
	Health    HealthState    `json:"health"`
	LastError string         `json:"last_error,omitempty"`
	Metrics   interface{}    `json:"metrics,omitempty"`
}

// HealthStatus represents overall health information
type HealthStatus struct {
	Status     HealthState            `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Components map[string]HealthState `json:"components"`
	Checks     []HealthCheck          `json:"checks"`
}

// HealthCheck represents an individual health check result
type HealthCheck struct {
	Name     string        `json:"name"`
	Status   HealthState   `json:"status"`
	Duration time.Duration `json:"duration"`
	Message  string        `json:"message,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// RuntimeState represents the state of the runtime
type RuntimeState string

const (
	RuntimeStateUnknown  RuntimeState = "unknown"
	RuntimeStateStarting RuntimeState = "starting"
	RuntimeStateRunning  RuntimeState = "running"
	RuntimeStateStopping RuntimeState = "stopping"
	RuntimeStateStopped  RuntimeState = "stopped"
	RuntimeStateError    RuntimeState = "error"
)

// ComponentState represents the state of a component
type ComponentState string

const (
	ComponentStateUnknown  ComponentState = "unknown"
	ComponentStateStarting ComponentState = "starting"
	ComponentStateRunning  ComponentState = "running"
	ComponentStateStopping ComponentState = "stopping"
	ComponentStateStopped  ComponentState = "stopped"
	ComponentStateError    ComponentState = "error"
)

// HealthState represents health status
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateUnhealthy HealthState = "unhealthy"
	HealthStateUnknown   HealthState = "unknown"
)

// Component represents a manageable component in the runtime
type Component interface {
	// GetName returns the component name
	GetName() string

	// Start starts the component
	Start(ctx context.Context) error

	// Stop stops the component
	Stop(ctx context.Context) error

	// Health checks component health
	Health(ctx context.Context) error

	// GetStatus returns component status
	GetStatus() ComponentStatus
}

// RuntimeConfig holds runtime-specific configuration
type RuntimeConfig struct {
	// HealthCheck configuration
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`

	// Graceful shutdown timeout
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`

	// Component startup timeout
	StartupTimeout time.Duration `yaml:"startup_timeout" json:"startup_timeout"`

	// Restart policy for failed components
	RestartPolicy RestartPolicy `yaml:"restart_policy" json:"restart_policy"`
}

// HealthCheckConfig configures health check behavior
type HealthCheckConfig struct {
	// Enabled controls whether health checks are enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Interval between health checks
	Interval time.Duration `yaml:"interval" json:"interval"`

	// Timeout for individual health checks
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// Port for health check HTTP server
	Port int `yaml:"port" json:"port"`

	// Path for health check endpoint
	Path string `yaml:"path" json:"path"`
}

// RestartPolicy defines how components should be restarted on failure
type RestartPolicy struct {
	// Enabled controls whether automatic restart is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MaxAttempts is the maximum number of restart attempts
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`

	// BackoffDuration is the initial backoff duration
	BackoffDuration time.Duration `yaml:"backoff_duration" json:"backoff_duration"`

	// MaxBackoffDuration is the maximum backoff duration
	MaxBackoffDuration time.Duration `yaml:"max_backoff_duration" json:"max_backoff_duration"`
}

// RuntimeFactory creates Runtime instances
type RuntimeFactory interface {
	// CreateRuntime creates a new Runtime instance
	CreateRuntime(config *types.Config) (Runtime, error)
}
