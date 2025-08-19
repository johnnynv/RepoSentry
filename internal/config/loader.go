package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/johnnynv/RepoSentry/pkg/types"
	"github.com/johnnynv/RepoSentry/pkg/utils"
)

// Loader handles configuration loading and processing
type Loader struct {
	envExpander *utils.EnvExpander
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromFile loads configuration from a YAML file
func (l *Loader) LoadFromFile(filePath string) (*types.Config, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Open and read file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	return l.LoadFromReader(file)
}

// LoadFromReader loads configuration from an io.Reader
func (l *Loader) LoadFromReader(reader io.Reader) (*types.Config, error) {
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	return l.LoadFromBytes(content)
}

// LoadFromBytes loads configuration from byte slice
func (l *Loader) LoadFromBytes(content []byte) (*types.Config, error) {
	// Parse YAML into raw map first
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(content, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Get security config first to set up environment variable expansion
	securityConfig, err := l.extractSecurityConfig(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security configuration: %w", err)
	}

	// Create environment expander with allowed variables
	l.envExpander = utils.NewEnvExpander(securityConfig.AllowedEnvVars)

	// Expand environment variables
	expandedConfig, err := l.envExpander.ExpandMap(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	// Marshal back to YAML and unmarshal into typed structure
	expandedBytes, err := yaml.Marshal(expandedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expanded configuration: %w", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(expandedBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Apply defaults
	l.applyDefaults(&config)

	return &config, nil
}

// extractSecurityConfig extracts security configuration before environment expansion
func (l *Loader) extractSecurityConfig(rawConfig map[string]interface{}) (*types.SecurityConfig, error) {
	// Default security config
	securityConfig := &types.SecurityConfig{
		AllowedEnvVars: []string{
			"GITHUB_TOKEN",
			"GITLAB_TOKEN",
			"GHE_TOKEN",
			"*_TOKEN",
		},
		RequireHTTPS: true,
	}

	// Override with configuration if present
	if securityRaw, exists := rawConfig["security"]; exists {
		securityBytes, err := yaml.Marshal(securityRaw)
		if err != nil {
			return nil, err
		}
		
		if err := yaml.Unmarshal(securityBytes, securityConfig); err != nil {
			return nil, err
		}
	}

	return securityConfig, nil
}

// applyDefaults applies default values to configuration
func (l *Loader) applyDefaults(config *types.Config) {
	// App defaults
	if config.App.Name == "" {
		config.App.Name = "reposentry"
	}
	if config.App.LogLevel == "" {
		config.App.LogLevel = "info"
	}
	if config.App.LogFormat == "" {
		config.App.LogFormat = "json"
	}
	if config.App.HealthCheckPort == 0 {
		config.App.HealthCheckPort = 8080
	}
	if config.App.DataDir == "" {
		config.App.DataDir = "./data"
	}

	// Polling defaults
	if config.Polling.Interval == 0 {
		config.Polling.Interval = 5 * time.Minute
	}
	if config.Polling.Timeout == 0 {
		config.Polling.Timeout = 30 * time.Second
	}
	if config.Polling.MaxWorkers == 0 {
		config.Polling.MaxWorkers = 5
	}
	if config.Polling.BatchSize == 0 {
		config.Polling.BatchSize = 10
	}

	// Storage defaults
	if config.Storage.Type == "" {
		config.Storage.Type = "sqlite"
	}
	if config.Storage.SQLite.Path == "" {
		config.Storage.SQLite.Path = filepath.Join(config.App.DataDir, "reposentry.db")
	}
	if config.Storage.SQLite.MaxConnections == 0 {
		config.Storage.SQLite.MaxConnections = 10
	}
	if config.Storage.SQLite.ConnectionTimeout == 0 {
		config.Storage.SQLite.ConnectionTimeout = 30 * time.Second
	}

	// Tekton defaults
	if config.Tekton.Timeout == 0 {
		config.Tekton.Timeout = 10 * time.Second
	}
	if config.Tekton.RetryAttempts == 0 {
		config.Tekton.RetryAttempts = 3
	}
	if config.Tekton.RetryBackoff == 0 {
		config.Tekton.RetryBackoff = 1 * time.Second
	}
	if config.Tekton.Headers == nil {
		config.Tekton.Headers = make(map[string]string)
	}
	if config.Tekton.Headers["Content-Type"] == "" {
		config.Tekton.Headers["Content-Type"] = "application/json"
	}
	if config.Tekton.Headers["X-Source"] == "" {
		config.Tekton.Headers["X-Source"] = "reposentry"
	}

	// Rate limit defaults
	if config.RateLimit.GitHub.RequestsPerHour == 0 {
		config.RateLimit.GitHub.RequestsPerHour = 4000
	}
	if config.RateLimit.GitHub.Burst == 0 {
		config.RateLimit.GitHub.Burst = 10
	}
	if config.RateLimit.GitLab.RequestsPerSecond == 0 {
		config.RateLimit.GitLab.RequestsPerSecond = 8
	}
	if config.RateLimit.GitLab.Burst == 0 {
		config.RateLimit.GitLab.Burst = 5
	}

	// Security defaults
	if len(config.Security.AllowedEnvVars) == 0 {
		config.Security.AllowedEnvVars = []string{
			"GITHUB_TOKEN",
			"GITLAB_TOKEN",
			"GHE_TOKEN",
			"*_TOKEN",
		}
	}

	// Repository defaults
	for i := range config.Repositories {
		repo := &config.Repositories[i]
		if repo.PollingInterval == 0 {
			repo.PollingInterval = config.Polling.Interval
		}
		if repo.Enabled == false && repo.Name != "" {
			// Default to enabled if not explicitly set
			repo.Enabled = true
		}
	}
}

// LoadWithDefaults loads configuration with fallback to defaults
func (l *Loader) LoadWithDefaults(filePath string) (*types.Config, error) {
	// Try to load from file first
	if filePath != "" {
		if config, err := l.LoadFromFile(filePath); err == nil {
			return config, nil
		}
	}

	// Fallback to minimal default configuration
	defaultConfig := &types.Config{}
	l.applyDefaults(defaultConfig)
	
	return defaultConfig, nil
}

// Validate validates loaded configuration
func (l *Loader) Validate(config *types.Config) error {
	validator := NewValidator()
	return validator.Validate(config)
}
