# RepoSentry Examples

This directory contains example configurations and usage scenarios for RepoSentry.

## 📁 Directory Structure

```
examples/
├── configs/           # Configuration examples
│   ├── basic.yaml        # Basic configuration
│   ├── minimal.yaml      # Minimal configuration
│   ├── development.yaml  # Development environment
│   ├── production.yaml   # Production environment
│   ├── user-repos.yaml.template  # User repository template
│   └── my-repos.yaml     # Example user configuration

├── kubernetes/        # Kubernetes examples
└── scripts/          # Utility scripts
```

## 🔧 Configuration Examples

### Basic Configuration (`configs/basic.yaml`)
A standard configuration suitable for most use cases with common Git providers.

### User Repository Configuration (`configs/user-repos.yaml`)
A simplified configuration file containing only repository information. Users can edit this file without being overwhelmed by system configuration options.

**Benefits:**
- **Separation of Concerns**: Repository configuration is separate from system configuration
- **User-Friendly**: Only shows what users need to configure
- **Easy Maintenance**: Simple structure for adding/removing repositories
- **Template-Based**: Start with a template and customize as needed

**Usage:**
1. Copy `user-repos.yaml.template` to `user-repos.yaml`
2. Edit with your repository information
3. Use `merge-config.sh` to combine with system configuration
4. Run RepoSentry with the merged configuration

**Example Structure:**
```yaml
# My GitHub repositories
github_repos:
  - name: "my-project"
    url: "https://github.com/username/my-project.git"
    branch_regex: "^(main|dev)$"
    enabled: true

# My GitLab repositories
gitlab_repos:
  - name: "internal-project"
    url: "https://gitlab.company.com/group/project.git"
    branch_regex: "^(main|develop)$"
    enabled: true
    api_base_url: "https://gitlab.company.com/api/v4"
```

### Minimal Configuration (`configs/minimal.yaml`)
The simplest possible configuration to get RepoSentry running.

### Development Configuration (`configs/development.yaml`)
Optimized for development with:
- Debug logging
- Frequent polling
- Local Tekton setup
- Relaxed security

### Production Configuration (`configs/production.yaml`)
Production-ready configuration with:
- Structured logging with rotation
- Optimized polling intervals
- Multiple repositories
- Security hardening

## 🚀 Quick Start

### Option 1: Interactive Setup (Recommended)
Use the interactive setup script for a guided experience:

```bash
./examples/scripts/start.sh
```

This will guide you through:
1. Building RepoSentry
2. Setting up environment variables
3. Managing repositories
4. Validating configuration
5. Merging user configuration (if using user-repos.yaml)
6. Starting the service

### Option 2: Manual Configuration

#### 1. Choose a Configuration

```bash
# Copy desired configuration
cp examples/configs/basic.yaml config.yaml

# Edit with your settings
vim config.yaml
```

### 2. Set Environment Variables

```bash
# GitHub token
export GITHUB_TOKEN="your_github_token"

# GitLab token
export GITLAB_TOKEN="your_gitlab_token"

# Enterprise GitLab (if needed)
export GITLAB_ENTERPRISE_TOKEN="your_enterprise_token"
```

#### 2a. Using User Repository Configuration (Recommended)

If you prefer to maintain a simple repository list:

```bash
# Copy the template
cp examples/configs/user-repos.yaml.template user-repos.yaml

# Edit with your repositories
vim user-repos.yaml

# Merge with system configuration
./examples/scripts/merge-config.sh

# Use the merged configuration
cp config-merged.yaml config.yaml
```

#### 2b. Using Full Configuration

Or edit the full configuration directly:

```bash
# Copy desired configuration
cp examples/configs/basic.yaml config.yaml

# Edit with your settings
vim config.yaml
```

### 3. Run RepoSentry

```bash
# Validate configuration
./reposentry config validate config.yaml

# Start service
./reposentry run --config config.yaml
```

## 🔍 Configuration Fields

### Application Settings
- `app.name`: Application name
- `app.log_level`: Log level (debug, info, warn, error)
- `app.log_format`: Log format (json, text)
- `app.health_check_port`: Health check port
- `app.data_dir`: Data directory path

### Polling Settings
- `polling.interval`: How often to check repositories
- `polling.timeout`: API request timeout
- `polling.max_workers`: Maximum concurrent workers
- `polling.batch_size`: Repositories processed per batch

### Storage Settings
- `storage.type`: Storage backend (currently only sqlite)
- `storage.sqlite.path`: Database file path
- `storage.sqlite.max_connections`: Connection pool size

### Tekton Integration
- `tekton.event_listener_url`: Tekton EventListener webhook URL
- `tekton.timeout`: Webhook request timeout
- `tekton.headers`: Custom headers for webhooks

### Repository Configuration
- `repositories[].name`: Unique repository name
- `repositories[].url`: Repository URL (HTTPS only)
- `repositories[].provider`: Git provider (github, gitlab)
- `repositories[].token`: API token (use environment variables)
- `repositories[].branch_regex`: Branch filter regex
- `repositories[].polling_interval`: Per-repository polling interval

## 🔐 Security Best Practices

### 1. Environment Variables
Always use environment variables for sensitive data:

```yaml
repositories:
  - name: my-repo
    token: "${GITHUB_TOKEN}"  # ✅ Good
    # token: "ghp_xxxxx"      # ❌ Bad - hardcoded
```

### 2. Branch Filtering
Use specific regex patterns:

```yaml
repositories:
  - name: prod-repo
    branch_regex: "^(main|release/.*)$"  # ✅ Specific
    # branch_regex: ".*"                 # ❌ Too broad
```

### 3. Log Security
Avoid logging sensitive information:

```yaml
app:
  log_level: info  # ✅ Production safe
  # log_level: debug  # ❌ May log sensitive data
```

## 📊 Monitoring Examples

### Health Check Script
```bash
#!/bin/bash
# examples/scripts/health_check.sh
response=$(curl -s http://localhost:8080/health)
healthy=$(echo $response | jq -r '.data.healthy')

if [ "$healthy" = "true" ]; then
  echo "✅ RepoSentry is healthy"
  exit 0
else
  echo "❌ RepoSentry is unhealthy"
  exit 1
fi
```

### Prometheus Configuration
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'reposentry'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## 🐳 Docker Examples

See `../deployments/docker/` for official Docker and Docker Compose configurations.

## ☸️ Kubernetes Examples

See `examples/kubernetes/` for Kubernetes deployment examples and Helm values.

## 🛠️ Troubleshooting

### Common Issues

1. **Configuration Validation Failed**
   ```bash
   # Check configuration syntax
   ./reposentry config validate config.yaml --check-env
   ```

2. **API Rate Limiting**
   ```yaml
   # Increase polling interval
   polling:
     interval: 10m  # Reduce API calls
   ```

3. **Permission Denied**
   ```bash
   # Check file permissions
   chmod 644 config.yaml
   chmod 750 ./data
   ```

## 📚 Additional Resources

- [API Documentation](../docs/API_EXAMPLES.md)
- [Deployment Guide](../deployments/README.md)
- [Configuration Reference](../docs/CONFIGURATION.md)
