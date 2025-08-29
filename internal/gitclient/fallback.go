package gitclient

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// FallbackClient implements Git operations using git commands
type FallbackClient struct {
	timeout time.Duration
	logger  *logger.Entry
}

// NewFallbackClient creates a new fallback client
func NewFallbackClient(parentLogger *logger.Entry) *FallbackClient {
	clientLogger := parentLogger.WithFields(logger.Fields{
		"component": "gitclient",
		"provider":  "git-fallback",
	})

	clientLogger.Info("Initializing Git fallback client")

	return &FallbackClient{
		timeout: 30 * time.Second,
		logger:  clientLogger,
	}
}

// GetBranches retrieves branches using git ls-remote
func (f *FallbackClient) GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error) {
	f.logger.WithFields(logger.Fields{
		"operation":  "get_branches",
		"repository": repo.Name,
		"url":        repo.URL,
	}).Info("Starting fallback branch retrieval using git ls-remote")

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Use git ls-remote to list branches
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", repo.URL)

	output, err := cmd.Output()
	if err != nil {
		f.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "get_branches",
			"repository": repo.Name,
			"url":        repo.URL,
		}).Error("Git ls-remote command failed")
		return nil, &NetworkError{
			Provider: "git-fallback",
			Err:      fmt.Errorf("git ls-remote failed: %w", err),
		}
	}

	branches, err := f.parseLsRemoteOutput(string(output))
	if err != nil {
		f.logger.WithError(err).WithFields(logger.Fields{
			"operation":  "get_branches",
			"repository": repo.Name,
		}).Error("Failed to parse git ls-remote output")
		return nil, err
	}

	f.logger.WithFields(logger.Fields{
		"operation":    "get_branches",
		"repository":   repo.Name,
		"branch_count": len(branches),
	}).Info("Successfully retrieved branches using git fallback")

	return branches, nil
}

// GetLatestCommit retrieves latest commit for a specific branch using git ls-remote
func (f *FallbackClient) GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Use git ls-remote to get specific branch
	refName := fmt.Sprintf("refs/heads/%s", branch)
	cmd := exec.CommandContext(ctx, "git", "ls-remote", repo.URL, refName)

	output, err := cmd.Output()
	if err != nil {
		return "", &NetworkError{
			Provider: "git-fallback",
			Err:      fmt.Errorf("git ls-remote failed for branch %s: %w", branch, err),
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", &RepositoryNotFoundError{
			Repository: fmt.Sprintf("%s (branch: %s)", repo.URL, branch),
			Provider:   "git-fallback",
		}
	}

	// Parse commit SHA from first line
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid git ls-remote output: %s", lines[0])
	}

	return parts[0], nil
}

// CheckPermissions checks if repository is accessible using git ls-remote
func (f *FallbackClient) CheckPermissions(ctx context.Context, repo types.Repository) error {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Try to list remote references
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", repo.URL)

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			switch exitError.ExitCode() {
			case 128:
				return &RepositoryNotFoundError{Repository: repo.URL, Provider: "git-fallback"}
			case 129:
				return &AuthenticationError{Provider: "git-fallback", Message: "access denied"}
			default:
				return &NetworkError{Provider: "git-fallback", Err: err}
			}
		}
		return &NetworkError{Provider: "git-fallback", Err: err}
	}

	return nil
}

// GetRateLimit returns unlimited rate limit for git commands
func (f *FallbackClient) GetRateLimit(ctx context.Context) (*types.RateLimit, error) {
	return &types.RateLimit{
		Limit:     999999,
		Remaining: 999999,
		Reset:     time.Now().Add(time.Hour),
	}, nil
}

// GetProvider returns the provider name
func (f *FallbackClient) GetProvider() string {
	return "git-fallback"
}

// Close releases any resources
func (f *FallbackClient) Close() error {
	return nil
}

// ListFiles retrieves files from a repository path (fallback implementation)
func (f *FallbackClient) ListFiles(ctx context.Context, repo types.Repository, commitSHA, path string) ([]string, error) {
	f.logger.WithFields(logger.Fields{
		"operation": "list_files",
		"repository": repo.Name,
		"commit": commitSHA,
		"path": path,
	}).Info("Starting fallback file listing")

	// Note: This is a simplified fallback implementation
	// In production, you might want to implement this using git archive or similar
	return nil, fmt.Errorf("ListFiles not implemented in fallback client - API client required")
}

