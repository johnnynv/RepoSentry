package tekton

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// StaticBootstrapGenerator generates static Bootstrap Pipeline YAML files for deployment
// This generator is used during RepoSentry installation to pre-deploy the Bootstrap Pipeline infrastructure
type StaticBootstrapGenerator struct {
	logger *logger.Entry
}

// NewStaticBootstrapGenerator creates a new static Bootstrap Pipeline generator
func NewStaticBootstrapGenerator(parentLogger *logger.Entry) *StaticBootstrapGenerator {
	generatorLogger := parentLogger.WithFields(logger.Fields{
		"component": "static-bootstrap-generator",
	})

	return &StaticBootstrapGenerator{
		logger: generatorLogger,
	}
}

// StaticBootstrapConfig contains configuration for generating static Bootstrap Pipeline
type StaticBootstrapConfig struct {
	// System configuration
	SystemNamespace string
	ServiceAccount  string
	CloneImage      string
	KubectlImage    string
	TektonImage     string

	// Resource limits and security
	ResourceLimits  map[string]string
	SecurityContext map[string]interface{}

	// Output configuration
	OutputDirectory string
}

// StaticBootstrapOutput contains all generated static files
type StaticBootstrapOutput struct {
	// Core Pipeline and Tasks
	Pipeline string
	Tasks    []string

	// Infrastructure resources
	Namespace      string
	ServiceAccount string
	Role           string
	RoleBinding    string

	// Generated metadata
	GeneratedAt string
	FilePaths   []string
}

// GenerateStaticBootstrapInfrastructure generates all static Bootstrap Pipeline infrastructure
func (sbg *StaticBootstrapGenerator) GenerateStaticBootstrapInfrastructure(config *StaticBootstrapConfig) (*StaticBootstrapOutput, error) {
	sbg.logger.WithFields(logger.Fields{
		"operation":        "generate_static_infrastructure",
		"system_namespace": config.SystemNamespace,
		"output_dir":       config.OutputDirectory,
	}).Info("Generating static Bootstrap Pipeline infrastructure")

	// Set defaults
	if err := sbg.setDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	output := &StaticBootstrapOutput{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		FilePaths:   []string{},
		Tasks:       []string{},
	}

	// Generate namespace
	namespace, err := sbg.generateSystemNamespace(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate namespace: %w", err)
	}
	output.Namespace = namespace

	// Generate static Bootstrap Pipeline
	pipeline, err := sbg.generateStaticBootstrapPipeline(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Bootstrap Pipeline: %w", err)
	}
	output.Pipeline = pipeline

	// Generate all Bootstrap Tasks
	tasks, err := sbg.generateStaticBootstrapTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Bootstrap Tasks: %w", err)
	}
	output.Tasks = tasks

	// Generate RBAC resources
	if err := sbg.generateStaticRBACResources(config, output); err != nil {
		return nil, fmt.Errorf("failed to generate RBAC resources: %w", err)
	}

	sbg.logger.WithFields(logger.Fields{
		"pipeline_generated": true,
		"tasks_count":        len(output.Tasks),
		"rbac_generated":     true,
	}).Info("Static Bootstrap Pipeline infrastructure generated successfully")

	return output, nil
}

// setDefaults sets default values for static configuration
func (sbg *StaticBootstrapGenerator) setDefaults(config *StaticBootstrapConfig) error {
	if config.SystemNamespace == "" {
		config.SystemNamespace = "reposentry-system"
	}
	if config.ServiceAccount == "" {
		config.ServiceAccount = "reposentry-bootstrap-sa"
	}
	if config.CloneImage == "" {
		config.CloneImage = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2"
	}
	if config.KubectlImage == "" {
		config.KubectlImage = "bitnami/kubectl:1.28"
	}
	if config.TektonImage == "" {
		config.TektonImage = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller:v0.40.2"
	}
	if config.ResourceLimits == nil {
		config.ResourceLimits = map[string]string{
			"cpu":    "500m",
			"memory": "512Mi",
		}
	}
	if config.SecurityContext == nil {
		config.SecurityContext = map[string]interface{}{
			"runAsNonRoot": true,
			"runAsUser":    65532,
			"fsGroup":      65532,
		}
	}
	if config.OutputDirectory == "" {
		config.OutputDirectory = "./deployments/tekton/bootstrap"
	}

	return nil
}

// generateSystemNamespace generates the system namespace YAML
func (sbg *StaticBootstrapGenerator) generateSystemNamespace(config *StaticBootstrapConfig) (string, error) {
	namespaceTemplate := `apiVersion: v1
kind: Namespace
metadata:
  name: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "system"
    reposentry.io/type: "bootstrap-infrastructure"
  annotations:
    reposentry.io/generated-at: "{{.GeneratedAt}}"
---`

	tmpl, err := template.New("system-namespace").Parse(namespaceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse namespace template: %w", err)
	}

	data := struct {
		SystemNamespace string
		GeneratedAt     string
	}{
		SystemNamespace: config.SystemNamespace,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("failed to execute namespace template: %w", err)
	}

	return buffer.String(), nil
}

