package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup RepoSentry configuration",
	Long:  "Setup RepoSentry configuration with interactive wizard",
}

var setupInteractiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Interactive configuration wizard",
	Long:  "Run an interactive wizard to configure RepoSentry repositories and settings",
	RunE:  runSetupInteractive,
}

func init() {
	setupCmd.AddCommand(setupInteractiveCmd)
	rootCmd.AddCommand(setupCmd)
}

// ConfigWizard holds the collected configuration data
type ConfigWizard struct {
	GitHubToken     string
	GitLabToken     string
	GitHubRepos     []RepositoryConfig
	GitLabRepos     []RepositoryConfig
	TektonURL       string
	PollingInterval int // in minutes
}

// RepositoryConfig represents a single repository configuration
type RepositoryConfig struct {
	Name     string
	URL      string
	Branches []string
	Provider string
}

func runSetupInteractive(cmd *cobra.Command, args []string) error {
	fmt.Println("🎯 RepoSentry Configuration Wizard")
	fmt.Println()

	wizard := &ConfigWizard{}
	scanner := bufio.NewScanner(os.Stdin)

	// Step 1: Access Tokens
	if err := collectAccessTokens(wizard, scanner); err != nil {
		return err
	}

	// Step 2: GitHub Repositories
	if wizard.GitHubToken != "" {
		if err := collectGitHubRepositories(wizard, scanner); err != nil {
			return err
		}
	}

	// Step 3: GitLab Repositories
	if wizard.GitLabToken != "" {
		if err := collectGitLabRepositories(wizard, scanner); err != nil {
			return err
		}
	}

	// Step 4: Tekton Configuration
	if err := collectTektonConfig(wizard, scanner); err != nil {
		return err
	}

	// Step 5: Polling Configuration
	if err := collectPollingConfig(wizard, scanner); err != nil {
		return err
	}

	// Step 6: Generate Files
	fmt.Println("🎉 Configuration collection completed! Generating configuration files...")
	fmt.Println()

	if err := generateConfigurationFiles(wizard); err != nil {
		return fmt.Errorf("failed to generate configuration files: %w", err)
	}

	// Step 7: Show completion message
	showCompletionMessage()

	return nil
}

func collectAccessTokens(wizard *ConfigWizard, scanner *bufio.Scanner) error {
	fmt.Println("=== 🔑 Access Token Configuration ===")

	// GitHub Token
	fmt.Println("❓ Please enter your GitHub access token (for API access):")
	fmt.Println("   💡 Get token at: https://github.com/settings/tokens")

	githubToken, err := readSensitiveInput("")
	if err != nil {
		return fmt.Errorf("failed to read GitHub token: %w", err)
	}
	wizard.GitHubToken = githubToken

	// GitLab Token
	fmt.Println()
	fmt.Println("❓ Please enter your GitLab access token (leave empty if not using GitLab):")
	fmt.Println("   💡 Get token at: https://gitlab-master.nvidia.com/-/profile/personal_access_tokens")

	gitlabToken, err := readSensitiveInput("")
	if err != nil {
		return fmt.Errorf("failed to read GitLab token: %w", err)
	}
	wizard.GitLabToken = gitlabToken

	fmt.Println()
	return nil
}

func collectGitHubRepositories(wizard *ConfigWizard, scanner *bufio.Scanner) error {
	fmt.Println("=== 📂 GitHub Repository Configuration ===")

	for {
		repo := RepositoryConfig{Provider: "github"}

		// Repository URL
		fmt.Println("❓ Please enter GitHub repository URL:")
		fmt.Print("   > ")
		if scanner.Scan() {
			repo.URL = strings.TrimSpace(scanner.Text())
		}
		if repo.URL == "" {
			break
		}

		// Extract repository name from URL
		repo.Name = extractRepoName(repo.URL)

		// Branches
		fmt.Println("❓ Please enter branch names to monitor (comma-separated, regex supported):")
		fmt.Println("   💡 Examples: main,develop or main,feature/.* or release/.*")
		fmt.Print("   > ")
		if scanner.Scan() {
			branchStr := strings.TrimSpace(scanner.Text())
			if branchStr != "" {
				branches := strings.Split(branchStr, ",")
				for _, branch := range branches {
					repo.Branches = append(repo.Branches, strings.TrimSpace(branch))
				}
			}
		}

		wizard.GitHubRepos = append(wizard.GitHubRepos, repo)
		fmt.Printf("✅ Added GitHub repository: %s (branches: %s)\n", repo.URL, strings.Join(repo.Branches, ","))
		fmt.Println()

		// Ask for more
		fmt.Println("❓ Would you like to add more GitHub repositories? [y/N]")
		fmt.Print("   > ")
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				break
			}
		}
		fmt.Println()
	}

	fmt.Println()
	return scanner.Err()
}

