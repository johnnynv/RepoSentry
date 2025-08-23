#!/bin/bash

# RepoSentry Configuration Merger
# This script merges user repository configuration with system configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default paths
SYSTEM_CONFIG="config.yaml"
USER_CONFIG="user-repos.yaml"
OUTPUT_CONFIG="config-merged.yaml"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -s, --system-config FILE    System configuration file (default: config.yaml)
    -u, --user-config FILE      User repository configuration file (default: user-repos.yaml)
    -o, --output FILE           Output merged configuration file (default: config-merged.yaml)
    -h, --help                  Show this help message

Examples:
    # Use default file names
    $0

    # Specify custom file names
    $0 -s system.yaml -u my-repos.yaml -o final-config.yaml

    # Merge and replace original config
    $0 -o config.yaml

Description:
    This script merges user repository configuration with system configuration,
    creating a complete configuration file that RepoSentry can use.
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--system-config)
            SYSTEM_CONFIG="$2"
            shift 2
            ;;
        -u|--user-config)
            USER_CONFIG="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_CONFIG="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Check if required files exist
if [[ ! -f "$SYSTEM_CONFIG" ]]; then
    print_error "System configuration file not found: $SYSTEM_CONFIG"
    exit 1
fi

if [[ ! -f "$USER_CONFIG" ]]; then
    print_error "User configuration file not found: $USER_CONFIG"
    exit 1
fi

print_info "Starting configuration merge..."
print_info "System config: $SYSTEM_CONFIG"
print_info "User config: $USER_CONFIG"
print_info "Output: $OUTPUT_CONFIG"

# Check if yq is available for YAML manipulation
if command -v yq &> /dev/null; then
    print_info "Using yq for YAML manipulation"
    MERGE_TOOL="yq"
elif command -v python3 &> /dev/null; then
    print_info "Using Python for YAML manipulation"
    MERGE_TOOL="python"
else
    print_error "Neither yq nor Python3 found. Please install one of them."
    print_info "Install yq: https://github.com/mikefarah/yq"
    print_info "Install Python3: sudo apt-get install python3 python3-pip"
    exit 1
fi

# Function to merge using yq
merge_with_yq() {
    print_info "Merging configurations using yq..."
    
    # Create a temporary file for the merge
    TEMP_FILE=$(mktemp)
    
    # Copy system config first
    cp "$SYSTEM_CONFIG" "$TEMP_FILE"
    
    # Merge GitHub repositories
    if yq eval '.github_repos' "$USER_CONFIG" &> /dev/null; then
        print_info "Merging GitHub repositories..."
        yq eval-all 'select(fileIndex == 0) * select(fileIndex == 1)' "$TEMP_FILE" <(
            yq eval '.repositories = .github_repos | del(.github_repos) | .repositories[].provider = "github" | .repositories[].token = "${GITHUB_TOKEN}"' "$USER_CONFIG"
        ) > "$OUTPUT_CONFIG"
        cp "$OUTPUT_CONFIG" "$TEMP_FILE"
    fi
    
    # Merge GitLab repositories
    if yq eval '.gitlab_repos' "$USER_CONFIG" &> /dev/null; then
        print_info "Merging GitLab repositories..."
        yq eval-all 'select(fileIndex == 0) * select(fileIndex == 1)' "$TEMP_FILE" <(
            yq eval '.repositories = .gitlab_repos | del(.gitlab_repos) | .repositories[].provider = "gitlab" | .repositories[].token = "${GITLAB_TOKEN}"' "$USER_CONFIG"
        ) > "$OUTPUT_CONFIG"
        cp "$OUTPUT_CONFIG" "$TEMP_FILE"
    fi
    
    # Apply polling overrides if specified
    if yq eval '.polling_overrides.test_mode' "$USER_CONFIG" &> /dev/null; then
        TEST_MODE=$(yq eval '.polling_overrides.test_mode' "$USER_CONFIG")
        if [[ "$TEST_MODE" == "true" ]]; then
            print_info "Applying test mode polling settings..."
            INTERVAL=$(yq eval '.polling_overrides.interval' "$USER_CONFIG")
            yq eval ".polling.interval = \"$INTERVAL\"" "$TEMP_FILE" > "$OUTPUT_CONFIG"
            cp "$OUTPUT_CONFIG" "$TEMP_FILE"
        fi
    fi
    
    # Clean up temp file
    rm "$TEMP_FILE"
}

# Function to merge using Python
merge_with_python() {
    print_info "Merging configurations using Python..."
    
    python3 -c "
import yaml
import sys

# Load system configuration
with open('$SYSTEM_CONFIG', 'r') as f:
    system_config = yaml.safe_load(f)

# Load user configuration
with open('$USER_CONFIG', 'r') as f:
    user_config = yaml.safe_load(f)

# Initialize repositories list if not exists
if 'repositories' not in system_config:
    system_config['repositories'] = []

# Merge GitHub repositories
if 'github_repos' in user_config:
    for repo in user_config['github_repos']:
        repo['provider'] = 'github'
        repo['token'] = '\${GITHUB_TOKEN}'
        system_config['repositories'].append(repo)

# Merge GitLab repositories
if 'gitlab_repos' in user_config:
    for repo in user_config['gitlab_repos']:
        repo['provider'] = 'gitlab'
        repo['token'] = '\${GITLAB_TOKEN}'
        system_config['repositories'].append(repo)

# Apply polling overrides
if 'polling_overrides' in user_config:
    if user_config['polling_overrides'].get('test_mode', False):
        interval = user_config['polling_overrides'].get('interval', '1m')
        system_config['polling']['interval'] = interval

# Write merged configuration
with open('$OUTPUT_CONFIG', 'w') as f:
    yaml.dump(system_config, f, default_flow_style=False, sort_keys=False)

print(f'Configuration merged successfully to {OUTPUT_CONFIG}')
"
}

# Perform the merge
if [[ "$MERGE_TOOL" == "yq" ]]; then
    merge_with_yq
else
    merge_with_python
fi

# Validate the merged configuration
print_info "Validating merged configuration..."
if ./bin/reposentry config validate "$OUTPUT_CONFIG" &> /dev/null; then
    print_success "Configuration validation passed!"
else
    print_warning "Configuration validation failed. Please check the merged file."
fi

print_success "Configuration merge completed!"
print_info "You can now use: ./bin/reposentry run --config $OUTPUT_CONFIG"
print_info "Or copy $OUTPUT_CONFIG to config.yaml to use as your main configuration"