// generateStaticBootstrapPipeline generates the main Bootstrap Pipeline
func (sbg *StaticBootstrapGenerator) generateStaticBootstrapPipeline(config *StaticBootstrapConfig) (string, error) {
	pipelineTemplate := `apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-bootstrap-pipeline
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "pipeline"
  annotations:
    reposentry.io/generated-at: "{{.GeneratedAt}}"
    reposentry.io/description: "Static Bootstrap Pipeline for processing user Tekton resources"
spec:
  description: "Bootstrap Pipeline to clone, validate, and apply user Tekton resources from remote repositories"
  params:
  - name: repo-url
    type: string
    description: "URL of the user repository to process"
  - name: repo-branch
    type: string
    description: "Branch name to checkout"
    default: "main"
  - name: commit-sha
    type: string
    description: "Commit SHA to checkout"
  - name: target-namespace
    type: string
    description: "Target namespace for user resources (computed from repo info)"
  - name: tekton-path
    type: string
    description: "Path to Tekton resources in the repository"
    default: ".tekton"
  workspaces:
  - name: source-workspace
    description: "Workspace for cloned source code"
  - name: tekton-workspace
    description: "Workspace for processed Tekton resources"
  tasks:
  - name: clone-user-repository
    taskRef:
      name: reposentry-bootstrap-clone
    params:
    - name: url
      value: $(params.repo-url)
    - name: revision
      value: $(params.commit-sha)
    workspaces:
    - name: output
      workspace: source-workspace
  - name: compute-target-namespace
    taskRef:
      name: reposentry-bootstrap-compute-namespace
    runAfter:
    - clone-user-repository
    params:
    - name: repo-url
      value: $(params.repo-url)
    results:
    - name: namespace-name
      description: "Computed namespace name for user resources"
  - name: validate-tekton-resources
    taskRef:
      name: reposentry-bootstrap-validate
    runAfter:
    - compute-target-namespace
    params:
    - name: tekton-path
      value: $(params.tekton-path)
    - name: target-namespace
      value: $(tasks.compute-target-namespace.results.namespace-name)
    workspaces:
    - name: source
      workspace: source-workspace
    - name: output
      workspace: tekton-workspace
  - name: ensure-user-namespace
    taskRef:
      name: reposentry-bootstrap-ensure-namespace
    runAfter:
    - validate-tekton-resources
    params:
    - name: namespace-name
      value: $(tasks.compute-target-namespace.results.namespace-name)
    - name: repo-url
      value: $(params.repo-url)
  - name: apply-user-resources
    taskRef:
      name: reposentry-bootstrap-apply
    runAfter:
    - ensure-user-namespace
    params:
    - name: target-namespace
      value: $(tasks.compute-target-namespace.results.namespace-name)
    workspaces:
    - name: resources
      workspace: tekton-workspace
---`

	tmpl, err := template.New("bootstrap-pipeline").Parse(pipelineTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse pipeline template: %w", err)
	}

	data := struct {
		SystemNamespace string
		GeneratedAt     string
	}{
		SystemNamespace: config.SystemNamespace,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("failed to execute pipeline template: %w", err)
	}

	return buffer.String(), nil
}

// generateStaticBootstrapTasks generates all Bootstrap Tasks
func (sbg *StaticBootstrapGenerator) generateStaticBootstrapTasks(config *StaticBootstrapConfig) ([]string, error) {
	var tasks []string

	// Clone task
	cloneTask, err := sbg.generateStaticCloneTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate clone task: %w", err)
	}
	tasks = append(tasks, cloneTask)

	// Compute namespace task
	namespaceTask, err := sbg.generateStaticComputeNamespaceTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compute namespace task: %w", err)
	}
	tasks = append(tasks, namespaceTask)

	// Validate task
	validateTask, err := sbg.generateStaticValidateTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate validate task: %w", err)
	}
	tasks = append(tasks, validateTask)

	// Ensure namespace task
	ensureNamespaceTask, err := sbg.generateStaticEnsureNamespaceTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ensure namespace task: %w", err)
	}
	tasks = append(tasks, ensureNamespaceTask)

	// Apply resources task
	applyTask, err := sbg.generateStaticApplyTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apply task: %w", err)
	}
	tasks = append(tasks, applyTask)

	return tasks, nil
}

