package tekton

import (
	"fmt"
	"strings"
	"text/template"
)

// generateApplyTasks generates tasks for apply mode
func (bpg *BootstrapPipelineGenerator) generateApplyTasks(config *BootstrapPipelineConfig) ([]string, error) {
	var tasks []string

	// Clone task
	cloneTask, err := bpg.generateCloneTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate clone task: %w", err)
	}
	tasks = append(tasks, cloneTask)

	// Validate task
	validateTask, err := bpg.generateValidateTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate validate task: %w", err)
	}
	tasks = append(tasks, validateTask)

	// Apply resources task
	applyTask, err := bpg.generateApplyResourcesTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apply task: %w", err)
	}
	tasks = append(tasks, applyTask)

	return tasks, nil
}

// generateTriggerTasks generates tasks for trigger mode
func (bpg *BootstrapPipelineGenerator) generateTriggerTasks(config *BootstrapPipelineConfig) ([]string, error) {
	var tasks []string

	// Get apply tasks first
	applyTasks, err := bpg.generateApplyTasks(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apply tasks: %w", err)
	}
	tasks = append(tasks, applyTasks...)

	// Add trigger runs task
	triggerTask, err := bpg.generateTriggerRunsTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate trigger task: %w", err)
	}
	tasks = append(tasks, triggerTask)

	return tasks, nil
}

// generateValidateTasks generates tasks for validate mode
func (bpg *BootstrapPipelineGenerator) generateValidateTasks(config *BootstrapPipelineConfig) ([]string, error) {
	var tasks []string

	// Clone task
	cloneTask, err := bpg.generateCloneTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate clone task: %w", err)
	}
	tasks = append(tasks, cloneTask)

	// Validate only task
	validateOnlyTask, err := bpg.generateValidateOnlyTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate validate-only task: %w", err)
	}
	tasks = append(tasks, validateOnlyTask)

	return tasks, nil
}

// generateSkipTasks generates tasks for skip mode
func (bpg *BootstrapPipelineGenerator) generateSkipTasks(config *BootstrapPipelineConfig) ([]string, error) {
	var tasks []string

	// Notify task
	notifyTask, err := bpg.generateNotifyTask(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate notify task: %w", err)
	}
	tasks = append(tasks, notifyTask)

	return tasks, nil
}

// generateCloneTask generates the repository clone task
func (bpg *BootstrapPipelineGenerator) generateCloneTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-clone
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "clone"
spec:
  description: "Clone repository from remote URL"
  params:
  - name: url
    type: string
    description: "Repository URL to clone"
  - name: revision
    type: string
    description: "Git revision to checkout"
    default: "main"
  - name: depth
    type: string
    description: "Git clone depth"
    default: "1"
  workspaces:
  - name: output
    description: "The workspace to clone the repository to"
  steps:
  - name: clone
    image: {{.CloneImage}}
    workingDir: $(workspaces.output.path)
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: {{.ResourceLimits.cpu}}
        memory: {{.ResourceLimits.memory}}
      requests:
        cpu: "100m"
        memory: "128Mi"
    script: |
      #!/bin/sh
      set -eu
      
      echo "Cloning repository: $(params.url)"
      echo "Revision: $(params.revision)"
      echo "Depth: $(params.depth)"
      
      # Clean any existing content
      rm -rf .git *
      
      # Clone repository
      /ko-app/git-init \
        -url="$(params.url)" \
        -revision="$(params.revision)" \
        -path="$(workspaces.output.path)" \
        -depth="$(params.depth)"
      
      echo "Repository cloned successfully"
      ls -la $(workspaces.output.path)
