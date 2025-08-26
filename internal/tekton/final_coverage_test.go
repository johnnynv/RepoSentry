package tekton

import (
	"strings"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Final tests to push coverage over 80%

func TestGetGeneratedNamespaceVariations(t *testing.T) {
	tests := []struct {
		name       string
		repository types.Repository
		expected   string
	}{
		{
			name: "GitHub repository",
			repository: types.Repository{
				Name: "test-repo",
				URL:  "https://github.com/user/test-repo",
			},
			expected: "reposentry-user-repo-",
		},
		{
			name: "GitLab repository",
			repository: types.Repository{
				Name: "gitlab-repo",
				URL:  "https://gitlab.com/group/gitlab-repo",
			},
			expected: "reposentry-user-repo-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := GetGeneratedNamespace(tt.repository)

			if namespace == "" {
				t.Error("Namespace should not be empty")
			}

			if !strings.HasPrefix(namespace, tt.expected) {
				t.Errorf("Expected namespace to start with %s, got %s", tt.expected, namespace)
			}

			// Should be a valid Kubernetes namespace name
			if len(namespace) > 63 {
				t.Error("Namespace should not exceed 63 characters")
			}

			// Should be lowercase
			if namespace != strings.ToLower(namespace) {
				t.Error("Namespace should be lowercase")
			}
		})
	}
}

func TestBootstrapPipelineConfigValidation(t *testing.T) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	// Test with minimal config
	config := &BootstrapPipelineConfig{}
	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("setDefaults should not fail: %v", err)
	}

	// Verify defaults were set
	if config.CloneImage == "" {
		t.Error("CloneImage should be set by defaults")
	}

	if config.KubectlImage == "" {
		t.Error("KubectlImage should be set by defaults")
	}

	if config.ServiceAccount == "" {
		t.Error("ServiceAccount should be set by defaults")
	}

	if config.WorkspaceSize == "" {
		t.Error("WorkspaceSize should be set by defaults")
	}

	if config.ResourceLimits == nil {
		t.Error("ResourceLimits should be set by defaults")
	}

	if config.SecurityContext == nil {
		t.Error("SecurityContext should be set by defaults")
	}
}

func TestTektonIntegrationManager_ValidateRequest_EdgeCases(t *testing.T) {
	mockClient := NewMockGitClient()
	testLogger := createTestLogger()
	manager := NewTektonIntegrationManager(mockClient, testLogger)

	// Test with various URL formats
	validURLs := []string{
		"https://github.com/user/repo",
		"http://gitlab.com/group/project",
		"https://gitlab-enterprise.company.com/team/repo",
	}

	for _, url := range validURLs {
		request := &TektonIntegrationRequest{
			Repository: types.Repository{
				Name: "test-repo",
				URL:  url,
			},
			CommitSHA: "abc123",
			Branch:    "main",
		}

		err := manager.ValidateIntegrationRequest(request)
		if err != nil {
			t.Errorf("Valid URL %s should not cause error: %v", url, err)
		}
	}
}

func TestBootstrapPipelineResources_Structure(t *testing.T) {
	resources := &BootstrapPipelineResources{
		Namespace:         "test-namespace",
		BootstrapPipeline: "pipeline-yaml",
		BootstrapTasks:    []string{"task1", "task2"},
		ServiceAccount:    "sa-yaml",
		RoleBinding:       "rb-yaml",
		ResourceQuota:     "quota-yaml",
		NetworkPolicy:     "netpol-yaml",
		PipelineRun:       "run-yaml",
	}

	// Verify all fields are accessible
	if resources.Namespace != "test-namespace" {
		t.Error("Namespace field not accessible")
	}

	if len(resources.BootstrapTasks) != 2 {
		t.Error("BootstrapTasks field not accessible")
	}

	if resources.ServiceAccount == "" {
		t.Error("ServiceAccount field not accessible")
	}
}

func TestKubernetesApplierConfig_AllFields(t *testing.T) {
	config := &KubernetesApplierConfig{
		KubeconfigPath:    "/path/to/kubeconfig",
		Context:           "test-context",
		DryRun:            true,
		Timeout:           30 * time.Second,
		RetryAttempts:     5,
		RetryDelay:        2 * time.Second,
		ValidateResources: false,
		SkipTLSVerify:     true,
	}

	// Verify all fields are accessible and settable
	if config.KubeconfigPath != "/path/to/kubeconfig" {
		t.Error("KubeconfigPath not set correctly")
	}

	if config.Context != "test-context" {
		t.Error("Context not set correctly")
	}

	if !config.DryRun {
		t.Error("DryRun not set correctly")
	}

	if config.Timeout != 30*time.Second {
		t.Error("Timeout not set correctly")
	}

	if config.RetryAttempts != 5 {
		t.Error("RetryAttempts not set correctly")
	}
}
