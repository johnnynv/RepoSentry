package gitclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"


	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// GitHubClient implements GitClient for GitHub API
type GitHubClient struct {
	config      ClientConfig
	httpClient  *http.Client
	rateLimiter RateLimiter
	fallback    *FallbackClient
	baseURL     string
	logger      *logger.Entry
}

// GitHubBranch represents a branch response from GitHub API
type GitHubBranch struct {
	Name      string `json:"name"`
	Commit    GitHubCommit `json:"commit"`
	Protected bool   `json:"protected"`
}

// GitHubCommit represents a commit in GitHub API response
type GitHubCommit struct {
	SHA string `json:"sha"`
}

// GitHubRepository represents a repository in GitHub API
type GitHubRepository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

// GitHubRateLimit represents GitHub's rate limit response
type GitHubRateLimit struct {
	Resources struct {
		Core struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"core"`
	} `json:"resources"`
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(config ClientConfig, rateLimiter RateLimiter, fallback *FallbackClient) (*GitHubClient, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	clientLogger := logger.GetDefaultLogger().WithFields(logger.Fields{
		"component": "gitclient",
		"provider":  "github",
		"base_url":  baseURL,
	})

	clientLogger.Info("Initializing GitHub client")

	return &GitHubClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
		fallback:    fallback,
		baseURL:     baseURL,
		logger:      clientLogger,
	}, nil
}

// GetBranches retrieves all branches for a repository
func (c *GitHubClient) GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error) {
	c.logger.WithFields(logger.Fields{
		"operation": "get_branches",
		"repository": repo.Name,
		"url": repo.URL,
	}).Info("Starting branch retrieval")

	owner, repoName, err := c.parseRepoURL(repo.URL)
	if err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "get_branches",
			"repository": repo.Name,
			"url": repo.URL,
		}).Error("Failed to parse repository URL")
		if c.config.EnableFallback {
			c.logger.Info("Attempting fallback for branch retrieval")
			return c.fallback.GetBranches(ctx, repo)
		}
		return nil, err
	}

	// Wait for rate limiter
	c.logger.WithFields(logger.Fields{
		"operation": "get_branches",
		"repository": repo.Name,
	}).Debug("Waiting for rate limiter")
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "get_branches",
			"repository": repo.Name,
		}).Error("Rate limiter wait failed")
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/branches", c.baseURL, owner, repoName)
	c.logger.WithFields(logger.Fields{
		"operation": "get_branches",
		"repository": repo.Name,
		"url": url,
	}).Debug("Making API request")
	
	var githubBranches []GitHubBranch
	if err := c.makeRequest(ctx, "GET", url, nil, &githubBranches); err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "get_branches",
			"repository": repo.Name,
			"url": url,
		}).Error("API request failed")
		if c.config.EnableFallback && IsRetryableError(err) {
			c.logger.Info("Attempting fallback after API failure")
			return c.fallback.GetBranches(ctx, repo)
		}
		return nil, err
	}

	// Convert to our types
	branches := make([]types.Branch, len(githubBranches))
	for i, gb := range githubBranches {
		branches[i] = types.Branch{
			Name:      gb.Name,
			CommitSHA: gb.Commit.SHA,
			Protected: gb.Protected,
		}
	}

	c.logger.WithFields(logger.Fields{
		"operation": "get_branches",
		"repository": repo.Name,
		"branch_count": len(branches),
	}).Info("Successfully retrieved branches")

	return branches, nil
}

// GetLatestCommit retrieves the latest commit SHA for a branch
func (c *GitHubClient) GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error) {
	c.logger.WithFields(logger.Fields{
		"operation": "get_latest_commit",
		"repository": repo.Name,
		"branch": branch,
	}).Info("Starting latest commit retrieval")

	owner, repoName, err := c.parseRepoURL(repo.URL)
	if err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "get_latest_commit",
			"repository": repo.Name,
			"branch": branch,
		}).Error("Failed to parse repository URL")
		if c.config.EnableFallback {
			c.logger.Info("Attempting fallback for latest commit")
			return c.fallback.GetLatestCommit(ctx, repo, branch)
		}
		return "", err
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/branches/%s", c.baseURL, owner, repoName, branch)
	
	var githubBranch GitHubBranch
	if err := c.makeRequest(ctx, "GET", url, nil, &githubBranch); err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "get_latest_commit",
			"repository": repo.Name,
			"branch": branch,
			"url": url,
		}).Error("API request failed")
		if c.config.EnableFallback && IsRetryableError(err) {
			c.logger.Info("Attempting fallback after API failure")
			return c.fallback.GetLatestCommit(ctx, repo, branch)
		}
		return "", err
	}

	c.logger.WithFields(logger.Fields{
		"operation": "get_latest_commit",
		"repository": repo.Name,
		"branch": branch,
		"commit_sha": githubBranch.Commit.SHA,
	}).Info("Successfully retrieved latest commit")

	return githubBranch.Commit.SHA, nil
}

// CheckPermissions verifies if the client has access to the repository
func (c *GitHubClient) CheckPermissions(ctx context.Context, repo types.Repository) error {
	c.logger.WithFields(logger.Fields{
		"operation": "check_permissions",
		"repository": repo.Name,
	}).Info("Checking repository permissions")

	owner, repoName, err := c.parseRepoURL(repo.URL)
	if err != nil {
		c.logger.WithError(err).WithFields(logger.Fields{
			"operation": "check_permissions",
			"repository": repo.Name,
		}).Error("Failed to parse repository URL")
		return err
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repoName)
	
	var githubRepo GitHubRepository
	if err := c.makeRequest(ctx, "GET", url, nil, &githubRepo); err != nil {
		return err
	}

	return nil
}

// GetRateLimit returns current rate limit status
func (c *GitHubClient) GetRateLimit(ctx context.Context) (*types.RateLimit, error) {
	url := fmt.Sprintf("%s/rate_limit", c.baseURL)
	
	var rateLimitResp GitHubRateLimit
	if err := c.makeRequest(ctx, "GET", url, nil, &rateLimitResp); err != nil {
		return nil, err
	}

	resetTime := time.Unix(int64(rateLimitResp.Resources.Core.Reset), 0)
	
	return &types.RateLimit{
		Limit:     rateLimitResp.Resources.Core.Limit,
		Remaining: rateLimitResp.Resources.Core.Remaining,
		Reset:     resetTime,
	}, nil
}

// GetProvider returns the provider name
func (c *GitHubClient) GetProvider() string {
	return "github"
}

// Close releases any resources
func (c *GitHubClient) Close() error {
	return nil
}

// makeRequest makes an HTTP request to the GitHub API
func (c *GitHubClient) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return &NetworkError{Provider: "github", Err: err}
	}

	// Set headers
	req.Header.Set("Authorization", "token "+c.config.Token)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryBackoff * time.Duration(attempt)):
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &NetworkError{Provider: "github", Err: err}
			if attempt < c.config.RetryAttempts {
				continue
			}
			return lastErr
		}

		// Update rate limiter from headers
		c.updateRateLimitFromHeaders(resp.Header)

		// Handle different status codes
		switch resp.StatusCode {
		case http.StatusOK:
			if result != nil {
				defer resp.Body.Close()
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return &NetworkError{Provider: "github", Err: err}
				}
			}
			return nil
		case http.StatusUnauthorized, http.StatusForbidden:
			resp.Body.Close()
			return &AuthenticationError{Provider: "github", Message: "invalid or insufficient permissions"}
		case http.StatusNotFound:
			resp.Body.Close()
			return &RepositoryNotFoundError{Repository: url, Provider: "github"}
		case http.StatusTooManyRequests:
			resp.Body.Close()
			resetTime := c.parseResetTime(resp.Header.Get("X-RateLimit-Reset"))
			return &RateLimitExceededError{Provider: "github", ResetTime: resetTime}
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			resp.Body.Close()
			lastErr = &NetworkError{Provider: "github", Err: fmt.Errorf("server error: %d", resp.StatusCode)}
			if attempt < c.config.RetryAttempts {
				continue
			}
			return lastErr
		default:
			resp.Body.Close()
			return &NetworkError{Provider: "github", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
		}
	}

	return lastErr
}

// parseRepoURL extracts owner and repository name from GitHub URL
func (c *GitHubClient) parseRepoURL(repoURL string) (owner, repo string, err error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Handle GitHub URLs: https://github.com/owner/repo
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repository URL format: %s", repoURL)
	}

	return pathParts[0], pathParts[1], nil
}

// updateRateLimitFromHeaders updates the rate limiter based on response headers
func (c *GitHubClient) updateRateLimitFromHeaders(headers http.Header) {
	limitStr := headers.Get("X-RateLimit-Limit")
	remainingStr := headers.Get("X-RateLimit-Remaining")
	resetStr := headers.Get("X-RateLimit-Reset")

	if limitStr != "" && remainingStr != "" && resetStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			if remaining, err := strconv.Atoi(remainingStr); err == nil {
				if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
					resetTime := time.Unix(resetUnix, 0)
					c.rateLimiter.UpdateLimit(limit, remaining, resetTime)
				}
			}
		}
	}
}

// parseResetTime parses the reset time from rate limit header
func (c *GitHubClient) parseResetTime(resetStr string) time.Time {
	if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
		return time.Unix(resetUnix, 0)
	}
	return time.Now().Add(time.Hour) // Default to 1 hour if parsing fails
}