`

	tmpl, err := template.New("clone-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse clone task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute clone task template: %w", err)
	}

	return taskBuffer.String(), nil
}

// generateValidateTask generates the validation task
func (bpg *BootstrapPipelineGenerator) generateValidateTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-validate
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "validate"
spec:
  description: "Validate and process Tekton resources"
  params:
  - name: tekton-path
    type: string
    description: "Path to Tekton resources in repository"
    default: ".tekton"
  workspaces:
  - name: source
    description: "Source repository workspace"
  - name: output
    description: "Output workspace for processed resources"
  steps:
  - name: validate-resources
    image: {{.KubectlImage}}
    workingDir: $(workspaces.source.path)
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: {{.ResourceLimits.cpu}}
        memory: {{.ResourceLimits.memory}}
      requests:
        cpu: "100m"
        memory: "128Mi"
    script: |
      #!/bin/bash
      set -eu
      
      TEKTON_PATH="$(params.tekton-path)"
      OUTPUT_PATH="$(workspaces.output.path)"
      
      echo "Validating Tekton resources in: $TEKTON_PATH"
      
      # Check if tekton directory exists
      if [ ! -d "$TEKTON_PATH" ]; then
        echo "ERROR: Tekton directory $TEKTON_PATH not found"
        exit 1
      fi
      
      # Create output directory
      mkdir -p "$OUTPUT_PATH"
      
      # Find and validate YAML files
      find "$TEKTON_PATH" -name "*.yaml" -o -name "*.yml" | while read -r file; do
        echo "Processing file: $file"
        
        # Basic YAML validation
        if ! kubectl --dry-run=client --validate=true -f "$file" > /dev/null 2>&1; then
          echo "WARNING: Invalid YAML in $file"
          continue
        fi
        
        # Check if it's a Tekton resource
        if kubectl --dry-run=client -f "$file" 2>&1 | grep -q "tekton.dev"; then
          echo "Valid Tekton resource found: $file"
          
          # Copy to output with namespace injection
          filename=$(basename "$file")
          sed 's/namespace: .*/namespace: {{.Namespace}}/' "$file" > "$OUTPUT_PATH/$filename"
          
          # If no namespace specified, add it
          if ! grep -q "namespace:" "$file"; then
            sed '/metadata:/a\  namespace: {{.Namespace}}' "$OUTPUT_PATH/$filename" > "$OUTPUT_PATH/$filename.tmp"
            mv "$OUTPUT_PATH/$filename.tmp" "$OUTPUT_PATH/$filename"
          fi
        else
          echo "Skipping non-Tekton resource: $file"
        fi
      done
      
      echo "Validation completed. Processed resources:"
      ls -la "$OUTPUT_PATH"
`

	tmpl, err := template.New("validate-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse validate task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute validate task template: %w", err)
	}

	return taskBuffer.String(), nil
}

// generateApplyResourcesTask generates the apply resources task
func (bpg *BootstrapPipelineGenerator) generateApplyResourcesTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-apply-resources
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "apply"
spec:
  description: "Apply Tekton resources to Kubernetes"
  params:
  - name: namespace
    type: string
    description: "Target namespace for resources"
  - name: resource-filter
    type: string
    description: "Filter resources by kind (comma-separated)"
    default: "Pipeline,Task,PipelineRun,TaskRun"
  workspaces:
  - name: resources
    description: "Workspace containing processed Tekton resources"
  steps:
  - name: apply-resources
    image: {{.KubectlImage}}
    workingDir: $(workspaces.resources.path)
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: {{.ResourceLimits.cpu}}
        memory: {{.ResourceLimits.memory}}
      requests:
        cpu: "200m"
        memory: "256Mi"
    script: |
      #!/bin/bash
      set -eu
      
      NAMESPACE="$(params.namespace)"
      RESOURCE_FILTER="$(params.resource-filter)"
      
      echo "Applying Tekton resources to namespace: $NAMESPACE"
      echo "Resource filter: $RESOURCE_FILTER"
      
      # Convert filter to array
      IFS=',' read -ra FILTERS <<< "$RESOURCE_FILTER"
      
      # Apply resources by priority order
      # First apply Tasks and Pipelines (definitions)
      for filter in "${FILTERS[@]}"; do
        filter=$(echo "$filter" | tr -d ' ')
        
        for file in *.yaml *.yml; do
          [ -f "$file" ] || continue
          
          # Check if file contains the specified kind
          if grep -q "kind: $filter" "$file"; then
            echo "Applying $filter from $file"
            
            # Apply with proper error handling
            if kubectl apply -f "$file" -n "$NAMESPACE"; then
              echo "Successfully applied $filter from $file"
            else
              echo "WARNING: Failed to apply $filter from $file"
              # Continue with other resources instead of failing
            fi
          fi
        done
      done
      
      echo "Resource application completed"
      
      # List applied resources
      echo "Resources in namespace $NAMESPACE:"
      kubectl get all -n "$NAMESPACE" --show-labels | grep "reposentry"
