package trigger

import (
	"testing"
)

func TestURLParser_ParseRepositoryURL(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		url      string
		expected *RepositoryInfo
		wantErr  bool
	}{
		{
			name: "GitHub public repository",
			url:  "https://github.com/torvalds/linux.git",
			expected: &RepositoryInfo{
				Provider:     "github",
				Instance:     "github.com",
				Namespace:    "torvalds",
				ProjectName:  "linux",
				FullName:     "torvalds/linux",
				CloneURL:     "https://github.com/torvalds/linux.git",
				HTMLURL:      "https://github.com/torvalds/linux",
				APIBaseURL:   "https://api.github.com",
				IsEnterprise: false,
			},
		},
		{
			name: "GitLab public repository",
			url:  "https://gitlab.com/gitlab-org/gitlab.git",
			expected: &RepositoryInfo{
				Provider:     "gitlab",
				Instance:     "gitlab.com",
				Namespace:    "gitlab-org",
				ProjectName:  "gitlab",
				FullName:     "gitlab-org/gitlab",
				CloneURL:     "https://gitlab.com/gitlab-org/gitlab.git",
				HTMLURL:      "https://gitlab.com/gitlab-org/gitlab",
				APIBaseURL:   "https://gitlab.com/api/v4",
				IsEnterprise: false,
			},
		},
		{
			name: "NVIDIA GitLab enterprise repository - multi-level namespace",
			url:  "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
			expected: &RepositoryInfo{
				Provider:     "gitlab",
				Instance:     "gitlab-master.nvidia.com",
				Namespace:    "chat-labs/OpenSource",
				ProjectName:  "rag",
				FullName:     "chat-labs/OpenSource/rag",
				CloneURL:     "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git",
				HTMLURL:      "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
				APIBaseURL:   "https://gitlab-master.nvidia.com/api/v4",
				IsEnterprise: true,
			},
		},
		{
			name: "NVIDIA GitLab enterprise repository - with .git suffix",
			url:  "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git",
			expected: &RepositoryInfo{
				Provider:     "gitlab",
				Instance:     "gitlab-master.nvidia.com",
				Namespace:    "chat-labs/OpenSource",
				ProjectName:  "rag",
				FullName:     "chat-labs/OpenSource/rag",
				CloneURL:     "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git",
				HTMLURL:      "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
				APIBaseURL:   "https://gitlab-master.nvidia.com/api/v4",
				IsEnterprise: true,
			},
		},

		{
			name: "GitHub Enterprise",
			url:  "https://github.enterprise.com/org/repo",
			expected: &RepositoryInfo{
				Provider:     "github",
				Instance:     "github.enterprise.com",
				Namespace:    "org",
				ProjectName:  "repo",
				FullName:     "org/repo",
				CloneURL:     "https://github.enterprise.com/org/repo.git",
				HTMLURL:      "https://github.enterprise.com/org/repo",
				APIBaseURL:   "https://github.enterprise.com/api/v3",
				IsEnterprise: true,
			},
		},
		{
			name: "GitLab with simple namespace",
			url:  "https://gitlab-internal.company.com/team/project",
			expected: &RepositoryInfo{
				Provider:     "gitlab",
				Instance:     "gitlab-internal.company.com",
				Namespace:    "team",
				ProjectName:  "project",
				FullName:     "team/project",
				CloneURL:     "https://gitlab-internal.company.com/team/project.git",
				HTMLURL:      "https://gitlab-internal.company.com/team/project",
				APIBaseURL:   "https://gitlab-internal.company.com/api/v4",
				IsEnterprise: true,
			},
		},
		{
			name:    "SSH URL (not supported)",
			url:     "git@gitlab-master.nvidia.com:chat-labs/OpenSource/rag.git",
			wantErr: true,
		},
		{
			name:    "HTTP URL (not supported)",
			url:     "http://github.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "Invalid URL",
			url:     "not-a-valid-url",
			wantErr: true,
		},
		{
			name:    "Empty path",
			url:     "https://github.com/",
			wantErr: true,
		},
		{
			name:    "Missing project name",
			url:     "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseRepositoryURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for URL %s, but got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL %s: %v", tt.url, err)
				return
			}

			if result == nil {
				t.Errorf("Expected result for URL %s, but got nil", tt.url)
				return
			}

			// Compare fields
			if result.Provider != tt.expected.Provider {
				t.Errorf("Provider: expected %s, got %s", tt.expected.Provider, result.Provider)
			}
			if result.Instance != tt.expected.Instance {
				t.Errorf("Instance: expected %s, got %s", tt.expected.Instance, result.Instance)
			}
			if result.Namespace != tt.expected.Namespace {
				t.Errorf("Namespace: expected %s, got %s", tt.expected.Namespace, result.Namespace)
			}
			if result.ProjectName != tt.expected.ProjectName {
				t.Errorf("ProjectName: expected %s, got %s", tt.expected.ProjectName, result.ProjectName)
			}
			if result.FullName != tt.expected.FullName {
				t.Errorf("FullName: expected %s, got %s", tt.expected.FullName, result.FullName)
			}
			if result.CloneURL != tt.expected.CloneURL {
				t.Errorf("CloneURL: expected %s, got %s", tt.expected.CloneURL, result.CloneURL)
			}
			if result.HTMLURL != tt.expected.HTMLURL {
				t.Errorf("HTMLURL: expected %s, got %s", tt.expected.HTMLURL, result.HTMLURL)
			}
			if result.APIBaseURL != tt.expected.APIBaseURL {
				t.Errorf("APIBaseURL: expected %s, got %s", tt.expected.APIBaseURL, result.APIBaseURL)
			}
			if result.IsEnterprise != tt.expected.IsEnterprise {
				t.Errorf("IsEnterprise: expected %t, got %t", tt.expected.IsEnterprise, result.IsEnterprise)
			}
		})
	}
}

