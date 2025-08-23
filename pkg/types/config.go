package types

import (
	"time"
)

// Config represents the main application configuration
type Config struct {
	App                AppConfig       `yaml:"app" json:"app"`
	Polling            PollingConfig   `yaml:"polling" json:"polling"`
	Storage            StorageConfig   `yaml:"storage" json:"storage"`
	Tekton             TektonConfig    `yaml:"tekton" json:"tekton"`
	RateLimit          RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	Security           SecurityConfig  `yaml:"security" json:"security"`
	Repositories       []Repository    `yaml:"repositories,omitempty" json:"repositories,omitempty"`               // Legacy: repositories in main config
	RepositoriesConfig string          `yaml:"repositories_config,omitempty" json:"repositories_config,omitempty"` // New: path to repositories config file
}

// AppConfig represents application-level configuration
type AppConfig struct {
	Name            string        `yaml:"name" json:"name"`
	LogLevel        string        `yaml:"log_level" json:"log_level"`
	LogFormat       string        `yaml:"log_format" json:"log_format"`
	LogFile         string        `yaml:"log_file" json:"log_file,omitempty"`
	LogFileRotation LogFileConfig `yaml:"log_file_rotation" json:"log_file_rotation,omitempty"`
	HealthCheckPort int           `yaml:"health_check_port" json:"health_check_port"`
	DataDir         string        `yaml:"data_dir" json:"data_dir"`
}

// LogFileConfig represents log file rotation configuration
type LogFileConfig struct {
	MaxSize    int  `yaml:"max_size" json:"max_size"`       // MB
	MaxBackups int  `yaml:"max_backups" json:"max_backups"` // number of backup files
	MaxAge     int  `yaml:"max_age" json:"max_age"`         // days
	Compress   bool `yaml:"compress" json:"compress"`       // compress rotated files
}

// PollingConfig represents polling-related configuration
type PollingConfig struct {
	Interval          time.Duration `yaml:"interval" json:"interval"`
	Timeout           time.Duration `yaml:"timeout" json:"timeout"`
	MaxWorkers        int           `yaml:"max_workers" json:"max_workers"`
	BatchSize         int           `yaml:"batch_size" json:"batch_size"`
	EnableAPIFallback bool          `yaml:"enable_api_fallback" json:"enable_api_fallback"`
	RetryAttempts     int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryBackoff      time.Duration `yaml:"retry_backoff" json:"retry_backoff"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type   string       `yaml:"type" json:"type"`
	SQLite SQLiteConfig `yaml:"sqlite" json:"sqlite"`
}

// SQLiteConfig represents SQLite-specific configuration
type SQLiteConfig struct {
	Path              string        `yaml:"path" json:"path"`
	MaxConnections    int           `yaml:"max_connections" json:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
}

// TektonConfig represents Tekton EventListener configuration
type TektonConfig struct {
	EventListenerURL string            `yaml:"event_listener_url" json:"event_listener_url"`
	Timeout          time.Duration     `yaml:"timeout" json:"timeout"`
	RetryAttempts    int               `yaml:"retry_attempts" json:"retry_attempts"`
	RetryBackoff     time.Duration     `yaml:"retry_backoff" json:"retry_backoff"`
	Headers          map[string]string `yaml:"headers" json:"headers"`
}

// RateLimitConfig represents rate limiting configuration for different providers
type RateLimitConfig struct {
	GitHub GitHubRateLimit `yaml:"github" json:"github"`
	GitLab GitLabRateLimit `yaml:"gitlab" json:"gitlab"`
}

// GitHubRateLimit represents GitHub-specific rate limiting
type GitHubRateLimit struct {
	RequestsPerHour int `yaml:"requests_per_hour" json:"requests_per_hour"`
	Burst           int `yaml:"burst" json:"burst"`
}

// GitLabRateLimit represents GitLab-specific rate limiting
type GitLabRateLimit struct {
	RequestsPerSecond int `yaml:"requests_per_second" json:"requests_per_second"`
	Burst             int `yaml:"burst" json:"burst"`
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	AllowedEnvVars []string `yaml:"allowed_env_vars" json:"allowed_env_vars"`
	RequireHTTPS   bool     `yaml:"require_https" json:"require_https"`
}

// RepositoriesConfig represents a separate repositories configuration file
type RepositoriesConfig struct {
	Repositories   []Repository             `yaml:"repositories" json:"repositories"`
	GlobalSettings RepositoryGlobalSettings `yaml:"global_settings,omitempty" json:"global_settings,omitempty"`
}

// RepositoryGlobalSettings represents global repository settings
type RepositoryGlobalSettings struct {
	DefaultPollingEnabled   bool `yaml:"default_polling_enabled" json:"default_polling_enabled"`
	DefaultWebhookEnabled   bool `yaml:"default_webhook_enabled" json:"default_webhook_enabled"`
	DefaultBranchProtection bool `yaml:"default_branch_protection" json:"default_branch_protection"`
}