`

	tmpl, err := template.New("apply-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse apply task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute apply task template: %w", err)
	}

	return taskBuffer.String(), nil
}

// generateTriggerRunsTask generates the trigger runs task
func (bpg *BootstrapPipelineGenerator) generateTriggerRunsTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-trigger-runs
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "trigger"
spec:
  description: "Trigger PipelineRuns and TaskRuns"
  params:
  - name: namespace
    type: string
    description: "Target namespace"
  workspaces:
  - name: resources
    description: "Workspace containing processed resources"
  steps:
  - name: trigger-runs
    image: {{.KubectlImage}}
    workingDir: $(workspaces.resources.path)
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: {{.ResourceLimits.cpu}}
        memory: {{.ResourceLimits.memory}}
      requests:
        cpu: "200m"
        memory: "256Mi"
    script: |
      #!/bin/bash
      set -eu
      
      NAMESPACE="$(params.namespace)"
      
      echo "Triggering runs in namespace: $NAMESPACE"
      
      # First apply PipelineRuns
      for file in *.yaml *.yml; do
        [ -f "$file" ] || continue
        
        if grep -q "kind: PipelineRun" "$file"; then
          echo "Triggering PipelineRun from $file"
          kubectl apply -f "$file" -n "$NAMESPACE"
        fi
      done
      
      # Then apply TaskRuns
      for file in *.yaml *.yml; do
        [ -f "$file" ] || continue
        
        if grep -q "kind: TaskRun" "$file"; then
          echo "Triggering TaskRun from $file"
          kubectl apply -f "$file" -n "$NAMESPACE"
        fi
      done
      
      echo "Triggered runs completed"
      
      # Show status
      echo "PipelineRuns:"
      kubectl get pipelineruns -n "$NAMESPACE"
      
      echo "TaskRuns:"
      kubectl get taskruns -n "$NAMESPACE"
`

	tmpl, err := template.New("trigger-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse trigger task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute trigger task template: %w", err)
	}

	return taskBuffer.String(), nil
}

// generateValidateOnlyTask generates the validate-only task
func (bpg *BootstrapPipelineGenerator) generateValidateOnlyTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-validate-only
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "validate-only"
spec:
  description: "Validate Tekton resources without applying"
  params:
  - name: tekton-path
    type: string
    description: "Path to Tekton resources"
    default: ".tekton"
  workspaces:
  - name: source
    description: "Source repository workspace"
  steps:
  - name: validate-only
    image: {{.KubectlImage}}
    workingDir: $(workspaces.source.path)
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: {{.ResourceLimits.cpu}}
        memory: {{.ResourceLimits.memory}}
    script: |
      #!/bin/bash
      set -eu
      
      TEKTON_PATH="$(params.tekton-path)"
      
      echo "Validating Tekton resources in: $TEKTON_PATH"
      
      if [ ! -d "$TEKTON_PATH" ]; then
        echo "ERROR: Tekton directory $TEKTON_PATH not found"
        exit 1
      fi
      
      VALID_COUNT=0
      INVALID_COUNT=0
      
      find "$TEKTON_PATH" -name "*.yaml" -o -name "*.yml" | while read -r file; do
        echo "Validating file: $file"
        
        if kubectl --dry-run=client --validate=true -f "$file" > /dev/null 2>&1; then
          echo "✓ Valid: $file"
          VALID_COUNT=$((VALID_COUNT + 1))
        else
          echo "✗ Invalid: $file"
          INVALID_COUNT=$((INVALID_COUNT + 1))
        fi
      done
      
      echo "Validation completed"
      echo "Valid files: $VALID_COUNT"
      echo "Invalid files: $INVALID_COUNT"
      
      if [ $INVALID_COUNT -gt 0 ]; then
        echo "WARNING: Some files failed validation"
        exit 1
      fi
`

	tmpl, err := template.New("validate-only-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse validate-only task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute validate-only task template: %w", err)
	}

	return taskBuffer.String(), nil
}

// generateNotifyTask generates the notification task
func (bpg *BootstrapPipelineGenerator) generateNotifyTask(config *BootstrapPipelineConfig) (string, error) {
	taskTemplate := `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: reposentry-bootstrap-notify
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-task"
    reposentry.io/function: "notify"
spec:
  description: "Send notification about processing status"
  params:
  - name: message
    type: string
    description: "Notification message"
  - name: status
    type: string
    description: "Processing status"
    default: "info"
  steps:
  - name: notify
    image: alpine:3.18
    securityContext:
      runAsNonRoot: {{.SecurityContext.runAsNonRoot}}
      runAsUser: {{.SecurityContext.runAsUser}}
    resources:
      limits:
        cpu: "100m"
        memory: "128Mi"
    script: |
      #!/bin/sh
      set -eu
      
      MESSAGE="$(params.message)"
      STATUS="$(params.status)"
      
      echo "=== RepoSentry Bootstrap Notification ==="
      echo "Status: $STATUS"
      echo "Message: $MESSAGE"
      echo "Timestamp: $(date -Iseconds)"
      echo "Repository: {{.Repository.Name}}"
      echo "Commit: {{.CommitSHA}}"
      echo "Branch: {{.Branch}}"
      echo "========================================="
      
      # TODO: Add webhook notification, Slack integration, etc.
`

	tmpl, err := template.New("notify-task").Parse(taskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse notify task template: %w", err)
	}

	var taskBuffer strings.Builder
	if err := tmpl.Execute(&taskBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute notify task template: %w", err)
	}

	return taskBuffer.String(), nil
}
