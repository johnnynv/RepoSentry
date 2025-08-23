package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Validator validates configuration
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// Validate validates the entire configuration
func (v *Validator) Validate(config *types.Config) error {
	v.errors = v.errors[:0] // Reset errors

	v.validateApp(&config.App)
	v.validatePolling(&config.Polling)
	v.validateStorage(&config.Storage)
	v.validateTekton(&config.Tekton)
	v.validateRateLimit(&config.RateLimit)
	v.validateSecurity(&config.Security)
	v.validateRepositories(config.Repositories)

	if len(v.errors) > 0 {
		return v.errors
	}

	return nil
}

// validateApp validates app configuration
func (v *Validator) validateApp(app *types.AppConfig) {
	if app.Name == "" {
		v.addError("app.name", app.Name, "application name is required")
	}

	if app.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		if !v.contains(validLevels, app.LogLevel) {
			v.addError("app.log_level", app.LogLevel, "invalid log level")
		}
	}

	if app.LogFormat != "" {
		validFormats := []string{"json", "text"}
		if !v.contains(validFormats, app.LogFormat) {
			v.addError("app.log_format", app.LogFormat, "invalid log format")
		}
	}

	if app.HealthCheckPort <= 0 || app.HealthCheckPort > 65535 {
		v.addError("app.health_check_port", fmt.Sprintf("%d", app.HealthCheckPort), "invalid port number")
	}

	if app.DataDir == "" {
		v.addError("app.data_dir", app.DataDir, "data directory is required")
	}
}

// validatePolling validates polling configuration
func (v *Validator) validatePolling(polling *types.PollingConfig) {
	if polling.Interval <= 0 {
		v.addError("polling.interval", polling.Interval.String(), "polling interval must be positive")
	}

	if polling.Timeout <= 0 {
		v.addError("polling.timeout", polling.Timeout.String(), "polling timeout must be positive")
	}

	if polling.MaxWorkers <= 0 {
		v.addError("polling.max_workers", fmt.Sprintf("%d", polling.MaxWorkers), "max workers must be positive")
	}

	if polling.BatchSize <= 0 {
		v.addError("polling.batch_size", fmt.Sprintf("%d", polling.BatchSize), "batch size must be positive")
	}
}

// validateStorage validates storage configuration
func (v *Validator) validateStorage(storage *types.StorageConfig) {
	if storage.Type == "" {
		v.addError("storage.type", storage.Type, "storage type is required")
	}

	if storage.Type == "sqlite" {
		v.validateSQLite(&storage.SQLite)
	}
}

// validateSQLite validates SQLite configuration
func (v *Validator) validateSQLite(sqlite *types.SQLiteConfig) {
	if sqlite.Path == "" {
		v.addError("storage.sqlite.path", sqlite.Path, "SQLite database path is required")
	}

	if sqlite.MaxConnections <= 0 {
		v.addError("storage.sqlite.max_connections", fmt.Sprintf("%d", sqlite.MaxConnections), "max connections must be positive")
	}

	if sqlite.ConnectionTimeout <= 0 {
		v.addError("storage.sqlite.connection_timeout", sqlite.ConnectionTimeout.String(), "connection timeout must be positive")
	}
}

// validateTekton validates Tekton configuration
func (v *Validator) validateTekton(tekton *types.TektonConfig) {
	if tekton.EventListenerURL == "" {
		v.addError("tekton.event_listener_url", tekton.EventListenerURL, "Tekton EventListener URL is required")
	} else {
		if _, err := url.Parse(tekton.EventListenerURL); err != nil {
			v.addError("tekton.event_listener_url", tekton.EventListenerURL, "invalid URL format")
		}
	}

	if tekton.Timeout <= 0 {
		v.addError("tekton.timeout", tekton.Timeout.String(), "timeout must be positive")
	}

	if tekton.RetryAttempts < 0 {
		v.addError("tekton.retry_attempts", fmt.Sprintf("%d", tekton.RetryAttempts), "retry attempts cannot be negative")
	}

	if tekton.RetryBackoff <= 0 {
		v.addError("tekton.retry_backoff", tekton.RetryBackoff.String(), "retry backoff must be positive")
	}
}

