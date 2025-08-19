# RepoSentry Helm Chart

A Helm chart for deploying RepoSentry, a lightweight Git repository monitoring service for Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installation

### Add Helm Repository

```bash
helm repo add reposentry https://charts.reposentry.io
helm repo update
```

### Install the Chart

```bash
# Basic installation
helm install reposentry reposentry/reposentry

# Install with custom values
helm install reposentry reposentry/reposentry -f values.yaml

# Install in specific namespace
helm install reposentry reposentry/reposentry -n monitoring --create-namespace
```

### Example Installation Commands

```bash
# Development environment
helm install reposentry ./deployments/helm/reposentry \
  -f deployments/helm/reposentry/values-development.yaml \
  -n reposentry-dev --create-namespace

# Production environment
helm install reposentry ./deployments/helm/reposentry \
  -f deployments/helm/reposentry/values-production.yaml \
  -n reposentry-prod --create-namespace
```

## Configuration

The following table lists the configurable parameters and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Global Docker image registry | `""` |
| `global.imagePullSecrets` | Global Docker registry secret names | `[]` |
| `global.storageClass` | Global storage class | `""` |

### Image Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.registry` | Image registry | `docker.io` |
| `image.repository` | Image repository | `reposentry/reposentry` |
| `image.tag` | Image tag | `""` (uses appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.pullSecrets` | Image pull secrets | `[]` |

### Deployment Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `updateStrategy.type` | Deployment update strategy | `RollingUpdate` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.targetPort` | Target port | `8080` |
| `service.annotations` | Service annotations | `{}` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress hosts | `[{host: reposentry.local, paths: [{path: /, pathType: Prefix}]}]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Persistence Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.storageClass` | Storage class | `""` |
| `persistence.accessMode` | Access mode | `ReadWriteOnce` |
| `persistence.size` | Storage size | `10Gi` |
| `persistence.annotations` | PVC annotations | `{}` |

### Resource Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Autoscaling Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `1` |
| `autoscaling.maxReplicas` | Maximum replicas | `3` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory utilization | `80` |

### RepoSentry Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.app.name` | Application name | `reposentry` |
| `config.app.logLevel` | Log level | `info` |
| `config.app.logFormat` | Log format | `json` |
| `config.app.dataDir` | Data directory | `/app/data` |
| `config.app.healthCheckPort` | Health check port | `8080` |
| `config.polling.interval` | Polling interval | `5m` |
| `config.polling.timeout` | Polling timeout | `30s` |
| `config.polling.maxWorkers` | Max workers | `2` |
| `config.polling.batchSize` | Batch size | `5` |
| `config.polling.retryAttempts` | Retry attempts | `3` |
| `config.polling.retryBackoff` | Retry backoff | `2s` |
| `config.storage.type` | Storage type | `sqlite` |
| `config.storage.sqlite.path` | SQLite database path | `/app/data/reposentry.db` |
| `config.storage.sqlite.maxConnections` | Max connections | `10` |
| `config.storage.sqlite.connectionTimeout` | Connection timeout | `30s` |
| `config.tekton.eventListenerURL` | Tekton EventListener URL | `""` (required) |
| `config.tekton.namespace` | Tekton namespace | `tekton-pipelines` |
| `config.tekton.timeout` | Tekton timeout | `30s` |
| `config.tekton.headers` | Custom headers | `{}` |
| `config.repositories` | Repository configurations | `[]` |

### Security Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podSecurityContext.runAsNonRoot` | Run as non-root | `true` |
| `podSecurityContext.runAsUser` | User ID | `1000` |
| `podSecurityContext.runAsGroup` | Group ID | `1000` |
| `podSecurityContext.fsGroup` | FS Group | `1000` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `securityContext.capabilities.drop` | Dropped capabilities | `[ALL]` |

### Monitoring Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceMonitor.enabled` | Enable ServiceMonitor | `false` |
| `serviceMonitor.interval` | Scrape interval | `30s` |
| `serviceMonitor.path` | Metrics path | `/metrics` |
| `serviceMonitor.port` | Metrics port | `http` |

## Usage Examples

### Basic Setup

```yaml
# values.yaml
config:
  tekton:
    eventListenerURL: "https://tekton.example.com/webhook"
  
  repositories:
    - name: my-app
      url: "https://github.com/org/my-app"
      provider: "github"
      token: "${GITHUB_TOKEN}"
      branchRegex: "^(main|master)$"

secrets:
  create: true
  data:
    GITHUB_TOKEN: "your_github_token"
```

### Production Setup

```yaml
# values-prod.yaml
replicaCount: 2

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5

persistence:
  enabled: true
  storageClass: "fast-ssd"
  size: 50Gi

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: reposentry.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: reposentry-tls
      hosts:
        - reposentry.example.com

config:
  app:
    logLevel: info
    logFormat: json
  
  polling:
    interval: "5m"
    maxWorkers: 5
    batchSize: 10
  
  tekton:
    eventListenerURL: "https://tekton.prod.example.com/webhook"
  
  repositories:
    - name: main-app
      url: "https://github.com/company/main-app"
      provider: "github"
      token: "${GITHUB_TOKEN}"
      branchRegex: "^(main|release/.*)$"

secrets:
  create: true
  data:
    GITHUB_TOKEN: ""  # Use external-secrets or sealed-secrets

serviceMonitor:
  enabled: true

networkPolicy:
  enabled: true
```

### Development Setup

```yaml
# values-dev.yaml
config:
  app:
    logLevel: debug
    logFormat: text
  
  polling:
    interval: "1m"
  
  tekton:
    eventListenerURL: "http://tekton.dev.local/webhook"
  
  repositories:
    - name: test-repo
      url: "https://github.com/example/test"
      provider: "github"
      token: "${GITHUB_TOKEN}"
      branchRegex: ".*"

secrets:
  create: true
  data:
    GITHUB_TOKEN: "dev_token"

service:
  type: NodePort

persistence:
  size: 5Gi
```

## Upgrading

```bash
# Upgrade to new version
helm upgrade reposentry reposentry/reposentry -f values.yaml

# Upgrade with new values
helm upgrade reposentry reposentry/reposentry -f new-values.yaml
```

## Uninstalling

```bash
helm uninstall reposentry -n reposentry
```

## Troubleshooting

### Common Issues

1. **Pod stuck in pending**: Check PVC and storage class
   ```bash
   kubectl describe pvc reposentry -n reposentry
   ```

2. **Health check failures**: Check configuration and logs
   ```bash
   kubectl logs deployment/reposentry -n reposentry
   ```

3. **Ingress not working**: Verify ingress controller and DNS
   ```bash
   kubectl describe ingress reposentry -n reposentry
   ```

### Debugging Commands

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=reposentry -n reposentry

# View logs
kubectl logs -f deployment/reposentry -n reposentry

# Check configuration
kubectl get configmap reposentry -n reposentry -o yaml

# Test health endpoint
kubectl port-forward svc/reposentry 8080:8080 -n reposentry
curl http://localhost:8080/health

# Run helm tests
helm test reposentry -n reposentry
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This Helm chart is licensed under the MIT License.
