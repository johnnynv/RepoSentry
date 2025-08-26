package tekton

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// BootstrapPipelineGenerator generates Bootstrap Pipeline YAML based on detection results
type BootstrapPipelineGenerator struct {
	logger *logger.Entry
}

// NewBootstrapPipelineGenerator creates a new Bootstrap Pipeline generator
func NewBootstrapPipelineGenerator(parentLogger *logger.Entry) *BootstrapPipelineGenerator {
	generatorLogger := parentLogger.WithFields(logger.Fields{
		"component": "bootstrap-pipeline-generator",
	})

	return &BootstrapPipelineGenerator{
		logger: generatorLogger,
	}
}

// BootstrapPipelineConfig contains configuration for generating Bootstrap Pipeline
type BootstrapPipelineConfig struct {
	// Repository information
	Repository types.Repository
	CommitSHA  string
	Branch     string

	// Detection results
	Detection *TektonDetection

	// Namespace for execution
	Namespace string

	// Bootstrap configuration
	CloneImage     string
	KubectlImage   string
	TektonImage    string
	WorkspaceSize  string
	ServiceAccount string

	// Security and resource limits
	ResourceLimits  map[string]string
	SecurityContext map[string]interface{}
	NetworkPolicy   string
}

// BootstrapPipelineResources contains all generated Tekton resources
type BootstrapPipelineResources struct {
	// Core bootstrap resources
	BootstrapPipeline string
	BootstrapTasks    []string

	// Supporting resources
	ServiceAccount string
	RoleBinding    string
	NetworkPolicy  string
	ResourceQuota  string

	// Execution resources
	PipelineRun string

	// Generated metadata
	GeneratedAt string
	Namespace   string
	Config      *BootstrapPipelineConfig
}

// GenerateBootstrapResources generates all necessary Bootstrap Pipeline resources
func (bpg *BootstrapPipelineGenerator) GenerateBootstrapResources(config *BootstrapPipelineConfig) (*BootstrapPipelineResources, error) {
	bpg.logger.WithFields(logger.Fields{
		"operation":        "generate_bootstrap_resources",
		"repository":       config.Repository.Name,
		"estimated_action": config.Detection.EstimatedAction,
		"namespace":        config.Namespace,
	}).Info("Generating Bootstrap Pipeline resources")

	// Set defaults if not provided
	if err := bpg.setDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	resources := &BootstrapPipelineResources{
		Namespace:      config.Namespace,
		Config:         config,
		GeneratedAt:    "2025-08-26T08:00:00Z", // TODO: Use actual timestamp
		BootstrapTasks: []string{},
	}

	// Generate based on estimated action
	switch config.Detection.EstimatedAction {
	case "apply":
		return bpg.generateApplyResources(config, resources)
	case "trigger":
		return bpg.generateTriggerResources(config, resources)
	case "validate":
		return bpg.generateValidateResources(config, resources)
	case "skip":
		return bpg.generateSkipResources(config, resources)
	default:
		return nil, fmt.Errorf("unsupported estimated action: %s", config.Detection.EstimatedAction)
	}
}

