package gitclient

import (
	"context"
	"encoding/base64"
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

// GitLabClient implements GitClient for GitLab API
type GitLabClient struct {
	config      ClientConfig
	httpClient  *http.Client
	rateLimiter RateLimiter
	fallback    *FallbackClient
	baseURL     string
	logger      *logger.Entry
}

// GitLabBranch represents a branch response from GitLab API
type GitLabBranch struct {
	Name      string       `json:"name"`
	Commit    GitLabCommit `json:"commit"`
	Protected bool         `json:"protected"`
}

// GitLabCommit represents a commit in GitLab API response
type GitLabCommit struct {
	ID string `json:"id"`
}

// GitLabProject represents a project in GitLab API
type GitLabProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	Visibility        string `json:"visibility"`
}

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(config ClientConfig, rateLimiter RateLimiter, fallback *FallbackClient, parentLogger *logger.Entry) (*GitLabClient, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		// Auto-detect GitLab API URL from repository URL
		if config.RepositoryURL != "" {
			baseURL = extractGitLabAPIURL(config.RepositoryURL)
		} else {
			baseURL = "https://gitlab.com/api/v4"
		}
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	clientLogger := parentLogger.WithFields(logger.Fields{
		"component": "gitclient",
		"provider":  "gitlab",
		"base_url":  baseURL,
	})

	clientLogger.Info("Initializing GitLab client")

	return &GitLabClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
		fallback:    fallback,
		baseURL:     baseURL,
		logger:      clientLogger,
	}, nil
}

// GetBranches retrieves all branches for a repository
func (c *GitLabClient) GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error) {
	projectID, err := c.getProjectID(ctx, repo.URL)
	if err != nil {
		if c.config.EnableFallback {
			return c.fallback.GetBranches(ctx, repo)
		}
		return nil, err
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/projects/%s/repository/branches", c.baseURL, projectID)

	var gitlabBranches []GitLabBranch
	if err := c.makeRequest(ctx, "GET", url, nil, &gitlabBranches); err != nil {
		if c.config.EnableFallback && IsRetryableError(err) {
			return c.fallback.GetBranches(ctx, repo)
		}
		return nil, err
	}

	// Convert to our types
	branches := make([]types.Branch, len(gitlabBranches))
	for i, gb := range gitlabBranches {
		branches[i] = types.Branch{
			Name:      gb.Name,
			CommitSHA: gb.Commit.ID,
			Protected: gb.Protected,
		}
	}

	return branches, nil
}

// GetLatestCommit retrieves the latest commit SHA for a branch
func (c *GitLabClient) GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error) {
	projectID, err := c.getProjectID(ctx, repo.URL)
	if err != nil {
		if c.config.EnableFallback {
			return c.fallback.GetLatestCommit(ctx, repo, branch)
		}
		return "", err
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s", c.baseURL, projectID, branch)

	var gitlabBranch GitLabBranch
	if err := c.makeRequest(ctx, "GET", url, nil, &gitlabBranch); err != nil {
		if c.config.EnableFallback && IsRetryableError(err) {
			return c.fallback.GetLatestCommit(ctx, repo, branch)
		}
		return "", err
	}

	return gitlabBranch.Commit.ID, nil
}

// CheckPermissions verifies if the client has access to the repository
func (c *GitLabClient) CheckPermissions(ctx context.Context, repo types.Repository) error {
	_, err := c.getProjectID(ctx, repo.URL)
	return err
}

// GetRateLimit returns current rate limit status
func (c *GitLabClient) GetRateLimit(ctx context.Context) (*types.RateLimit, error) {
	// GitLab doesn't have a dedicated rate limit endpoint
	// We return the current state from our rate limiter
	limitInfo := c.rateLimiter.GetLimit()

	return &types.RateLimit{
		Limit:     limitInfo.Limit,
		Remaining: limitInfo.Remaining,
		Reset:     limitInfo.ResetTime,
	}, nil
}

// GetProvider returns the provider name
func (c *GitLabClient) GetProvider() string {
	return "gitlab"
}

// Close releases any resources
func (c *GitLabClient) Close() error {
	return nil
}

// getProjectID retrieves the project ID from a GitLab repository URL
func (c *GitLabClient) getProjectID(ctx context.Context, repoURL string) (string, error) {
	namespace, project, err := c.parseRepoURL(repoURL)
	if err != nil {
		return "", err
	}

	projectPath := url.QueryEscape(fmt.Sprintf("%s/%s", namespace, project))

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/projects/%s", c.baseURL, projectPath)

	var gitlabProject GitLabProject
	if err := c.makeRequest(ctx, "GET", url, nil, &gitlabProject); err != nil {
		return "", err
	}

	return strconv.Itoa(gitlabProject.ID), nil
}

// makeRequest makes an HTTP request to the GitLab API
func (c *GitLabClient) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return &NetworkError{Provider: "gitlab", Err: err}
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Content-Type", "application/json")

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
			lastErr = &NetworkError{Provider: "gitlab", Err: err}
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
					return &NetworkError{Provider: "gitlab", Err: err}
				}
			}
			return nil
		case http.StatusUnauthorized, http.StatusForbidden:
			resp.Body.Close()
			return &AuthenticationError{Provider: "gitlab", Message: "invalid or insufficient permissions"}
		case http.StatusNotFound:
			resp.Body.Close()
			return &RepositoryNotFoundError{Repository: url, Provider: "gitlab"}
		case http.StatusTooManyRequests:
			resp.Body.Close()
			resetTime := c.parseResetTime(resp.Header.Get("RateLimit-ResetTime"))
			return &RateLimitExceededError{Provider: "gitlab", ResetTime: resetTime}
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			resp.Body.Close()
			lastErr = &NetworkError{Provider: "gitlab", Err: fmt.Errorf("server error: %d", resp.StatusCode)}
			if attempt < c.config.RetryAttempts {
				continue
			}
			return lastErr
		default:
			resp.Body.Close()
			return &NetworkError{Provider: "gitlab", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
		}
	}

	return lastErr
}

