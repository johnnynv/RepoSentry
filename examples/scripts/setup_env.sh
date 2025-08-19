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

echo "📝 Setting up Git provider tokens..."
echo

# GitHub token
if [ -z "$GITHUB_TOKEN" ]; then
    echo "🐙 GitHub Configuration"
    prompt_sensitive "Enter your GitHub personal access token" "GITHUB_TOKEN"
else
    echo "✅ GitHub token already set"
fi

echo

# GitLab token
if [ -z "$GITLAB_TOKEN" ]; then
    echo "🦊 GitLab Configuration"
    prompt_sensitive "Enter your GitLab personal access token" "GITLAB_TOKEN"
else
    echo "✅ GitLab token already set"
fi

echo

# Enterprise GitLab (optional)
read -p "🏢 Do you need Enterprise GitLab access? (y/N): " need_enterprise
if [[ "$need_enterprise" =~ ^[Yy]$ ]]; then
    if [ -z "$GITLAB_ENTERPRISE_TOKEN" ]; then
        prompt_sensitive "Enter your Enterprise GitLab token" "GITLAB_ENTERPRISE_TOKEN"
    else
        echo "✅ Enterprise GitLab token already set"
    fi
fi

echo

# Tekton configuration
echo "🚀 Tekton Configuration"
if [ -z "$TEKTON_WEBHOOK_URL" ]; then
    prompt_with_default "Tekton EventListener webhook URL" "http://localhost:8081/webhook" "TEKTON_WEBHOOK_URL"
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

# Git Provider Tokens
GITHUB_TOKEN=$GITHUB_TOKEN
GITLAB_TOKEN=$GITLAB_TOKEN
EOF

if [ -n "$GITLAB_ENTERPRISE_TOKEN" ]; then
    echo "GITLAB_ENTERPRISE_TOKEN=$GITLAB_ENTERPRISE_TOKEN" >> "$ENV_FILE"
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
echo "📋 Next steps:"
echo "   1. Review and edit $ENV_FILE if needed"
echo "   2. Source the environment file: source $ENV_FILE"
echo "   3. Copy example configuration: cp examples/configs/basic.yaml config.yaml"
echo "   4. Edit config.yaml with your repository settings"
echo "   5. Validate configuration: ./reposentry config validate config.yaml"
echo "   6. Start RepoSentry: ./reposentry run --config config.yaml"
echo
echo "🔐 Security Note: Keep your $ENV_FILE secure and add it to .gitignore"