// setDefaults sets default values for configuration
func (bpg *BootstrapPipelineGenerator) setDefaults(config *BootstrapPipelineConfig) error {
	if config.CloneImage == "" {
		config.CloneImage = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2"
	}
	if config.KubectlImage == "" {
		config.KubectlImage = "bitnami/kubectl:1.28"
	}
	if config.TektonImage == "" {
		config.TektonImage = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller:v0.40.2"
	}
	if config.WorkspaceSize == "" {
		config.WorkspaceSize = "1Gi"
	}
	if config.ServiceAccount == "" {
		config.ServiceAccount = "reposentry-bootstrap-sa"
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

	return nil
}

// generateApplyResources generates resources for "apply" action
func (bpg *BootstrapPipelineGenerator) generateApplyResources(config *BootstrapPipelineConfig, resources *BootstrapPipelineResources) (*BootstrapPipelineResources, error) {
	bpg.logger.Info("Generating apply-mode Bootstrap Pipeline")

	// Generate bootstrap pipeline for applying user resources
	pipelineTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-bootstrap-apply
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/action: "apply"
    reposentry.io/repository: "{{.Repository.Name}}"
spec:
  description: "Bootstrap Pipeline to clone repository, validate and apply Tekton resources"
  params:
  - name: repo-url
    type: string
    description: "Repository URL to clone"
  - name: commit-sha
    type: string
    description: "Commit SHA to checkout"
  - name: branch
    type: string
    description: "Branch name"
    default: "main"
  - name: tekton-path
    type: string
    description: "Path to Tekton resources"
    default: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source
    description: "Workspace for source code"
  - name: tekton-resources
    description: "Workspace for processed Tekton resources"
  tasks:
  - name: clone-repository
    taskRef:
      name: reposentry-bootstrap-clone
    params:
    - name: url
      value: $(params.repo-url)
    - name: revision
      value: $(params.commit-sha)
    workspaces:
    - name: output
      workspace: source
  - name: validate-tekton-resources
    taskRef:
      name: reposentry-bootstrap-validate
    runAfter:
    - clone-repository
    params:
    - name: tekton-path
      value: $(params.tekton-path)
    workspaces:
    - name: source
      workspace: source
    - name: output
      workspace: tekton-resources
  - name: apply-tekton-resources
    taskRef:
      name: reposentry-bootstrap-apply-resources
    runAfter:
    - validate-tekton-resources
    params:
    - name: namespace
      value: "{{.Namespace}}"
    workspaces:
    - name: resources
      workspace: tekton-resources
`

	tmpl, err := template.New("bootstrap-pipeline").Parse(pipelineTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline template: %w", err)
	}

	var pipelineBuffer strings.Builder
	if err := tmpl.Execute(&pipelineBuffer, config); err != nil {
		return nil, fmt.Errorf("failed to execute pipeline template: %w", err)
	}

	resources.BootstrapPipeline = pipelineBuffer.String()

	// Generate supporting tasks
	tasks, err := bpg.generateApplyTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apply tasks: %w", err)
	}
	resources.BootstrapTasks = tasks

	// Generate supporting resources
	if err := bpg.generateSupportingResources(config, resources); err != nil {
		return nil, fmt.Errorf("failed to generate supporting resources: %w", err)
	}

	// Generate PipelineRun
	pipelineRun, err := bpg.generateApplyPipelineRun(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PipelineRun: %w", err)
	}
	resources.PipelineRun = pipelineRun

	return resources, nil
}

// generateTriggerResources generates resources for "trigger" action
func (bpg *BootstrapPipelineGenerator) generateTriggerResources(config *BootstrapPipelineConfig, resources *BootstrapPipelineResources) (*BootstrapPipelineResources, error) {
	bpg.logger.Info("Generating trigger-mode Bootstrap Pipeline")

	// For trigger mode, we apply resources first, then trigger them
	pipelineTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-bootstrap-trigger
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/action: "trigger"
    reposentry.io/repository: "{{.Repository.Name}}"
spec:
  description: "Bootstrap Pipeline to apply and trigger user Tekton resources"
  params:
  - name: repo-url
    type: string
  - name: commit-sha
    type: string
  - name: branch
    type: string
    default: "main"
  - name: tekton-path
    type: string
    default: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source
  - name: tekton-resources
  tasks:
  - name: clone-repository
    taskRef:
      name: reposentry-bootstrap-clone
    params:
    - name: url
      value: $(params.repo-url)
    - name: revision
      value: $(params.commit-sha)
    workspaces:
    - name: output
      workspace: source
  - name: validate-tekton-resources
    taskRef:
      name: reposentry-bootstrap-validate
    runAfter:
    - clone-repository
    params:
    - name: tekton-path
      value: $(params.tekton-path)
    workspaces:
    - name: source
      workspace: source
    - name: output
      workspace: tekton-resources
  - name: apply-definitions
    taskRef:
      name: reposentry-bootstrap-apply-resources
    runAfter:
    - validate-tekton-resources
    params:
    - name: namespace
      value: "{{.Namespace}}"
    - name: resource-filter
      value: "Pipeline,Task"
    workspaces:
    - name: resources
      workspace: tekton-resources
  - name: trigger-runs
    taskRef:
      name: reposentry-bootstrap-trigger-runs
    runAfter:
    - apply-definitions
    params:
    - name: namespace
      value: "{{.Namespace}}"
    workspaces:
    - name: resources
      workspace: tekton-resources
`

	tmpl, err := template.New("bootstrap-trigger").Parse(pipelineTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trigger template: %w", err)
	}

	var pipelineBuffer strings.Builder
	if err := tmpl.Execute(&pipelineBuffer, config); err != nil {
		return nil, fmt.Errorf("failed to execute trigger template: %w", err)
	}

	resources.BootstrapPipeline = pipelineBuffer.String()

	// Generate trigger-specific tasks
	tasks, err := bpg.generateTriggerTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate trigger tasks: %w", err)
	}
	resources.BootstrapTasks = tasks

	// Generate supporting resources
	if err := bpg.generateSupportingResources(config, resources); err != nil {
		return nil, fmt.Errorf("failed to generate supporting resources: %w", err)
	}

	// Generate PipelineRun
	pipelineRun, err := bpg.generateTriggerPipelineRun(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PipelineRun: %w", err)
	}
	resources.PipelineRun = pipelineRun

	return resources, nil
}

