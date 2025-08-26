package gitclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// GitClient defines the interface for Git providers
type GitClient interface {
	// GetBranches retrieves all branches for a repository
	GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error)

	// GetLatestCommit retrieves the latest commit SHA for a branch
	GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error)

	// CheckPermissions verifies if the client has access to the repository
	CheckPermissions(ctx context.Context, repo types.Repository) error

	// GetRateLimit returns current rate limit status
	GetRateLimit(ctx context.Context) (*types.RateLimit, error)

	// GetProvider returns the provider name
	GetProvider() string

	// Close releases any resources
	Close() error

	// ListFiles retrieves all files in a specific path for a commit
	ListFiles(ctx context.Context, repo types.Repository, commitSHA, path string) ([]string, error)

	// GetFileContent retrieves the content of a specific file
	GetFileContent(ctx context.Context, repo types.Repository, commitSHA, filePath string) ([]byte, error)

	// CheckDirectoryExists checks if a directory exists in the repository
	CheckDirectoryExists(ctx context.Context, repo types.Repository, commitSHA, dirPath string) (bool, error)
}

// ClientConfig represents common configuration for Git clients
type ClientConfig struct {
	Token          string        `json:"-"` // Hidden for security
	BaseURL        string        `json:"base_url,omitempty"`
	RepositoryURL  string        `json:"repository_url,omitempty"` // For auto-detecting API URLs
	Timeout        time.Duration `json:"timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryBackoff   time.Duration `json:"retry_backoff"`
	UserAgent      string        `json:"user_agent"`
	EnableFallback bool          `json:"enable_fallback"`
}

// ClientFactory creates Git clients based on provider
type ClientFactory struct {
	mu           sync.RWMutex
	rateLimiters map[string]RateLimiter
	fallback     *FallbackClient
	logger       *logger.Entry
}

// NewClientFactory creates a new client factory
func NewClientFactory(parentLogger *logger.Entry) *ClientFactory {
	return &ClientFactory{
		rateLimiters: make(map[string]RateLimiter),
		fallback:     NewFallbackClient(parentLogger),
		logger:       parentLogger,
	}
}

// CreateClient creates a client for the specified repository
func (f *ClientFactory) CreateClient(repo types.Repository, config ClientConfig) (GitClient, error) {
	// Set repository URL for auto-detection
	config.RepositoryURL = repo.URL

	switch repo.Provider {
	case "github":
		rateLimiter := f.getRateLimiter("github", NewGitHubRateLimiter())
		return NewGitHubClient(config, rateLimiter, f.fallback, f.logger)
	case "gitlab":
		rateLimiter := f.getRateLimiter("gitlab", NewGitLabRateLimiter())
		return NewGitLabClient(config, rateLimiter, f.fallback, f.logger)
	default:
		return nil, &UnsupportedProviderError{Provider: repo.Provider}
	}
}

// getRateLimiter returns or creates a rate limiter for a provider
func (f *ClientFactory) getRateLimiter(provider string, defaultLimiter RateLimiter) RateLimiter {
	f.mu.Lock()
	defer f.mu.Unlock()

	if limiter, exists := f.rateLimiters[provider]; exists {
		return limiter
	}
	f.rateLimiters[provider] = defaultLimiter
	return defaultLimiter
}

// Client errors
type UnsupportedProviderError struct {
	Provider string
}

func (e *UnsupportedProviderError) Error() string {
	return fmt.Sprintf("unsupported Git provider: %s", e.Provider)
}

type AuthenticationError struct {
	Provider string
	Message  string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed for %s: %s", e.Provider, e.Message)
}

type RepositoryNotFoundError struct {
	Repository string
	Provider   string
}

func (e *RepositoryNotFoundError) Error() string {
	return fmt.Sprintf("repository not found: %s on %s", e.Repository, e.Provider)
}

type RateLimitExceededError struct {
	Provider  string
	ResetTime time.Time
}

func (e *RateLimitExceededError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s, resets at %s",
		e.Provider, e.ResetTime.Format(time.RFC3339))
}

type NetworkError struct {
	Provider string
	Err      error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error for %s: %v", e.Provider, e.Err)
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	switch err.(type) {
	case *NetworkError:
		return true
	case *RateLimitExceededError:
		return false // Should wait, not retry immediately
	case *AuthenticationError:
		return false
	case *RepositoryNotFoundError:
		return false
	default:
		return false
	}
}

// GetDefaultConfig returns default client configuration
func GetDefaultConfig() ClientConfig {
	return ClientConfig{
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		RetryBackoff:   1 * time.Second,
		UserAgent:      "RepoSentry/1.0",
		EnableFallback: true,
	}
}
