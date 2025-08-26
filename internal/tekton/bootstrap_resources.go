package tekton

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"text/template"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// generateSupportingResources generates ServiceAccount, RBAC, ResourceQuota, etc.
func (bpg *BootstrapPipelineGenerator) generateSupportingResources(config *BootstrapPipelineConfig, resources *BootstrapPipelineResources) error {
	// Generate ServiceAccount
	serviceAccount, err := bpg.generateServiceAccount(config)
	if err != nil {
		return fmt.Errorf("failed to generate ServiceAccount: %w", err)
	}
	resources.ServiceAccount = serviceAccount

	// Generate RoleBinding
	roleBinding, err := bpg.generateRoleBinding(config)
	if err != nil {
		return fmt.Errorf("failed to generate RoleBinding: %w", err)
	}
	resources.RoleBinding = roleBinding

	// Generate ResourceQuota
	resourceQuota, err := bpg.generateResourceQuota(config)
	if err != nil {
		return fmt.Errorf("failed to generate ResourceQuota: %w", err)
	}
	resources.ResourceQuota = resourceQuota

	// Generate NetworkPolicy
	networkPolicy, err := bpg.generateNetworkPolicy(config)
	if err != nil {
		return fmt.Errorf("failed to generate NetworkPolicy: %w", err)
	}
	resources.NetworkPolicy = networkPolicy

	return nil
}

// generateServiceAccount generates ServiceAccount for Bootstrap Pipeline
func (bpg *BootstrapPipelineGenerator) generateServiceAccount(config *BootstrapPipelineConfig) (string, error) {
	saTemplate := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{.ServiceAccount}}
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/repository: "{{.Repository.Name}}"
  annotations:
    reposentry.io/repository-url: "{{.Repository.URL}}"
    reposentry.io/commit-sha: "{{.CommitSHA}}"
    reposentry.io/branch: "{{.Branch}}"
automountServiceAccountToken: true
---
apiVersion: v1
kind: Secret
metadata:
  name: {{.ServiceAccount}}-token
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
  annotations:
    kubernetes.io/service-account.name: {{.ServiceAccount}}
