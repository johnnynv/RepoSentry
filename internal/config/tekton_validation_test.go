package config

import (
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// createValidConfig creates a minimal valid configuration for testing
func createValidConfig() *types.Config {
	return &types.Config{
		App: types.AppConfig{
			Name:            "test-app",
			DataDir:         "/tmp/test",
			HealthCheckPort: 8080,
		},
		Polling: types.PollingConfig{
			Interval:   30 * time.Second,
			Timeout:    10 * time.Second,
			MaxWorkers: 2,
			BatchSize:  5,
		},
		Storage: types.StorageConfig{
			Type: "sqlite",
			SQLite: types.SQLiteConfig{
				Path:              "/tmp/test.db",
				MaxConnections:    10,
				ConnectionTimeout: 5 * time.Second,
			},
		},
		RateLimit: types.RateLimitConfig{
			GitHub: types.GitHubRateLimit{
				RequestsPerHour: 5000,
				Burst:           100,
			},
			GitLab: types.GitLabRateLimit{
				RequestsPerSecond: 10,
				Burst:             20,
			},
		},
		Security: types.SecurityConfig{
			AllowedEnvVars: []string{"HOME", "PATH"},
		},
		Repositories: []types.Repository{
			{
				Name:        "test-repo",
				URL:         "https://github.com/test/repo",
				Provider:    "github",
				Token:       "test-token",
				BranchRegex: "main|master",
				Enabled:     true,
			},
		},
	}
}

func TestValidator_ValidateTekton_Valid(t *testing.T) {
	validator := NewValidator()

	config := createValidConfig()
	config.Tekton = types.TektonConfig{
		EventListenerURL:  "http://localhost:8080/webhook",
		SystemNamespace:   "reposentry-system",
		BootstrapPipeline: "reposentry-bootstrap-pipeline",
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryBackoff:      5 * time.Second,
	}

	err := validator.Validate(config)
	if err != nil {
		t.Errorf("Expected no validation errors, got: %v", err)
	}
}

func TestValidator_ValidateTekton_InvalidFields(t *testing.T) {
	validator := NewValidator()

	config := createValidConfig()
	config.Tekton = types.TektonConfig{
		// Invalid fields should cause validation errors
		EventListenerURL: "invalid-url",
		Timeout:          -1 * time.Second,
		RetryBackoff:     -1 * time.Second,
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("Expected validation errors for invalid Tekton config, got none")
	}
}

func TestValidator_ValidateTekton_InvalidEventListenerURL(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid HTTP URL",
			url:         "http://localhost:8080/webhook",
			expectError: false,
		},
		{
			name:        "Valid HTTPS URL",
			url:         "https://tekton.example.com/webhook",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Invalid scheme",
			url:         "ftp://example.com/webhook",
			expectError: true,
		},
		{
			name:        "Empty URL (valid - optional field)",
			url:         "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewValidator()

			config := createValidConfig()
			config.Tekton = types.TektonConfig{

				EventListenerURL: tc.url,
				Timeout:          30 * time.Second,
				RetryBackoff:     5 * time.Second,
			}

			err := validator.Validate(config)
			if tc.expectError && err == nil {
				t.Errorf("Expected validation error for URL '%s', got none", tc.url)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error for URL '%s', got: %v", tc.url, err)
			}
		})
	}
}

func TestValidator_ValidateTekton_InvalidKubernetesNames(t *testing.T) {
	testCases := []struct {
		name        string
		namespace   string
		pipeline    string
		expectError bool
	}{
		{
			name:        "Valid names",
			namespace:   "reposentry-system",
			pipeline:    "bootstrap-pipeline",
			expectError: false,
		},
		{
			name:        "Valid single character",
			namespace:   "a",
			pipeline:    "b",
			expectError: false,
		},
		{
			name:        "Invalid uppercase",
			namespace:   "RepoSentry-System",
			pipeline:    "Bootstrap-Pipeline",
			expectError: true,
		},
		{
			name:        "Invalid starting with dash",
			namespace:   "-reposentry",
			pipeline:    "-pipeline",
			expectError: true,
		},
		{
			name:        "Invalid ending with dash",
			namespace:   "reposentry-",
			pipeline:    "pipeline-",
			expectError: true,
		},
		{
			name:        "Invalid special characters",
			namespace:   "repo_sentry",
			pipeline:    "pipeline.yaml",
			expectError: true,
		},
		{
			name:        "Empty names (valid - optional fields)",
			namespace:   "",
			pipeline:    "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewValidator()

			config := createValidConfig()
			config.Tekton = types.TektonConfig{

				SystemNamespace:   tc.namespace,
				BootstrapPipeline: tc.pipeline,
				Timeout:           30 * time.Second,
				RetryBackoff:      5 * time.Second,
			}

			err := validator.Validate(config)
			if tc.expectError && err == nil {
				t.Errorf("Expected validation error for namespace '%s', pipeline '%s', got none", tc.namespace, tc.pipeline)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error for namespace '%s', pipeline '%s', got: %v", tc.namespace, tc.pipeline, err)
			}
		})
	}
}

