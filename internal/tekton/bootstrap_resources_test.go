package tekton

import (
	"strings"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Helper function to create generator and config for testing
func createTestGeneratorAndConfig() (*BootstrapPipelineGenerator, *BootstrapPipelineConfig) {
	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	config := &BootstrapPipelineConfig{
		Repository: types.Repository{
			Name: "test-repo",
			URL:  "https://github.com/test-org/test-repo",
		},
		Namespace: "test-namespace",
	}

	// Set defaults
	err := generator.setDefaults(config)
	if err != nil {
		panic("Failed to set defaults in test helper: " + err.Error())
	}

	return generator, config
}

func TestGenerateServiceAccount(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	yaml, err := generator.generateServiceAccount(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify YAML structure
	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Error("Missing apiVersion")
	}

	if !strings.Contains(yaml, "kind: ServiceAccount") {
		t.Error("Missing kind")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-sa") {
		t.Error("Missing ServiceAccount name")
	}

	if !strings.Contains(yaml, "namespace: "+config.Namespace) {
		t.Error("Namespace not included")
	}

	// Verify labels
	if !strings.Contains(yaml, "reposentry.io/type: \"bootstrap\"") {
		t.Error("Missing type label")
	}

	if !strings.Contains(yaml, "reposentry.io/repository: \"test-repo\"") {
		t.Error("Missing repository label")
	}
}

func TestGenerateRoleBinding(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	yaml, err := generator.generateRoleBinding(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify YAML structure
	if !strings.Contains(yaml, "apiVersion: rbac.authorization.k8s.io/v1") {
		t.Error("Missing apiVersion")
	}

	if !strings.Contains(yaml, "kind: RoleBinding") {
		t.Error("Missing kind")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-binding") {
		t.Error("Missing RoleBinding name")
	}

	// Verify role reference
	if !strings.Contains(yaml, "roleRef:") {
		t.Error("Missing roleRef")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-role") {
		t.Error("Missing role reference")
	}

	// Verify subjects
	if !strings.Contains(yaml, "subjects:") {
		t.Error("Missing subjects")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-sa") {
		t.Error("Missing ServiceAccount subject")
	}

	if !strings.Contains(yaml, "namespace: "+config.Namespace) {
		t.Error("Subject namespace not included")
	}
}

func TestGenerateResourceQuota(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	yaml, err := generator.generateResourceQuota(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify YAML structure
	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Error("Missing apiVersion")
	}

	if !strings.Contains(yaml, "kind: ResourceQuota") {
		t.Error("Missing kind")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-quota") {
		t.Error("Missing ResourceQuota name")
	}

	// Verify spec
	if !strings.Contains(yaml, "spec:") {
		t.Error("Missing spec")
	}

	if !strings.Contains(yaml, "hard:") {
		t.Error("Missing hard limits")
	}

	// Check for resource limits
	resourceLimits := []string{
		"requests.cpu:",
		"requests.memory:",
		"limits.cpu:",
		"limits.memory:",
		"persistentvolumeclaims:",
		"pods:",
		"count/pipelineruns.tekton.dev:",
		"count/taskruns.tekton.dev:",
	}

	for _, limit := range resourceLimits {
		if !strings.Contains(yaml, limit) {
			t.Errorf("Missing resource limit: %s", limit)
		}
	}
}

func TestGenerateNetworkPolicy(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	yaml, err := generator.generateNetworkPolicy(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify YAML structure
	if !strings.Contains(yaml, "apiVersion: networking.k8s.io/v1") {
		t.Error("Missing apiVersion")
	}

	if !strings.Contains(yaml, "kind: NetworkPolicy") {
		t.Error("Missing kind")
	}

	if !strings.Contains(yaml, "name: reposentry-bootstrap-netpol") {
		t.Error("Missing NetworkPolicy name")
	}

	// Verify spec
	if !strings.Contains(yaml, "spec:") {
		t.Error("Missing spec")
	}

	if !strings.Contains(yaml, "podSelector:") {
		t.Error("Missing podSelector")
	}

	if !strings.Contains(yaml, "policyTypes:") {
		t.Error("Missing policyTypes")
	}

	// Verify ingress and egress rules
	if !strings.Contains(yaml, "ingress:") {
		t.Error("Missing ingress rules")
	}

	if !strings.Contains(yaml, "egress:") {
		t.Error("Missing egress rules")
	}

	// Check for DNS and API server access
	if !strings.Contains(yaml, "ports:") {
		t.Error("Missing port definitions")
	}

	if !strings.Contains(yaml, "protocol: TCP") {
		t.Error("Missing TCP protocol")
	}
}

func TestGenerateSupportingResources(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	resources := &BootstrapPipelineResources{
		Namespace: config.Namespace,
	}

	err := generator.generateSupportingResources(config, resources)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all resources were generated
	if resources.ServiceAccount == "" {
		t.Error("ServiceAccount not generated")
	}

	if resources.RoleBinding == "" {
		t.Error("RoleBinding not generated")
	}

	if resources.ResourceQuota == "" {
		t.Error("ResourceQuota not generated")
	}

	if resources.NetworkPolicy == "" {
		t.Error("NetworkPolicy not generated")
	}

	// Verify each resource contains expected namespace
	allResources := []struct {
		name string
		yaml string
	}{
		{"ServiceAccount", resources.ServiceAccount},
		{"RoleBinding", resources.RoleBinding},
		{"ResourceQuota", resources.ResourceQuota},
		{"NetworkPolicy", resources.NetworkPolicy},
	}

	for _, resource := range allResources {
		if !strings.Contains(resource.yaml, config.Namespace) {
			t.Errorf("%s does not contain namespace %s", resource.name, config.Namespace)
		}
	}
}

func TestResourceGeneration_DifferentRepositories(t *testing.T) {
	testCases := []struct {
		name       string
		repository types.Repository
		namespace  string
	}{
		{
			name: "GitHub repository",
			repository: types.Repository{
				Name:     "github-repo",
				URL:      "https://github.com/org/github-repo",
				Provider: "github",
			},
			namespace: "github-namespace",
		},
		{
			name: "GitLab repository",
			repository: types.Repository{
				Name:     "gitlab-repo",
				URL:      "https://gitlab.com/org/gitlab-repo",
				Provider: "gitlab",
			},
			namespace: "gitlab-namespace",
		},
	}

	testLogger := createTestLogger()
	generator := NewBootstrapPipelineGenerator(testLogger)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &BootstrapPipelineConfig{
				Repository: tc.repository,
				Namespace:  tc.namespace,
			}

			// Test all resource generators
			tests := []struct {
				name string
				fn   func(*BootstrapPipelineConfig) (string, error)
			}{
				{"ServiceAccount", generator.generateServiceAccount},
				{"RoleBinding", generator.generateRoleBinding},
				{"ResourceQuota", generator.generateResourceQuota},
				{"NetworkPolicy", generator.generateNetworkPolicy},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					yaml, err := test.fn(config)
					if err != nil {
						t.Fatalf("Unexpected error: %v", err)
					}

					// Verify namespace is included
					if !strings.Contains(yaml, tc.namespace) {
						t.Errorf("Namespace %s not found in %s", tc.namespace, test.name)
					}

					// Verify basic YAML structure
					if !strings.Contains(yaml, "apiVersion:") {
						t.Errorf("Missing apiVersion in %s", test.name)
					}

					if !strings.Contains(yaml, "kind:") {
						t.Errorf("Missing kind in %s", test.name)
					}

					if !strings.Contains(yaml, "metadata:") {
						t.Errorf("Missing metadata in %s", test.name)
					}
				})
			}
		})
	}
}

func TestAllResourcesHaveLabels(t *testing.T) {
	generator, config := createTestGeneratorAndConfig()

	tests := []struct {
		name string
		fn   func(*BootstrapPipelineConfig) (string, error)
	}{
		{"ServiceAccount", generator.generateServiceAccount},
		{"RoleBinding", generator.generateRoleBinding},
		{"ResourceQuota", generator.generateResourceQuota},
		{"NetworkPolicy", generator.generateNetworkPolicy},
	}

	requiredLabels := []string{
		"reposentry.io/type: \"bootstrap\"",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			yaml, err := test.fn(config)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			for _, label := range requiredLabels {
				if !strings.Contains(yaml, label) {
					t.Errorf("Missing required label %s in %s", label, test.name)
				}
			}
		})
	}
}