func collectGitLabRepositories(wizard *ConfigWizard, scanner *bufio.Scanner) error {
	fmt.Println("=== 🦊 GitLab Repository Configuration ===")

	for {
		repo := RepositoryConfig{Provider: "gitlab"}

		// Repository URL
		fmt.Println("❓ Please enter GitLab repository URL:")
		fmt.Println("   💡 Enterprise GitLab format: https://gitlab-master.nvidia.com/username/project")
		fmt.Print("   > ")
		if scanner.Scan() {
			repo.URL = strings.TrimSpace(scanner.Text())
		}
		if repo.URL == "" {
			break
		}

		// Extract repository name from URL
		repo.Name = extractRepoName(repo.URL)

		// Branches
		fmt.Println("❓ Please enter branch names to monitor (comma-separated, regex supported):")
		fmt.Print("   > ")
		if scanner.Scan() {
			branchStr := strings.TrimSpace(scanner.Text())
			if branchStr != "" {
				branches := strings.Split(branchStr, ",")
				for _, branch := range branches {
					repo.Branches = append(repo.Branches, strings.TrimSpace(branch))
				}
			}
		}

		wizard.GitLabRepos = append(wizard.GitLabRepos, repo)
		fmt.Printf("✅ Added GitLab repository: %s (branches: %s)\n", repo.URL, strings.Join(repo.Branches, ","))
		fmt.Println()

		// Ask for more
		fmt.Println("❓ Would you like to add more GitLab repositories? [y/N]")
		fmt.Print("   > ")
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				break
			}
		}
		fmt.Println()
	}

	fmt.Println()
	return scanner.Err()
}

func collectTektonConfig(wizard *ConfigWizard, scanner *bufio.Scanner) error {
	fmt.Println("=== 🎯 Tekton Configuration ===")
	fmt.Println("❓ Please enter your Tekton EventListener URL:")
	fmt.Println("   💡 Example: http://webhook.10.78.14.61.nip.io")
	fmt.Print("   > ")
	if scanner.Scan() {
		wizard.TektonURL = strings.TrimSpace(scanner.Text())
	}

	fmt.Println()
	return scanner.Err()
}

func collectPollingConfig(wizard *ConfigWizard, scanner *bufio.Scanner) error {
	fmt.Println("=== ⏰ Polling Configuration ===")
	fmt.Println("❓ Set repository check interval in minutes:")
	fmt.Println("   💡 Recommended: 1-2 minutes for development, 5-10 minutes for production")
	fmt.Print("   > ")
	if scanner.Scan() {
		intervalStr := strings.TrimSpace(scanner.Text())
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			wizard.PollingInterval = interval
		} else {
			wizard.PollingInterval = 5 // default
		}
	}

	fmt.Println()
	return scanner.Err()
}

func extractRepoName(url string) string {
	// Remove trailing .git if present
	url = strings.TrimSuffix(url, ".git")

	// Extract path after last slash
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s-%s", parts[len(parts)-2], parts[len(parts)-1])
	}

	// Fallback to last part
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown-repo"
}

func generateConfigurationFiles(wizard *ConfigWizard) error {
	// Generate application config
	if err := generateAppConfig(wizard); err != nil {
		return fmt.Errorf("failed to generate app config: %w", err)
	}

	// Generate repository config
	if err := generateRepositoryConfig(wizard); err != nil {
		return fmt.Errorf("failed to generate repository config: %w", err)
	}

	// Generate .env file
	if err := generateEnvFile(wizard); err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}

	// Generate .gitignore
	if err := generateGitignore(); err != nil {
		return fmt.Errorf("failed to generate .gitignore: %w", err)
	}

	// Generate start.sh
	if err := generateStartScript(); err != nil {
		return fmt.Errorf("failed to generate start script: %w", err)
	}

	// Generate stop.sh
	if err := generateStopScript(); err != nil {
		return fmt.Errorf("failed to generate stop script: %w", err)
	}

	// Generate README.md
	if err := generateReadme(wizard); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	return nil
}

