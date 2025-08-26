package tekton

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// KubernetesApplier handles applying Bootstrap Pipeline resources to Kubernetes
type KubernetesApplier struct {
	logger *logger.Entry
	config *KubernetesApplierConfig
}

// KubernetesApplierConfig configures the Kubernetes applier
type KubernetesApplierConfig struct {
	// Kubernetes connection settings
	KubeconfigPath string `json:"kubeconfig_path"`
	Context        string `json:"context"`

	// Apply settings
	DryRun        bool          `json:"dry_run"`
	Timeout       time.Duration `json:"timeout"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`

	// Validation settings
	ValidateResources bool `json:"validate_resources"`
	SkipTLSVerify     bool `json:"skip_tls_verify"`
}

// NewKubernetesApplier creates a new Kubernetes applier
func NewKubernetesApplier(parentLogger *logger.Entry) *KubernetesApplier {
	applierLogger := parentLogger.WithFields(logger.Fields{
		"component": "kubernetes-applier",
	})

	return &KubernetesApplier{
		logger: applierLogger,
		config: getDefaultApplierConfig(),
	}
}

// getDefaultApplierConfig returns default applier configuration
func getDefaultApplierConfig() *KubernetesApplierConfig {
	return &KubernetesApplierConfig{
		DryRun:            false,
		Timeout:           10 * time.Minute,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		ValidateResources: true,
		SkipTLSVerify:     false,
	}
}