// parseRepoURL extracts namespace and project name from GitLab URL
func (c *GitLabClient) parseRepoURL(repoURL string) (namespace, project string, err error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Handle GitLab URLs: https://gitlab.com/namespace/project
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("invalid GitLab repository URL format: %s", repoURL)
	}

	// For nested groups: namespace/subgroup/project
	if len(pathParts) > 2 {
		namespace = strings.Join(pathParts[:len(pathParts)-1], "/")
		project = pathParts[len(pathParts)-1]
	} else {
		namespace = pathParts[0]
		project = pathParts[1]
	}

	return namespace, project, nil
}

// updateRateLimitFromHeaders updates the rate limiter based on response headers
func (c *GitLabClient) updateRateLimitFromHeaders(headers http.Header) {
	limitStr := headers.Get("RateLimit-Limit")
	remainingStr := headers.Get("RateLimit-Remaining")
	resetStr := headers.Get("RateLimit-ResetTime")

	if limitStr != "" && remainingStr != "" && resetStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			if remaining, err := strconv.Atoi(remainingStr); err == nil {
				if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
					c.rateLimiter.UpdateLimit(limit, remaining, resetTime)
				}
			}
		}
	}
}

// parseResetTime parses the reset time from rate limit header
func (c *GitLabClient) parseResetTime(resetStr string) time.Time {
	if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
		return resetTime
	}
	return time.Now().Add(time.Minute) // Default to 1 minute if parsing fails
}

// extractGitLabAPIURL extracts GitLab API URL from repository URL
func extractGitLabAPIURL(repoURL string) string {
	// Parse the repository URL to extract the host
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "https://gitlab.com/api/v4" // Default fallback
	}

	// Construct API URL from the host
	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = "https"
	}

	// For our specific use case, support these two GitLab instances
	host := parsedURL.Host
	switch host {
	case "gitlab-master.nvidia.com":
		return fmt.Sprintf("%s://%s/api/v4", scheme, host)
	case "gitlab.com":
		return "https://gitlab.com/api/v4"
	default:
		// For any other GitLab instance, construct API URL
		return fmt.Sprintf("%s://%s/api/v4", scheme, host)
	}
}

