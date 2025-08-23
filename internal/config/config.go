package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Manager manages application configuration
type Manager struct {
	config    *types.Config
	loader    *Loader
	validator *Validator
	logger    *logger.Logger
	mu        sync.RWMutex

	// Configuration file path for hot reload
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(logger *logger.Logger) *Manager {
	return &Manager{
		loader:    NewLoader(),
		validator: NewValidator(),
		logger:    logger,
	}
}

// Load loads configuration from file with validation
func (m *Manager) Load(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.WithComponent("config").
		WithField("path", configPath).
		Info("Loading configuration")

	// Load configuration with repositories support
	config, err := m.loader.LoadWithDefaults(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := m.validator.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Ensure data directory exists
	if err := m.ensureDataDirectory(config.App.DataDir); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	m.config = config
	m.configPath = configPath

	m.logger.WithComponent("config").
		WithField("repositories", len(config.Repositories)).
		Info("Configuration loaded successfully")

	return nil
}

// LoadWithDefaults loads configuration with fallback to defaults
func (m *Manager) LoadWithDefaults(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.WithComponent("config").
		WithField("path", configPath).
		Info("Loading configuration with defaults")

	// Load with defaults
	config, err := m.loader.LoadWithDefaults(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration with defaults: %w", err)
	}

	// Validate configuration
	if err := m.validator.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Ensure data directory exists
	if err := m.ensureDataDirectory(config.App.DataDir); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	m.config = config
	m.configPath = configPath

	m.logger.WithComponent("config").
		WithField("repositories", len(config.Repositories)).
		Info("Configuration loaded with defaults")

	return nil
}

// Get returns the current configuration (thread-safe)
func (m *Manager) Get() *types.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil
	}

	// Return a copy to prevent external modifications
	configCopy := *m.config
	return &configCopy
}

// GetRepositories returns enabled repositories
func (m *Manager) GetRepositories() []types.Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil
	}

	var enabled []types.Repository
	for _, repo := range m.config.Repositories {
		if repo.Enabled {
			enabled = append(enabled, repo)
		}
	}

	return enabled
}

// GetRepository returns a specific repository by name
func (m *Manager) GetRepository(name string) (*types.Repository, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, false
	}

	for _, repo := range m.config.Repositories {
		if repo.Name == name {
			return &repo, true
		}
	}

	return nil, false
}

// Reload reloads configuration from the same file
func (m *Manager) Reload() error {
	if m.configPath == "" {
		// No config file path set, this is not an error in test scenarios
		m.logger.WithComponent("config").
			Debug("No configuration file path set for reload")
		return nil
	}

	m.logger.WithComponent("config").
		Info("Reloading configuration")

	return m.Load(m.configPath)
}

// Validate validates a configuration without loading it
func (m *Manager) Validate(configPath string) error {
	config, err := m.loader.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration for validation: %w", err)
	}

	return m.validator.Validate(config)
}

// CheckPermissions checks if all required environment variables are set
func (m *Manager) CheckPermissions() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	var missingTokens []string

	for _, repo := range m.config.Repositories {
		if !repo.Enabled {
			continue
		}

		// Check if token is an environment variable reference
		if repo.Token == "" || (len(repo.Token) > 3 && repo.Token[:2] == "${" && repo.Token[len(repo.Token)-1:] == "}") {
			// Extract variable name and check if it's set
			if repo.Token == "" {
				missingTokens = append(missingTokens, fmt.Sprintf("repository '%s' has no token", repo.Name))
			} else {
				varName := repo.Token[2 : len(repo.Token)-1]
				if os.Getenv(varName) == "" {
					missingTokens = append(missingTokens, fmt.Sprintf("environment variable '%s' for repository '%s'", varName, repo.Name))
				}
			}
		}
	}

	if len(missingTokens) > 0 {
		return fmt.Errorf("missing required tokens: %v", missingTokens)
	}

	return nil
}

// GetConfigPath returns the current configuration file path
func (m *Manager) GetConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configPath
}

// SetConfig sets the configuration directly (for runtime initialization)
func (m *Manager) SetConfig(config *types.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	// Don't set configPath for programmatically set configs

	m.logger.WithComponent("config").
		WithField("repositories", len(config.Repositories)).
		Info("Configuration set programmatically")
}

// ensureDataDirectory creates the data directory if it doesn't exist
func (m *Manager) ensureDataDirectory(dataDir string) error {
	// Create absolute path
	absPath, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return err
	}

	// Verify it's writable
	testFile := filepath.Join(absPath, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("data directory is not writable: %w", err)
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// GetLoggerConfig returns logger configuration
func (m *Manager) GetLoggerConfig() logger.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return logger.Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}
	}

	// Use configuration file settings
	logConfig := logger.Config{
		Level:  m.config.App.LogLevel,
		Format: m.config.App.LogFormat,
		Output: "stdout", // Default to stdout
	}

	// Set file output if configured
	if m.config.App.LogFile != "" {
		logConfig.Output = m.config.App.LogFile
		logConfig.File = logger.FileConfig{
			MaxSize:    m.config.App.LogFileRotation.MaxSize,
			MaxBackups: m.config.App.LogFileRotation.MaxBackups,
			MaxAge:     m.config.App.LogFileRotation.MaxAge,
			Compress:   m.config.App.LogFileRotation.Compress,
		}
	}

	return logConfig
}
