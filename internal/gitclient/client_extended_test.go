package gitclient

import (
	"context"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Test the extended GitClient interface methods
func TestGitClientExtendedInterface(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{
			name:     "GitHub client implements extended interface",
			provider: "github",
		},
		{
			name:     "GitLab client implements extended interface",
			provider: "gitlab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			testLogger, err := logger.NewLogger(logger.Config{
				Level:  "info",
				Format: "text",
			})
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Create mock repository
			repo := types.Repository{
				Name:     "test-repo",
				URL:      "https://github.com/test/repo",
				Provider: tt.provider,
			}

			// Create client factory
			factory := NewClientFactory(testLogger.WithField("test", "client"))
			config := GetDefaultConfig()
			config.Token = "test-token"

			// Create client
			client, err := factory.CreateClient(repo, config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			// Test that client implements extended interface
			extendedClient, ok := client.(interface {
				ListFiles(ctx context.Context, repo types.Repository, commitSHA, path string) ([]string, error)
				GetFileContent(ctx context.Context, repo types.Repository, commitSHA, filePath string) ([]byte, error)
				CheckDirectoryExists(ctx context.Context, repo types.Repository, commitSHA, dirPath string) (bool, error)
			})

			if !ok {
				t.Errorf("Client does not implement extended interface methods")
			}

			// Verify methods exist (we can't test actual functionality without real API)
			ctx := context.Background()
			
			// Test ListFiles method exists
			_, err = extendedClient.ListFiles(ctx, repo, "main", ".tekton")
			// We expect this to fail due to invalid credentials, but method should exist
			if err == nil {
				t.Log("ListFiles method callable (unexpected success - might indicate mock data)")
			}

			// Test GetFileContent method exists  
			_, err = extendedClient.GetFileContent(ctx, repo, "main", ".tekton/pipeline.yaml")
			// We expect this to fail due to invalid credentials, but method should exist
			if err == nil {
				t.Log("GetFileContent method callable (unexpected success - might indicate mock data)")
			}

			// Test CheckDirectoryExists method exists
			_, err = extendedClient.CheckDirectoryExists(ctx, repo, "main", ".tekton")
			// We expect this to fail due to invalid credentials, but method should exist
			if err == nil {
				t.Log("CheckDirectoryExists method callable (unexpected success - might indicate mock data)")
			}
		})
	}
}

func TestFallbackClientExtendedMethods(t *testing.T) {
	// Create test logger
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "info",
		Format: "text",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	fallbackClient := NewFallbackClient(testLogger.WithField("test", "fallback"))
	defer fallbackClient.Close()

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}

	ctx := context.Background()

	// Test ListFiles - should return error (not implemented)
	_, listErr := fallbackClient.ListFiles(ctx, repo, "main", ".tekton")
	if listErr == nil {
		t.Error("Expected error for unimplemented ListFiles in fallback client")
	}
	if listErr.Error() != "ListFiles not implemented in fallback client - API client required" {
		t.Errorf("Unexpected error message: %v", listErr)
	}

	// Test GetFileContent - should return error (not implemented)
	_, contentErr := fallbackClient.GetFileContent(ctx, repo, "main", ".tekton/pipeline.yaml")
	if contentErr == nil {
		t.Error("Expected error for unimplemented GetFileContent in fallback client")
	}
	if contentErr.Error() != "GetFileContent not implemented in fallback client - API client required" {
		t.Errorf("Unexpected error message: %v", contentErr)
	}

	// Test CheckDirectoryExists - should return error (not implemented)
	_, dirErr := fallbackClient.CheckDirectoryExists(ctx, repo, "main", ".tekton")
	if dirErr == nil {
		t.Error("Expected error for unimplemented CheckDirectoryExists in fallback client")
	}
	if dirErr.Error() != "CheckDirectoryExists not implemented in fallback client - API client required" {
		t.Errorf("Unexpected error message: %v", dirErr)
	}
}

func TestGitHubBase64Decode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple base64 content",
			input:    "SGVsbG8gV29ybGQ=", // "Hello World"
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Base64 with newlines",
			input:    "SGVsbG8g\nV29ybGQ=", // "Hello World" with newline
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Base64 with spaces",
			input:    "SGVsbG8g V29ybGQ=", // "Hello World" with space
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Invalid base64",
			input:    "invalid!!!",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := base64DecodeContent(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestGitLabProjectPathParsing(t *testing.T) {
	// Create a GitLab client for testing
	config := GetDefaultConfig()
	config.Token = "test-token"
	
	client := &GitLabClient{
		config:  config,
		baseURL: "https://gitlab.com/api/v4",
	}

	tests := []struct {
		name     string
		repoURL  string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple GitLab URL",
			repoURL:  "https://gitlab.com/group/project",
			expected: "group/project",
			wantErr:  false,
		},
		{
			name:     "GitLab URL with .git suffix",
			repoURL:  "https://gitlab.com/group/project.git",
			expected: "group/project",
			wantErr:  false,
		},
		{
			name:     "GitLab subgroup URL",
			repoURL:  "https://gitlab-master.nvidia.com/group/subgroup/project",
			expected: "group/subgroup/project",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			repoURL:  "not-a-url",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty path",
			repoURL:  "https://gitlab.com/",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseProjectPath(tt.repoURL)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
