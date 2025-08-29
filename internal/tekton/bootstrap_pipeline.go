package tekton

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// BootstrapPipelineGenerator generates PipelineRun YAML for the pre-deployed Bootstrap Pipeline
// Note: The actual Bootstrap Pipeline and Tasks are deployed statically using YAML files in deployments/tekton/bootstrap/
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

// BootstrapPipelineResources contains generated PipelineRun for triggering the pre-deployed Bootstrap Pipeline
type BootstrapPipelineResources struct {
	// Execution resources (only PipelineRun is generated at runtime)
	PipelineRun string

	// Generated metadata
	GeneratedAt     string
	TargetNamespace string
	Config          *BootstrapPipelineConfig
}

// GeneratePipelineRun generates a PipelineRun to trigger the pre-deployed Bootstrap Pipeline
func (bpg *BootstrapPipelineGenerator) GeneratePipelineRun(config *BootstrapPipelineConfig) (*BootstrapPipelineResources, error) {
	bpg.logger.WithFields(logger.Fields{
		"operation":        "generate_pipeline_run",
		"repository":       config.Repository.Name,
		"estimated_action": config.Detection.EstimatedAction,
		"target_namespace": config.Namespace,
	}).Info("Generating PipelineRun for pre-deployed Bootstrap Pipeline")

	// Set defaults if not provided
	if err := bpg.setDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	resources := &BootstrapPipelineResources{
		TargetNamespace: config.Namespace,
		Config:          config,
		GeneratedAt:     fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Generate PipelineRun for the static Bootstrap Pipeline
	pipelineRun, err := bpg.generateBootstrapPipelineRun(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PipelineRun: %w", err)
	}
	resources.PipelineRun = pipelineRun

	bpg.logger.WithFields(logger.Fields{
		"pipeline_run_generated": true,
		"target_namespace":       config.Namespace,
	}).Info("PipelineRun generated successfully")

	return resources, nil
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

// generateBootstrapPipelineRun generates a PipelineRun to trigger the static Bootstrap Pipeline
func (bpg *BootstrapPipelineGenerator) generateBootstrapPipelineRun(config *BootstrapPipelineConfig) (string, error) {
	pipelineRunTemplate := `apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: reposentry-bootstrap-run-
  namespace: reposentry-system
  labels:
    reposentry.io/repository: "{{.Repository.Name}}"
    reposentry.io/commit-sha: "{{.CommitSHA}}"
    reposentry.io/estimated-action: "{{.Detection.EstimatedAction}}"
  annotations:
    reposentry.io/repository-url: "{{.Repository.URL}}"
spec:
  pipelineRef:
    name: reposentry-bootstrap-pipeline
  params:
  - name: repo-url
    value: "{{.Repository.URL}}"
  - name: repo-branch
    value: "{{.Branch}}"
  - name: commit-sha
    value: "{{.CommitSHA}}"
  - name: target-namespace
    value: "{{.Namespace}}"
  - name: tekton-path
    value: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source-workspace
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
  - name: tekton-workspace
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
  serviceAccountName: {{.ServiceAccount}}
---`

	tmpl, err := template.New("bootstrap-pipelinerun").Parse(pipelineRunTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse PipelineRun template: %w", err)
	}

	var buffer strings.Builder
	if err := tmpl.Execute(&buffer, config); err != nil {
		return "", fmt.Errorf("failed to execute PipelineRun template: %w", err)
	}

	return buffer.String(), nil
}
