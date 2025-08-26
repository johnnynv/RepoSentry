package tekton

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/johnnynv/RepoSentry/internal/gitclient"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
	"gopkg.in/yaml.v3"
)

// TektonDetector detects and analyzes Tekton resources in REMOTE repositories
// IMPORTANT: This detector scans .tekton/ directories in REMOTE user repositories
// (configured in RepoSentry's monitoring YAML), NOT local RepoSentry directories.
// It uses GitClient to access remote repository content via GitHub/GitLab APIs.
type TektonDetector struct {
	gitClient gitclient.GitClient
	logger    *logger.Entry
	config    DetectorConfig
}

// DetectorConfig configures the Tekton detector behavior
type DetectorConfig struct {
	// ScanPath is the base path to scan for Tekton files (default: ".tekton")
	ScanPath string `json:"scan_path"`

	// FileExtensions are the file extensions to consider as Tekton files
	FileExtensions []string `json:"file_extensions"`

	// MaxFileSize is the maximum file size to process (in bytes)
	MaxFileSize int64 `json:"max_file_size"`

	// Timeout for detection operations
	Timeout time.Duration `json:"timeout"`
}

// TektonDetection represents the result of Tekton resource detection
type TektonDetection struct {
	// Repository information
	Repository types.Repository `json:"repository"`
	CommitSHA  string           `json:"commit_sha"`
	Branch     string           `json:"branch"`

	// Detection results
	HasTektonDirectory bool             `json:"has_tekton_directory"`
	TektonFiles        []TektonFile     `json:"tekton_files"`
	Resources          []TektonResource `json:"resources"`

	// Detection metadata
	DetectedAt time.Time `json:"detected_at"`
	ScanPath   string    `json:"scan_path"`
	TotalFiles int       `json:"total_files"`
	ValidFiles int       `json:"valid_files"`

	// Processing results
	EstimatedAction string   `json:"estimated_action"` // "apply", "trigger", "validate", "skip"
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

// TektonFile represents a single Tekton YAML file
type TektonFile struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified,omitempty"`
	IsValid      bool      `json:"is_valid"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// TektonResource represents a parsed Tekton resource from YAML
type TektonResource struct {
	// Kubernetes resource fields
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`

	// Source information
	FilePath      string `json:"file_path"`
	ResourceIndex int    `json:"resource_index"` // Index within the file if multiple resources

	// Tekton-specific information
	ResourceType string                 `json:"resource_type"` // "Task", "Pipeline", "PipelineRun", etc.
	Spec         map[string]interface{} `json:"spec,omitempty"`

	// Validation
	IsValid      bool     `json:"is_valid"`
	Errors       []string `json:"errors,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"` // Referenced resources
}

// NewTektonDetector creates a new Tekton detector
func NewTektonDetector(gitClient gitclient.GitClient, parentLogger *logger.Entry) *TektonDetector {
	detectorLogger := parentLogger.WithFields(logger.Fields{
		"component": "tekton-detector",
	})

	return &TektonDetector{
		gitClient: gitClient,
		logger:    detectorLogger,
		config:    getDefaultDetectorConfig(),
	}
}

// getDefaultDetectorConfig returns the default detector configuration
func getDefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		ScanPath:       ".tekton",
		FileExtensions: []string{".yaml", ".yml"},
		MaxFileSize:    1 * 1024 * 1024, // 1MB
		Timeout:        30 * time.Second,
	}
}

