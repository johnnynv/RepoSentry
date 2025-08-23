package trigger

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// RepositoryInfo represents parsed repository information
type RepositoryInfo struct {
	Provider     string `json:"provider"`      // "github" or "gitlab"
	Instance     string `json:"instance"`      // "github.com", "gitlab-master.nvidia.com"
	Namespace    string `json:"namespace"`     // "owner" or "group/subgroup"
	ProjectName  string `json:"project_name"`  // "repo"
	FullName     string `json:"full_name"`     // "owner/repo" or "group/subgroup/repo"
	CloneURL     string `json:"clone_url"`     // Original or normalized clone URL
	HTMLURL      string `json:"html_url"`      // Web URL
	APIBaseURL   string `json:"api_base_url"`  // API base URL
	IsEnterprise bool   `json:"is_enterprise"` // Whether it's enterprise instance
}

// URLParser provides intelligent URL parsing for Git repositories
type URLParser struct {
	logger *logger.Entry
}

// NewURLParser creates a new URL parser
func NewURLParser(parentLogger *logger.Entry) *URLParser {
	return &URLParser{
		logger: parentLogger.WithFields(logger.Fields{
			"component": "trigger",
			"module":    "url_parser",
		}),
	}
}

// ParseRepositoryURL parses a repository URL and extracts provider information
func (p *URLParser) ParseRepositoryURL(repoURL string) (*RepositoryInfo, error) {
	// Trim whitespace from URL for robustness
	repoURL = strings.TrimSpace(repoURL)

	p.logger.WithFields(logger.Fields{
		"operation": "parse_repository_url",
		"url":       repoURL,
	}).Debug("Parsing repository URL")

	// Validate URL format first
	if err := p.validateURLFormat(repoURL); err != nil {
		p.logger.WithFields(logger.Fields{
			"operation": "parse_repository_url",
			"url":       repoURL,
			"error":     err.Error(),
		}).Error("Invalid repository URL format")
		return nil, err
	}

	// Parse URL
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		p.logger.WithFields(logger.Fields{
			"operation": "parse_repository_url",
			"url":       repoURL,
			"error":     err.Error(),
		}).Error("Failed to parse repository URL")
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Normalize URL (remove .git suffix, HTTPS only)
	normalizedURL := p.normalizeURL(parsedURL)

	// Detect provider type
	provider := p.detectProvider(normalizedURL.Host)

	// Parse path components
	namespace, projectName, err := p.parseRepoPath(normalizedURL.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository path: %w", err)
	}

	// Build full name
	fullName := namespace
	if projectName != "" {
		if namespace != "" {
			fullName = namespace + "/" + projectName
		} else {
			fullName = projectName
		}
	}

	// Determine if enterprise instance
	isEnterprise := p.isEnterpriseInstance(normalizedURL.Host, provider)

	// Build repository info
	repoInfo := &RepositoryInfo{
		Provider:     provider,
		Instance:     normalizedURL.Host,
		Namespace:    namespace,
		ProjectName:  projectName,
		FullName:     fullName,
		CloneURL:     p.buildCloneURL(normalizedURL, provider),
		HTMLURL:      p.buildHTMLURL(normalizedURL, provider),
		APIBaseURL:   p.buildAPIBaseURL(normalizedURL.Host, provider),
		IsEnterprise: isEnterprise,
	}

	p.logger.WithFields(logger.Fields{
		"operation":     "parse_repository_url",
		"original_url":  repoURL,
		"provider":      repoInfo.Provider,
		"instance":      repoInfo.Instance,
		"full_name":     repoInfo.FullName,
		"is_enterprise": repoInfo.IsEnterprise,
	}).Info("Successfully parsed repository URL")

	return repoInfo, nil
}

// normalizeURL normalizes the repository URL (HTTPS only)
func (p *URLParser) normalizeURL(parsedURL *url.URL) *url.URL {
	normalized := *parsedURL

	// Remove .git suffix from path
	if strings.HasSuffix(normalized.Path, ".git") {
		normalized.Path = strings.TrimSuffix(normalized.Path, ".git")
	}

	// Ensure leading slash
	if !strings.HasPrefix(normalized.Path, "/") {
		normalized.Path = "/" + normalized.Path
	}

	return &normalized
}

// detectProvider detects the Git provider based on hostname
func (p *URLParser) detectProvider(hostname string) string {
	hostname = strings.ToLower(hostname)

	// GitHub detection
	if hostname == "github.com" || strings.Contains(hostname, "github") {
		return "github"
	}

	// GitLab detection
	if hostname == "gitlab.com" || strings.Contains(hostname, "gitlab") {
		return "gitlab"
	}

	// Default fallback based on common patterns
	if strings.Contains(hostname, "git") {
		// If hostname contains "git", assume GitLab (more common for enterprise)
		return "gitlab"
	}

	// Ultimate fallback to GitLab (as it's more commonly self-hosted)
	return "gitlab"
}

