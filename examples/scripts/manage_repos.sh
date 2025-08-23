#!/bin/bash
# RepoSentry Repository Management Script
# This script provides interactive management of repository configurations

set -e

CONFIG_FILE="config.yaml"
TEMP_CONFIG="config.yaml.tmp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to prompt for input with default
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    
    if [ -n "$default" ]; then
        read -p "$prompt [$default]: " input
        export $var_name="${input:-$default}"
    else
        read -p "$prompt: " input
        export $var_name="$input"
    fi
}

# Function to prompt for sensitive input
prompt_sensitive() {
    local prompt="$1"
    local var_name="$2"
    
    echo -n "$prompt: "
    read -s input
    echo
    export $var_name="$input"
}

# Function to prompt for yes/no with default
prompt_yes_no() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    
    if [ "$default" = "y" ]; then
        read -p "$prompt (Y/n): " input
        if [[ "$input" =~ ^[Nn]$ ]]; then
            export $var_name="n"
        else
            export $var_name="y"
        fi
    else
        read -p "$prompt (y/N): " input
        if [[ "$input" =~ ^[Yy]$ ]]; then
            export $var_name="y"
        else
            export $var_name="n"
        fi
    fi
}

# Function to check if config file exists
check_config() {
    if [ ! -f "$CONFIG_FILE" ]; then
        print_color $RED "❌ Configuration file $CONFIG_FILE not found!"
        print_color $YELLOW "Please run setup_env.sh first or create a config.yaml file."
        exit 1
    fi
}

# Function to display main menu
show_menu() {
    echo
    print_color $BLUE "🔧 RepoSentry Repository Management"
    print_color $BLUE "=================================="
    echo
    echo "1. 📋 List all repositories"
    echo "2. ➕ Add new repository"
    echo "3. ✏️  Edit repository"
    echo "4. 🗑️  Delete repository"
    echo "5. 🔍 View repository details"
    echo "6. ✅ Validate configuration"
    echo "7. 🚀 Test webhook connection"
    echo "8. 📁 Show configuration file"
    echo "0. 🚪 Exit"
    echo
}

# Function to list all repositories
list_repositories() {
    print_color $BLUE "📋 Current Repositories:"
    echo "=========================="
    
    # Extract repository names and basic info using yq or grep
    if command -v yq &> /dev/null; then
        yq eval '.repositories[] | "• \(.name) (\(.provider)) - \(.url)"' "$CONFIG_FILE" 2>/dev/null || echo "No repositories found"
    else
        # Fallback to grep if yq is not available
        grep -A 5 "name:" "$CONFIG_FILE" | grep -E "(name:|provider:|url:)" | sed 's/^[[:space:]]*//' | paste -d ' ' - - - | sed 's/name: //;s/provider: //;s/url: //;s/^/• /'
    fi
}

# Function to add new repository
add_repository() {
    print_color $BLUE "➕ Add New Repository"
    echo "====================="
    
    # Get repository name
    read -p "Repository name (e.g., my-project): " repo_name
    if [ -z "$repo_name" ]; then
        print_color $RED "❌ Repository name cannot be empty!"
        return
    fi
    
    # Get repository URL
    read -p "Repository URL: " repo_url
    if [ -z "$repo_url" ]; then
        print_color $RED "❌ Repository URL cannot be empty!"
        return
    fi
    
    # Determine provider from URL
    if [[ "$repo_url" == *"github.com"* ]]; then
        if [[ "$repo_url" == *"api.github.com"* ]] || [[ "$repo_url" == *"github.enterprise"* ]]; then
            provider="github"
            prompt_with_default "GitHub Enterprise API base URL" "https://api.github.com" "api_base_url"
        else
            provider="github"
            api_base_url="https://api.github.com"
        fi
    elif [[ "$repo_url" == *"gitlab.com"* ]]; then
        provider="gitlab"
        api_base_url="https://gitlab.com/api/v4"
    elif [[ "$repo_url" == *"nvidia.com"* ]] || [[ "$repo_url" == *"gitlab.enterprise"* ]]; then
        provider="gitlab"
        prompt_with_default "GitLab Enterprise API base URL" "https://gitlab-master.nvidia.com/api/v4" "api_base_url"
    else
        provider="gitlab"
        prompt_with_default "GitLab API base URL" "https://gitlab.com/api/v4" "api_base_url"
    fi
    
    # Get branches
    prompt_with_default "Branch regex pattern" "^(main|master|develop|feature/.*)$" "branch_regex"
    
    # Get polling interval
    prompt_with_default "Polling interval (e.g., 3m)" "3m" "polling_interval"
    
    # Enable/disable
    prompt_yes_no "Enable this repository?" "y" "enabled"
    
    # Create repository entry
    cat >> "$CONFIG_FILE" << EOF

  # $repo_name
  - name: "$repo_name"
    url: "$repo_url"
    provider: "$provider"
    token: "\${${provider^^}_TOKEN}"
    branch_regex: "$branch_regex"
    enabled: $([ "$enabled" = "y" ] && echo "true" || echo "false")
    polling_interval: "$polling_interval"
EOF
    
    if [ "$provider" = "gitlab" ] && [ "$api_base_url" != "https://gitlab.com/api/v4" ]; then
        echo "    api_base_url: \"$api_base_url\"" >> "$CONFIG_FILE"
    fi
    
    print_color $GREEN "✅ Repository '$repo_name' added successfully!"
}

