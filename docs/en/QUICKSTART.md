# RepoSentry Quick Start Guide

## 🚀 Overview

RepoSentry is a lightweight, cloud-native Git repository monitoring sentinel that supports monitoring GitHub and GitLab repositories for changes and triggering Tekton pipelines.

## ⚡ 5-Minute Quick Start

### Prerequisites

- Go 1.21+ (if building from source)
- Docker (if using container deployment)
- Kubernetes (if using Helm deployment)
- GitHub/GitLab API Token
- Tekton EventListener URL

### Step 1: Get RepoSentry

#### Option 1: Download Pre-built Binary (Recommended)
```bash
# Download latest version (assuming releases are available)
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-linux-amd64
chmod +x reposentry-linux-amd64
sudo mv reposentry-linux-amd64 /usr/local/bin/reposentry
```

#### Option 2: Build from Source
```bash
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry
make build
sudo cp bin/reposentry /usr/local/bin/
```

### Step 2: Prepare Configuration File

Generate basic configuration:

```bash
# Generate basic configuration
reposentry config init --type=basic > config.yaml
```

**Or** manually create `config.yaml`:

```yaml
# Application configuration
app:
  name: "reposentry"
  log_level: "info"
  log_format: "json"
  health_check_port: 8080

# Polling configuration
polling:
  interval: "5m"
  timeout: "30s"
  max_workers: 5
  batch_size: 10

# Storage configuration
storage:
  type: "sqlite"
  sqlite:
    path: "./data/reposentry.db"

# Tekton integration
tekton:
  event_listener_url: "http://your-tekton-listener:8080"
  timeout: "10s"

# Monitored repositories list
repositories:
  - name: "my-github-repo"
    url: "https://github.com/username/repository"
    provider: "github"
    token: "${GITHUB_TOKEN}"
    branch_regex: "^(main|develop|release/.*)$"
    
  - name: "my-gitlab-repo"
    url: "https://gitlab.example.com/group/project"
    provider: "gitlab"
    token: "${GITLAB_TOKEN}"
    branch_regex: "^(main|master|hotfix/.*)$"
```

### Step 3: Set Environment Variables

```bash
# GitHub Token
export GITHUB_TOKEN="ghp_your_github_token_here"

# GitLab Token
export GITLAB_TOKEN="glpat-your_gitlab_token_here"

# Enterprise GitLab (if needed)
export GITLAB_ENTERPRISE_TOKEN="glpat-your_enterprise_token"
```

### Step 4: Validate Configuration

```bash
# Validate configuration file syntax
reposentry config validate config.yaml

# Validate environment variables and connectivity
reposentry config validate config.yaml --check-env --check-connectivity
```

### Step 5: Start RepoSentry

```bash
# Run in foreground (for testing)
reposentry run --config=config.yaml

# Run in background
reposentry run --config=config.yaml --daemon
```

### Step 6: Verify Running Status

```bash
# Check health status
curl http://localhost:8080/health

# View running status
reposentry status

# View monitored repositories
reposentry repo list

# View event history
curl http://localhost:8080/api/v1/events
```

## 🐳 Docker Deployment

### Quick Start

```bash
# Clone repository
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry/deployments/docker

# Edit configuration file
cp ../../examples/configs/basic.yaml config.yaml
vim config.yaml  # Modify your settings

# Set environment variables
export GITHUB_TOKEN="your_github_token"
export GITLAB_TOKEN="your_gitlab_token"

# Start service
docker-compose up -d
```

### View Logs

```bash
# View service logs
docker-compose logs -f reposentry

# Check health status
curl http://localhost:8080/health
```

### Stop Service

```bash
docker-compose down
```

## ☸️ Kubernetes (Helm) Deployment

### Quick Deployment

```bash
# Add necessary Secret
kubectl create secret generic reposentry-tokens \
  --from-literal=github-token="your_github_token" \
  --from-literal=gitlab-token="your_gitlab_token"

# Deploy using example configuration
helm install reposentry ./deployments/helm/reposentry \
  -f examples/kubernetes/helm-values-prod.yaml
```