// ApplyBootstrapResources applies Bootstrap Pipeline resources to Kubernetes
func (ka *KubernetesApplier) ApplyBootstrapResources(ctx context.Context, resources *BootstrapPipelineResources) error {
	ka.logger.WithFields(logger.Fields{
		"operation": "apply_bootstrap_resources",
		"namespace": resources.Namespace,
	}).Info("Starting Bootstrap Pipeline resources application")

	// Create namespace first
	if err := ka.createNamespace(ctx, resources.Namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Apply resources in order
	applyOrder := []struct {
		name     string
		content  string
		required bool
	}{
		{"ServiceAccount", resources.ServiceAccount, true},
		{"RoleBinding", resources.RoleBinding, true},
		{"ResourceQuota", resources.ResourceQuota, true},
		{"NetworkPolicy", resources.NetworkPolicy, false}, // Optional if NetworkPolicy not supported
		{"BootstrapTasks", strings.Join(resources.BootstrapTasks, "\n---\n"), true},
		{"BootstrapPipeline", resources.BootstrapPipeline, true},
		{"PipelineRun", resources.PipelineRun, false}, // May be empty for some modes
	}

	for _, resource := range applyOrder {
		if resource.content == "" && resource.required {
			return fmt.Errorf("required resource %s is empty", resource.name)
		}

		if resource.content != "" {
			ka.logger.WithFields(logger.Fields{
				"resource_type": resource.name,
				"namespace":     resources.Namespace,
			}).Debug("Applying resource")

			if err := ka.applyYAMLContent(ctx, resource.content, resources.Namespace); err != nil {
				if resource.required {
					return fmt.Errorf("failed to apply required resource %s: %w", resource.name, err)
				} else {
					ka.logger.WithError(err).WithFields(logger.Fields{
						"resource_type": resource.name,
					}).Warn("Failed to apply optional resource")
				}
			}
		}
	}

	ka.logger.WithFields(logger.Fields{
		"namespace": resources.Namespace,
	}).Info("Bootstrap Pipeline resources applied successfully")

	return nil
}

// createNamespace creates the namespace if it doesn't exist
func (ka *KubernetesApplier) createNamespace(ctx context.Context, namespace string) error {
	ka.logger.WithFields(logger.Fields{
		"operation": "create_namespace",
		"namespace": namespace,
	}).Debug("Creating namespace")

	// Check if namespace exists
	exists, err := ka.checkNamespaceExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	if exists {
		ka.logger.WithFields(logger.Fields{
			"namespace": namespace,
		}).Debug("Namespace already exists")
		return nil
	}

	// Create namespace
	namespaceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    reposentry.io/managed: "true"
    reposentry.io/type: "user-repository"
  annotations:
    reposentry.io/created-at: "%s"
`, namespace, time.Now().Format(time.RFC3339))

	if err := ka.applyYAMLContent(ctx, namespaceYAML, ""); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	ka.logger.WithFields(logger.Fields{
		"namespace": namespace,
	}).Info("Namespace created successfully")

	return nil
}

// applyYAMLContent applies YAML content to Kubernetes
func (ka *KubernetesApplier) applyYAMLContent(ctx context.Context, yamlContent, namespace string) error {
	// In a real implementation, this would use kubectl or Kubernetes client-go
	// For now, we'll simulate the apply operation

	ka.logger.WithFields(logger.Fields{
		"operation":    "apply_yaml_content",
		"namespace":    namespace,
		"dry_run":      ka.config.DryRun,
		"content_size": len(yamlContent),
	}).Debug("Applying YAML content")

	if ka.config.DryRun {
		ka.logger.WithFields(logger.Fields{
			"namespace": namespace,
		}).Info("DRY RUN: Would apply YAML content")
		return nil
	}

	// Simulate network delay and potential failures
	time.Sleep(100 * time.Millisecond)

	// Basic validation
	if !strings.Contains(yamlContent, "apiVersion") {
		return fmt.Errorf("invalid YAML: missing apiVersion")
	}

	if !strings.Contains(yamlContent, "kind") {
		return fmt.Errorf("invalid YAML: missing kind")
	}

	// TODO: Replace with actual kubectl apply or client-go implementation
	// Example commands that would be executed:
	// kubectl apply -f - --namespace=<namespace> < yamlContent
	// Or use client-go to apply resources programmatically

	ka.logger.WithFields(logger.Fields{
		"namespace": namespace,
	}).Debug("YAML content applied successfully (simulated)")

	return nil
}

// checkNamespaceExists checks if a namespace exists
func (ka *KubernetesApplier) checkNamespaceExists(ctx context.Context, namespace string) (bool, error) {
	// TODO: Replace with actual kubectl or client-go implementation
	// kubectl get namespace <namespace> --ignore-not-found=true

	ka.logger.WithFields(logger.Fields{
		"operation": "check_namespace_exists",
		"namespace": namespace,
	}).Debug("Checking namespace existence")

	// Simulate check - in reality this would query Kubernetes API
	// For now, assume namespace doesn't exist
	return false, nil
}

// GetNamespaceStatus gets the status of a namespace
func (ka *KubernetesApplier) GetNamespaceStatus(ctx context.Context, namespace string) (*NamespaceStatus, error) {
	ka.logger.WithFields(logger.Fields{
		"operation": "get_namespace_status",
		"namespace": namespace,
	}).Debug("Getting namespace status")

	// Check if namespace exists
	exists, err := ka.checkNamespaceExists(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check namespace: %w", err)
	}

	status := &NamespaceStatus{
		Exists:    exists,
		Phase:     "Active",
		Resources: make(map[string]int),
	}

	if !exists {
		status.Phase = "NotFound"
		return status, nil
	}

	// TODO: Replace with actual Kubernetes API calls
	// Get PipelineRuns
	pipelineRuns, err := ka.getPipelineRuns(ctx, namespace)
	if err != nil {
		ka.logger.WithError(err).Warn("Failed to get PipelineRuns")
	} else {
		status.PipelineRuns = pipelineRuns
		status.Resources["PipelineRun"] = len(pipelineRuns)
	}

	// Get TaskRuns
	taskRuns, err := ka.getTaskRuns(ctx, namespace)
	if err != nil {
		ka.logger.WithError(err).Warn("Failed to get TaskRuns")
	} else {
		status.TaskRuns = taskRuns
		status.Resources["TaskRun"] = len(taskRuns)
	}

	// Get resource usage
	resourceUsage, err := ka.getResourceUsage(ctx, namespace)
	if err != nil {
		ka.logger.WithError(err).Warn("Failed to get resource usage")
	} else {
		status.ResourceUsage = resourceUsage
	}

	// Determine last activity
	lastActivity := ka.getLastActivity(status.PipelineRuns, status.TaskRuns)
	if lastActivity != nil {
		status.LastActivity = lastActivity
	}

	return status, nil
}

// getPipelineRuns gets PipelineRuns in a namespace
func (ka *KubernetesApplier) getPipelineRuns(ctx context.Context, namespace string) ([]PipelineRunStatus, error) {
	// TODO: Replace with actual kubectl or client-go implementation
	// kubectl get pipelineruns -n <namespace> -o json

	// Simulate some PipelineRuns
	now := time.Now()
	return []PipelineRunStatus{
		{
			Name:           "reposentry-bootstrap-apply-abc123",
			Status:         "Running",
			StartTime:      &now,
			CompletionTime: nil,
			Message:        "Pipeline is running",
		},
	}, nil
}

// getTaskRuns gets TaskRuns in a namespace
func (ka *KubernetesApplier) getTaskRuns(ctx context.Context, namespace string) ([]TaskRunStatus, error) {
	// TODO: Replace with actual kubectl or client-go implementation
	// kubectl get taskruns -n <namespace> -o json

	// Simulate some TaskRuns
	now := time.Now()
	duration := 2 * time.Minute
	return []TaskRunStatus{
		{
			Name:           "clone-repository-abc123",
			Status:         "Succeeded",
			StartTime:      &now,
			CompletionTime: &now,
			Duration:       &duration,
			Message:        "Task completed successfully",
			PipelineRun:    "reposentry-bootstrap-apply-abc123",
		},
	}, nil
}

// getResourceUsage gets resource usage in a namespace
func (ka *KubernetesApplier) getResourceUsage(ctx context.Context, namespace string) (*NamespaceResourceUsage, error) {
	// TODO: Replace with actual kubectl or client-go implementation
	// kubectl top pods -n <namespace>

	// Simulate resource usage
	return &NamespaceResourceUsage{
		CPU:    "100m",
		Memory: "256Mi",
		Pods:   2,
		PVCs:   1,
	}, nil
}

// getLastActivity determines the last activity time from runs
func (ka *KubernetesApplier) getLastActivity(pipelineRuns []PipelineRunStatus, taskRuns []TaskRunStatus) *time.Time {
	var lastActivity *time.Time

	// Check PipelineRuns
	for _, pr := range pipelineRuns {
		if pr.CompletionTime != nil && (lastActivity == nil || pr.CompletionTime.After(*lastActivity)) {
			lastActivity = pr.CompletionTime
		} else if pr.StartTime != nil && (lastActivity == nil || pr.StartTime.After(*lastActivity)) {
			lastActivity = pr.StartTime
		}
	}

	// Check TaskRuns
	for _, tr := range taskRuns {
		if tr.CompletionTime != nil && (lastActivity == nil || tr.CompletionTime.After(*lastActivity)) {
			lastActivity = tr.CompletionTime
		} else if tr.StartTime != nil && (lastActivity == nil || tr.StartTime.After(*lastActivity)) {
			lastActivity = tr.StartTime
		}
	}

	return lastActivity
}

// CleanupNamespace cleans up a namespace and all its resources
func (ka *KubernetesApplier) CleanupNamespace(ctx context.Context, namespace string) error {
	ka.logger.WithFields(logger.Fields{
		"operation": "cleanup_namespace",
		"namespace": namespace,
	}).Info("Starting namespace cleanup")

	// Check if namespace exists
	exists, err := ka.checkNamespaceExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	if !exists {
		ka.logger.WithFields(logger.Fields{
			"namespace": namespace,
		}).Info("Namespace does not exist, nothing to cleanup")
		return nil
	}

	// TODO: Replace with actual kubectl or client-go implementation
	// kubectl delete namespace <namespace>

	if ka.config.DryRun {
		ka.logger.WithFields(logger.Fields{
			"namespace": namespace,
		}).Info("DRY RUN: Would delete namespace")
		return nil
	}

	// Simulate cleanup
	time.Sleep(2 * time.Second)

	ka.logger.WithFields(logger.Fields{
		"namespace": namespace,
	}).Info("Namespace cleanup completed (simulated)")

	return nil
}

// SetConfig updates the applier configuration
func (ka *KubernetesApplier) SetConfig(config *KubernetesApplierConfig) {
	ka.config = config
}

// GetConfig returns the current applier configuration
func (ka *KubernetesApplier) GetConfig() *KubernetesApplierConfig {
	return ka.config
}