// GitLabTreeItem represents a single item in GitLab's repository tree API response
type GitLabTreeItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// GitLabFile represents GitLab's file API response
type GitLabFile struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	Size         int    `json:"size"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
	ContentSHA256 string `json:"content_sha256"`
	Ref          string `json:"ref"`
	BlobID       string `json:"blob_id"`
	CommitID     string `json:"commit_id"`
	LastCommitID string `json:"last_commit_id"`
}

// ListFiles retrieves all files in a specific path for a commit
func (c *GitLabClient) ListFiles(ctx context.Context, repo types.Repository, commitSHA, path string) ([]string, error) {
	projectPath, err := c.parseProjectPath(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Use repository tree API
	encodedPath := url.QueryEscape(projectPath)
	apiURL := fmt.Sprintf("%s/projects/%s/repository/tree?ref=%s&recursive=true", 
		c.baseURL, encodedPath, commitSHA)
	
	if path != "" {
		apiURL += "&path=" + url.QueryEscape(path)
	}

	var tree []GitLabTreeItem
	if err := c.makeRequest(ctx, "GET", apiURL, nil, &tree); err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	var files []string
	for _, item := range tree {
		// Only include files (blobs)
		if item.Type == "blob" {
			files = append(files, item.Path)
		}
	}

	return files, nil
}

// GetFileContent retrieves the content of a specific file
func (c *GitLabClient) GetFileContent(ctx context.Context, repo types.Repository, commitSHA, filePath string) ([]byte, error) {
	projectPath, err := c.parseProjectPath(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Use files API
	encodedPath := url.QueryEscape(projectPath)
	encodedFilePath := url.QueryEscape(filePath)
	apiURL := fmt.Sprintf("%s/projects/%s/repository/files/%s?ref=%s", 
		c.baseURL, encodedPath, encodedFilePath, commitSHA)

	var file GitLabFile
	if err := c.makeRequest(ctx, "GET", apiURL, nil, &file); err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	// Decode base64 content
	if file.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode content: %w", err)
		}
		return decoded, nil
	}

	return []byte(file.Content), nil
}

// CheckDirectoryExists checks if a directory exists in the repository
func (c *GitLabClient) CheckDirectoryExists(ctx context.Context, repo types.Repository, commitSHA, dirPath string) (bool, error) {
	projectPath, err := c.parseProjectPath(repo.URL)
	if err != nil {
		return false, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Use repository tree API to check if directory has content
	encodedPath := url.QueryEscape(projectPath)
	apiURL := fmt.Sprintf("%s/projects/%s/repository/tree?ref=%s&path=%s", 
		c.baseURL, encodedPath, commitSHA, url.QueryEscape(dirPath))

	var tree []GitLabTreeItem
	if err := c.makeRequest(ctx, "GET", apiURL, nil, &tree); err != nil {
		// If 404, directory doesn't exist
		if networkErr, ok := err.(*NetworkError); ok {
			if strings.Contains(networkErr.Err.Error(), "404") {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check directory: %w", err)
	}

	return len(tree) > 0, nil
}

// parseProjectPath extracts the project path from GitLab repository URL
func (c *GitLabClient) parseProjectPath(repoURL string) (string, error) {
	// Parse URLs like:
	// https://gitlab.com/group/project
	// https://gitlab-master.nvidia.com/group/subgroup/project
	
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate that it's a proper HTTP/HTTPS URL
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing scheme or host")
	}
	
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	path := strings.TrimPrefix(parsedURL.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	
	if path == "" {
		return "", fmt.Errorf("empty project path")
	}

	return path, nil
}
