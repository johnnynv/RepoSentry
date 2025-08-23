package trigger

import (
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestEventTransformer_TransformToGitHub(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	tests := []struct {
		name     string
		event    types.Event
		expected GitHubPayload
		wantErr  bool
	}{
		{
			name: "GitHub event with repository URL",
			event: types.Event{
				ID:         "event_123",
				Type:       types.EventTypeBranchUpdated,
				Repository: "torvalds/linux",
				Branch:     "master",
				CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
				PrevCommit: "1234567890abcdef1234567890abcdef12345678",
				Provider:   "github",
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Metadata: map[string]string{
					"repository_url": "https://github.com/torvalds/linux",
				},
			},
			expected: GitHubPayload{
				Repository: GitHubRepository{
					Name:     "torvalds/linux",
					FullName: "torvalds/linux",
					CloneURL: "https://github.com/torvalds/linux.git",
					HTMLURL:  "https://github.com/torvalds/linux",
				},
				After:    "abcd1234567890abcdef1234567890abcdef1234",
				ShortSHA: "abcd1234",
				Ref:      "refs/heads/master",
				Before:   "1234567890abcdef1234567890abcdef12345678",
			},
		},
		{
			name: "NVIDIA GitLab event with multi-level namespace",
			event: types.Event{
				ID:         "event_456",
				Type:       types.EventTypeBranchCreated,
				Repository: "chat-labs/OpenSource/rag",
				Branch:     "main",
				CommitSHA:  "xyz789xyz789xyz789xyz789xyz789xyz789xyz7",
				Provider:   "gitlab",
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Metadata: map[string]string{
					"repository_url": "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
					"protected":      "true",
				},
			},
			expected: GitHubPayload{
				Repository: GitHubRepository{
					Name:     "chat-labs/OpenSource/rag",
					FullName: "chat-labs/OpenSource/rag",
					CloneURL: "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git",
					HTMLURL:  "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
				},
				After:    "xyz789xyz789xyz789xyz789xyz789xyz789xyz7",
				ShortSHA: "xyz789xy",
				Ref:      "refs/heads/main",
				Before:   "",
			},
		},
		{
			name: "Event without repository URL (fallback)",
			event: types.Event{
				ID:         "event_789",
				Type:       types.EventTypeBranchUpdated,
				Repository: "owner/repo",
				Branch:     "develop",
				CommitSHA:  "fedcba9876543210fedcba9876543210fedcba98",
				Provider:   "github",
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Metadata:   map[string]string{},
			},
			expected: GitHubPayload{
				Repository: GitHubRepository{
					Name:     "owner/repo",
					FullName: "owner/repo",
					CloneURL: "https://github.com/owner/repo.git",
					HTMLURL:  "https://github.com/owner/repo",
				},
				After:    "fedcba9876543210fedcba9876543210fedcba98",
				ShortSHA: "fedcba98",
				Ref:      "refs/heads/develop",
				Before:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.TransformToGitHub(tt.event)

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

			// Compare key fields
			if result.Repository.Name != tt.expected.Repository.Name {
				t.Errorf("Repository.Name: expected %s, got %s", tt.expected.Repository.Name, result.Repository.Name)
			}
			if result.Repository.CloneURL != tt.expected.Repository.CloneURL {
				t.Errorf("Repository.CloneURL: expected %s, got %s", tt.expected.Repository.CloneURL, result.Repository.CloneURL)
			}
			if result.After != tt.expected.After {
				t.Errorf("After: expected %s, got %s", tt.expected.After, result.After)
			}
			if result.ShortSHA != tt.expected.ShortSHA {
				t.Errorf("ShortSHA: expected %s, got %s", tt.expected.ShortSHA, result.ShortSHA)
			}
			if result.Ref != tt.expected.Ref {
				t.Errorf("Ref: expected %s, got %s", tt.expected.Ref, result.Ref)
			}
		})
	}
}

func TestEventTransformer_TransformToTekton(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	event := types.Event{
		ID:         "event_tekton_123",
		Type:       types.EventTypeBranchUpdated,
		Repository: "chat-labs/OpenSource/rag",
		Branch:     "main",
		CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
		Provider:   "gitlab",
		Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Metadata: map[string]string{
			"repository_url": "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
			"protected":      "true",
			"author_name":    "John Doe",
		},
	}

	result, err := transformer.TransformToTekton(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify GitHubPayload portion
	if result.Repository.Name != "chat-labs/OpenSource/rag" {
		t.Errorf("Repository.Name: expected %s, got %s", "chat-labs/OpenSource/rag", result.Repository.Name)
	}
	if result.After != "abcd1234567890abcdef1234567890abcdef1234" {
		t.Errorf("After: expected %s, got %s", "abcd1234567890abcdef1234567890abcdef1234", result.After)
	}
	if result.ShortSHA != "abcd1234" {
		t.Errorf("ShortSHA: expected %s, got %s", "abcd1234", result.ShortSHA)
	}

	// Verify Tekton-specific fields
	if result.Source != "reposentry" {
		t.Errorf("Source: expected %s, got %s", "reposentry", result.Source)
	}
	if result.EventID != event.ID {
		t.Errorf("EventID: expected %s, got %s", event.ID, result.EventID)
	}

	// Verify metadata
	if result.Metadata["event_type"] != string(event.Type) {
		t.Errorf("Metadata[event_type]: expected %s, got %v", string(event.Type), result.Metadata["event_type"])
	}
	if result.Metadata["provider"] != event.Provider {
		t.Errorf("Metadata[provider]: expected %s, got %v", event.Provider, result.Metadata["provider"])
	}
	if result.Metadata["protected"] != true {
		t.Errorf("Metadata[protected]: expected true, got %v", result.Metadata["protected"])
	}
	if result.Metadata["custom_author_name"] != "John Doe" {
		t.Errorf("Metadata[custom_author_name]: expected %s, got %v", "John Doe", result.Metadata["custom_author_name"])
	}
}

func TestEventTransformer_TransformToGeneric(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	event := types.Event{
		ID:         "event_generic_123",
		Type:       types.EventTypeBranchDeleted,
		Repository: "owner/repo",
		Branch:     "feature-branch",
		CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
		Provider:   "github",
		Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Metadata: map[string]string{
			"repository_url": "https://github.com/owner/repo",
			"custom_field":   "custom_value",
		},
	}

	result, err := transformer.TransformToGeneric(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify basic fields
	if result.Event.ID != event.ID {
		t.Errorf("Event.ID: expected %s, got %s", event.ID, result.Event.ID)
	}
	if result.Source != "reposentry" {
		t.Errorf("Source: expected %s, got %s", "reposentry", result.Source)
	}

	// Verify repository map
	if result.Repository["name"] != event.Repository {
		t.Errorf("Repository[name]: expected %s, got %v", event.Repository, result.Repository["name"])
	}
	if result.Repository["provider"] != event.Provider {
		t.Errorf("Repository[provider]: expected %s, got %v", event.Provider, result.Repository["provider"])
	}

	// Verify metadata
	if result.Metadata["event_type"] != string(event.Type) {
		t.Errorf("Metadata[event_type]: expected %s, got %v", string(event.Type), result.Metadata["event_type"])
	}
	if result.Metadata["custom_field"] != "custom_value" {
		t.Errorf("Metadata[custom_field]: expected %s, got %v", "custom_value", result.Metadata["custom_field"])
	}
}

func TestEventTransformer_GetShortSHA(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	tests := []struct {
		input    string
		expected string
	}{
		{"abcd1234567890abcdef1234567890abcdef1234", "abcd1234"},
		{"12345678", "12345678"},
		{"abc", "abc"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := transformer.getShortSHA(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEventTransformer_GetBranchRef(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	tests := []struct {
		input    string
		expected string
	}{
		{"main", "refs/heads/main"},
		{"feature/auth", "refs/heads/feature/auth"},
		{"refs/heads/main", "refs/heads/main"},
		{"refs/tags/v1.0", "refs/tags/v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := transformer.getBranchRef(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEventTransformer_ExtractRepositoryInfo_NvidiaGitLab(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	// Test with NVIDIA GitLab URL
	event := types.Event{
		ID:         "nvidia_test",
		Repository: "chat-labs/OpenSource/rag",
		Provider:   "gitlab",
		Metadata: map[string]string{
			"repository_url": "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag",
		},
	}

	info, err := transformer.extractRepositoryInfo(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if info.FullName != "chat-labs/OpenSource/rag" {
		t.Errorf("FullName: expected %s, got %s", "chat-labs/OpenSource/rag", info.FullName)
	}
	if info.CloneURL != "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git" {
		t.Errorf("CloneURL: expected %s, got %s", "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag.git", info.CloneURL)
	}
	if info.HTMLURL != "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag" {
		t.Errorf("HTMLURL: expected %s, got %s", "https://gitlab-master.nvidia.com/chat-labs/OpenSource/rag", info.HTMLURL)
	}
}

func TestEventTransformer_CreateCommitFromEvent(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "transformer"))

	tests := []struct {
		name     string
		event    types.Event
		expected *GitHubCommit
	}{
		{
			name: "Event with commit information",
			event: types.Event{
				CommitSHA: "abcd1234567890abcdef1234567890abcdef1234",
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Metadata: map[string]string{
					"commit_message": "Fix bug in authentication",
					"author_name":    "John Doe",
					"author_email":   "john@example.com",
					"commit_url":     "https://github.com/owner/repo/commit/abcd1234",
				},
			},
			expected: &GitHubCommit{
				ID:        "abcd1234567890abcdef1234567890abcdef1234",
				Message:   "Fix bug in authentication",
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				URL:       "https://github.com/owner/repo/commit/abcd1234",
				Author: GitHubUser{
					Name:  "John Doe",
					Email: "john@example.com",
				},
			},
		},
		{
			name: "Event without commit SHA",
			event: types.Event{
				CommitSHA: "",
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.createCommitFromEvent(tt.event)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected commit, got nil")
				return
			}

			if result.ID != tt.expected.ID {
				t.Errorf("ID: expected %s, got %s", tt.expected.ID, result.ID)
			}
			if result.Message != tt.expected.Message {
				t.Errorf("Message: expected %s, got %s", tt.expected.Message, result.Message)
			}
			if result.Author.Name != tt.expected.Author.Name {
				t.Errorf("Author.Name: expected %s, got %s", tt.expected.Author.Name, result.Author.Name)
			}
		})
	}
}

func TestEventTransformer_TransformToCloudEvents(t *testing.T) {
	transformer := NewEventTransformer(logger.GetDefaultLogger().WithField("test", "cloudevents"))

	tests := []struct {
		name     string
		event    types.Event
		expected CloudEventsPayload
		wantErr  bool
	}{
		{
			name: "GitHub event with repository URL",
			event: types.Event{
				ID:         "event_123",
				Type:       types.EventTypeBranchUpdated,
				Repository: "test-repo",
				Branch:     "main",
				CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
				Provider:   "github",
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Metadata: map[string]string{
					"repository_url": "https://github.com/owner/test-repo",
				},
			},
			expected: CloudEventsPayload{
				SpecVersion:     "1.0",
				Type:            "dev.reposentry.repository.branch_updated",
				Source:          "reposentry/github",
				DataContentType: "application/json",
				Data: CloudEventsData{
					Repository: CloudEventsRepository{
						Provider:     "github",
						Organization: "owner",
						Name:         "test-repo",
						FullName:     "owner/test-repo",
						URL:          "https://github.com/owner/test-repo",
					},
					Branch: CloudEventsBranch{
						Name: "main",
						Ref:  "refs/heads/main",
					},
					Commit: CloudEventsCommit{
						SHA:      "abcd1234567890abcdef1234567890abcdef1234",
						ShortSHA: "abcd1234",
					},
					Event: CloudEventsEvent{
						Type:          "branch_updated",
						TriggerSource: "reposentry",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.TransformToCloudEvents(tt.event)

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

			if result.SpecVersion != tt.expected.SpecVersion {
				t.Errorf("Expected SpecVersion %s, got %s", tt.expected.SpecVersion, result.SpecVersion)
			}

			if result.Type != tt.expected.Type {
				t.Errorf("Expected Type %s, got %s", tt.expected.Type, result.Type)
			}

			if result.Source != tt.expected.Source {
				t.Errorf("Expected Source %s, got %s", tt.expected.Source, result.Source)
			}

			if result.Data.Repository.Name != tt.expected.Data.Repository.Name {
				t.Errorf("Expected Repository.Name %s, got %s", tt.expected.Data.Repository.Name, result.Data.Repository.Name)
			}

			if result.Data.Repository.Organization != tt.expected.Data.Repository.Organization {
				t.Errorf("Expected Repository.Organization %s, got %s", tt.expected.Data.Repository.Organization, result.Data.Repository.Organization)
			}

			if result.Data.Branch.Name != tt.expected.Data.Branch.Name {
				t.Errorf("Expected Branch.Name %s, got %s", tt.expected.Data.Branch.Name, result.Data.Branch.Name)
			}

			if result.Data.Commit.ShortSHA != tt.expected.Data.Commit.ShortSHA {
				t.Errorf("Expected Commit.ShortSHA %s, got %s", tt.expected.Data.Commit.ShortSHA, result.Data.Commit.ShortSHA)
			}
		})
	}
}
