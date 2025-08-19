package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  "Manage RepoSentry configuration files, validate settings, and generate examples",
}

var showCmd = &cobra.Command{
	Use:   "show [config-file]",
	Short: "Show current configuration",
	Long:  "Display the current configuration with resolved environment variables",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runShowConfig,
}

var initCmd = &cobra.Command{
	Use:   "init [config-file]",
	Short: "Initialize a new configuration file",
	Long:  "Create a new configuration file with example values",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInitConfig,
}

var (
	showFormat      string
	showSecrets     bool
	initForce       bool
	initTemplate    string
)

func init() {
	// Show command flags
	showCmd.Flags().StringVar(&showFormat, "format", "yaml", "Output format (yaml, json)")
	showCmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Show sensitive values (tokens, passwords)")
	
	// Init command flags
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing configuration file")
	initCmd.Flags().StringVar(&initTemplate, "template", "basic", "Configuration template (basic, advanced, minimal)")
	
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(initCmd)
	
	rootCmd.AddCommand(configCmd)
}

func runShowConfig(cmd *cobra.Command, args []string) error {
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
	
	// Load configuration
	configManager := config.NewManager(appLogger)
	err := configManager.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg := configManager.Get()

	// Mask sensitive data if not showing secrets
	if !showSecrets {
		cfg = maskSensitiveData(cfg)
	}

	// Output in requested format
	switch showFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		
	case "yaml":
		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(yamlBytes))
		
	default:
		return fmt.Errorf("unsupported format: %s (supported: yaml, json)", showFormat)
	}

	return nil
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	// Determine config file
	configFile := "./config.yaml"
	if len(args) > 0 {
		configFile = args[0]
	}

	// Check if file exists
	if _, err := os.Stat(configFile); err == nil && !initForce {
		return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configFile)
	}

	// Create directory if needed
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate configuration based on template
	var cfg *types.Config
	
	switch initTemplate {
	case "minimal":
		cfg = generateMinimalConfig()
	case "advanced":
		cfg = generateAdvancedConfig()
	default: // basic
		cfg = generateBasicConfig()
	}

	// Write configuration to file
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(configFile, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration file created: %s\n", configFile)
	fmt.Printf("📝 Template: %s\n", initTemplate)
	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   1. Edit the configuration file to match your environment\n")
	fmt.Printf("   2. Set required environment variables (tokens, URLs)\n")
	fmt.Printf("   3. Validate configuration: reposentry config validate %s\n", configFile)
	fmt.Printf("   4. Start RepoSentry: reposentry run --config %s\n", configFile)

	return nil
}

func maskSensitiveData(cfg *types.Config) *types.Config {
	// Create a copy to avoid modifying original
	result := *cfg
	
	// Mask repository tokens
	for i := range result.Repositories {
		if result.Repositories[i].Token != "" {
			result.Repositories[i].Token = "***MASKED***"
		}
	}
	
	// TODO: Mask Tekton auth token when field is available
	
	return &result
}

func generateMinimalConfig() *types.Config {
	return &types.Config{
		App: types.AppConfig{
			Name:            "reposentry",
			LogLevel:        "info",
			LogFormat:       "json",
			HealthCheckPort: 8080,
			DataDir:         "./data",
		},
		Polling: types.PollingConfig{
			Interval:      5 * time.Minute,
			Timeout:       30 * time.Second,
			MaxWorkers:    2,
			BatchSize:     5,
			RetryAttempts: 3,
			RetryBackoff:  2 * time.Second,
		},
		Storage: types.StorageConfig{
			Type: "sqlite",
			SQLite: types.SQLiteConfig{
				Path:              "./data/reposentry.db",
				MaxConnections:    10,
				ConnectionTimeout: 30 * time.Second,
			},
		},
		Tekton: types.TektonConfig{
			EventListenerURL: "https://tekton.example.com/webhook",
			Timeout:          30 * time.Second,
		},
		Repositories: []types.Repository{
			{
				Name:            "example-repo",
				URL:             "https://github.com/example/repo",
				Provider:        "github",
				Token:           "${GITHUB_TOKEN}",
				BranchRegex:     "^(main|master)$",
				PollingInterval: 5 * time.Minute,
			},
		},
	}
}

func generateBasicConfig() *types.Config {
	cfg := generateMinimalConfig()
	
	// Add more examples
	cfg.Repositories = append(cfg.Repositories, types.Repository{
		Name:            "gitlab-repo",
		URL:             "https://gitlab.com/example/repo",
					Provider:        "gitlab",
			Token:           "${GITLAB_TOKEN}",
			BranchRegex:     "^(main|develop|release/.*)$",
			PollingInterval: 10 * time.Minute,
	})
	
	// Add some headers and auth
	cfg.Tekton.Headers = map[string]string{
		"X-Custom-Header": "reposentry",
	}
	
	return cfg
}

func generateAdvancedConfig() *types.Config {
	cfg := generateBasicConfig()
	
	// More advanced settings
	cfg.App.LogFile = "./logs/reposentry.log"
	// TODO: Add log file rotation configuration when available
	
	cfg.Polling.MaxWorkers = 5
	cfg.Polling.BatchSize = 10
	
	cfg.Storage.SQLite.MaxConnections = 20
	
	cfg.Tekton.Timeout = 60 * time.Second
	// TODO: Add TLS configuration when available
	
	// Add enterprise GitLab example
	cfg.Repositories = append(cfg.Repositories, types.Repository{
		Name:            "enterprise-gitlab",
		URL:             "https://gitlab-enterprise.company.com/team/project",
		Provider:        "gitlab",
		Token:           "${GITLAB_ENTERPRISE_TOKEN}",
		BranchRegex:     "^(main|staging|production)$",
		PollingInterval: 15 * time.Minute,
	})
	
	return cfg
}