// generateStaticRBACResources generates RBAC resources for the Bootstrap Pipeline
func (sbg *StaticBootstrapGenerator) generateStaticRBACResources(config *StaticBootstrapConfig, output *StaticBootstrapOutput) error {
	// Generate ServiceAccount
	serviceAccount, err := sbg.generateStaticServiceAccount(config)
	if err != nil {
		return fmt.Errorf("failed to generate ServiceAccount: %w", err)
	}
	output.ServiceAccount = serviceAccount

	// Generate Role
	role, err := sbg.generateStaticRole(config)
	if err != nil {
		return fmt.Errorf("failed to generate Role: %w", err)
	}
	output.Role = role

	// Generate RoleBinding
	roleBinding, err := sbg.generateStaticRoleBinding(config)
	if err != nil {
		return fmt.Errorf("failed to generate RoleBinding: %w", err)
	}
	output.RoleBinding = roleBinding

	return nil
}

// WriteToFiles writes all generated YAML to files in the output directory
func (sbg *StaticBootstrapGenerator) WriteToFiles(output *StaticBootstrapOutput, outputDir string) error {
	sbg.logger.WithFields(logger.Fields{
		"output_directory": outputDir,
		"files_to_write":   6, // namespace, pipeline, tasks, serviceaccount, role, rolebinding
	}).Info("Writing static Bootstrap Pipeline files")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Define files to write
	filesToWrite := map[string]string{
		filepath.Join(outputDir, "00-namespace.yaml"):      output.Namespace,
		filepath.Join(outputDir, "01-pipeline.yaml"):       output.Pipeline,
		filepath.Join(outputDir, "02-tasks.yaml"):          strings.Join(output.Tasks, "\n"),
		filepath.Join(outputDir, "03-serviceaccount.yaml"): output.ServiceAccount,
		filepath.Join(outputDir, "04-role.yaml"):           output.Role,
		filepath.Join(outputDir, "05-rolebinding.yaml"):    output.RoleBinding,
	}

	// Write each file
	for filePath, content := range filesToWrite {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		sbg.logger.WithFields(logger.Fields{
			"file_path": filePath,
			"file_size": len(content),
		}).Debug("File written successfully")
	}

	sbg.logger.WithFields(logger.Fields{
		"files_written":    len(filesToWrite),
		"output_directory": outputDir,
	}).Info("Static Bootstrap Pipeline files written successfully")

	return nil
}

