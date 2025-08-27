package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnnynv/RepoSentry/internal/tekton"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	// Generate command flags
	generateOutputDir      string
	generateSystemNS       string
	generateServiceAccount string
	generateCloneImage     string
	generateKubectlImage   string
	generateTektonImage    string
	generateDryRun         bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate deployment resources",
	Long: `Generate various deployment resources for RepoSentry.

This command provides subcommands to generate different types of deployment
resources such as Bootstrap Pipeline YAML files.`,
}

// generateBootstrapCmd represents the bootstrap-pipeline subcommand
var generateBootstrapCmd = &cobra.Command{
	Use:   "bootstrap-pipeline",
	Short: "Generate Bootstrap Pipeline YAML for deployment",
	Long: `Generate Bootstrap Pipeline YAML files for static deployment.

This command creates all necessary Tekton resources including:
- Bootstrap Pipeline
- Bootstrap Tasks
- RBAC resources (ServiceAccount, Role, RoleBinding)
- System namespace

The generated YAML files can be deployed to a Kubernetes cluster
before starting RepoSentry to enable Tekton integration.`,
	Example: `  # Generate Bootstrap Pipeline YAML to default directory
  reposentry generate bootstrap-pipeline

  # Generate to specific directory
  reposentry generate bootstrap-pipeline --output ./my-deployments/

  # Generate with custom system namespace
  reposentry generate bootstrap-pipeline --system-namespace my-reposentry-system

  # Dry run - output to stdout only
  reposentry generate bootstrap-pipeline --dry-run`,
	RunE: runGenerateBootstrap,
}

func runGenerateBootstrap(cmd *cobra.Command, args []string) error {
	// Initialize logger
	loggerManager, err := logger.NewLogger(logger.Config{
		Level:  globalLogLevel,
		Format: globalLogFormat,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	parentLogger := loggerManager.WithField("component", "generate-bootstrap")

	parentLogger.WithFields(logger.Fields{
		"output_dir":       generateOutputDir,
		"system_namespace": generateSystemNS,
		"dry_run":          generateDryRun,
	}).Info("Starting Bootstrap Pipeline generation")

	// Create static bootstrap generator
	generator := tekton.NewStaticBootstrapGenerator(parentLogger)

	// Configure generation
	config := &tekton.StaticBootstrapConfig{
		SystemNamespace: generateSystemNS,
		ServiceAccount:  generateServiceAccount,
		CloneImage:      generateCloneImage,
		KubectlImage:    generateKubectlImage,
		TektonImage:     generateTektonImage,
		OutputDirectory: generateOutputDir,
	}

	// Generate static infrastructure
	output, err := generator.GenerateStaticBootstrapInfrastructure(config)
	if err != nil {
		return fmt.Errorf("failed to generate Bootstrap Pipeline infrastructure: %w", err)
	}

	// Handle output
	if generateDryRun {
		return outputToStdout(output)
	} else {
		return writeToFiles(output, generateOutputDir, parentLogger)
	}
}

// outputToStdout outputs all generated YAML to stdout
func outputToStdout(output *tekton.StaticBootstrapOutput) error {
	fmt.Println("---")
	fmt.Println("# Generated Bootstrap Pipeline Infrastructure")
	fmt.Printf("# Generated at: %s\n", output.GeneratedAt)
	fmt.Println("---")

	// Output namespace
	if output.Namespace != "" {
		fmt.Println(output.Namespace)
	}

	// Output pipeline
	if output.Pipeline != "" {
		fmt.Println(output.Pipeline)
	}

	// Output tasks
	for _, task := range output.Tasks {
		if task != "" {
			fmt.Println(task)
		}
	}

	// Output RBAC resources
	if output.ServiceAccount != "" {
		fmt.Println(output.ServiceAccount)
	}

	if output.Role != "" {
		fmt.Println(output.Role)
	}

	if output.RoleBinding != "" {
		fmt.Println(output.RoleBinding)
	}

	return nil
}

// writeToFiles writes all generated YAML to files in the output directory
func writeToFiles(output *tekton.StaticBootstrapOutput, outputDir string, logger *logger.Entry) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	logger.WithField("output_directory", outputDir).Info("Writing generated files")

	// Define files to write
	filesToWrite := map[string]string{
		"00-namespace.yaml":      output.Namespace,
		"01-pipeline.yaml":       output.Pipeline,
		"02-tasks.yaml":          joinTasks(output.Tasks),
		"03-serviceaccount.yaml": output.ServiceAccount,
		"04-role.yaml":           output.Role,
		"05-rolebinding.yaml":    output.RoleBinding,
	}

	// Write files
	for filename, content := range filesToWrite {
		if content == "" {
			continue // Skip empty content
		}

		filePath := filepath.Join(outputDir, filename)
		if err := writeYAMLFile(filePath, content); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}

		logger.WithField("file", filename).Info("Generated file")
	}

	// Create README file
	readmePath := filepath.Join(outputDir, "README.md")
	if err := writeReadmeFile(readmePath, output); err != nil {
		logger.WithError(err).Warn("Failed to write README file")
	}

	logger.WithField("files_written", len(filesToWrite)).
		WithField("output_dir", outputDir).
		Info("Bootstrap Pipeline generation completed successfully")

	return nil
}

