package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate RepoSentry configuration file",
	Long: `Validate the syntax and content of a RepoSentry configuration file.
	
This command checks:
- YAML syntax validation
- Required fields presence
- Field value validation
- Repository URL format
- Environment variable resolution
- Tekton configuration validity`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

var (
	validateEnvironment bool
	validateConnections bool
	validateFormat      string
)

func init() {
	validateCmd.Flags().BoolVar(&validateEnvironment, "check-env", false, "Validate environment variables")
	validateCmd.Flags().BoolVar(&validateConnections, "check-connections", false, "Test connectivity to external services")
	validateCmd.Flags().StringVar(&validateFormat, "format", "text", "Output format (text, json)")
	
	configCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Determine config file
	configFile := "./config.yaml"
	if len(args) > 0 {
		configFile = args[0]
	}
	// Use global config file if set via flag
	if configFileFlag := cmd.Flag("config").Value.String(); configFileFlag != "" {
		configFile = configFileFlag
	}

	// Initialize logger
	appLogger := logger.GetDefaultLogger()
	
	appLogger.WithFields(logger.Fields{
		"config_file":        configFile,
		"check_environment":  validateEnvironment,
		"check_connections":  validateConnections,
		"format":            validateFormat,
	}).Info("Starting configuration validation")

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configFile)
	}

	// Create config manager
	configManager := config.NewManager(appLogger)
	
	// Load and validate configuration
	err := configManager.Load(configFile)
	if err != nil {
		if validateFormat == "json" {
			return printValidationResultJSON(configFile, false, []string{err.Error()}, nil)
		}
		
		fmt.Printf("❌ Configuration validation FAILED\n\n")
		fmt.Printf("File: %s\n", configFile)
		fmt.Printf("Error: %v\n", err)
		return fmt.Errorf("configuration validation failed")
	}

	// Get the loaded configuration
	cfg := configManager.Get()

	var warnings []string
	var errors []string

	// Additional validations
	if validateEnvironment {
		envWarnings := validateEnvironmentVariables(cfg)
		warnings = append(warnings, envWarnings...)
	}

	if validateConnections {
		connErrors := validateConnectivity(cfg, appLogger)
		errors = append(errors, connErrors...)
	}

	// Basic configuration checks
	configWarnings := performBasicChecks(cfg)
	warnings = append(warnings, configWarnings...)

	// Print results
	if validateFormat == "json" {
		return printValidationResultJSON(configFile, len(errors) == 0, errors, warnings)
	}

	return printValidationResultText(configFile, cfg, errors, warnings)
}

func validateEnvironmentVariables(cfg *types.Config) []string {
	var warnings []string
	
	// Check repositories for environment variables
	for i, repo := range cfg.Repositories {
		if repo.Token == "" {
			warnings = append(warnings, fmt.Sprintf("Repository %d (%s) has no token configured", i, repo.Name))
		}
		
		// Check if token looks like environment variable pattern
		if len(repo.Token) > 0 && repo.Token[0] == '$' {
			envVar := repo.Token[1:] // Remove $
			if os.Getenv(envVar) == "" {
				warnings = append(warnings, fmt.Sprintf("Environment variable %s is not set for repository %s", envVar, repo.Name))
			}
		}
	}

	// TODO: Add Tekton auth token check when field is available

	return warnings
}

func validateConnectivity(cfg *types.Config, logger *logger.Logger) []string {
	var errors []string
	
	// TODO: Add connectivity tests
	// - Test Tekton EventListener URL
	// - Test repository URLs (if tokens available)
	
	logger.Info("Connectivity validation not yet implemented")
	
	return errors
}

func performBasicChecks(cfg *types.Config) []string {
	var warnings []string
	
	// Check polling intervals
	if cfg.Polling.Interval < 60 {
		warnings = append(warnings, "Polling interval is less than 1 minute, this may cause API rate limiting")
	}
	
	// Check for duplicate repository names
	namesSeen := make(map[string]bool)
	for _, repo := range cfg.Repositories {
		if namesSeen[repo.Name] {
			warnings = append(warnings, fmt.Sprintf("Duplicate repository name: %s", repo.Name))
		}
		namesSeen[repo.Name] = true
	}
	
	// Check storage configuration
	if cfg.Storage.SQLite.MaxConnections > 100 {
		warnings = append(warnings, "SQLite max connections is very high, consider reducing for better performance")
	}
	
	return warnings
}

func printValidationResultJSON(file string, valid bool, errors, warnings []string) error {
	result := map[string]interface{}{
		"file":     file,
		"valid":    valid,
		"errors":   errors,
		"warnings": warnings,
	}
	
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	fmt.Println(string(jsonBytes))
	
	if !valid {
		os.Exit(1)
	}
	
	return nil
}

func printValidationResultText(file string, cfg *types.Config, errors, warnings []string) error {
	if len(errors) == 0 {
		fmt.Printf("✅ Configuration validation PASSED\n\n")
	} else {
		fmt.Printf("❌ Configuration validation FAILED\n\n")
	}
	
	fmt.Printf("File: %s\n", file)
	fmt.Printf("App: %s\n", cfg.App.Name)
	fmt.Printf("Repositories: %d\n", len(cfg.Repositories))
	fmt.Printf("Storage: %s\n", cfg.Storage.Type)
	fmt.Printf("Tekton URL: %s\n", cfg.Tekton.EventListenerURL)
	fmt.Println()
	
	if len(errors) > 0 {
		fmt.Printf("🚨 Errors:\n")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		fmt.Println()
	}
	
	if len(warnings) > 0 {
		fmt.Printf("⚠️  Warnings:\n")
		for _, warn := range warnings {
			fmt.Printf("  - %s\n", warn)
		}
		fmt.Println()
	}
	
	if len(errors) == 0 && len(warnings) == 0 {
		fmt.Printf("🎉 No issues found!\n")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed with %d error(s)", len(errors))
	}
	
	return nil
}