func generateAppConfig(wizard *ConfigWizard) error {
	config := fmt.Sprintf(`# RepoSentry Application Configuration
# Generated by: reposentry setup interactive
# Generated at: %s

app:
  name: "reposentry"
  log_level: "info"
  log_format: "json"
  log_file: "./logs/reposentry.log"
  log_file_rotation:
    max_size: 100
    max_age: 30
    max_backups: 5
    compress: true
  health_check_port: 8080
  data_dir: "./data"

polling:
  interval: "%dm"
  timeout: "30s"
  max_workers: 4
  batch_size: 10
  enable_api_fallback: true
  retry_attempts: 3
  retry_backoff: "10s"

tekton:
  event_listener_url: "%s"
  timeout: "30s"
  retry_attempts: 3
  retry_backoff: "5s"

storage:
  type: "sqlite"
  sqlite:
    path: "./data/reposentry.db"
    max_connections: 10
    connection_timeout: "30s"

rate_limit:
  github:
    requests_per_hour: 5000
    burst: 10
  gitlab:
    requests_per_second: 10
    burst: 5

security:
  allowed_env_vars: ["GITHUB_TOKEN", "GITLAB_TOKEN"]
  require_https: true

# Repository configuration file path
repositories_config: "./repositories.yaml"
`, time.Now().Format("2006-01-02 15:04:05"), wizard.PollingInterval, wizard.TektonURL)

	return os.WriteFile("config.yaml", []byte(config), 0644)
}

func generateRepositoryConfig(wizard *ConfigWizard) error {
	config := fmt.Sprintf(`# RepoSentry Repository Configuration
# Generated by: reposentry setup interactive
# Generated at: %s
# This file contains all repository definitions

repositories:
`, time.Now().Format("2006-01-02 15:04:05"))

	// Add GitHub repositories
	for _, repo := range wizard.GitHubRepos {
		branchRegex := convertBranchesToRegex(repo.Branches)
		config += fmt.Sprintf(`  - name: "%s"
    provider: "github"
    url: "%s"
    token: "%s"
    branch_regex: "%s"
    enabled: true
    
`, repo.Name, repo.URL, wizard.GitHubToken, branchRegex)
	}

	// Add GitLab repositories
	for _, repo := range wizard.GitLabRepos {
		branchRegex := convertBranchesToRegex(repo.Branches)
		config += fmt.Sprintf(`  - name: "%s"
    provider: "gitlab"
    url: "%s"
    token: "%s"
    branch_regex: "%s"
    enabled: true
    
`, repo.Name, repo.URL, wizard.GitLabToken, branchRegex)
	}

	config += `
# Global repository settings
global_settings:
  default_polling_enabled: true
  default_webhook_enabled: true
  default_branch_protection: false
`

	return os.WriteFile("repositories.yaml", []byte(config), 0644)
}

func convertBranchesToRegex(branches []string) string {
	if len(branches) == 0 {
		return ".*" // Match all branches
	}

	// Convert branch patterns to regex
	var patterns []string
	for _, branch := range branches {
		// If it's already a regex pattern (contains . * [ ] etc), use as-is
		if strings.ContainsAny(branch, ".*[](){}^$+?|\\") {
			patterns = append(patterns, branch)
		} else {
			// Escape special regex characters and make exact match
			escaped := strings.ReplaceAll(branch, ".", "\\.")
			escaped = strings.ReplaceAll(escaped, "*", "\\*")
			escaped = strings.ReplaceAll(escaped, "+", "\\+")
			escaped = strings.ReplaceAll(escaped, "?", "\\?")
			patterns = append(patterns, escaped) // Remove extra ^$ since we'll add them below
		}
	}

	if len(patterns) == 1 {
		// Single pattern - check if it already has anchors
		if strings.HasPrefix(patterns[0], "^") && strings.HasSuffix(patterns[0], "$") {
			return patterns[0]
		}
		return "^" + patterns[0] + "$"
	}

	// Multiple patterns - combine with OR
	return "^(" + strings.Join(patterns, "|") + ")$"
}

func formatBranchesAsYaml(branches []string) string {
	if len(branches) == 0 {
		return "[]"
	}
	if len(branches) == 1 {
		return fmt.Sprintf(`["%s"]`, branches[0])
	}

	result := "["
	for i, branch := range branches {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, branch)
	}
	result += "]"
	return result
}

func generateEnvFile(wizard *ConfigWizard) error {
	env := fmt.Sprintf(`# RepoSentry Environment Variables Configuration
# Generated by: reposentry setup interactive
# Generated at: %s
# 
# ⚠️  SECURITY WARNING: This file contains sensitive information
# 🔒 Keep this file secure and never commit it to version control
# 💡 Add .env to your .gitignore file

# GitHub access token%s
GITHUB_TOKEN=%s

# GitLab access token%s
GITLAB_TOKEN=%s

# Optional configuration (uncomment if needed)
# RS_LOG_LEVEL=info
# RS_CONFIG_PATH=./config.yaml
`,
		time.Now().Format("2006-01-02 15:04:05"),
		func() string {
			if wizard.GitHubToken != "" {
				return ""
			}
			return " (not provided)"
		}(),
		wizard.GitHubToken,
		func() string {
			if wizard.GitLabToken != "" {
				return ""
			}
			return " (not provided)"
		}(),
		wizard.GitLabToken,
	)

	return os.WriteFile(".env", []byte(env), 0600) // More restrictive permissions for secrets
}

