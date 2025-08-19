# RepoSentry Deployment Guide

RepoSentry supports multiple deployment methods to fit different environments and requirements.

## Quick Start

1. **Configuration**: Create your configuration file
   ```bash
   ./reposentry config init config.yaml --template=basic
   # Edit config.yaml with your settings
   ```

2. **Environment Variables**: Set your API tokens
   ```bash
   export GITHUB_TOKEN="your_github_token"
   export GITLAB_TOKEN="your_gitlab_token"
   ```

3. **Run**: Start RepoSentry
   ```bash
   ./reposentry run --config=config.yaml
   ```

## Deployment Methods

### 1. Systemd Service (Recommended for VMs/Bare Metal)

**Pros**: Native Linux integration, automatic startup, resource management
**Cons**: Linux-only

#### Installation

1. **Build for Linux**:
   ```bash
   make build-linux
   ```

2. **Install using script**:
   ```bash
   sudo ./deployments/systemd/install.sh
   ```

3. **Configure**:
   ```bash
   sudo vim /etc/reposentry/config.yaml
   sudo vim /etc/reposentry/environment  # Add tokens
   ```

4. **Start service**:
   ```bash
   sudo systemctl start reposentry
   sudo systemctl status reposentry
   ```

#### Management Commands

```bash
# Status
sudo systemctl status reposentry

# Logs
sudo journalctl -u reposentry -f

# Restart
sudo systemctl restart reposentry

# Stop
sudo systemctl stop reposentry

# Disable auto-start
sudo systemctl disable reposentry
```

#### Uninstallation

```bash
sudo ./deployments/systemd/install.sh uninstall
```

### 2. Docker Container (Recommended for Development)

**Pros**: Portable, isolated, easy to update
**Cons**: Additional Docker overhead

#### Quick Start

1. **Create configuration**:
   ```bash
   cp configs/example.yaml deployments/docker/config.yaml
   # Edit config.yaml
   ```

2. **Run with Docker Compose**:
   ```bash
   cd deployments/docker
   docker-compose up -d
   ```

3. **Check status**:
   ```bash
   docker-compose ps
   docker-compose logs -f reposentry
   ```

#### Manual Docker Run

```bash
# Build image
docker build -f deployments/docker/Dockerfile -t reposentry:latest .

# Run container
docker run -d \
  --name reposentry \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/configs/config.yaml:ro \
  -v reposentry-data:/app/data \
  -e GITHUB_TOKEN=your_token \
  -e GITLAB_TOKEN=your_token \
  reposentry:latest
```

#### Docker Management

```bash
# Status
docker ps | grep reposentry

# Logs
docker logs -f reposentry

# Restart
docker restart reposentry

# Stop
docker stop reposentry

# Remove
docker rm -f reposentry
```

### 3. Kubernetes with Helm (Recommended for K8s)

**Pros**: Cloud-native, scalable, declarative, production-ready
**Cons**: Complex setup, requires Kubernetes knowledge

#### Quick Installation

1. **Install from local chart**:
   ```bash
   # Development
   helm install reposentry ./deployments/helm/reposentry \
     -f deployments/helm/reposentry/values-development.yaml \
     -n reposentry-dev --create-namespace
   
   # Production
   helm install reposentry ./deployments/helm/reposentry \
     -f deployments/helm/reposentry/values-production.yaml \
     -n reposentry-prod --create-namespace
   ```

2. **Create custom values file**:
   ```yaml
   # values.yaml
   config:
     tekton:
       eventListenerURL: "https://tekton.example.com/webhook"
     
     repositories:
       - name: my-repo
         url: "https://github.com/org/repo"
         provider: github
         token: "${GITHUB_TOKEN}"
         branchRegex: "^(main|develop)$"
         pollingInterval: "5m"
   
   secrets:
     create: true
     data:
       GITHUB_TOKEN: "your_github_token"
       GITLAB_TOKEN: "your_gitlab_token"
   
   ingress:
     enabled: true
     hosts:
       - host: reposentry.example.com
         paths:
           - path: /
             pathType: Prefix
   
   persistence:
     enabled: true
     size: 20Gi
   
   autoscaling:
     enabled: true
     minReplicas: 2
     maxReplicas: 5
   ```

3. **Install with custom values**:
   ```bash
   helm install reposentry ./deployments/helm/reposentry -f values.yaml
   ```

#### Kubernetes Management

```bash
# Status
kubectl get pods -l app.kubernetes.io/name=reposentry

# Logs
kubectl logs -f deployment/reposentry

# Scale manually
kubectl scale deployment reposentry --replicas=3

# Update configuration
helm upgrade reposentry ./deployments/helm/reposentry -f values.yaml

# Run tests
helm test reposentry

# Uninstall
helm uninstall reposentry
```