### Custom Deployment

```bash
# Copy and edit configuration
cp examples/kubernetes/helm-values-prod.yaml my-values.yaml
vim my-values.yaml

# Deploy
helm install reposentry ./deployments/helm/reposentry -f my-values.yaml
```

### Verify Deployment

```bash
# Check Pod status
kubectl get pods -l app.kubernetes.io/name=reposentry

# Check service
kubectl get svc -l app.kubernetes.io/name=reposentry

# Port forward for testing
kubectl port-forward svc/reposentry 8080:8080

# Test health check
curl http://localhost:8080/health
```

## 🔧 Systemd Deployment

### Installation Setup

```bash
# Copy binary file
sudo cp bin/reposentry /usr/local/bin/

# Create configuration directory
sudo mkdir -p /etc/reposentry

# Copy configuration file
sudo cp config.yaml /etc/reposentry/

# Create data directory
sudo mkdir -p /var/lib/reposentry
sudo chown reposentry:reposentry /var/lib/reposentry

# Install systemd service
sudo cp deployments/systemd/reposentry.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### Set Environment Variables

```bash
# Edit service file to add environment variables
sudo systemctl edit reposentry

# Add the following content:
[Service]
Environment="GITHUB_TOKEN=your_github_token"
Environment="GITLAB_TOKEN=your_gitlab_token"
```

### Start Service

```bash
# Enable and start service
sudo systemctl enable reposentry
sudo systemctl start reposentry

# Check status
sudo systemctl status reposentry

# View logs
sudo journalctl -u reposentry -f
```

## ⚙️ Required Configuration Fields

### Core Required Fields

| Field Path | Type | Description | Example |
|------------|------|-------------|---------|
| `tekton.event_listener_url` | string | Tekton EventListener URL | `http://tekton:8080` |
| `repositories[].name` | string | Repository unique identifier | `my-app` |
| `repositories[].url` | string | Repository HTTPS URL | `https://github.com/user/repo` |
| `repositories[].provider` | string | Git provider | `github` or `gitlab` |
| `repositories[].token` | string | API access token | `${GITHUB_TOKEN}` |
| `repositories[].branch_regex` | string | Branch filter regex | `^(main\|develop)$` |

### Optional but Recommended

| Field Path | Type | Default | Description |
|------------|------|---------|-------------|
| `app.log_level` | string | `info` | Log level |
| `app.health_check_port` | int | `8080` | Health check port |
| `polling.interval` | string | `5m` | Polling interval |
| `storage.sqlite.path` | string | `./data/reposentry.db` | Database path |

## 🔍 Verification Checklist

After startup, please check the following items:

- [ ] ✅ Configuration file syntax correct: `reposentry config validate config.yaml`
- [ ] ✅ Environment variables set: `reposentry config validate --check-env`
- [ ] ✅ Network connection normal: `curl http://localhost:8080/health`
- [ ] ✅ Repository access normal: `reposentry repo list`
- [ ] ✅ Tekton connection normal: Check EventListener logs
- [ ] ✅ Polling working normally: Observe event logs

## 🚨 Common Issues

### 1. Configuration Validation Failed
```bash
# Check configuration syntax
reposentry config validate config.yaml

# Check environment variables
echo $GITHUB_TOKEN
echo $GITLAB_TOKEN
```

### 2. Permission Insufficient
```bash
# Check Token permissions
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# GitLab check
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/user
```

### 3. Network Connection Issues
```bash
# Test Tekton connection
curl -X POST $TEKTON_EVENTLISTENER_URL/health

# Check firewall settings
sudo ufw status
```

### 4. Database Permissions
```bash
# Check data directory permissions
ls -la ./data/
chmod 755 ./data/
```

## 📖 Next Steps

- Read [User Manual](USER_MANUAL.md) for detailed configuration
- View [Technical Architecture](ARCHITECTURE.md) to understand how it works
- Visit Swagger API documentation: `http://localhost:8080/swagger/`
- Check [Deployment Guide](../deployments/README.md) for production deployment