func generateStartScript() error {
	script := `#!/bin/bash
# RepoSentry Startup Script
# Generated by: reposentry setup interactive

echo "🚀 Starting RepoSentry..."

# Check if binary exists
if [[ ! -f "./reposentry" ]]; then
    echo "❌ Error: reposentry binary not found in current directory"
    echo "💡 Please ensure reposentry binary is in the same directory as this script"
    exit 1
fi

# Check environment variables
check_env() {
    if [[ -z "$GITHUB_TOKEN" ]]; then
        echo "💡 Loading environment variables from .env file..."
        if [[ -f ".env" ]]; then
            source .env
        else
            echo "❌ .env file not found and GITHUB_TOKEN not set"
            echo "💡 Please create .env file or export GITHUB_TOKEN"
            exit 1
        fi
    fi
}

# Setup directories
setup_dirs() {
    mkdir -p logs data
    echo "✅ Directories ready"
}

# Validate configurations
validate_config() {
    if [[ ! -f "config.yaml" ]]; then
        echo "❌ config.yaml not found"
        exit 1
    fi
    
    if [[ ! -f "repositories.yaml" ]]; then
        echo "❌ repositories.yaml not found"
        exit 1
    fi
    
    echo "✅ Configuration files validated"
}

# Validate configuration using RepoSentry config command
validate_reposentry_config() {
    echo "🔍 Validating RepoSentry configuration..."
    ./reposentry config validate --config config.yaml
    if [ $? -eq 0 ]; then
        echo "✅ Configuration validation passed"
    else
        echo "❌ Configuration validation failed"
        echo "💡 Please fix configuration issues before starting"
        exit 1
    fi
}

# Main function
main() {
    check_env
    setup_dirs
    validate_config
    validate_reposentry_config
    
    echo "🎯 Starting RepoSentry service..."
    nohup ./reposentry run \
        --config config.yaml \
        --log-level debug \
        --log-file ./logs/reposentry.log \
        > ./logs/startup.log 2>&1 &
    
    # Get the PID
    REPOSENTRY_PID=$!
    echo $REPOSENTRY_PID > ./reposentry.pid
    
    echo "✅ RepoSentry started successfully!"
    echo "📋 Process ID: $REPOSENTRY_PID"
    echo "📁 PID file: ./reposentry.pid"
    echo "📄 Log file: ./logs/reposentry.log"
    echo ""
    echo "📊 Monitor logs with:"
    echo "   tail -f ./logs/reposentry.log"
    echo ""
    echo "🛑 Stop service with:"
    echo "   ./stop.sh"
}

main "$@"
`

	if err := os.WriteFile("start.sh", []byte(script), 0755); err != nil {
		return err
	}

	return nil
}

func generateStopScript() error {
	script := `#!/bin/bash
# RepoSentry Stop Script
# Generated by: reposentry setup interactive

echo "🛑 Stopping RepoSentry..."

# First try to stop using PID file
if [[ -f "./reposentry.pid" ]]; then
    PID=$(cat ./reposentry.pid)
    if [[ -n "$PID" ]]; then
        echo "📍 Found PID file with process ID: $PID"
        if kill -0 "$PID" 2>/dev/null; then
            echo "🔄 Stopping process $PID gracefully..."
            kill "$PID"
            sleep 3
            
            # Check if process is stopped
            if ! kill -0 "$PID" 2>/dev/null; then
                echo "✅ RepoSentry stopped successfully"
                rm -f ./reposentry.pid
                exit 0
            else
                echo "⚠️  Process still running, using force kill..."
                kill -9 "$PID" 2>/dev/null
                sleep 1
                rm -f ./reposentry.pid
            fi
        else
            echo "💡 Process $PID not running, cleaning up PID file"
            rm -f ./reposentry.pid
        fi
    fi
fi

# Fallback: Find and kill RepoSentry processes by name
if pgrep -f "reposentry run" > /dev/null; then
    echo "📍 Found running RepoSentry processes (fallback method)"
    pkill -f "reposentry run"
    sleep 2
    
    # Check if process is still running
    if pgrep -f "reposentry run" > /dev/null; then
        echo "⚠️  Process still running, using force kill..."
        pkill -9 -f "reposentry run"
        sleep 1
    fi
    
    if ! pgrep -f "reposentry run" > /dev/null; then
        echo "✅ RepoSentry stopped successfully"
        rm -f ./reposentry.pid
    else
        echo "❌ Failed to stop RepoSentry"
        exit 1
    fi
else
    echo "💡 No running RepoSentry processes found"
fi
`

	return os.WriteFile("stop.sh", []byte(script), 0755)
}