# Function to delete repository
delete_repository() {
    print_color $BLUE "🗑️  Delete Repository"
    echo "===================="
    
    list_repositories
    echo
    read -p "Enter repository name to delete: " repo_name
    
    if [ -z "$repo_name" ]; then
        print_color $RED "❌ Repository name cannot be empty!"
        return
    fi
    
    # Create temporary file without the repository
    if command -v yq &> /dev/null; then
        # Use yq to remove repository
        yq eval "del(.repositories[] | select(.name == \"$repo_name\"))" "$CONFIG_FILE" > "$TEMP_CONFIG"
        if [ $? -eq 0 ]; then
            mv "$TEMP_CONFIG" "$CONFIG_FILE"
            print_color $GREEN "✅ Repository '$repo_name' deleted successfully!"
        else
            print_color $RED "❌ Failed to delete repository!"
            rm -f "$TEMP_CONFIG"
        fi
    else
        # Fallback to sed (less reliable)
        print_color $YELLOW "⚠️  yq not found, using sed fallback (less reliable)"
        prompt_yes_no "Continue with sed fallback?" "n" "continue_sed"
        if [ "$continue_sed" = "y" ]; then
            # This is a basic sed approach - may need manual verification
            sed -i "/name: \"$repo_name\"/,/^  -/d" "$CONFIG_FILE"
            print_color $GREEN "✅ Repository '$repo_name' deleted (please verify config file)"
        fi
    fi
}

# Function to validate configuration
validate_config() {
    print_color $BLUE "✅ Validating Configuration"
    echo "========================="
    
    if [ -f "./bin/reposentry" ]; then
        ./bin/reposentry config validate "$CONFIG_FILE"
    else
        print_color $YELLOW "⚠️  RepoSentry binary not found. Please run 'make build' first."
    fi
}

# Function to test webhook connection
test_webhook() {
    print_color $BLUE "🚀 Testing Webhook Connection"
    echo "============================="
    
    if [ -f "./bin/reposentry" ]; then
        ./bin/reposentry test-webhook --config "$CONFIG_FILE"
    else
        print_color $YELLOW "⚠️  RepoSentry binary not found. Please run 'make build' first."
    fi
}

# Function to show configuration file
show_config() {
    print_color $BLUE "📁 Configuration File: $CONFIG_FILE"
    echo "=========================================="
    cat "$CONFIG_FILE"
}

# Main script logic
main() {
    check_config
    
    while true; do
        show_menu
        read -p "Select an option (0-8): " choice
        
        case $choice in
            1)
                list_repositories
                ;;
            2)
                add_repository
                ;;
            3)
                print_color $YELLOW "⚠️  Edit functionality coming soon. Please edit $CONFIG_FILE manually for now."
                ;;
            4)
                delete_repository
                ;;
            5)
                print_color $YELLOW "⚠️  View details functionality coming soon. Please check $CONFIG_FILE manually for now."
                ;;
            6)
                validate_config
                ;;
            7)
                test_webhook
                ;;
            8)
                show_config
                ;;
            0)
                print_color $GREEN "👋 Goodbye!"
                exit 0
                ;;
            *)
                print_color $RED "❌ Invalid option. Please select 0-8."
                ;;
        esac
        
        echo
        read -p "Press Enter to continue..."
    done
}

# Run main function
main

