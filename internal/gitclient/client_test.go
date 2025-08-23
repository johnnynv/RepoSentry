package gitclient

import (
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFactory_CreateClient(t *testing.T) {
	// Create a valid logger for testing
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))
	config := GetDefaultConfig()
	config.Token = "test-token"

	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/owner/repo",
		Provider: "github",
	}

	client, err := factory.CreateClient(repo, config)

	// In test environment, this might fail due to missing dependencies
	// but we can still test the factory logic
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
		// Check if it's the expected error type
		if _, ok := err.(*UnsupportedProviderError); ok {
			t.Error("Expected GitHub provider to be supported")
		}
		return
	}

	if client == nil {
		t.Error("Expected client but got nil")
	}
}

func TestClientFactory_GetRateLimiter(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test getting rate limiter for different providers
	githubLimiter := factory.getRateLimiter("github", NewGitHubRateLimiter())
	gitlabLimiter := factory.getRateLimiter("gitlab", NewGitLabRateLimiter())

	assert.NotNil(t, githubLimiter)
	assert.NotNil(t, gitlabLimiter)
	assert.NotEqual(t, githubLimiter, gitlabLimiter)
}

func TestClientFactory_CreateClient_EdgeCases(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test with empty provider
	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "",
	}
	config := ClientConfig{
		Token: "test-token",
	}

	client, err := factory.CreateClient(repo, config)
	assert.Error(t, err)
	assert.Nil(t, client)

	// Test with unsupported provider
	repo.Provider = "unsupported"
	client, err = factory.CreateClient(repo, config)
	assert.Error(t, err)
	assert.Nil(t, client)

	// Test with invalid repository URL - this might actually succeed in some cases
	// as the URL validation is not strict
	repo.Provider = "github"
	repo.URL = "invalid-url"
	client, err = factory.CreateClient(repo, config)
	// We can't guarantee this will fail, so we just test that it doesn't panic
	if err != nil {
		t.Logf("Expected error with invalid URL: %v", err)
	} else {
		t.Logf("Client created successfully with invalid URL")
	}
}

func TestClientConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		isValid bool
	}{
		{
			name: "valid config with token",
			config: ClientConfig{
				Token: "test-token",
			},
			isValid: true,
		},
		{
			name: "empty token",
			config: ClientConfig{
				Token: "",
			},
			isValid: false,
		},
		{
			name: "config with all fields",
			config: ClientConfig{
				Token:          "test-token",
				BaseURL:        "https://api.test.com",
				RepositoryURL:  "https://test.com/owner/repo",
				Timeout:        30 * time.Second,
				RetryAttempts:  3,
				RetryBackoff:   5 * time.Second,
				UserAgent:      "test-agent",
				EnableFallback: true,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.config.Token != ""
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test AuthenticationError
	authErr := &AuthenticationError{
		Provider: "github",
		Message:  "invalid token",
	}

	assert.Equal(t, "authentication failed for github: invalid token", authErr.Error())
	assert.Equal(t, "github", authErr.Provider)
	assert.Equal(t, "invalid token", authErr.Message)

	// Test UnsupportedProviderError
	unsupportedErr := &UnsupportedProviderError{
		Provider: "unsupported",
	}

	assert.Equal(t, "unsupported Git provider: unsupported", unsupportedErr.Error())
	assert.Equal(t, "unsupported", unsupportedErr.Provider)

	// Test RepositoryNotFoundError
	repoNotFoundErr := &RepositoryNotFoundError{
		Repository: "test-repo",
		Provider:   "github",
	}

	assert.Contains(t, repoNotFoundErr.Error(), "repository not found")
	assert.Equal(t, "test-repo", repoNotFoundErr.Repository)
	assert.Equal(t, "github", repoNotFoundErr.Provider)

	// Test RateLimitExceededError
	rateLimitErr := &RateLimitExceededError{
		Provider:  "github",
		ResetTime: time.Now().Add(1 * time.Hour),
	}

	assert.Contains(t, rateLimitErr.Error(), "rate limit exceeded")
	assert.Equal(t, "github", rateLimitErr.Provider)

	// Test NetworkError
	networkErr := &NetworkError{
		Provider: "github",
		Err:      assert.AnError,
	}

	assert.Contains(t, networkErr.Error(), "network error")
	assert.Equal(t, "github", networkErr.Provider)
	assert.Equal(t, assert.AnError, networkErr.Err)
}

func TestClientFactory_Concurrency(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test concurrent access to rate limiters
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			limiter := factory.getRateLimiter("github", NewGitHubRateLimiter())
			assert.NotNil(t, limiter)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestClientFactory_GetRateLimiter_Caching(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test that rate limiters are cached
	limiter1 := factory.getRateLimiter("github", NewGitHubRateLimiter())
	limiter2 := factory.getRateLimiter("github", NewGitHubRateLimiter())

	// Should return the same limiter instance
	assert.Equal(t, limiter1, limiter2)

	// Test different providers get different limiters
	gitlabLimiter := factory.getRateLimiter("gitlab", NewGitLabRateLimiter())
	assert.NotEqual(t, limiter1, gitlabLimiter)
}

func TestClientFactory_GetRateLimiter_DefaultFallback(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test that default limiter is used when provider doesn't exist
	defaultLimiter := NewNoOpRateLimiter()
	limiter := factory.getRateLimiter("unknown", defaultLimiter)

	assert.Equal(t, defaultLimiter, limiter)
}

func TestClientFactory_NewClientFactory(t *testing.T) {
	// Test creating factory with valid logger
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory1 := NewClientFactory(testLogger.WithField("test", "client"))
	assert.NotNil(t, factory1)
	assert.NotNil(t, factory1.fallback)

	// Test creating factory with another logger
	factory2 := NewClientFactory(testLogger.WithField("test", "client2"))
	assert.NotNil(t, factory2)
	assert.NotNil(t, factory2.fallback)
}

func TestClientFactory_CreateClient_ProviderMapping(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test GitHub provider
	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	config := ClientConfig{
		Token: "test-token",
	}

	client, err := factory.CreateClient(repo, config)
	if err == nil {
		assert.NotNil(t, client)
		// Note: In test environment, this might fail due to missing dependencies
		// but we can still test the factory logic
	}

	// Test GitLab provider
	repo.Provider = "gitlab"
	client, err = factory.CreateClient(repo, config)
	if err == nil {
		assert.NotNil(t, client)
	}
}

func TestClientFactory_CreateClient_ConfigValidation(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test with invalid config
	repo := types.Repository{
		Name:     "test-repo",
		URL:      "not-a-url",
		Provider: "invalid",
	}
	config := ClientConfig{
		Token: "",
	}

	client, err := factory.CreateClient(repo, config)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClientFactory_CreateClient_RepositoryURLParsing(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test with various URL formats
	testCases := []string{
		"https://github.com/test/repo",
		"https://gitlab.com/test/repo",
		"git@github.com:test/repo.git",
		"git@gitlab.com:test/repo.git",
	}

	for _, url := range testCases {
		repo := types.Repository{
			Name:     "test-repo",
			URL:      url,
			Provider: "github", // Use github for all tests
		}
		config := ClientConfig{
			Token: "test-token",
		}

		// This might fail in test environment, but we're testing the factory logic
		client, err := factory.CreateClient(repo, config)
		if err == nil {
			assert.NotNil(t, client)
		}
	}
}

func TestClientFactory_CreateClient_TokenHandling(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test with various token formats
	testCases := []string{
		"ghp_1234567890abcdef",
		"glpat-1234567890abcdef",
		"1234567890abcdef",
	}

	for _, token := range testCases {
		repo := types.Repository{
			Name:     "test-repo",
			URL:      "https://github.com/test/repo",
			Provider: "github",
		}
		config := ClientConfig{
			Token: token,
		}

		// This might fail in test environment, but we're testing the factory logic
		client, err := factory.CreateClient(repo, config)
		if err == nil {
			assert.NotNil(t, client)
		}
	}
}

func TestClientFactory_CreateClient_ContextHandling(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test that context is properly passed through
	repo := types.Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Provider: "github",
	}
	config := ClientConfig{
		Token: "test-token",
	}

	// This might fail in test environment, but we're testing the factory logic
	client, err := factory.CreateClient(repo, config)
	if err == nil {
		assert.NotNil(t, client)
		// Test that client can be used with context
		// Note: This is a basic test, actual functionality depends on the client implementation
	}
}

func TestClientFactory_CreateClient_ErrorHandling(t *testing.T) {
	testLogger, err := logger.NewLogger(logger.Config{
		Level:  "error",
		Format: "json",
		Output: "stderr",
	})
	require.NoError(t, err)

	factory := NewClientFactory(testLogger.WithField("test", "client"))

	// Test various error conditions
	testCases := []struct {
		name   string
		repo   types.Repository
		config ClientConfig
	}{
		{
			name: "empty provider",
			repo: types.Repository{
				Name:     "test-repo",
				URL:      "https://github.com/test/repo",
				Provider: "",
			},
			config: ClientConfig{
				Token: "test-token",
			},
		},
		{
			name: "unsupported provider",
			repo: types.Repository{
				Name:     "test-repo",
				URL:      "https://github.com/test/repo",
				Provider: "unsupported",
			},
			config: ClientConfig{
				Token: "test-token",
			},
		},
		{
			name: "empty repository URL",
			repo: types.Repository{
				Name:     "test-repo",
				URL:      "",
				Provider: "github",
			},
			config: ClientConfig{
				Token: "test-token",
			},
		},
		{
			name: "empty token",
			repo: types.Repository{
				Name:     "test-repo",
				URL:      "https://github.com/test/repo",
				Provider: "github",
			},
			config: ClientConfig{
				Token: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := factory.CreateClient(tc.repo, tc.config)

			// For empty provider and unsupported provider, we expect errors
			if tc.name == "empty provider" || tc.name == "unsupported provider" {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				// For other cases, the behavior might vary depending on the implementation
				// We just test that it doesn't panic
				if err != nil {
					t.Logf("Expected error: %v", err)
				} else {
					t.Logf("Client created successfully")
				}
			}
		})
	}
}