// DetectTektonResources detects Tekton resources in a REMOTE repository
// This function scans the .tekton/ directory in the specified REMOTE repository
// using GitClient APIs (GitHub/GitLab) to access remote repository content.
func (d *TektonDetector) DetectTektonResources(ctx context.Context, repo types.Repository, commitSHA, branch string) (*TektonDetection, error) {
	startTime := time.Now()

	d.logger.WithFields(logger.Fields{
		"operation":  "detect_tekton_resources",
		"repository": repo.Name,
		"commit":     commitSHA,
		"branch":     branch,
		"scan_path":  d.config.ScanPath,
	}).Info("Starting Tekton resource detection")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	detection := &TektonDetection{
		Repository:  repo,
		CommitSHA:   commitSHA,
		Branch:      branch,
		DetectedAt:  startTime,
		ScanPath:    d.config.ScanPath,
		TektonFiles: []TektonFile{},
		Resources:   []TektonResource{},
		Errors:      []string{},
		Warnings:    []string{},
	}

	// Check if .tekton directory exists in the REMOTE repository
	hasDir, err := d.gitClient.CheckDirectoryExists(ctx, repo, commitSHA, d.config.ScanPath)
	if err != nil {
		detection.Errors = append(detection.Errors, fmt.Sprintf("Failed to check .tekton directory: %v", err))
		return detection, nil // Return partial results
	}

	detection.HasTektonDirectory = hasDir
	if !hasDir {
		d.logger.WithFields(logger.Fields{
			"repository": repo.Name,
			"scan_path":  d.config.ScanPath,
		}).Info("No .tekton directory found")
		detection.EstimatedAction = "skip"
		return detection, nil
	}

	// List all files in .tekton directory of the REMOTE repository
	files, err := d.gitClient.ListFiles(ctx, repo, commitSHA, d.config.ScanPath)
	if err != nil {
		detection.Errors = append(detection.Errors, fmt.Sprintf("Failed to list files in .tekton directory: %v", err))
		return detection, nil
	}

	detection.TotalFiles = len(files)

	// Process each file
	for _, filePath := range files {
		if d.isTektonFile(filePath) {
			tektonFile, resources, err := d.processFile(ctx, repo, commitSHA, filePath)
			if err != nil {
				d.logger.WithError(err).WithFields(logger.Fields{
					"file_path":  filePath,
					"repository": repo.Name,
				}).Warn("Failed to process Tekton file")

				tektonFile.IsValid = false
				tektonFile.ErrorMessage = err.Error()
			} else {
				tektonFile.IsValid = true
				detection.ValidFiles++
				detection.Resources = append(detection.Resources, resources...)
			}

			detection.TektonFiles = append(detection.TektonFiles, tektonFile)
		}
	}

	// Determine estimated action based on detected resources
	detection.EstimatedAction = d.determineEstimatedAction(detection)

	d.logger.WithFields(logger.Fields{
		"repository":       repo.Name,
		"total_files":      detection.TotalFiles,
		"valid_files":      detection.ValidFiles,
		"resources":        len(detection.Resources),
		"estimated_action": detection.EstimatedAction,
		"duration":         time.Since(startTime),
	}).Info("Completed Tekton resource detection")

	return detection, nil
}

// isTektonFile checks if a file should be processed as a Tekton file
func (d *TektonDetector) isTektonFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, validExt := range d.config.FileExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

// processFile processes a single Tekton YAML file
func (d *TektonDetector) processFile(ctx context.Context, repo types.Repository, commitSHA, filePath string) (TektonFile, []TektonResource, error) {
	tektonFile := TektonFile{
		Path:    filePath,
		IsValid: false,
	}

	// Get file content from REMOTE repository
	content, err := d.gitClient.GetFileContent(ctx, repo, commitSHA, filePath)
	if err != nil {
		return tektonFile, nil, fmt.Errorf("failed to get file content: %w", err)
	}

	tektonFile.Size = int64(len(content))

	// Check file size limit
	if tektonFile.Size > d.config.MaxFileSize {
		return tektonFile, nil, fmt.Errorf("file size %d exceeds limit %d", tektonFile.Size, d.config.MaxFileSize)
	}

	// Parse YAML content
	resources, err := d.parseYAMLContent(content, filePath)
	if err != nil {
		return tektonFile, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	tektonFile.IsValid = len(resources) > 0
	return tektonFile, resources, nil
}

// parseYAMLContent parses YAML content and extracts Tekton resources
func (d *TektonDetector) parseYAMLContent(content []byte, filePath string) ([]TektonResource, error) {
	var resources []TektonResource

	// Split content by document separator for multi-document YAML
	docs := strings.Split(string(content), "---")

	for docIndex, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			return nil, fmt.Errorf("invalid YAML in document %d: %w", docIndex, err)
		}

		// Extract basic Kubernetes resource fields
		apiVersion, _ := resource["apiVersion"].(string)
		kind, _ := resource["kind"].(string)

		if apiVersion == "" || kind == "" {
			continue // Skip resources without required fields
		}

		// Check if this is a Tekton resource
		if !d.isTektonResource(apiVersion, kind) {
			continue
		}

		// Extract metadata
		metadata, _ := resource["metadata"].(map[string]interface{})
		var name, namespace string
		if metadata != nil {
			name, _ = metadata["name"].(string)
			namespace, _ = metadata["namespace"].(string)
		}

		// Extract spec
		spec, _ := resource["spec"].(map[string]interface{})

		tektonResource := TektonResource{
			APIVersion:    apiVersion,
			Kind:          kind,
			Name:          name,
			Namespace:     namespace,
			FilePath:      filePath,
			ResourceIndex: docIndex,
			ResourceType:  kind,
			Spec:          spec,
			IsValid:       true,
			Errors:        []string{},
			Dependencies:  []string{},
		}

		// Basic validation
		if name == "" {
			tektonResource.IsValid = false
			tektonResource.Errors = append(tektonResource.Errors, "resource name is required")
		}

		resources = append(resources, tektonResource)
	}

	return resources, nil
}