// generateStaticCloneTask generates the clone task for Bootstrap Pipeline
func (sbg *StaticBootstrapGenerator) generateStaticCloneTask(config *StaticBootstrapConfig) (string, error) {
	cloneTaskTemplate := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-clone
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "task"
spec:
  description: "Clone user repository for Bootstrap Pipeline"
  params:
  - name: url
    type: string
    description: "Repository URL to clone"
  - name: revision
    type: string
    description: "Git revision to checkout"
    default: "main"
  workspaces:
  - name: output
    description: "The workspace where the repo will be cloned"
  steps:
  - name: clone
    image: {{.CloneImage}}
    script: |
      set -e
      git clone "$(params.url)" "$(workspaces.output.path)/"
      cd "$(workspaces.output.path)/"
      git checkout "$(params.revision)"
      echo "Repository cloned successfully"
---`

	tmpl, err := template.New("clone-task").Parse(cloneTaskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse clone task template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute clone task template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticComputeNamespaceTask(config *StaticBootstrapConfig) (string, error) {
	computeNamespaceTemplate := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-compute-namespace
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "task"
spec:
  description: "Compute target namespace name from repository URL"
  params:
  - name: repo-url
    type: string
    description: "Repository URL"
  results:
  - name: namespace-name
    description: "Computed namespace name"
  steps:
  - name: compute
    image: {{.KubectlImage}}
    script: |
      set -e
      REPO_URL="$(params.repo-url)"
      # Extract owner/repo from URL and create hash
      REPO_PATH=$(echo "$REPO_URL" | sed -E 's#.*[:/]([^/]+/[^/]+)\.git$#\1#' | tr '/' '-')
      REPO_HASH=$(echo "$REPO_PATH" | sha256sum | cut -c1-8)
      NAMESPACE_NAME="reposentry-user-repo-$REPO_HASH"
      echo -n "$NAMESPACE_NAME" | tee $(results.namespace-name.path)
      echo "Computed namespace: $NAMESPACE_NAME"
---`

	tmpl, err := template.New("compute-namespace-task").Parse(computeNamespaceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse compute namespace task template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute compute namespace task template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticValidateTask(config *StaticBootstrapConfig) (string, error) {
	validateTaskTemplate := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-validate
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "task"
spec:
  description: "Validate Tekton resources in user repository"
  params:
  - name: tekton-path
    type: string
    description: "Path to Tekton resources"
    default: ".tekton"
  - name: target-namespace
    type: string
    description: "Target namespace for validation"
  workspaces:
  - name: source
    description: "Source code workspace"
  - name: output
    description: "Output workspace for processed resources"
  steps:
  - name: validate
    image: {{.KubectlImage}}
    script: |
      set -e
      TEKTON_PATH="$(workspaces.source.path)/$(params.tekton-path)"
      OUTPUT_PATH="$(workspaces.output.path)"
      
      if [ ! -d "$TEKTON_PATH" ]; then
        echo "No Tekton directory found at $TEKTON_PATH"
        exit 1
      fi
      
      echo "Validating Tekton resources in $TEKTON_PATH"
      
      # Copy and validate YAML files
      mkdir -p "$OUTPUT_PATH"
      find "$TEKTON_PATH" -name "*.yaml" -o -name "*.yml" | while read file; do
        echo "Validating $file"
        kubectl --dry-run=client apply -f "$file" --namespace="$(params.target-namespace)"
        cp "$file" "$OUTPUT_PATH/"
      done
      
      echo "Validation completed successfully"
---`

	tmpl, err := template.New("validate-task").Parse(validateTaskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse validate task template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute validate task template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticEnsureNamespaceTask(config *StaticBootstrapConfig) (string, error) {
	ensureNamespaceTemplate := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-ensure-namespace
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "task"
spec:
  description: "Ensure user namespace exists with proper RBAC and quotas"
  params:
  - name: namespace-name
    type: string
    description: "Namespace name to create"
  - name: repo-url
    type: string
    description: "Repository URL for labeling"
  steps:
  - name: ensure-namespace
    image: {{.KubectlImage}}
    script: |
      set -e
      NAMESPACE="$(params.namespace-name)"
      REPO_URL="$(params.repo-url)"
      
      echo "Ensuring namespace $NAMESPACE exists"
      
      # Create namespace if it doesn't exist
      kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | \
        kubectl apply -f -
      
      # Label namespace
      kubectl label namespace "$NAMESPACE" \
        reposentry.io/managed=true \
        reposentry.io/repository="$REPO_URL" \
        --overwrite
      
      echo "Namespace $NAMESPACE is ready"
---`

	tmpl, err := template.New("ensure-namespace-task").Parse(ensureNamespaceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse ensure namespace task template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute ensure namespace task template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticApplyTask(config *StaticBootstrapConfig) (string, error) {
	applyTaskTemplate := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-apply
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "task"
spec:
  description: "Apply user Tekton resources to target namespace"
  params:
  - name: target-namespace
    type: string
    description: "Target namespace for resources"
  workspaces:
  - name: resources
    description: "Validated Tekton resources workspace"
  steps:
  - name: apply
    image: {{.KubectlImage}}
    script: |
      set -e
      TARGET_NS="$(params.target-namespace)"
      RESOURCES_PATH="$(workspaces.resources.path)"
      
      echo "Applying Tekton resources to namespace $TARGET_NS"
      
      # Apply all YAML files to target namespace
      find "$RESOURCES_PATH" -name "*.yaml" -o -name "*.yml" | while read file; do
        echo "Applying $file"
        kubectl apply -f "$file" --namespace="$TARGET_NS"
      done
      
      echo "Resources applied successfully to $TARGET_NS"
---`

	tmpl, err := template.New("apply-task").Parse(applyTaskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse apply task template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute apply task template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticServiceAccount(config *StaticBootstrapConfig) (string, error) {
	serviceAccountTemplate := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{.ServiceAccount}}
  namespace: {{.SystemNamespace}}
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "serviceaccount"
---`

	tmpl, err := template.New("serviceaccount").Parse(serviceAccountTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse ServiceAccount template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute ServiceAccount template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticRole(config *StaticBootstrapConfig) (string, error) {
	roleTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: reposentry-bootstrap-role
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "clusterrole"
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "create", "update", "patch"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelines", "tasks", "pipelineruns", "taskruns"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "create", "update", "patch"]
---`

	tmpl, err := template.New("role").Parse(roleTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Role template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute Role template: %w", err)
	}

	return buffer.String(), nil
}

func (sbg *StaticBootstrapGenerator) generateStaticRoleBinding(config *StaticBootstrapConfig) (string, error) {
	roleBindingTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: reposentry-bootstrap-binding
  labels:
    reposentry.io/component: "bootstrap"
    reposentry.io/type: "clusterrolebinding"
subjects:
- kind: ServiceAccount
  name: {{.ServiceAccount}}
  namespace: {{.SystemNamespace}}
roleRef:
  kind: ClusterRole
  name: reposentry-bootstrap-role
  apiGroup: rbac.authorization.k8s.io
---`

	tmpl, err := template.New("rolebinding").Parse(roleBindingTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse RoleBinding template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute RoleBinding template: %w", err)
	}

	return buffer.String(), nil
}