// joinTasks joins multiple task YAML strings
func joinTasks(tasks []string) string {
	var result string
	for _, task := range tasks {
		if task != "" {
			result += task + "\n"
		}
	}
	return result
}

// writeYAMLFile writes content to a YAML file
func writeYAMLFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write YAML header comment
	_, err = file.WriteString(fmt.Sprintf("# Generated by RepoSentry\n# File: %s\n---\n", filepath.Base(filePath)))
	if err != nil {
		return err
	}

	// Write content
	_, err = file.WriteString(content)
	return err
}

// writeReadmeFile writes a README.md file with deployment instructions
func writeReadmeFile(filePath string, output *tekton.StaticBootstrapOutput) error {
	readme := "# RepoSentry Bootstrap Pipeline\n\n"
	readme += fmt.Sprintf("Generated at: %s\n\n", output.GeneratedAt)
	readme += "## Overview\n\n"
	readme += "This directory contains the static Bootstrap Pipeline infrastructure for RepoSentry Tekton integration.\n\n"
	readme += "## Files\n\n"
	readme += "- **00-namespace.yaml**: System namespace\n"
	readme += "- **01-pipeline.yaml**: Bootstrap Pipeline definition\n"
	readme += "- **02-tasks.yaml**: Bootstrap Tasks\n"
	readme += "- **03-serviceaccount.yaml**: Service account for Bootstrap Pipeline\n"
	readme += "- **04-role.yaml**: RBAC role definition\n"
	readme += "- **05-rolebinding.yaml**: Role binding\n\n"
	readme += "## Deployment\n\n"
	readme += "To deploy the Bootstrap Pipeline infrastructure:\n\n"
	readme += "### 1. Apply all resources\n"
	readme += "```bash\n"
	readme += "kubectl apply -f .\n"
	readme += "```\n\n"
	readme += "### 2. Verify deployment\n"
	readme += "```bash\n"
	readme += "# Check namespace\n"
	readme += "kubectl get namespace reposentry-system\n\n"
	readme += "# Check pipeline\n"
	readme += "kubectl get pipeline -n reposentry-system\n\n"
	readme += "# Check tasks\n"
	readme += "kubectl get task -n reposentry-system\n\n"
	readme += "# Check RBAC\n"
	readme += "kubectl get serviceaccount,role,rolebinding -n reposentry-system\n"
	readme += "```\n\n"
	readme += "### 3. Configure RepoSentry\n"
	readme += "Ensure your RepoSentry configuration has Tekton enabled:\n\n"
	readme += "```yaml\n"
	readme += "tekton:\n"
	readme += "  enabled: true\n"
	readme += "  # Other Tekton configuration...\n"
	readme += "```\n\n"
	readme += "## Next Steps\n\n"
	readme += "1. Deploy these resources to your Kubernetes cluster\n"
	readme += "2. Configure RepoSentry with proper Tekton settings\n"
	readme += "3. Start RepoSentry - it will automatically trigger the Bootstrap Pipeline when detecting Tekton resources in monitored repositories\n\n"
	readme += "## Troubleshooting\n\n"
	readme += "- Ensure your cluster has Tekton Pipelines installed\n"
	readme += "- Verify RBAC permissions for the reposentry-bootstrap-sa service account\n"
	readme += "- Check Bootstrap Pipeline logs: `kubectl logs -n reposentry-system -l tekton.dev/pipeline=reposentry-bootstrap-pipeline`\n"

	return os.WriteFile(filePath, []byte(readme), 0644)
}

func init() {
	// Add generate command to root
	rootCmd.AddCommand(generateCmd)

	// Add bootstrap-pipeline subcommand to generate
	generateCmd.AddCommand(generateBootstrapCmd)

	// Bootstrap pipeline flags
	generateBootstrapCmd.Flags().StringVarP(&generateOutputDir, "output", "o", "./deployments/tekton/bootstrap", "Output directory for generated files")
	generateBootstrapCmd.Flags().StringVar(&generateSystemNS, "system-namespace", "reposentry-system", "System namespace for Bootstrap Pipeline")
	generateBootstrapCmd.Flags().StringVar(&generateServiceAccount, "service-account", "reposentry-bootstrap-sa", "Service account name")
	generateBootstrapCmd.Flags().StringVar(&generateCloneImage, "clone-image", "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2", "Git clone image")
	generateBootstrapCmd.Flags().StringVar(&generateKubectlImage, "kubectl-image", "bitnami/kubectl:1.28", "kubectl image")
	generateBootstrapCmd.Flags().StringVar(&generateTektonImage, "tekton-image", "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller:v0.40.2", "Tekton controller image")
	generateBootstrapCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Output to stdout instead of files")
}