func generateReadme(wizard *ConfigWizard) error {
	totalRepos := len(wizard.GitHubRepos) + len(wizard.GitLabRepos)

	// Use a more manageable template approach
	readmeTemplate := `# RepoSentry Configuration

This directory contains your RepoSentry configuration, generated on %s.

> **📂 Directory**: repository-monitor  
> **🎯 Purpose**: Complete repository monitoring setup

## 🚀 Quick Start

1. **Start the service:**
   ` + "```bash" + `
   ./start.sh
   ` + "```" + `

2. **Monitor logs:**
   ` + "```bash" + `
   tail -f logs/reposentry.log
   ` + "```" + `

3. **Stop the service:**
   ` + "```bash" + `
   ./stop.sh
   ` + "```" + `

## 📁 Files Overview

- **reposentry** - The main binary
- **config.yaml** - Application configuration (polling, logging, Tekton)
- **repositories.yaml** - Repository definitions and monitoring settings
- **start.sh** / **stop.sh** - Control scripts
- **.env** - Environment variables (access tokens)
- **README.md** - This usage guide

## 📝 Configuration Modifications

### Repository Management
Edit **repositories.yaml** to add/remove repositories.

### Application Settings
Edit **config.yaml** for application settings like polling interval and Tekton URL.

### Access Tokens
Edit **.env** file to update GitHub and GitLab tokens.

## 🎯 Current Configuration Summary

**Monitoring Setup:**
- GitHub repositories: %d
- GitLab repositories: %d
- Total repositories: %d
- Polling interval: %d minutes
- Tekton EventListener: %s

**Files:**
- Application config: config.yaml
- Repository config: repositories.yaml
- Environment variables: .env
- Logs directory: ./logs/
- Database: ./data/reposentry.db

## 📚 Additional Resources

- **Documentation:** https://github.com/johnnynv/RepoSentry/wiki
- **Issue Tracker:** https://github.com/johnnynv/RepoSentry/issues

For help and support, please visit the project documentation.
`

	readme := fmt.Sprintf(readmeTemplate,
		time.Now().Format("2006-01-02 15:04:05"),
		len(wizard.GitHubRepos),
		len(wizard.GitLabRepos),
		totalRepos,
		wizard.PollingInterval,
		wizard.TektonURL,
	)

	return os.WriteFile("README.md", []byte(readme), 0644)
}

func showCompletionMessage() {
	fmt.Println("✅ Generation completed! Created the following files:")
	fmt.Println("📄 config.yaml          # Application configuration")
	fmt.Println("📂 repositories.yaml    # Repository definitions")
	fmt.Println("🚀 start.sh             # Startup script")
	fmt.Println("🛑 stop.sh              # Stop script")
	fmt.Println("🔧 .env                 # Environment variables (tokens included)")
	fmt.Println("🔒 .gitignore           # Git ignore file (protects sensitive data)")
	fmt.Println("📖 README.md            # Usage instructions")
	fmt.Println()
	fmt.Println("🎯 Next Steps:")
	fmt.Println("1️⃣ Start service: ./start.sh")
	fmt.Println("2️⃣ Monitor logs: tail -f logs/reposentry.log")
	fmt.Println("3️⃣ Stop service: ./stop.sh")
	fmt.Println()
	fmt.Println("📝 For configuration modifications, see README.md")
	fmt.Println()
	fmt.Println("💡 For help: https://github.com/johnnynv/RepoSentry/wiki")
}

// readSensitiveInput reads sensitive input (like tokens) without echoing to screen
func readSensitiveInput(prompt string) (string, error) {
	fmt.Print(prompt)
	fmt.Println("🔒 (input hidden for security)")
	fmt.Print("   > ")

	// Read password without echoing
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}

	fmt.Println() // Add newline after hidden input
	return strings.TrimSpace(string(bytePassword)), nil
}

func generateGitignore() error {
	gitignore := `# RepoSentry generated files and sensitive data
# Generated by: reposentry setup interactive

# Environment variables (contains sensitive tokens)
.env

# Runtime files
reposentry.pid
logs/
data/

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# Backup files
*.bak
*.backup
`

	return os.WriteFile(".gitignore", []byte(gitignore), 0644)
}