// parseRepoPath parses repository path to extract namespace and project name
func (p *URLParser) parseRepoPath(repoPath string) (namespace, projectName string, err error) {
	// Clean path
	cleanPath := strings.Trim(repoPath, "/")
	if cleanPath == "" {
		return "", "", fmt.Errorf("empty repository path")
	}

	// Split path components
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository path: must contain at least owner/repo")
	}

	// Last part is always project name
	projectName = parts[len(parts)-1]

	// Everything before last part is namespace
	if len(parts) > 1 {
		namespaceParts := parts[:len(parts)-1]
		namespace = strings.Join(namespaceParts, "/")
	}

	return namespace, projectName, nil
}

// isEnterpriseInstance determines if the instance is enterprise/self-hosted
func (p *URLParser) isEnterpriseInstance(hostname, provider string) bool {
	hostname = strings.ToLower(hostname)

	// Public instances
	publicInstances := map[string]bool{
		"github.com": true,
		"gitlab.com": true,
	}

	return !publicInstances[hostname]
}

// buildCloneURL builds the clone URL
func (p *URLParser) buildCloneURL(parsedURL *url.URL, provider string) string {
	cloneURL := fmt.Sprintf("https://%s%s", parsedURL.Host, parsedURL.Path)
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}
	return cloneURL
}

// buildHTMLURL builds the web URL
func (p *URLParser) buildHTMLURL(parsedURL *url.URL, provider string) string {
	return fmt.Sprintf("https://%s%s", parsedURL.Host, parsedURL.Path)
}

// buildAPIBaseURL builds the API base URL
func (p *URLParser) buildAPIBaseURL(hostname, provider string) string {
	switch provider {
	case "github":
		if hostname == "github.com" {
			return "https://api.github.com"
		}
		// GitHub Enterprise: https://hostname/api/v3
		return fmt.Sprintf("https://%s/api/v3", hostname)
	case "gitlab":
		// GitLab (both .com and self-hosted): https://hostname/api/v4
		return fmt.Sprintf("https://%s/api/v4", hostname)
	default:
		// Default to GitLab API format
		return fmt.Sprintf("https://%s/api/v4", hostname)
	}
}

// ValidateRepositoryURL validates that a repository URL is properly formatted
func (p *URLParser) ValidateRepositoryURL(repoURL string) error {
	_, err := p.ParseRepositoryURL(repoURL)
	return err
}

// GetProviderType extracts just the provider type from URL (quick operation)
func (p *URLParser) GetProviderType(repoURL string) string {
	// Trim whitespace for robustness
	repoURL = strings.TrimSpace(repoURL)

	// Validate URL format first
	if err := p.validateURLFormat(repoURL); err != nil {
		p.logger.WithFields(logger.Fields{
			"operation": "get_provider_type",
			"url":       repoURL,
			"error":     err.Error(),
		}).Error("Invalid URL format for provider detection")
		return "unknown"
	}

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		p.logger.WithFields(logger.Fields{
			"operation": "get_provider_type",
			"url":       repoURL,
			"error":     err.Error(),
		}).Error("Failed to parse URL for provider detection")
		return "unknown"
	}

	return p.detectProvider(parsedURL.Host)
}

// BuildRepoURLs builds various repository URLs from components
func (p *URLParser) BuildRepoURLs(instance, fullName, provider string) *RepositoryInfo {
	// Construct normalized URL
	baseURL := fmt.Sprintf("https://%s/%s", instance, fullName)

	// Parse to get full info
	repoInfo, _ := p.ParseRepositoryURL(baseURL)
	if repoInfo == nil {
		// Fallback construction
		namespace, projectName := path.Split(fullName)
		namespace = strings.Trim(namespace, "/")

		repoInfo = &RepositoryInfo{
			Provider:     provider,
			Instance:     instance,
			Namespace:    namespace,
			ProjectName:  projectName,
			FullName:     fullName,
			CloneURL:     baseURL + ".git",
			HTMLURL:      baseURL,
			APIBaseURL:   p.buildAPIBaseURL(instance, provider),
			IsEnterprise: p.isEnterpriseInstance(instance, provider),
		}
	}

	return repoInfo
}

// validateURLFormat validates that URL meets our strict requirements
func (p *URLParser) validateURLFormat(repoURL string) error {
	// Trim whitespace for robustness
	repoURL = strings.TrimSpace(repoURL)

	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Check for SSH format (not supported)
	if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") && !strings.Contains(repoURL, "://") {
		return fmt.Errorf("SSH URLs are not supported, please use HTTPS format (e.g., https://gitlab-master.nvidia.com/owner/repo)")
	}

	// Must start with https://
	if !strings.HasPrefix(repoURL, "https://") {
		return fmt.Errorf("only HTTPS URLs are supported, URL must start with 'https://' (got: %s)", repoURL)
	}

	// Basic URL parsing check
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	// Must have a hostname
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must contain a valid hostname")
	}

	// Must have a path with at least 2 components (owner/repo)
	cleanPath := strings.Trim(parsedURL.Path, "/")
	if cleanPath == "" {
		return fmt.Errorf("URL must contain repository path (e.g., /owner/repo)")
	}

	pathParts := strings.Split(cleanPath, "/")
	if len(pathParts) < 2 {
		return fmt.Errorf("URL path must contain at least owner/repo (got: %s)", cleanPath)
	}

	// Validate each path component is not empty
	for i, part := range pathParts {
		if part == "" {
			return fmt.Errorf("URL path cannot contain empty components (position %d)", i)
		}
	}

	return nil
}
