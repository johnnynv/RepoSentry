package gitclient

import (
	"context"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestClientFactory_CreateClient(t *testing.T) {
	factory := NewClientFactory()
	config := GetDefaultConfig()
	config.Token = "test-token"

	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{
			name:     "GitHub client",
			provider: "github",
			wantErr:  false,
		},
		{
			name:     "GitLab client",
			provider: "gitlab",
			wantErr:  false,
		},
		{
			name:     "Unsupported provider",
			provider: "bitbucket",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := types.Repository{
				Name:     "test-repo",
				URL:      "https://example.com/test/repo",
				Provider: tt.provider,
				Token:    "test-token",
			}

			client, err := factory.CreateClient(repo, config)
			
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

			if client == nil {
				t.Error("Expected client but got nil")
				return
			}

			if client.GetProvider() != tt.provider {
				t.Errorf("Expected provider %s, got %s", tt.provider, client.GetProvider())
			}

			// Test cleanup
			if err := client.Close(); err != nil {
				t.Errorf("Failed to close client: %v", err)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "Network error",
			err:  &NetworkError{Provider: "test", Err: nil},
			want: true,
		},
		{
			name: "Rate limit error",
			err:  &RateLimitExceededError{Provider: "test"},
			want: false,
		},
		{
			name: "Authentication error",
			err:  &AuthenticationError{Provider: "test"},
			want: false,
		},
		{
			name: "Repository not found error",
			err:  &RepositoryNotFoundError{Repository: "test"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", config.RetryAttempts)
	}

	if config.RetryBackoff != 1*time.Second {
		t.Errorf("Expected 1s retry backoff, got %v", config.RetryBackoff)
	}

	if config.UserAgent != "RepoSentry/1.0" {
		t.Errorf("Expected UserAgent 'RepoSentry/1.0', got %s", config.UserAgent)
	}

	if !config.EnableFallback {
		t.Error("Expected fallback to be enabled by default")
	}
}

// Mock client for testing
type MockGitClient struct {
	provider  string
	branches  []types.Branch
	commitSHA string
	rateLimit *types.RateLimit
	err       error
}

func NewMockGitClient(provider string) *MockGitClient {
	return &MockGitClient{
		provider: provider,
		branches: []types.Branch{
			{Name: "main", CommitSHA: "abc123", Protected: false},
			{Name: "develop", CommitSHA: "def456", Protected: true},
		},
		commitSHA: "abc123",
		rateLimit: &types.RateLimit{
			Limit:     5000,
			Remaining: 4999,
			Reset:     time.Now().Add(time.Hour),
		},
	}
}

func (m *MockGitClient) GetBranches(ctx context.Context, repo types.Repository) ([]types.Branch, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.branches, nil
}

func (m *MockGitClient) GetLatestCommit(ctx context.Context, repo types.Repository, branch string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.commitSHA, nil
}

func (m *MockGitClient) CheckPermissions(ctx context.Context, repo types.Repository) error {
	return m.err
}

func (m *MockGitClient) GetRateLimit(ctx context.Context) (*types.RateLimit, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rateLimit, nil
}

func (m *MockGitClient) GetProvider() string {
	return m.provider
}

func (m *MockGitClient) Close() error {
	return nil
}

func TestMockGitClient(t *testing.T) {
	client := NewMockGitClient("mock")
	ctx := context.Background()

	repo := types.Repository{
		Name:     "test",
		URL:      "https://example.com/test/repo",
		Provider: "mock",
	}

	// Test GetBranches
	branches, err := client.GetBranches(ctx, repo)
	if err != nil {
		t.Errorf("GetBranches failed: %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("Expected 2 branches, got %d", len(branches))
	}

	// Test GetLatestCommit
	commit, err := client.GetLatestCommit(ctx, repo, "main")
	if err != nil {
		t.Errorf("GetLatestCommit failed: %v", err)
	}
	if commit != "abc123" {
		t.Errorf("Expected commit abc123, got %s", commit)
	}

	// Test CheckPermissions
	if err := client.CheckPermissions(ctx, repo); err != nil {
		t.Errorf("CheckPermissions failed: %v", err)
	}

	// Test GetRateLimit
	rateLimit, err := client.GetRateLimit(ctx)
	if err != nil {
		t.Errorf("GetRateLimit failed: %v", err)
	}
	if rateLimit.Limit != 5000 {
		t.Errorf("Expected rate limit 5000, got %d", rateLimit.Limit)
	}

	// Test GetProvider
	if provider := client.GetProvider(); provider != "mock" {
		t.Errorf("Expected provider 'mock', got %s", provider)
	}
}