// generateValidateResources generates resources for "validate" action
func (bpg *BootstrapPipelineGenerator) generateValidateResources(config *BootstrapPipelineConfig, resources *BootstrapPipelineResources) (*BootstrapPipelineResources, error) {
	bpg.logger.Info("Generating validate-mode Bootstrap Pipeline")

	// For validate mode, we only clone and validate, no apply
	pipelineTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-bootstrap-validate
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/action: "validate"
    reposentry.io/repository: "{{.Repository.Name}}"
spec:
  description: "Bootstrap Pipeline to validate Tekton resources only"
  params:
  - name: repo-url
    type: string
  - name: commit-sha
    type: string
  - name: tekton-path
    type: string
    default: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source
  tasks:
  - name: clone-repository
    taskRef:
      name: reposentry-bootstrap-clone
    params:
    - name: url
      value: $(params.repo-url)
    - name: revision
      value: $(params.commit-sha)
    workspaces:
    - name: output
      workspace: source
  - name: validate-only
    taskRef:
      name: reposentry-bootstrap-validate-only
    runAfter:
    - clone-repository
    params:
    - name: tekton-path
      value: $(params.tekton-path)
    workspaces:
    - name: source
      workspace: source
`

	tmpl, err := template.New("bootstrap-validate").Parse(pipelineTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse validate template: %w", err)
	}

	var pipelineBuffer strings.Builder
	if err := tmpl.Execute(&pipelineBuffer, config); err != nil {
		return nil, fmt.Errorf("failed to execute validate template: %w", err)
	}

	resources.BootstrapPipeline = pipelineBuffer.String()

	// Generate validate-specific tasks
	tasks, err := bpg.generateValidateTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate validate tasks: %w", err)
	}
	resources.BootstrapTasks = tasks

	// Generate supporting resources (minimal for validation)
	if err := bpg.generateSupportingResources(config, resources); err != nil {
		return nil, fmt.Errorf("failed to generate supporting resources: %w", err)
	}

	return resources, nil
}

// generateSkipResources generates minimal resources for "skip" action
func (bpg *BootstrapPipelineGenerator) generateSkipResources(config *BootstrapPipelineConfig, resources *BootstrapPipelineResources) (*BootstrapPipelineResources, error) {
	bpg.logger.Info("Generating skip-mode resources (minimal)")

	// For skip mode, we just generate a simple notification task
	pipelineTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-bootstrap-skip
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/action: "skip"
    reposentry.io/repository: "{{.Repository.Name}}"
spec:
  description: "Bootstrap Pipeline for repositories without Tekton resources"
  tasks:
  - name: notify-skip
    taskRef:
      name: reposentry-bootstrap-notify
    params:
    - name: message
      value: "No Tekton resources found in repository {{.Repository.Name}}"
    - name: status
      value: "skipped"
`

	tmpl, err := template.New("bootstrap-skip").Parse(pipelineTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skip template: %w", err)
	}

	var pipelineBuffer strings.Builder
	if err := tmpl.Execute(&pipelineBuffer, config); err != nil {
		return nil, fmt.Errorf("failed to execute skip template: %w", err)
	}

	resources.BootstrapPipeline = pipelineBuffer.String()

	// Generate minimal tasks
	tasks, err := bpg.generateSkipTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate skip tasks: %w", err)
	}
	resources.BootstrapTasks = tasks

	return resources, nil
}
