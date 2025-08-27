package tekton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStaticBootstrapGenerator(t *testing.T) {
	parentLogger := createTestLogger()

	generator := NewStaticBootstrapGenerator(parentLogger)

	if generator == nil {
		t.Fatal("Expected generator to be created, got nil")
	}

	if generator.logger == nil {
		t.Fatal("Expected logger to be set")
	}
}

func TestStaticBootstrapGenerator_SetDefaults(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{}

	err := generator.setDefaults(config)
	if err != nil {
		t.Fatalf("Expected no error setting defaults, got: %v", err)
	}

	// Verify defaults are set
	if config.SystemNamespace != "reposentry-system" {
		t.Errorf("Expected SystemNamespace to be 'reposentry-system', got: %s", config.SystemNamespace)
	}

	if config.ServiceAccount != "reposentry-bootstrap-sa" {
		t.Errorf("Expected ServiceAccount to be 'reposentry-bootstrap-sa', got: %s", config.ServiceAccount)
	}

	if config.CloneImage == "" {
		t.Error("Expected CloneImage to be set")
	}

	if config.KubectlImage == "" {
		t.Error("Expected KubectlImage to be set")
	}

	if config.TektonImage == "" {
		t.Error("Expected TektonImage to be set")
	}

	if config.ResourceLimits == nil {
		t.Error("Expected ResourceLimits to be set")
	} else {
		if config.ResourceLimits["cpu"] == "" {
			t.Error("Expected CPU limit to be set")
		}
		if config.ResourceLimits["memory"] == "" {
			t.Error("Expected memory limit to be set")
		}
	}

	if config.SecurityContext == nil {
		t.Error("Expected SecurityContext to be set")
	}

	if config.OutputDirectory != "./deployments/tekton/bootstrap" {
		t.Errorf("Expected OutputDirectory to be './deployments/tekton/bootstrap', got: %s", config.OutputDirectory)
	}
}

func TestStaticBootstrapGenerator_GenerateSystemNamespace(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "test-system",
	}

	namespace, err := generator.generateSystemNamespace(config)
	if err != nil {
		t.Fatalf("Expected no error generating namespace, got: %v", err)
	}

	if namespace == "" {
		t.Fatal("Expected namespace YAML to be generated")
	}

	// Verify namespace contains expected content
	if !strings.Contains(namespace, "name: test-system") {
		t.Error("Expected namespace YAML to contain namespace name")
	}

	if !strings.Contains(namespace, "reposentry.io/component: \"system\"") {
		t.Error("Expected namespace YAML to contain component label")
	}

	if !strings.Contains(namespace, "reposentry.io/type: \"bootstrap-infrastructure\"") {
		t.Error("Expected namespace YAML to contain type label")
	}
}

func TestStaticBootstrapGenerator_GenerateStaticBootstrapPipeline(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "test-system",
	}

	pipeline, err := generator.generateStaticBootstrapPipeline(config)
	if err != nil {
		t.Fatalf("Expected no error generating pipeline, got: %v", err)
	}

	if pipeline == "" {
		t.Fatal("Expected pipeline YAML to be generated")
	}

	// Verify pipeline contains expected content
	expectedContent := []string{
		"name: reposentry-bootstrap-pipeline",
		"namespace: test-system",
		"reposentry.io/component: \"bootstrap\"",
		"reposentry.io/type: \"pipeline\"",
		"repo-url",
		"repo-branch",
		"commit-sha",
		"target-namespace",
		"tekton-path",
		"clone-user-repository",
		"compute-target-namespace",
		"validate-tekton-resources",
		"ensure-user-namespace",
		"apply-user-resources",
	}

	for _, content := range expectedContent {
		if !strings.Contains(pipeline, content) {
			t.Errorf("Expected pipeline YAML to contain: %s", content)
		}
	}
}

func TestStaticBootstrapGenerator_GenerateStaticBootstrapTasks(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "test-system",
	}

	tasks, err := generator.generateStaticBootstrapTasks(config)
	if err != nil {
		t.Fatalf("Expected no error generating tasks, got: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatal("Expected tasks to be generated")
	}

	// Verify we have the expected number of tasks
	expectedTaskCount := 5 // clone, compute-namespace, validate, ensure-namespace, apply
	if len(tasks) != expectedTaskCount {
		t.Errorf("Expected %d tasks, got: %d", expectedTaskCount, len(tasks))
	}

	// Verify all tasks are non-empty strings (placeholder check)
	for i, task := range tasks {
		if task == "" {
			t.Errorf("Expected task %d to be non-empty", i)
		}
	}
}

func TestStaticBootstrapGenerator_GenerateStaticRBACResources(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "test-system",
		ServiceAccount:  "test-sa",
	}

	output := &StaticBootstrapOutput{}

	err := generator.generateStaticRBACResources(config, output)
	if err != nil {
		t.Fatalf("Expected no error generating RBAC resources, got: %v", err)
	}

	// Verify RBAC resources are generated
	if output.ServiceAccount == "" {
		t.Error("Expected ServiceAccount to be generated")
	}

	if output.Role == "" {
		t.Error("Expected Role to be generated")
	}

	if output.RoleBinding == "" {
		t.Error("Expected RoleBinding to be generated")
	}
}