func TestValidator_ValidateTekton_InvalidTimeouts(t *testing.T) {
	testCases := []struct {
		name         string
		timeout      time.Duration
		retryBackoff time.Duration
		expectError  bool
	}{
		{
			name:         "Valid timeouts",
			timeout:      30 * time.Second,
			retryBackoff: 5 * time.Second,
			expectError:  false,
		},
		{
			name:         "Zero timeout (invalid)",
			timeout:      0,
			retryBackoff: 5 * time.Second,
			expectError:  true,
		},
		{
			name:         "Negative timeout (invalid)",
			timeout:      -10 * time.Second,
			retryBackoff: 5 * time.Second,
			expectError:  true,
		},
		{
			name:         "Zero retry backoff (invalid)",
			timeout:      30 * time.Second,
			retryBackoff: 0,
			expectError:  true,
		},
		{
			name:         "Negative retry backoff (invalid)",
			timeout:      30 * time.Second,
			retryBackoff: -5 * time.Second,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewValidator()

			config := createValidConfig()
			config.Tekton = types.TektonConfig{

				Timeout:      tc.timeout,
				RetryBackoff: tc.retryBackoff,
			}

			err := validator.Validate(config)
			if tc.expectError && err == nil {
				t.Errorf("Expected validation error for timeout %v, retryBackoff %v, got none", tc.timeout, tc.retryBackoff)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error for timeout %v, retryBackoff %v, got: %v", tc.timeout, tc.retryBackoff, err)
			}
		})
	}
}

func TestValidator_ValidateTekton_InvalidRetryAttempts(t *testing.T) {
	testCases := []struct {
		name          string
		retryAttempts int
		expectError   bool
	}{
		{
			name:          "Valid retry attempts (0)",
			retryAttempts: 0,
			expectError:   false,
		},
		{
			name:          "Valid retry attempts (positive)",
			retryAttempts: 3,
			expectError:   false,
		},
		{
			name:          "Invalid retry attempts (negative)",
			retryAttempts: -1,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewValidator()

			config := createValidConfig()
			config.Tekton = types.TektonConfig{

				RetryAttempts: tc.retryAttempts,
				Timeout:       30 * time.Second,
				RetryBackoff:  5 * time.Second,
			}

			err := validator.Validate(config)
			if tc.expectError && err == nil {
				t.Errorf("Expected validation error for retryAttempts %d, got none", tc.retryAttempts)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error for retryAttempts %d, got: %v", tc.retryAttempts, err)
			}
		})
	}
}

func TestValidator_IsValidKubernetesName(t *testing.T) {
	validator := NewValidator()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid simple name",
			input:    "test",
			expected: true,
		},
		{
			name:     "Valid name with dashes",
			input:    "test-pipeline-name",
			expected: true,
		},
		{
			name:     "Valid name with numbers",
			input:    "test123",
			expected: true,
		},
		{
			name:     "Valid single character",
			input:    "a",
			expected: true,
		},
		{
			name:     "Invalid empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Invalid uppercase",
			input:    "Test",
			expected: false,
		},
		{
			name:     "Invalid starting with dash",
			input:    "-test",
			expected: false,
		},
		{
			name:     "Invalid ending with dash",
			input:    "test-",
			expected: false,
		},
		{
			name:     "Invalid underscore",
			input:    "test_name",
			expected: false,
		},
		{
			name:     "Invalid dot",
			input:    "test.name",
			expected: false,
		},
		{
			name:     "Invalid special characters",
			input:    "test@name",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.isValidKubernetesName(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v for input '%s', got %v", tc.expected, tc.input, result)
			}
		})
	}
}

func TestValidator_ValidateTekton_ZeroConfig(t *testing.T) {
	validator := NewValidator()

	config := createValidConfig()
	// Tekton is not set, should use zero value - this will have validation errors
	// because Timeout and RetryBackoff default to 0

	err := validator.Validate(config)
	if err == nil {
		t.Error("Expected validation errors with zero-value Tekton config, got none")
	}
}