// validateRateLimit validates rate limit configuration
func (v *Validator) validateRateLimit(rateLimit *types.RateLimitConfig) {
	if rateLimit.GitHub.RequestsPerHour <= 0 {
		v.addError("rate_limit.github.requests_per_hour", fmt.Sprintf("%d", rateLimit.GitHub.RequestsPerHour), "requests per hour must be positive")
	}

	if rateLimit.GitHub.Burst <= 0 {
		v.addError("rate_limit.github.burst", fmt.Sprintf("%d", rateLimit.GitHub.Burst), "burst must be positive")
	}

	if rateLimit.GitLab.RequestsPerSecond <= 0 {
		v.addError("rate_limit.gitlab.requests_per_second", fmt.Sprintf("%d", rateLimit.GitLab.RequestsPerSecond), "requests per second must be positive")
	}

	if rateLimit.GitLab.Burst <= 0 {
		v.addError("rate_limit.gitlab.burst", fmt.Sprintf("%d", rateLimit.GitLab.Burst), "burst must be positive")
	}
}

// validateSecurity validates security configuration
func (v *Validator) validateSecurity(security *types.SecurityConfig) {
	if len(security.AllowedEnvVars) == 0 {
		v.addError("security.allowed_env_vars", "[]", "at least one allowed environment variable is required")
	}
}

// validateRepositories validates repository configurations
func (v *Validator) validateRepositories(repositories []types.Repository) {
	if len(repositories) == 0 {
		v.addError("repositories", "[]", "at least one repository is required")
		return
	}

	names := make(map[string]bool)

	for i, repo := range repositories {
		prefix := fmt.Sprintf("repositories[%d]", i)

		// Validate unique names
		if repo.Name == "" {
			v.addError(prefix+".name", repo.Name, "repository name is required")
		} else if names[repo.Name] {
			v.addError(prefix+".name", repo.Name, "repository name must be unique")
		} else {
			names[repo.Name] = true
		}

		// Validate URL
		if repo.URL == "" {
			v.addError(prefix+".url", repo.URL, "repository URL is required")
		} else {
			if err := v.validateURL(repo.URL); err != nil {
				v.addError(prefix+".url", repo.URL, err.Error())
			}
		}

		// Validate provider
		validProviders := []string{"github", "gitlab"}
		if !v.contains(validProviders, repo.Provider) {
			v.addError(prefix+".provider", repo.Provider, "invalid provider, must be 'github' or 'gitlab'")
		}

		// Validate token (should be set or be an env var reference)
		if repo.Token == "" {
			v.addError(prefix+".token", repo.Token, "repository token is required")
		}

		// Validate branch regex
		if repo.BranchRegex == "" {
			v.addError(prefix+".branch_regex", repo.BranchRegex, "branch regex is required")
		} else {
			if _, err := regexp.Compile(repo.BranchRegex); err != nil {
				v.addError(prefix+".branch_regex", repo.BranchRegex, "invalid regular expression: "+err.Error())
			}
		}

		// Validate polling interval if set
		if repo.PollingInterval > 0 && repo.PollingInterval < time.Minute {
			v.addError(prefix+".polling_interval", repo.PollingInterval.String(),
				"polling interval cannot be less than 1 minute (to protect against API rate limits and avoid service abuse)")
		}

		// Validate API base URL if set
		if repo.APIBaseURL != "" {
			// Trim whitespace for robustness
			cleanAPIURL := strings.TrimSpace(repo.APIBaseURL)
			if _, err := url.Parse(cleanAPIURL); err != nil {
				v.addError(prefix+".api_base_url", repo.APIBaseURL, "invalid API base URL format")
			}
		}
	}
}

// validateURL validates repository URL format and security requirements
func (v *Validator) validateURL(repoURL string) error {
	// Trim whitespace for robustness
	repoURL = strings.TrimSpace(repoURL)

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("URL scheme must be http or https")
	}

	// Check host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// addError adds a validation error
func (v *Validator) addError(field, value, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// contains checks if a slice contains a string
func (v *Validator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
