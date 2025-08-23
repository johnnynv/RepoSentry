#!/bin/bash
# RepoSentry Environment Setup Script
# This script helps set up environment variables for RepoSentry

set -e

echo "🔧 RepoSentry Environment Setup"
echo "================================"

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

echo "📝 Setting up Git provider configurations..."
echo

# GitHub Configuration
echo "🐙 GitHub Configuration"
prompt_yes_no "Do you need GitHub access?" "n" "NEED_GITHUB"

if [ "$NEED_GITHUB" = "y" ]; then
    prompt_yes_no "Is this GitHub Enterprise (self-hosted)?" "n" "GITHUB_ENTERPRISE"
    
    if [ "$GITHUB_ENTERPRISE" = "y" ]; then
        prompt_with_default "GitHub Enterprise API base URL" "https://github.enterprise.com/api/v3" "GITHUB_API_URL"
        prompt_sensitive "Enter your GitHub Enterprise personal access token" "GITHUB_TOKEN"
    else
        export GITHUB_API_URL="https://api.github.com"
        prompt_sensitive "Enter your GitHub personal access token" "GITHUB_TOKEN"
    fi
fi

echo

# GitLab Configuration
echo "🦊 GitLab Configuration"
prompt_yes_no "Do you need GitLab access?" "y" "NEED_GITLAB"

if [ "$NEED_GITLAB" = "y" ]; then
    prompt_yes_no "Is this GitLab Enterprise (self-hosted)?" "y" "GITLAB_ENTERPRISE"
    
    if [ "$GITLAB_ENTERPRISE" = "y" ]; then
        prompt_with_default "GitLab Enterprise API base URL" "https://gitlab-master.nvidia.com/api/v4" "GITLAB_API_URL"
        prompt_sensitive "Enter your GitLab Enterprise personal access token" "GITLAB_TOKEN"
    else
        export GITLAB_API_URL="https://gitlab.com/api/v4"
        prompt_sensitive "Enter your GitLab personal access token" "GITLAB_TOKEN"
    fi
fi

echo

# Tekton configuration
echo "🚀 Tekton Configuration"
if [ -z "$TEKTON_WEBHOOK_URL" ]; then
    prompt_with_default "Tekton EventListener webhook URL" "http://webhook.10.78.14.61.nip.io" "TEKTON_WEBHOOK_URL"
else
    echo "✅ Tekton webhook URL already set: $TEKTON_WEBHOOK_URL"
fi

echo

# Generate environment file
ENV_FILE=".env"
echo "💾 Creating environment file: $ENV_FILE"

cat > "$ENV_FILE" << EOF
# RepoSentry Environment Variables
# Generated on $(date)

EOF

# Add GitHub configuration
if [ "$NEED_GITHUB" = "y" ]; then
    cat >> "$ENV_FILE" << EOF
# GitHub Configuration
GITHUB_TOKEN=$GITHUB_TOKEN
GITHUB_API_URL=$GITHUB_API_URL
EOF
fi

# Add GitLab configuration
if [ "$NEED_GITLAB" = "y" ]; then
    cat >> "$ENV_FILE" << EOF

# GitLab Configuration
GITLAB_TOKEN=$GITLAB_TOKEN
GITLAB_API_URL=$GITLAB_API_URL
EOF
fi

cat >> "$ENV_FILE" << EOF

# Tekton Configuration
TEKTON_WEBHOOK_URL=$TEKTON_WEBHOOK_URL

# Optional: RepoSentry Configuration
# RS_LOG_LEVEL=info
# RS_LOG_FORMAT=json
# RS_DATA_DIR=./data
EOF

echo "✅ Environment file created successfully!"
echo
echo "📋 Configuration Summary:"
if [ "$NEED_GITHUB" = "y" ]; then
    echo "   🐙 GitHub: $([ "$GITHUB_ENTERPRISE" = "y" ] && echo "Enterprise" || echo "Standard") - $GITHUB_API_URL"
fi
if [ "$NEED_GITLAB" = "y" ]; then
    echo "   🦊 GitLab: $([ "$GITLAB_ENTERPRISE" = "y" ] && echo "Enterprise" || echo "Standard") - $GITLAB_API_URL"
fi
echo "   🚀 Tekton: $TEKTON_WEBHOOK_URL"
echo
echo "📋 Next steps:"
echo "   1. Review and edit $ENV_FILE if needed"
echo "   2. Source the environment file: source $ENV_FILE"
echo "   3. Copy example configuration: cp examples/configs/basic.yaml config.yaml"
echo "   4. Edit config.yaml with your repository settings"
echo "   5. Validate configuration: ./bin/reposentry config validate config.yaml"
echo "   6. Start RepoSentry: ./bin/reposentry run --config config.yaml"
echo
echo "🔐 Security Note: Keep your $ENV_FILE secure and add it to .gitignore"
