package gitclient

import (
	"context"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestFallbackClient_parseLsRemoteOutput(t *testing.T) {
	client := NewFallbackClient(logger.GetDefaultLogger().WithField("test", "fallback"))

	tests := []struct {
		name          string
		output        string
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "Normal output",
			output:        "abc123def456789012345678901234567890abcd\trefs/heads/main\ndef456789012345678901234567890abcdef123\trefs/heads/develop\n1234567890abcdef123456789012345678901234\trefs/heads/feature/test",
			expectedCount: 3,
			expectedFirst: "main",
		},
		{
			name:          "Empty output",
			output:        "",
			expectedCount: 0,
		},
		{
			name:          "Single branch",
			output:        "abc123def456789012345678901234567890abcd\trefs/heads/master",
			expectedCount: 1,
			expectedFirst: "master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Input: %q", tt.output)
			branches, err := client.parseLsRemoteOutput(tt.output)
			if err != nil {
				t.Errorf("parseLsRemoteOutput() error = %v", err)
				return
			}

			if len(branches) != tt.expectedCount {
				t.Errorf("Expected %d branches, got %d", tt.expectedCount, len(branches))
				for i, branch := range branches {
					t.Logf("Branch %d: %s -> %s", i, branch.Name, branch.CommitSHA)
				}
				return
			}

			if tt.expectedCount > 0 {
				if branches[0].Name != tt.expectedFirst {
					t.Errorf("Expected first branch %s, got %s", tt.expectedFirst, branches[0].Name)
				}

				if len(branches[0].CommitSHA) == 0 {
					t.Error("Expected commit SHA to be set")
				}

				// Fallback client can't determine protection status
				if branches[0].Protected {
					t.Error("Expected protected to be false for fallback client")
				}
			}
		})
	}
}

func TestFallbackClient_GetProvider(t *testing.T) {
	client := NewFallbackClient(logger.GetDefaultLogger().WithField("test", "fallback"))

	if provider := client.GetProvider(); provider != "git-fallback" {
		t.Errorf("Expected provider 'git-fallback', got %s", provider)
	}
}

func TestFallbackClient_GetRateLimit(t *testing.T) {
	client := NewFallbackClient(logger.GetDefaultLogger().WithField("test", "fallback"))
	ctx := context.Background()

	rateLimit, err := client.GetRateLimit(ctx)
	if err != nil {
		t.Errorf("GetRateLimit() error = %v", err)
		return
	}

	if rateLimit.Limit != 999999 {
		t.Errorf("Expected unlimited rate limit, got %d", rateLimit.Limit)
	}

	if rateLimit.Remaining != 999999 {
		t.Errorf("Expected unlimited remaining, got %d", rateLimit.Remaining)
	}
}

func TestFallbackClient_Close(t *testing.T) {
	client := NewFallbackClient(logger.GetDefaultLogger().WithField("test", "fallback"))

	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestTestGitAvailability(t *testing.T) {
	ctx := context.Background()

	// This test will pass if git is installed, skip if not
	err := TestGitAvailability(ctx)
	if err != nil {
		t.Skipf("Git not available: %v", err)
	}
}

func TestParseGitVersion(t *testing.T) {
	ctx := context.Background()

	// This test will pass if git is installed, skip if not
	version, err := ParseGitVersion(ctx)
	if err != nil {
		t.Skipf("Git not available: %v", err)
	}

	if version == "" {
		t.Error("Expected non-empty version string")
	}

	t.Logf("Git version: %s", version)
}

func TestValidateGitRepository(t *testing.T) {
	ctx := context.Background()

	// Test with a public repository (this may fail in CI without network access)
	err := ValidateGitRepository(ctx, "https://github.com/torvalds/linux.git")
	if err != nil {
		t.Skipf("Could not validate public repository (network issue?): %v", err)
	}
}

func TestGetRemoteInfo(t *testing.T) {
	ctx := context.Background()

	// Test with a public repository (this may fail in CI without network access)
	info, err := GetRemoteInfo(ctx, "https://github.com/torvalds/linux.git")
	if err != nil {
		t.Skipf("Could not get remote info (network issue?): %v", err)
	}

	if info["url"] != "https://github.com/torvalds/linux.git" {
		t.Errorf("Expected URL to be set correctly")
	}

	if info["branches"] == "" {
		t.Error("Expected branches count to be set")
	}

	if info["tags"] == "" {
		t.Error("Expected tags count to be set")
	}

	t.Logf("Remote info: %+v", info)
}

// Integration test for FallbackClient (requires git and network access)
func TestFallbackClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Check if git is available
	if err := TestGitAvailability(ctx); err != nil {
		t.Skipf("Git not available: %v", err)
	}

	client := NewFallbackClient(logger.GetDefaultLogger().WithField("test", "fallback"))
	repo := types.Repository{
		Name:     "linux",
		URL:      "https://github.com/torvalds/linux.git",
		Provider: "github",
	}

	// Test GetBranches
	branches, err := client.GetBranches(ctx, repo)
	if err != nil {
		t.Skipf("GetBranches failed (network issue?): %v", err)
	}

	if len(branches) == 0 {
		t.Error("Expected at least one branch")
	}

	// Find master branch
	var masterBranch *types.Branch
	for i, branch := range branches {
		if branch.Name == "master" {
			masterBranch = &branches[i]
			break
		}
	}

	if masterBranch == nil {
		t.Skip("Master branch not found, skipping commit test")
	}

	// Test GetLatestCommit
	commitSHA, err := client.GetLatestCommit(ctx, repo, "master")
	if err != nil {
		t.Errorf("GetLatestCommit failed: %v", err)
	}

	if commitSHA == "" {
		t.Error("Expected non-empty commit SHA")
	}

	if commitSHA != masterBranch.CommitSHA {
		t.Errorf("Commit SHA mismatch: GetLatestCommit=%s, GetBranches=%s",
			commitSHA, masterBranch.CommitSHA)
	}

	// Test CheckPermissions
	if err := client.CheckPermissions(ctx, repo); err != nil {
		t.Errorf("CheckPermissions failed: %v", err)
	}

	t.Logf("Successfully tested fallback client with %d branches", len(branches))
}