// isTektonResource checks if the given apiVersion and kind represent a Tekton resource
func (d *TektonDetector) isTektonResource(apiVersion, kind string) bool {
	// Tekton API versions and kinds
	tektonPatterns := map[string][]string{
		"tekton.dev/v1beta1":           {"Task", "Pipeline", "PipelineRun", "TaskRun"},
		"tekton.dev/v1":                {"Task", "Pipeline", "PipelineRun", "TaskRun"},
		"triggers.tekton.dev/v1beta1":  {"EventListener", "Trigger", "TriggerBinding", "TriggerTemplate", "ClusterTriggerBinding"},
		"triggers.tekton.dev/v1alpha1": {"EventListener", "Trigger", "TriggerBinding", "TriggerTemplate", "ClusterTriggerBinding"},
	}

	if kinds, exists := tektonPatterns[apiVersion]; exists {
		for _, validKind := range kinds {
			if kind == validKind {
				return true
			}
		}
	}

	return false
}

// determineEstimatedAction determines what action should be taken based on detected resources
func (d *TektonDetector) determineEstimatedAction(detection *TektonDetection) string {
	if len(detection.Resources) == 0 {
		return "skip"
	}

	hasRunnableResources := false
	hasDefinitionResources := false

	for _, resource := range detection.Resources {
		switch resource.Kind {
		case "PipelineRun", "TaskRun":
			hasRunnableResources = true
		case "Pipeline", "Task":
			hasDefinitionResources = true
		}
	}

	// If we have runnable resources (PipelineRun/TaskRun), trigger them
	if hasRunnableResources {
		return "trigger"
	}

	// If we only have definitions, apply them and validate
	if hasDefinitionResources {
		return "apply"
	}

	// Default to validation
	return "validate"
}

// SetConfig updates the detector configuration
func (d *TektonDetector) SetConfig(config DetectorConfig) {
	d.config = config
}

// GetConfig returns the current detector configuration
func (d *TektonDetector) GetConfig() DetectorConfig {
	return d.config
}

// ValidateResource performs additional validation on a Tekton resource
func (d *TektonDetector) ValidateResource(resource *TektonResource) error {
	if resource.Name == "" {
		return fmt.Errorf("resource name is required")
	}

	// Validate name format (Kubernetes naming rules)
	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validName.MatchString(resource.Name) {
		return fmt.Errorf("invalid resource name format: %s", resource.Name)
	}

	// Additional kind-specific validation can be added here
	switch resource.Kind {
	case "Pipeline":
		return d.validatePipeline(resource)
	case "Task":
		return d.validateTask(resource)
	case "PipelineRun":
		return d.validatePipelineRun(resource)
	case "TaskRun":
		return d.validateTaskRun(resource)
	}

	return nil
}

// validatePipeline validates a Pipeline resource
func (d *TektonDetector) validatePipeline(resource *TektonResource) error {
	if resource.Spec == nil {
		return fmt.Errorf("pipeline spec is required")
	}

	// Check for required fields in spec
	if _, hasParams := resource.Spec["params"]; hasParams {
		// Validate params if present
	}

	if tasks, hasTasks := resource.Spec["tasks"]; hasTasks {
		if taskList, ok := tasks.([]interface{}); ok && len(taskList) == 0 {
			return fmt.Errorf("pipeline must have at least one task")
		}
	}

	return nil
}

// validateTask validates a Task resource
func (d *TektonDetector) validateTask(resource *TektonResource) error {
	if resource.Spec == nil {
		return fmt.Errorf("task spec is required")
	}

	// Check for steps
	if steps, hasSteps := resource.Spec["steps"]; hasSteps {
		if stepList, ok := steps.([]interface{}); ok && len(stepList) == 0 {
			return fmt.Errorf("task must have at least one step")
		}
	}

	return nil
}

// validatePipelineRun validates a PipelineRun resource
func (d *TektonDetector) validatePipelineRun(resource *TektonResource) error {
	if resource.Spec == nil {
		return fmt.Errorf("pipelineRun spec is required")
	}

	// Check for pipeline reference
	if pipelineRef, hasPipelineRef := resource.Spec["pipelineRef"]; hasPipelineRef {
		if pipelineRefMap, ok := pipelineRef.(map[string]interface{}); ok {
			if name, hasName := pipelineRefMap["name"]; !hasName || name == "" {
				return fmt.Errorf("pipelineRun must reference a pipeline by name")
			}
		}
	} else if _, hasPipelineSpec := resource.Spec["pipelineSpec"]; !hasPipelineSpec {
		return fmt.Errorf("pipelineRun must have either pipelineRef or pipelineSpec")
	}

	return nil
}

// validateTaskRun validates a TaskRun resource
func (d *TektonDetector) validateTaskRun(resource *TektonResource) error {
	if resource.Spec == nil {
		return fmt.Errorf("taskRun spec is required")
	}

	// Check for task reference
	if taskRef, hasTaskRef := resource.Spec["taskRef"]; hasTaskRef {
		if taskRefMap, ok := taskRef.(map[string]interface{}); ok {
			if name, hasName := taskRefMap["name"]; !hasName || name == "" {
				return fmt.Errorf("taskRun must reference a task by name")
			}
		}
	} else if _, hasTaskSpec := resource.Spec["taskSpec"]; !hasTaskSpec {
		return fmt.Errorf("taskRun must have either taskRef or taskSpec")
	}

	return nil
}