// GetFileContent retrieves file content (fallback implementation)
func (f *FallbackClient) GetFileContent(ctx context.Context, repo types.Repository, commitSHA, filePath string) ([]byte, error) {
	f.logger.WithFields(logger.Fields{
		"operation": "get_file_content",
		"repository": repo.Name,
		"commit": commitSHA,
		"file_path": filePath,
	}).Info("Starting fallback file content retrieval")

	// Note: This is a simplified fallback implementation
	// In production, you might want to implement this using git show or similar
	return nil, fmt.Errorf("GetFileContent not implemented in fallback client - API client required")
}

// CheckDirectoryExists checks if directory exists (fallback implementation)
func (f *FallbackClient) CheckDirectoryExists(ctx context.Context, repo types.Repository, commitSHA, dirPath string) (bool, error) {
	f.logger.WithFields(logger.Fields{
		"operation": "check_directory_exists",
		"repository": repo.Name,
		"commit": commitSHA,
		"dir_path": dirPath,
	}).Info("Starting fallback directory existence check")

	// Note: This is a simplified fallback implementation
	// In production, you might want to implement this using git ls-tree or similar
	return false, fmt.Errorf("CheckDirectoryExists not implemented in fallback client - API client required")
}

// parseLsRemoteOutput parses git ls-remote output to extract branches
func (f *FallbackClient) parseLsRemoteOutput(output string) ([]types.Branch, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []types.Branch{}, nil
	}

	var branches []types.Branch
	branchRegex := regexp.MustCompile(`^([a-f0-9A-F]+)\s+refs/heads/(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := branchRegex.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		commitSHA := matches[1]
		branchName := matches[2]

		branches = append(branches, types.Branch{
			Name:      branchName,
			CommitSHA: commitSHA,
			Protected: false, // Can't determine protection status from git ls-remote
		})
	}

	return branches, nil
}

// Wait implements a simple delay for fallback client
func (f *FallbackClient) Wait(ctx context.Context) error {
	// Simple rate limiting for git commands
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond): // Small delay between git commands
		return nil
	}
}

// Allow always returns true for fallback client
func (f *FallbackClient) Allow() bool {
	return true
}

// TestGitAvailability checks if git command is available
func TestGitAvailability(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command not available: %w", err)
	}

	return nil
}

// ParseGitVersion extracts git version from git --version output
func ParseGitVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git version: %w", err)
	}

	// Parse version from output like "git version 2.34.1"
	versionRegex := regexp.MustCompile(`git version (\d+\.\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return strings.TrimSpace(string(output)), nil
	}

	return matches[1], nil
}

// ConfigureGitCredentials configures git credentials for authentication
func ConfigureGitCredentials(ctx context.Context, repo types.Repository) error {
	if repo.Token == "" {
		return nil // No token to configure
	}

	// For HTTP(S) URLs, we can configure credentials via git config
	if strings.HasPrefix(repo.URL, "https://") {
		// Configure credential helper to use token
		// This is a simplified approach - in production you might want more sophisticated credential management
		return nil
	}

	return nil
}

// CleanupGitCredentials removes configured credentials
func CleanupGitCredentials(ctx context.Context, repo types.Repository) error {
	// In a real implementation, you would clean up any configured credentials
	return nil
}

// ValidateGitRepository checks if a URL is a valid git repository
func ValidateGitRepository(ctx context.Context, repoURL string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", repoURL)
	return cmd.Run()
}

// GetRemoteInfo retrieves information about a remote repository
func GetRemoteInfo(ctx context.Context, repoURL string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	branchCount := 0
	tagCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "refs/heads/") {
			branchCount++
		} else if strings.Contains(line, "refs/tags/") {
			tagCount++
		}
	}

	info["branches"] = fmt.Sprintf("%d", branchCount)
	info["tags"] = fmt.Sprintf("%d", tagCount)
	info["url"] = repoURL

	return info, nil
}