func TestURLParser_GetProviderType(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub public",
			url:      "https://github.com/owner/repo",
			expected: "github",
		},
		{
			name:     "GitLab public",
			url:      "https://gitlab.com/owner/repo",
			expected: "gitlab",
		},
		{
			name:     "NVIDIA GitLab",
			url:      "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
			expected: "gitlab",
		},
		{
			name:     "GitHub Enterprise",
			url:      "https://github.enterprise.com/org/repo",
			expected: "github",
		},
		{
			name:     "Unknown git provider",
			url:      "https://git.example.com/owner/repo",
			expected: "gitlab", // Default fallback
		},
		{
			name:     "SSH URL (not supported)",
			url:      "git@gitlab-master.nvidia.com:chat-labs/OpenSource/rag.git",
			expected: "unknown",
		},
		{
			name:     "HTTP URL (not supported)",
			url:      "http://github.com/owner/repo",
			expected: "unknown",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.GetProviderType(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestURLParser_ValidateRepositoryURL(t *testing.T) {
	parser := NewURLParser()

	validURLs := []string{
		"https://github.com/torvalds/linux",
		"https://gitlab.com/gitlab-org/gitlab",
		"https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
		"https://github.enterprise.com/org/repo",
	}

	invalidURLs := []string{
		"git@gitlab-master.nvidia.com:chat-labs/OpenSource/rag.git", // SSH not supported
		"http://github.com/owner/repo",                              // HTTP not supported
		"not-a-url",
		"https://github.com/",
		"https://github.com/owner",
		"",
		"ftp://invalid.com/repo",
	}

	for _, url := range validURLs {
		t.Run("Valid: "+url, func(t *testing.T) {
			err := parser.ValidateRepositoryURL(url)
			if err != nil {
				t.Errorf("Expected valid URL %s, but got error: %v", url, err)
			}
		})
	}

	for _, url := range invalidURLs {
		t.Run("Invalid: "+url, func(t *testing.T) {
			err := parser.ValidateRepositoryURL(url)
			if err == nil {
				t.Errorf("Expected invalid URL %s, but got no error", url)
			}
		})
	}
}

func TestURLParser_BuildRepoURLs(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name      string
		instance  string
		fullName  string
		provider  string
		expected  *RepositoryInfo
	}{
		{
			name:     "GitHub public",
			instance: "github.com",
			fullName: "owner/repo",
			provider: "github",
			expected: &RepositoryInfo{
				Provider:     "github",
				Instance:     "github.com",
				Namespace:    "owner",
				ProjectName:  "repo",
				FullName:     "owner/repo",
				CloneURL:     "https://github.com/owner/repo.git",
				HTMLURL:      "https://github.com/owner/repo",
				APIBaseURL:   "https://api.github.com",
				IsEnterprise: false,
			},
		},
		{
			name:     "NVIDIA GitLab multi-level",
			instance: "gitlab-master.nvidia.com",
			fullName: "chat-labs/OpenSource/rag",
			provider: "gitlab",
			expected: &RepositoryInfo{
				Provider:     "gitlab",
				Instance:     "gitlab-master.nvidia.com",
				Namespace:    "chat-labs/OpenSource",
				ProjectName:  "rag",
				FullName:     "chat-labs/OpenSource/rag",
				CloneURL:     "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git",
				HTMLURL:      "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
				APIBaseURL:   "https://gitlab-master.nvidia.com/api/v4",
				IsEnterprise: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.BuildRepoURLs(tt.instance, tt.fullName, tt.provider)

			if result.Provider != tt.expected.Provider {
				t.Errorf("Provider: expected %s, got %s", tt.expected.Provider, result.Provider)
			}
			if result.Instance != tt.expected.Instance {
				t.Errorf("Instance: expected %s, got %s", tt.expected.Instance, result.Instance)
			}
			if result.FullName != tt.expected.FullName {
				t.Errorf("FullName: expected %s, got %s", tt.expected.FullName, result.FullName)
			}
			if result.CloneURL != tt.expected.CloneURL {
				t.Errorf("CloneURL: expected %s, got %s", tt.expected.CloneURL, result.CloneURL)
			}
			if result.HTMLURL != tt.expected.HTMLURL {
				t.Errorf("HTMLURL: expected %s, got %s", tt.expected.HTMLURL, result.HTMLURL)
			}
			if result.IsEnterprise != tt.expected.IsEnterprise {
				t.Errorf("IsEnterprise: expected %t, got %t", tt.expected.IsEnterprise, result.IsEnterprise)
			}
		})
	}
}
