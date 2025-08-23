package runtime

import (
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestDefaultRuntimeFactory_CreateRuntime(t *testing.T) {
	factory := NewDefaultRuntimeFactory()

	tests := []struct {
		name    string
		config  *types.Config
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &types.Config{
				App: types.AppConfig{
					Name:    "test-reposentry",
					DataDir: "/tmp/test",
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: "/tmp/test-reposentry-factory.db",
					},
				},
				Polling: types.PollingConfig{
					Interval: 1 * time.Minute,
				},
				Tekton: types.TektonConfig{
					EventListenerURL: "http://localhost:8080/webhook",
					Timeout:          10 * time.Second,
				},
				Repositories: []types.Repository{
					{
						Name: "test-repo",
						URL:  "https://github.com/owner/repo",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil configuration",
			config:  nil,
			wantErr: true,
		},
		{
			name: "Missing app name",
			config: &types.Config{
				App: types.AppConfig{
					DataDir: "/tmp/test",
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: ":memory:",
					},
				},
				Repositories: []types.Repository{
					{
						Name: "test-repo",
						URL:  "https://github.com/owner/repo",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing data directory",
			config: &types.Config{
				App: types.AppConfig{
					Name: "test-reposentry",
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: ":memory:",
					},
				},
				Repositories: []types.Repository{
					{
						Name: "test-repo",
						URL:  "https://github.com/owner/repo",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid health check port",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 99999, // Invalid port
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: ":memory:",
					},
				},
				Repositories: []types.Repository{
					{
						Name: "test-repo",
						URL:  "https://github.com/owner/repo",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "No repositories",
			config: &types.Config{
				App: types.AppConfig{
					Name:    "test-reposentry",
					DataDir: "/tmp/test",
				},
				Storage: types.StorageConfig{
					SQLite: types.SQLiteConfig{
						Path: ":memory:",
					},
				},
				Repositories: []types.Repository{}, // Empty
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test logger manager for factory
			loggerManager, _ := logger.NewManager(logger.Config{
				Level:  "error",
				Format: "json",
				Output: "stderr",
			})

			runtime, err := factory.CreateRuntime(tt.config, loggerManager)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if runtime != nil {
					t.Error("Expected nil runtime on error")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if runtime == nil {
				t.Error("Expected runtime but got nil")
				return
			}

			// Verify runtime can be type-asserted to RuntimeManager
			if _, ok := runtime.(*RuntimeManager); !ok {
				t.Error("Expected RuntimeManager implementation")
			}
		})
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *types.Config
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 8080,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid configuration with port 0 (disabled)",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 0,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing app name",
			config: &types.Config{
				App: types.AppConfig{
					DataDir:         "/tmp/test",
					HealthCheckPort: 8080,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing data directory",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					HealthCheckPort: 8080,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: true,
		},
		{
			name: "Negative health check port",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: -1,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: true,
		},
		{
			name: "Health check port too high",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 70000,
				},
				Repositories: []types.Repository{
					{Name: "repo1", URL: "https://github.com/owner/repo1"},
				},
			},
			wantErr: true,
		},
		{
			name: "No repositories",
			config: &types.Config{
				App: types.AppConfig{
					Name:            "test-reposentry",
					DataDir:         "/tmp/test",
					HealthCheckPort: 8080,
				},
				Repositories: []types.Repository{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRuntimeConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