#### Production Features

- **High Availability**: Multi-replica deployment with PodDisruptionBudget
- **Auto-scaling**: HorizontalPodAutoscaler based on CPU/Memory
- **Security**: NetworkPolicy, SecurityContext, non-root containers
- **Monitoring**: ServiceMonitor for Prometheus integration
- **Persistence**: PersistentVolume for data storage
- **Health Checks**: Liveness, readiness, and startup probes
- **Ingress**: HTTPS with cert-manager integration

## Configuration

### Environment Variables

RepoSentry supports environment variable substitution in configuration files:

```yaml
repositories:
  - name: github-repo
    token: "${GITHUB_TOKEN}"    # From environment
    url: "https://github.com/org/repo"
```

### Common Environment Variables

- `GITHUB_TOKEN`: GitHub API token
- `GITLAB_TOKEN`: GitLab API token  
- `RS_LOG_LEVEL`: Log level (debug, info, warn, error)
- `RS_LOG_FORMAT`: Log format (json, text)
- `RS_DATA_DIR`: Data directory path

### Security Considerations

1. **File Permissions**:
   - Configuration: `640` (root:reposentry)
   - Environment file: `600` (root only)
   - Data directory: `750` (reposentry:reposentry)

2. **Network Security**:
   - Only expose port 8080 if external access needed
   - Use HTTPS for webhook URLs
   - Validate Tekton EventListener certificates

3. **Resource Limits**:
   - Memory: 512MB recommended
   - CPU: 0.5 cores recommended
   - Disk: Monitor data directory growth

## Monitoring

### Health Checks

```bash
# Check health
curl http://localhost:8080/health

# Get status
./reposentry status --port 8080

# View metrics
curl http://localhost:8080/metrics
```

### Logs

RepoSentry uses structured JSON logging by default:

```bash
# Systemd
journalctl -u reposentry -f --output=json

# Docker
docker logs reposentry 2>&1 | jq .

# Direct
./reposentry run --log-format=json 2>&1 | jq .
```

### Metrics

Basic metrics are available at `/metrics`:
- Request counts
- Error rates  
- Repository status
- Event processing stats

## Troubleshooting

### Common Issues

1. **Permission Denied**:
   ```bash
   # Check file permissions
   ls -la /etc/reposentry/
   
   # Fix ownership
   sudo chown -R reposentry:reposentry /var/lib/reposentry
   ```

2. **Config Validation Errors**:
   ```bash
   # Validate configuration
   ./reposentry config validate /etc/reposentry/config.yaml
   
   # Check environment variables
   ./reposentry config validate --check-env
   ```

3. **Network Issues**:
   ```bash
   # Test connectivity
   ./reposentry config validate --check-connections
   
   # Check Tekton EventListener
   curl -X POST https://your-tekton-url/webhook
   ```

4. **High Memory Usage**:
   - Reduce polling frequency
   - Limit max workers
   - Check for memory leaks in logs

### Debug Mode

Enable debug logging for troubleshooting:

```bash
./reposentry run --log-level=debug --config=config.yaml
```

## Updates

### Binary Updates

1. **Stop service**:
   ```bash
   sudo systemctl stop reposentry
   ```

2. **Replace binary**:
   ```bash
   sudo cp new-reposentry /usr/local/bin/reposentry
   ```

3. **Start service**:
   ```bash
   sudo systemctl start reposentry
   ```

### Docker Updates

```bash
# Pull new image
docker pull reposentry:latest

# Restart with new image
docker-compose up -d --force-recreate
```

### Configuration Updates

RepoSentry supports configuration hot-reload:

```bash
# Systemd
sudo systemctl reload reposentry

# Docker
docker kill -s SIGHUP reposentry

# Direct
kill -HUP $(pgrep reposentry)
```

## Performance Tuning

### Polling Configuration

```yaml
polling:
  interval: "5m"        # Reduce for faster detection
  max_workers: 5        # Increase for more repositories
  batch_size: 10        # Process multiple repos together
  timeout: "30s"        # API call timeout
```

### Resource Limits

```yaml
app:
  health_check_port: 8080

storage:
  sqlite:
    max_connections: 20     # Increase for high load
    connection_timeout: "30s"
```

## Support

- **Documentation**: https://github.com/johnnynv/RepoSentry/docs
- **Issues**: https://github.com/johnnynv/RepoSentry/issues
- **Discussions**: https://github.com/johnnynv/RepoSentry/discussions