func TestStaticBootstrapGenerator_GenerateStaticBootstrapInfrastructure(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "test-system",
		OutputDirectory: "./test-output",
	}

	output, err := generator.GenerateStaticBootstrapInfrastructure(config)
	if err != nil {
		t.Fatalf("Expected no error generating infrastructure, got: %v", err)
	}

	if output == nil {
		t.Fatal("Expected output to be generated")
	}

	// Verify all components are generated
	if output.Namespace == "" {
		t.Error("Expected Namespace to be generated")
	}

	if output.Pipeline == "" {
		t.Error("Expected Pipeline to be generated")
	}

	if len(output.Tasks) == 0 {
		t.Error("Expected Tasks to be generated")
	}

	if output.ServiceAccount == "" {
		t.Error("Expected ServiceAccount to be generated")
	}

	if output.Role == "" {
		t.Error("Expected Role to be generated")
	}

	if output.RoleBinding == "" {
		t.Error("Expected RoleBinding to be generated")
	}

	if output.GeneratedAt == "" {
		t.Error("Expected GeneratedAt timestamp to be set")
	}
}

func TestStaticBootstrapGenerator_GenerateStaticBootstrapInfrastructure_WithCustomConfig(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	config := &StaticBootstrapConfig{
		SystemNamespace: "custom-system",
		ServiceAccount:  "custom-sa",
		CloneImage:      "custom/clone:latest",
		KubectlImage:    "custom/kubectl:latest",
		TektonImage:     "custom/tekton:latest",
		ResourceLimits: map[string]string{
			"cpu":    "1000m",
			"memory": "1Gi",
		},
		SecurityContext: map[string]interface{}{
			"runAsUser": 1000,
		},
		OutputDirectory: "./custom-output",
	}

	output, err := generator.GenerateStaticBootstrapInfrastructure(config)
	if err != nil {
		t.Fatalf("Expected no error generating infrastructure with custom config, got: %v", err)
	}

	if output == nil {
		t.Fatal("Expected output to be generated")
	}

	// Verify custom configuration is preserved
	if !strings.Contains(output.Namespace, "custom-system") {
		t.Error("Expected custom namespace name to be used")
	}
}

func TestStaticBootstrapGenerator_WriteToFiles(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bootstrap-write-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create test output
	output := &StaticBootstrapOutput{
		Namespace:      "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: test-system",
		Pipeline:       "apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: test-pipeline",
		Tasks:          []string{"apiVersion: tekton.dev/v1beta1\nkind: Task\nmetadata:\n  name: test-task"},
		ServiceAccount: "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: test-sa",
		Role:           "apiVersion: rbac.authorization.k8s.io/v1\nkind: Role\nmetadata:\n  name: test-role",
		RoleBinding:    "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: test-rb",
	}

	// Test successful write
	err = generator.WriteToFiles(output, tempDir)
	if err != nil {
		t.Fatalf("Expected no error writing files, got: %v", err)
	}

	// Verify all files were created
	expectedFiles := []string{
		"00-namespace.yaml",
		"01-pipeline.yaml",
		"02-tasks.yaml",
		"03-serviceaccount.yaml",
		"04-role.yaml",
		"05-rolebinding.yaml",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to be created", filename)
		}

		// Verify file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", filename, err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("Expected file %s to have content", filename)
		}
	}

	// Verify tasks file contains all tasks
	tasksFile := filepath.Join(tempDir, "02-tasks.yaml")
	tasksContent, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Fatalf("Failed to read tasks file: %v", err)
	}

	if !strings.Contains(string(tasksContent), "test-task") {
		t.Error("Expected tasks file to contain test-task")
	}
}

func TestStaticBootstrapGenerator_WriteToFiles_DirectoryCreation(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bootstrap-write-nested-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Test with nested directory that doesn't exist
	nestedDir := filepath.Join(tempDir, "nested", "path")

	output := &StaticBootstrapOutput{
		Namespace:      "test-namespace",
		Pipeline:       "test-pipeline",
		Tasks:          []string{"test-task"},
		ServiceAccount: "test-sa",
		Role:           "test-role",
		RoleBinding:    "test-rb",
	}

	err = generator.WriteToFiles(output, nestedDir)
	if err != nil {
		t.Fatalf("Expected no error creating nested directory, got: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Expected nested directory to be created")
	}

	// Verify files were created in nested directory
	namespaceFile := filepath.Join(nestedDir, "00-namespace.yaml")
	if _, err := os.Stat(namespaceFile); os.IsNotExist(err) {
		t.Error("Expected namespace file to be created in nested directory")
	}
}

func TestStaticBootstrapGenerator_WriteToFiles_PermissionError(t *testing.T) {
	parentLogger := createTestLogger()
	generator := NewStaticBootstrapGenerator(parentLogger)

	// Test with invalid directory (root directory with no permissions)
	invalidDir := "/invalid/nonexistent/path"

	output := &StaticBootstrapOutput{
		Namespace:      "test-namespace",
		Pipeline:       "test-pipeline",
		Tasks:          []string{"test-task"},
		ServiceAccount: "test-sa",
		Role:           "test-role",
		RoleBinding:    "test-rb",
	}

	err := generator.WriteToFiles(output, invalidDir)
	if err == nil {
		t.Error("Expected error writing to invalid directory")
	}

	if !strings.Contains(err.Error(), "failed to create output directory") {
		t.Errorf("Expected error message about directory creation, got: %v", err)
	}
}