type: kubernetes.io/service-account-token
`

	tmpl, err := template.New("service-account").Parse(saTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse ServiceAccount template: %w", err)
	}

	var saBuffer strings.Builder
	if err := tmpl.Execute(&saBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute ServiceAccount template: %w", err)
	}

	return saBuffer.String(), nil
}

// generateRoleBinding generates RBAC for Bootstrap Pipeline
func (bpg *BootstrapPipelineGenerator) generateRoleBinding(config *BootstrapPipelineConfig) (string, error) {
	rbacTemplate := `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: reposentry-bootstrap-role
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
rules:
# Tekton resources
- apiGroups: ["tekton.dev"]
  resources: ["tasks", "pipelines", "pipelineruns", "taskruns"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
- apiGroups: ["tekton.dev"]
  resources: ["clustertasks"]
  verbs: ["get", "list"]
# Core resources needed for pipeline execution
- apiGroups: [""]
  resources: ["pods", "pods/log", "configmaps", "secrets", "persistentvolumeclaims"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "update", "patch"]
# Triggers resources (if using Tekton Triggers)
- apiGroups: ["triggers.tekton.dev"]
  resources: ["eventlisteners", "triggerbindings", "triggertemplates", "triggers"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: reposentry-bootstrap-binding
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
subjects:
- kind: ServiceAccount
  name: {{.ServiceAccount}}
  namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: reposentry-bootstrap-role
  apiGroup: rbac.authorization.k8s.io
`

	tmpl, err := template.New("role-binding").Parse(rbacTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse RBAC template: %w", err)
	}

	var rbacBuffer strings.Builder
	if err := tmpl.Execute(&rbacBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute RBAC template: %w", err)
	}

	return rbacBuffer.String(), nil
}

// generateResourceQuota generates ResourceQuota for namespace isolation
func (bpg *BootstrapPipelineGenerator) generateResourceQuota(config *BootstrapPipelineConfig) (string, error) {
	quotaTemplate := `
apiVersion: v1
kind: ResourceQuota
metadata:
  name: reposentry-bootstrap-quota
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
    reposentry.io/repository: "{{.Repository.Name}}"
spec:
  hard:
    # Compute resources
    requests.cpu: "2"
    requests.memory: "4Gi"
    limits.cpu: "4"
    limits.memory: "8Gi"
    
    # Storage resources
    requests.storage: "10Gi"
    persistentvolumeclaims: "10"
    
    # Object counts
    pods: "20"
    secrets: "20"
    configmaps: "20"
    services: "5"
    
    # Tekton resources
    count/pipelineruns.tekton.dev: "10"
    count/taskruns.tekton.dev: "50"
    count/pipelines.tekton.dev: "10"
    count/tasks.tekton.dev: "20"
---
apiVersion: v1
kind: LimitRange
metadata:
  name: reposentry-bootstrap-limits
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
spec:
  limits:
  - type: Container
    default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
    max:
      cpu: "2"
      memory: "2Gi"
    min:
      cpu: "50m"
      memory: "64Mi"
  - type: Pod
    max:
      cpu: "4"
      memory: "4Gi"
  - type: PersistentVolumeClaim
    max:
      storage: "5Gi"
    min:
      storage: "1Gi"
`

	tmpl, err := template.New("resource-quota").Parse(quotaTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse ResourceQuota template: %w", err)
	}

	var quotaBuffer strings.Builder
	if err := tmpl.Execute(&quotaBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute ResourceQuota template: %w", err)
	}

	return quotaBuffer.String(), nil
}

// generateNetworkPolicy generates NetworkPolicy for network isolation
func (bpg *BootstrapPipelineGenerator) generateNetworkPolicy(config *BootstrapPipelineConfig) (string, error) {
	npTemplate := `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: reposentry-bootstrap-netpol
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap"
spec:
  podSelector:
    matchLabels:
      reposentry.io/managed: "true"
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Allow ingress from same namespace
  - from:
    - namespaceSelector:
        matchLabels:
          name: {{.Namespace}}
  # Allow ingress from Tekton system
  - from:
    - namespaceSelector:
        matchLabels:
          name: tekton-pipelines
  egress:
  # Allow egress to same namespace
  - to:
    - namespaceSelector:
        matchLabels:
          name: {{.Namespace}}
  # Allow egress to Tekton system
  - to:
    - namespaceSelector:
        matchLabels:
          name: tekton-pipelines
  # Allow egress to kube-system (for DNS)
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # Allow egress to external Git repositories (HTTPS)
  - to: []
    ports:
    - protocol: TCP
      port: 443
  # Allow egress to external Git repositories (SSH)
  - to: []
    ports:
    - protocol: TCP
      port: 22
`

	tmpl, err := template.New("network-policy").Parse(npTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse NetworkPolicy template: %w", err)
	}

	var npBuffer strings.Builder
	if err := tmpl.Execute(&npBuffer, config); err != nil {
		return "", fmt.Errorf("failed to execute NetworkPolicy template: %w", err)
	}

	return npBuffer.String(), nil
}

// generateApplyPipelineRun generates PipelineRun for apply mode
func (bpg *BootstrapPipelineGenerator) generateApplyPipelineRun(config *BootstrapPipelineConfig) (string, error) {
	prTemplate := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: reposentry-bootstrap-apply-{{.RunID}}
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-run"
    reposentry.io/action: "apply"
    reposentry.io/repository: "{{.Repository.Name}}"
  annotations:
    reposentry.io/repository-url: "{{.Repository.URL}}"
    reposentry.io/commit-sha: "{{.CommitSHA}}"
    reposentry.io/branch: "{{.Branch}}"
    reposentry.io/scan-path: "{{.Detection.ScanPath}}"
spec:
  pipelineRef:
    name: reposentry-bootstrap-apply
  serviceAccountName: {{.ServiceAccount}}
  params:
  - name: repo-url
    value: "{{.Repository.URL}}"
  - name: commit-sha
    value: "{{.CommitSHA}}"
  - name: branch
    value: "{{.Branch}}"
  - name: tekton-path
    value: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
        storageClassName: standard
  - name: tekton-resources
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
        storageClassName: standard
  timeouts:
    pipeline: "30m"
    tasks: "20m"
    finally: "5m"
`

	// Generate unique run ID
	runID := bpg.generateRunID(config)
	
	// Create template data with RunID
	templateData := struct {
		*BootstrapPipelineConfig
		RunID string
	}{
		BootstrapPipelineConfig: config,
		RunID:                   runID,
	}

	tmpl, err := template.New("apply-pipelinerun").Parse(prTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse apply PipelineRun template: %w", err)
	}

	var prBuffer strings.Builder
	if err := tmpl.Execute(&prBuffer, templateData); err != nil {
		return "", fmt.Errorf("failed to execute apply PipelineRun template: %w", err)
	}

	return prBuffer.String(), nil
}

// generateTriggerPipelineRun generates PipelineRun for trigger mode
func (bpg *BootstrapPipelineGenerator) generateTriggerPipelineRun(config *BootstrapPipelineConfig) (string, error) {
	prTemplate := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: reposentry-bootstrap-trigger-{{.RunID}}
  namespace: {{.Namespace}}
  labels:
    reposentry.io/type: "bootstrap-run"
    reposentry.io/action: "trigger"
    reposentry.io/repository: "{{.Repository.Name}}"
  annotations:
    reposentry.io/repository-url: "{{.Repository.URL}}"
    reposentry.io/commit-sha: "{{.CommitSHA}}"
    reposentry.io/branch: "{{.Branch}}"
spec:
  pipelineRef:
    name: reposentry-bootstrap-trigger
  serviceAccountName: {{.ServiceAccount}}
  params:
  - name: repo-url
    value: "{{.Repository.URL}}"
  - name: commit-sha
    value: "{{.CommitSHA}}"
  - name: branch
    value: "{{.Branch}}"
  - name: tekton-path
    value: "{{.Detection.ScanPath}}"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
  - name: tekton-resources
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: {{.WorkspaceSize}}
  timeouts:
    pipeline: "1h"
    tasks: "45m"
    finally: "10m"
`

	runID := bpg.generateRunID(config)
	templateData := struct {
		*BootstrapPipelineConfig
		RunID string
	}{
		BootstrapPipelineConfig: config,
		RunID:                   runID,
	}

	tmpl, err := template.New("trigger-pipelinerun").Parse(prTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse trigger PipelineRun template: %w", err)
	}

	var prBuffer strings.Builder
	if err := tmpl.Execute(&prBuffer, templateData); err != nil {
		return "", fmt.Errorf("failed to execute trigger PipelineRun template: %w", err)
	}

	return prBuffer.String(), nil
}

// generateRunID generates a unique ID for PipelineRun
func (bpg *BootstrapPipelineGenerator) generateRunID(config *BootstrapPipelineConfig) string {
	// Create unique ID based on repository, commit, and timestamp
	data := fmt.Sprintf("%s:%s:%s:%s", 
		config.Repository.Name, 
		config.CommitSHA, 
		config.Branch,
		config.Detection.DetectedAt.Format("20060102-150405"))
	
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:6]) // Use first 6 bytes for shorter ID
}

// GetGeneratedNamespace generates a secure namespace name for a repository
func GetGeneratedNamespace(repository types.Repository) string {
	// Use the implementation from bootstrap-pipeline-architecture.md
	repoString := fmt.Sprintf("%s/%s", extractOwnerFromURL(repository.URL), repository.Name)
	hash := sha256.Sum256([]byte(repoString))
	return fmt.Sprintf("reposentry-user-repo-%x", hash[:8])
}
