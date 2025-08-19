# RepoSentry User Manual

## 📖 Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration Details](#configuration-details)
4. [CLI Commands](#cli-commands)
5. [API Interface](#api-interface)
6. [Configuration Hot Reload](#configuration-hot-reload)
7. [Monitoring and Logging](#monitoring-and-logging)
8. [Security Best Practices](#security-best-practices)
9. [Troubleshooting](#troubleshooting)
10. [Advanced Usage](#advanced-usage)

## 🎯 Overview

RepoSentry is a Git repository monitoring tool designed specifically for the Tekton ecosystem, providing:

- **Intelligent Polling**: API-first with git command fallback
- **Multi-platform Support**: GitHub, GitLab (including Enterprise)
- **Flexible Configuration**: YAML configuration + environment variables
- **Event-driven**: Real-time Tekton pipeline triggering
- **Cloud-native**: Supports Docker, Kubernetes deployment

## 🔧 Installation

### System Requirements

- **Operating System**: Linux, macOS, Windows
- **Memory**: Minimum 128MB, recommended 512MB
- **Storage**: 100MB (including database)
- **Network**: Access to Git provider APIs and Tekton EventListener required

### Installation Methods

#### 1. Binary Installation (Recommended)

```bash
# Download latest version
curl -L -o reposentry https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-linux-amd64

# Set executable permissions
chmod +x reposentry

# Move to system path
sudo mv reposentry /usr/local/bin/
```

#### 2. Build from Source

```bash
# Clone repository
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry

# Build
make build

# Install
sudo cp bin/reposentry /usr/local/bin/
```

#### 3. Docker Installation

```bash
# Pull image
docker pull reposentry:latest

# Or build from source
docker build -t reposentry:latest .
```

## ⚙️ Configuration Details

### Configuration File Structure

RepoSentry uses YAML format configuration file, mainly containing the following sections:

```yaml
app:           # Application configuration
polling:       # Polling configuration
storage:       # Storage configuration
tekton:        # Tekton integration configuration
repositories:  # Repository list configuration
```

### Application Configuration (app)

```yaml
app:
  name: "reposentry"                    # Application name
  log_level: "info"                     # Log level: debug, info, warn, error
  log_format: "json"                    # Log format: json, text
  log_file: "/var/log/reposentry.log"   # Log file path (optional)
  log_file_rotation:                    # Log rotation configuration (optional)
    max_size: 100                       # Maximum file size (MB)
    max_backups: 5                      # Maximum backup files count
    max_age: 30                         # Maximum retention days
    compress: true                      # Whether to compress
  health_check_port: 8080               # Health check and API port
  data_dir: "./data"                    # Data directory
```

#### Important Field Descriptions

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `log_level` | No | `info` | Recommend `info` for production, `debug` for debugging |
| `log_format` | No | `json` | JSON format convenient for log aggregation analysis |
| `health_check_port` | No | `8080` | REST API and health check port |
| `data_dir` | No | `./data` | Database and log file storage directory |

### Polling Configuration (polling)

```yaml
polling:
  interval: "5m"          # Global polling interval
  timeout: "30s"          # API request timeout
  max_workers: 5          # Maximum concurrent worker goroutines
  batch_size: 10          # Number of repositories per batch
  retry_attempts: 3       # Retry attempts on failure
  retry_backoff: "30s"    # Retry interval
```

#### Performance Tuning Guide

| Repository Count | Recommended Configuration | Description |
|------------------|---------------------------|-------------|
| 1-10 | `max_workers: 2, batch_size: 5` | Small-scale deployment |
| 11-50 | `max_workers: 5, batch_size: 10` | Medium-scale |
| 51-200 | `max_workers: 10, batch_size: 20` | Large-scale deployment |
| 200+ | `max_workers: 20, batch_size: 50` | Enterprise deployment |

### Storage Configuration (storage)

```yaml
storage:
  type: "sqlite"
  sqlite:
    path: "./data/reposentry.db"
    max_connections: 10
    connection_timeout: "30s"
    busy_timeout: "5s"
```

#### SQLite Configuration Description

- **path**: Database file path, recommend using absolute path
- **max_connections**: Connection pool size, generally no need to adjust
- **connection_timeout**: Connection timeout
- **busy_timeout**: Database lock wait time

### Tekton Integration Configuration

```yaml
tekton:
  event_listener_url: "http://tekton-listener:8080"
  timeout: "10s"
  headers:
    Content-Type: "application/json"
    X-Custom-Header: "reposentry"
  retry_attempts: 3
  retry_backoff: "5s"
```

#### Required Fields

- **event_listener_url**: Complete URL of Tekton EventListener
- Other fields are optional with reasonable defaults

### Repository Configuration (repositories)

This is the core configuration section of RepoSentry:

```yaml
repositories:
  - name: "frontend-app"                              # Repository unique identifier
    url: "https://github.com/company/frontend-app"    # Repository HTTPS URL
    provider: "github"                                # Provider: github or gitlab
    token: "${GITHUB_TOKEN}"                          # API Token (use environment variable)
    branch_regex: "^(main|develop|release/.*)$"       # Branch filter regex
    polling_interval: "3m"                            # Repository-specific polling interval (optional)
    metadata:                                         # Custom metadata (optional)
      team: "frontend"
      env: "production"
    
  - name: "backend-service"
    url: "https://gitlab-master.nvidia.com/team/backend"
    provider: "gitlab"
    token: "${GITLAB_ENTERPRISE_TOKEN}"
    branch_regex: "^(master|hotfix/.*)$"
    polling_interval: "10m"
```

#### Repository Configuration Field Details

| Field | Required | Type | Description | Example |
|-------|----------|------|-------------|---------|
| `name` | ✅ | string | Repository unique identifier, cannot duplicate | `my-app` |
| `url` | ✅ | string | Repository HTTPS URL, SSH not supported | `https://github.com/user/repo` |
| `provider` | ✅ | string | `github` or `gitlab` | `github` |
| `token` | ✅ | string | API access token, **must** use environment variable | `${GITHUB_TOKEN}` |
| `branch_regex` | ✅ | string | Branch filter regex | `^(main\|develop)$` |
| `polling_interval` | No | string | Override global polling interval | `2m` |
| `metadata` | No | map | Custom metadata, passed to Tekton | `team: frontend` |

#### Branch Regex Examples

```yaml
# Monitor only main branch
branch_regex: "^main$"

# Monitor main and develop branches
branch_regex: "^(main|develop)$"

# Monitor release branches
branch_regex: "^release/.*$"

# Monitor specific prefixes
branch_regex: "^(feature|bugfix)/.*$"

# Monitor multiple patterns
branch_regex: "^(main|develop|release/.*|hotfix/.*)$"
```

### Environment Variable Configuration

RepoSentry supports using environment variables in configuration files:

#### Supported Formats

```yaml
# Standard format
token: "${GITHUB_TOKEN}"

# With default value
url: "${TEKTON_URL:-http://localhost:8080}"

# Complex environment variables
token: "${GITLAB_ENTERPRISE_TOKEN}"
```

#### Environment Variable Whitelist

For security reasons, only environment variables with the following patterns are allowed:

- `*_TOKEN`
- `*_SECRET`
- `*_PASSWORD`
- `*_KEY`
- `*_URL`
- `*_HOST`
- `*_PORT`

## 🖥️ CLI Commands

### Main Commands

#### 1. Configuration Management

```bash
# Generate configuration file
reposentry config init --type=basic > config.yaml
reposentry config init --type=minimal > minimal.yaml

# Validate configuration
reposentry config validate config.yaml
reposentry config validate config.yaml --check-env
reposentry config validate config.yaml --check-connectivity

# Show current configuration
reposentry config show --config=config.yaml
reposentry config show --config=config.yaml --hide-secrets
```

#### 2. Run Service

```bash
# Run in foreground
reposentry run --config=config.yaml

# Run in background
reposentry run --config=config.yaml --daemon

# Specify log level
reposentry run --config=config.yaml --log-level=debug

# Custom port
reposentry run --config=config.yaml --port=9090
```

#### 3. Status Check

```bash
# Check service status
reposentry status

# Check specific host
reposentry status --host=remote-server --port=8080
```

#### 4. Repository Management

```bash
# List all repositories
reposentry repo list

# Show repository details
reposentry repo show my-repo-name

# Test repository connection
reposentry repo test my-repo-name
```

#### 5. Other Tool Commands

```bash
# View version
reposentry version

# Test webhook
reposentry test-webhook --url=http://tekton:8080 --payload='{"test": true}'

# View help
reposentry --help
reposentry run --help
```

### CLI Configuration File Search Order

RepoSentry searches for configuration files in the following order:

1. File specified by `--config` parameter
2. `RS_CONFIG_PATH` environment variable
3. `./config.yaml`
4. `./reposentry.yaml`
5. `~/.reposentry/config.yaml`
6. `/etc/reposentry/config.yaml`

## 🌐 API Interface

RepoSentry provides complete RESTful API interface.

### Swagger UI Documentation

After starting the service, visit the online Swagger documentation:

```
http://localhost:8080/swagger/
```

### Main Interfaces

#### 1. Health Check

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health check
curl http://localhost:8080/healthz

# Readiness check
curl http://localhost:8080/ready
```

#### 2. Service Status

```bash
# Get runtime status
curl http://localhost:8080/api/v1/status

# Get service version
curl http://localhost:8080/api/v1/version

# Get metrics information
curl http://localhost:8080/api/v1/metrics
```

#### 3. Repository Management

```bash
# List all repositories
curl http://localhost:8080/api/v1/repositories

# Get specific repository information
curl http://localhost:8080/api/v1/repositories/my-repo

# Get repository status
curl http://localhost:8080/api/v1/repositories/my-repo/status
```

#### 4. Event Query

```bash
# Get all events
curl http://localhost:8080/api/v1/events

# Get recent events
curl http://localhost:8080/api/v1/events/recent

# Get specific event
curl http://localhost:8080/api/v1/events/{event-id}

# Filter by repository
curl "http://localhost:8080/api/v1/events?repository=my-repo"

# Filter by time range
curl "http://localhost:8080/api/v1/events?since=2024-01-01T00:00:00Z"
```

### API Authentication

Current version API doesn't require authentication, but recommend restricting access through firewall or reverse proxy in production environment.

### API Response Format

All API responses follow a unified format:

```json
{
  "success": true,
  "message": "Operation successful",
  "data": {
    // Response data
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Error response:

```json
{
  "success": false,
  "message": "Error description",
  "error": "Detailed error information",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🔄 Configuration Hot Reload

RepoSentry supports runtime configuration hot reload without service restart.

### Trigger Hot Reload

#### Method 1: Send Signal (Linux/macOS)

```bash
# Send SIGHUP signal
sudo kill -HUP $(pgrep reposentry)

# Or use systemctl (if using systemd)
sudo systemctl reload reposentry
```

#### Method 2: API Interface

```bash
# Reload configuration
curl -X POST http://localhost:8080/api/v1/config/reload
```

#### Method 3: CLI Command

```bash
# Reload configuration
reposentry config reload --host=localhost --port=8080
```

### Hot Reload Notes

#### ✅ Supports Hot Reload

- Repository list (`repositories`)
- Polling interval (`polling.interval`)
- Log level (`app.log_level`)
- Tekton configuration (`tekton`)

#### ❌ Doesn't Support Hot Reload

- Port configuration (`app.health_check_port`)
- Storage configuration (`storage`)
- Data directory (`app.data_dir`)

These configurations require service restart to take effect.

### Verify Hot Reload

```bash
# 1. Modify configuration file
vim config.yaml

# 2. Trigger reload
curl -X POST http://localhost:8080/api/v1/config/reload

# 3. Check if configuration is effective
reposentry config show --host=localhost --port=8080
```

## 📊 Monitoring and Logging

### Log Configuration

#### Log Levels

- **debug**: Detailed debugging information, contains all operation details
- **info**: General information, recommended for production
- **warn**: Warning information, needs attention but doesn't affect operation
- **error**: Error information, needs immediate handling

#### Log Format

```yaml
# JSON format (recommended for production)
app:
  log_format: "json"

# Text format (suitable for development and debugging)
app:
  log_format: "text"
```

#### Log Files

```yaml
app:
  log_file: "/var/log/reposentry.log"
  log_file_rotation:
    max_size: 100      # 100MB
    max_backups: 5     # Keep 5 backups
    max_age: 30        # Keep for 30 days
    compress: true     # Compress old logs
```

### Key Log Fields

JSON format logs contain the following key fields:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "poller",
  "module": "github_client",
  "operation": "fetch_branches",
  "repository": "my-repo",
  "duration": 1250,
  "message": "Successfully fetched branches",
  "metadata": {
    "branch_count": 5,
    "api_rate_remaining": 4999
  }
}
```

### Monitoring Metrics

Get runtime metrics through API:

```bash
curl http://localhost:8080/api/v1/metrics | jq
```

Response example:

```json
{
  "success": true,
  "data": {
    "uptime": "2h30m15s",
    "repositories": {
      "total": 10,
      "healthy": 9,
      "error": 1
    },
    "polling": {
      "last_cycle": "2024-01-15T10:30:00Z",
      "next_cycle": "2024-01-15T10:35:00Z",
      "cycle_duration": "45s"
    },
    "events": {
      "total": 156,
      "today": 23,
      "last_hour": 3
    },
    "api_calls": {
      "github_remaining": 4950,
      "gitlab_remaining": 1850
    }
  }
}
```

### Health Check

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed component health status
curl http://localhost:8080/healthz
```

Health check response:

```json
{
  "success": true,
  "data": {
    "healthy": true,
    "components": {
      "config": {"healthy": true, "message": "OK"},
      "storage": {"healthy": true, "message": "Database connected"},
      "git_client": {"healthy": true, "message": "All clients ready"},
      "trigger": {"healthy": true, "message": "Tekton reachable"},
      "poller": {"healthy": true, "message": "Polling active"}
    }
  }
}
```

## 🔐 Security Best Practices

### 1. Sensitive Information Management

#### ✅ Correct Approach

```yaml
repositories:
  - name: "my-repo"
    token: "${GITHUB_TOKEN}"  # Use environment variable
```

#### ❌ Wrong Approach

```yaml
repositories:
  - name: "my-repo"
    token: "ghp_xxxxxxxxxxxx"  # Hard-coded Token
```

### 2. Token Permission Control

#### GitHub Token Permissions

- **Public repositories**: `public_repo` permission
- **Private repositories**: `repo` permission
- **Organization repositories**: Organization authorization required

#### GitLab Token Permissions

- **Project access**: `read_repository` permission
- **API access**: `read_api` permission
- **Enterprise version**: May require additional access permissions

### 3. Network Security

```yaml
# Restrict listening address (production environment)
app:
  health_check_bind: "127.0.0.1:8080"  # Local access only

# Use HTTPS (through reverse proxy)
tekton:
  event_listener_url: "https://tekton.example.com:8080"
```

### 4. File Permissions

```bash
# Configuration file permissions
chmod 600 config.yaml
chown reposentry:reposentry config.yaml

# Data directory permissions
chmod 750 ./data
chown reposentry:reposentry ./data
```

### 5. Container Security

```yaml
# docker-compose.yml security configuration
services:
  reposentry:
    user: "1000:1000"      # Non-root user
    read_only: true        # Read-only filesystem
    cap_drop:
      - ALL                # Remove all capabilities
    cap_add:
      - NET_BIND_SERVICE   # Keep only necessary capabilities
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Configuration File Issues

**Symptom**: Configuration validation fails during startup

```bash
# Troubleshooting steps
# 1. Check YAML syntax
reposentry config validate config.yaml

# 2. Check environment variables
reposentry config validate config.yaml --check-env

# 3. Check network connectivity
reposentry config validate config.yaml --check-connectivity
```

#### 2. API Token Issues

**Symptom**: Repository access denied

```bash
# GitHub Token test
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/owner/repo

# GitLab Token test
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/projects/owner%2Frepo
```

#### 3. Network Connection Issues

**Symptom**: Cannot connect to Tekton EventListener

```bash
# Test connection
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"test": true}' \
  $TEKTON_EVENTLISTENER_URL

# Check DNS resolution
nslookup tekton-listener.example.com

# Check port connectivity
telnet tekton-listener.example.com 8080
```

#### 4. Permission Issues

**Symptom**: Database creation failed

```bash
# Check directory permissions
ls -la ./data/

# Fix permissions
mkdir -p ./data
chmod 755 ./data
chown $USER:$USER ./data
```

#### 5. Performance Issues

**Symptom**: Slow polling speed

```bash
# Tuning configuration
polling:
  max_workers: 10        # Increase concurrency
  batch_size: 20         # Increase batch size
  timeout: "60s"         # Increase timeout
```

### Log Analysis

#### Enable Detailed Logs

```yaml
app:
  log_level: "debug"
```

#### Key Log Patterns

```bash
# Filter error logs
grep '"level":"error"' /var/log/reposentry.log

# View polling status
grep '"component":"poller"' /var/log/reposentry.log

# Monitor API calls
grep '"operation":"api_call"' /var/log/reposentry.log
```

### Database Recovery

#### Backup Database

```bash
# Stop service
sudo systemctl stop reposentry

# Backup database
cp ./data/reposentry.db ./data/reposentry.db.backup

# Restart service
sudo systemctl start reposentry
```

#### Reset Database

```bash
# Stop service
sudo systemctl stop reposentry

# Delete database (all historical data lost)
rm ./data/reposentry.db

# Restart service (will automatically create new database)
sudo systemctl start reposentry
```

## 🚀 Advanced Usage

### 1. Multi-environment Deployment

#### Development Environment Configuration

```yaml
app:
  log_level: "debug"
  log_format: "text"

polling:
  interval: "1m"         # Frequent polling for testing
  
repositories:
  - name: "test-repo"
    url: "https://github.com/user/test-repo"
    provider: "github"
    token: "${GITHUB_TOKEN}"
    branch_regex: ".*"   # Monitor all branches
```

#### Production Environment Configuration

```yaml
app:
  log_level: "info"
  log_format: "json"
  log_file: "/var/log/reposentry.log"
  log_file_rotation:
    max_size: 100
    max_backups: 10
    max_age: 90

polling:
  interval: "10m"        # Longer interval to reduce API calls
  max_workers: 20
  
repositories:
  - name: "prod-app"
    url: "https://github.com/company/prod-app"
    provider: "github"
    token: "${GITHUB_PROD_TOKEN}"
    branch_regex: "^(main|release/.*)$"  # Production branches only
```

### 2. Enterprise GitLab Integration

```yaml
repositories:
  - name: "enterprise-project"
    url: "https://gitlab-master.nvidia.com/ai/chat-bot"
    provider: "gitlab"
    token: "${GITLAB_ENTERPRISE_TOKEN}"
    branch_regex: "^(master|develop|feature/.*)$"
    polling_interval: "15m"
    metadata:
      team: "ai-research"
      priority: "high"
      environment: "production"
```

### 3. Branch Strategy Patterns

#### Git Flow Pattern

```yaml
repositories:
  - name: "gitflow-repo"
    branch_regex: "^(master|develop|release/.*|hotfix/.*)$"
```

#### GitHub Flow Pattern

```yaml
repositories:
  - name: "githubflow-repo"
    branch_regex: "^(main|feature/.*)$"
```

#### Custom Pattern

```yaml
repositories:
  - name: "custom-repo"
    branch_regex: "^(main|staging|prod|feature/.*|bugfix/.*|hotfix/.*)$"
```

### 4. Monitoring Integration

#### Prometheus Metrics

Although RepoSentry doesn't directly support Prometheus, you can collect metrics through scripts:

```bash
#!/bin/bash
# prometheus-exporter.sh

metrics=$(curl -s http://localhost:8080/api/v1/metrics)
echo "reposentry_uptime_seconds $(echo $metrics | jq -r '.data.uptime_seconds')"
echo "reposentry_repositories_total $(echo $metrics | jq -r '.data.repositories.total')"
echo "reposentry_events_total $(echo $metrics | jq -r '.data.events.total')"
```

#### Log Aggregation

Use ELK Stack or similar tools to aggregate logs:

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  paths:
    - /var/log/reposentry.log
  json.keys_under_root: true
  json.add_error_key: true
  
output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "reposentry-%{+yyyy.MM.dd}"
```

### 5. High Availability Deployment

#### Master-Slave Mode (Shared Database)

```yaml
# Master node - Enable polling
polling:
  enabled: true
  interval: "5m"

# Slave node - API service only
polling:
  enabled: false
```

#### Load Balancing

```nginx
# nginx.conf
upstream reposentry {
    server 192.168.1.10:8080;
    server 192.168.1.11:8080;
    server 192.168.1.12:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://reposentry;
    }
}
```

### 6. Automated Operations

#### Health Check Script

```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/health.json $HEALTH_URL)

if [ "$RESPONSE" != "200" ]; then
    echo "RepoSentry unhealthy, restarting..."
    sudo systemctl restart reposentry
    
    # Send alert
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"RepoSentry service restarted"}' \
        $SLACK_WEBHOOK_URL
fi
```

#### Configuration Sync Script

```bash
#!/bin/bash
# sync-config.sh

# Pull latest configuration from Git repository
cd /etc/reposentry/
git pull origin main

# Validate configuration
if reposentry config validate config.yaml; then
    # Reload configuration
    curl -X POST http://localhost:8080/api/v1/config/reload
    echo "Configuration updated successfully"
else
    echo "Configuration validation failed"
    exit 1
fi
```

## 📝 References

- [Quick Start Guide](QUICKSTART.md)
- [Technical Architecture](ARCHITECTURE.md)
- [Deployment Guide](../deployments/README.md)
- [API Examples](../API_EXAMPLES.md)
- [Configuration Examples](../examples/README.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)

---

**Tip**: If you encounter issues, please first check log files or use `reposentry status` command to diagnose problems.
