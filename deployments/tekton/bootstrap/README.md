# RepoSentry Bootstrap Pipeline

Generated at: 2025-08-27T02:46:05Z

## Overview

This directory contains the static Bootstrap Pipeline infrastructure for RepoSentry Tekton integration. The Bootstrap Pipeline is a pre-deployed Tekton Pipeline that handles the automatic deployment and execution of user-defined Tekton resources from monitored repositories.

## 🚀 Quick Start

### Option 1: One-Click Installation (Recommended)
```bash
# Auto-detect and install in one command
./install.sh

# Verify installation
./validate.sh
```

### Option 2: Custom Configuration
```bash
# Install with custom settings
./install.sh --ingress-class nginx --webhook-host webhook.your-domain.com

# Or use environment variables
export BOOTSTRAP_INGRESS_CLASS=traefik
export BOOTSTRAP_WEBHOOK_HOST=webhook.example.com
./install.sh

# Verify installation
./validate.sh
```

## 📋 Management Scripts

The Bootstrap Pipeline includes these management scripts:

- **`install.sh`**: 🚀 **One-click installation** with auto-detection and configuration
  - Automatically detects Ingress Controller type (nginx/traefik/istio)
  - Smart webhook host configuration using cluster IP
  - Integrated configuration functionality (no separate configure step needed)
  - Supports all original installation features plus new auto-configuration

- **`uninstall.sh`**: 🗑️ Clean removal of all Bootstrap Pipeline resources

- **`validate.sh`**: ✅ Comprehensive validation of deployed resources



### New Auto-Configuration Features

The `install.sh` script now includes intelligent auto-detection:
- **Ingress Controller Detection**: Automatically finds nginx, traefik, or istio
- **Smart Webhook Host**: Uses cluster IP with nip.io for easy local development
- **Flexible Override**: Command-line arguments override auto-detected values
- **Environment Variables**: Full support for scripted deployments

## 📁 Files

### 🏗️ Infrastructure Layer
- **01-namespace.yaml**: System namespace (`reposentry-system`)
- **02-serviceaccount.yaml**: Service account for Bootstrap Pipeline
- **03-clusterrole.yaml**: RBAC ClusterRole definition with comprehensive permissions
- **04-clusterrolebinding.yaml**: ClusterRoleBinding linking ServiceAccount to ClusterRole

### ⚙️ Pipeline Layer
- **05-tasks.yaml**: All Bootstrap Tasks (clone, validate, apply, run, etc.)
- **06-pipeline.yaml**: Main Bootstrap Pipeline definition with 6-stage workflow

### 🔔 Triggers Layer
- **07-triggerbinding.yaml**: TriggerBinding to extract CloudEvent parameters
- **08-triggertemplate.yaml**: TriggerTemplate to create Bootstrap PipelineRuns
- **09-eventlistener.yaml**: EventListener to receive CloudEvents with CEL filtering

### 🌐 Network Layer
- **10-service.yaml**: Internal Service for EventListener exposure
- **11-ingress.yaml**: External Ingress to expose webhook URL

### Management Scripts
- **install.sh**: Installation script with integrated auto-configuration
- **uninstall.sh**: Uninstallation script for clean removal
- **validate.sh**: Validation script to verify installation health

## 🔄 Deployment Order

The Bootstrap Pipeline follows a strict deployment order to ensure dependencies are properly resolved:

### Stage 1: Infrastructure Foundation
1. **01-namespace.yaml** - Creates the `reposentry-system` namespace
2. **02-serviceaccount.yaml** - Creates the Bootstrap ServiceAccount

### Stage 2: Security & Permissions
3. **03-clusterrole.yaml** - Defines comprehensive RBAC permissions
4. **04-clusterrolebinding.yaml** - Binds permissions to ServiceAccount

### Stage 3: Core Pipeline Components
5. **05-tasks.yaml** - Deploys all Bootstrap Tasks (6 tasks total)
6. **06-pipeline.yaml** - Deploys the main Bootstrap Pipeline

### Stage 4: Event Processing
7. **07-triggerbinding.yaml** - Configures CloudEvent parameter extraction
8. **08-triggertemplate.yaml** - Defines PipelineRun creation template
9. **09-eventlistener.yaml** - Sets up webhook event listener with CEL filtering

### Stage 5: Network Access
10. **10-service.yaml** - Exposes EventListener internally
11. **11-ingress.yaml** - Provides external webhook access

**Note**: Uninstallation follows the reverse order (11 → 1) to ensure clean resource removal.



## 🔧 Configuration

### Supported Ingress Controllers

| Controller | IngressClass | Notes |
|------------|--------------|-------|
| **Nginx** | `nginx` | Default, most common |
| **Traefik** | `traefik` | Cloud-native proxy |
| **Istio** | `istio` | Service mesh |
| **HAProxy** | `haproxy` | High performance |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `INGRESS_CLASS` | Ingress controller class | `nginx` |
| `WEBHOOK_HOST` | Webhook domain/host | `webhook.127.0.0.1.nip.io` |
| `SSL_REDIRECT` | Enable SSL redirect | `false` |
| `SYSTEM_NAMESPACE` | System namespace | `reposentry-system` |

### Installation Examples

#### Auto-Detection (Recommended)
```bash
# One-click installation with auto-detection
./install.sh

# Auto-detection with verbose output
./install.sh --verbose
```

#### Custom Configuration
```bash
# Override Ingress Controller
./install.sh --ingress-class nginx --webhook-host webhook.example.com

# Traefik with SSL
./install.sh --ingress-class traefik --webhook-host webhook.example.com --ssl-redirect true

# Istio setup
./install.sh --ingress-class istio --webhook-host webhook.example.com

# Custom namespace
./install.sh --namespace custom-reposentry
```

#### Environment Variable Configuration
```bash
# Set environment variables
export BOOTSTRAP_INGRESS_CLASS=nginx
export BOOTSTRAP_WEBHOOK_HOST=webhook.dev.example.com
export BOOTSTRAP_SSL_REDIRECT=true

# Install with environment variables
./install.sh
```

## 📋 Deployment Steps

### 1. Prerequisites
- Kubernetes cluster with Tekton Pipelines installed
- Ingress controller deployed (Nginx, Traefik, Istio, etc.)
- `kubectl` configured to access your cluster
- Cluster admin permissions for RBAC setup

### 2. Installation (Configuration is automatic)
```bash
# One-click installation with auto-detection
./install.sh

# Or with custom settings
./install.sh --ingress-class <YOUR_INGRESS_CLASS> --webhook-host <YOUR_DOMAIN>

# Verify installation
./validate.sh

# Check webhook URL
curl -X POST http://<YOUR_WEBHOOK_HOST>/
```

### 4. Verification
```bash
# Check all resources
kubectl get all -n reposentry-system

# Check webhook accessibility
./validate.sh --connectivity

# View recent activity
kubectl get pipelineruns -n reposentry-system
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Webhook Returns 404
```bash
# Check ingress configuration
kubectl describe ingress reposentry-eventlistener-ingress -n reposentry-system

# Verify ingress class
kubectl get ingress -n reposentry-system -o yaml | grep ingressClassName
```

#### 2. EventListener Not Ready
```bash
# Check EventListener status
kubectl get eventlistener -n reposentry-system

# Check EventListener logs
kubectl logs -n reposentry-system -l eventlistener=reposentry-standard-eventlistener
```

#### 3. RBAC Permissions
```bash
# Check ServiceAccount
kubectl get serviceaccount reposentry-bootstrap-sa -n reposentry-system

# Check ClusterRole
kubectl describe clusterrole reposentry-bootstrap-role
```

### Advanced Troubleshooting
```bash
# Full system validation
./validate.sh --verbose

# Check connectivity
./validate.sh --connectivity

# Dry run installation
./install.sh --dry-run
```

## 🛠️ Customization

### Custom Ingress Configuration

Use the integrated configuration options in install.sh:

```bash
# Automatic detection and configuration
./install.sh

# Or specify custom settings
./install.sh --ingress-class traefik --webhook-host webhook.example.com

# Manual editing (if needed)
vim 10-ingress.yaml
```

3. Apply changes:
```bash
kubectl apply -f 10-ingress.yaml
```

### Custom Namespace

1. Configure namespace:
```bash
./configure.sh --namespace custom-reposentry
```

2. All files will be updated automatically.

### Custom Annotations

Edit `10-ingress.yaml` to add controller-specific annotations:

```yaml
annotations:
  # Nginx
  nginx.ingress.kubernetes.io/rewrite-target: /
  
  # Traefik
  traefik.ingress.kubernetes.io/rewrite-target: /
  
  # Istio (usually none needed)
```

## 📚 Architecture

The Bootstrap Pipeline follows this flow:

1. **CloudEvent Reception**: EventListener receives CloudEvents from RepoSentry
2. **Parameter Extraction**: TriggerBinding extracts repository information
3. **PipelineRun Creation**: TriggerTemplate creates a Bootstrap PipelineRun
4. **Repository Processing**: Pipeline clones user repo, validates Tekton resources
5. **Resource Deployment**: Pipeline deploys user resources to computed namespace
6. **User Pipeline Trigger**: User-defined Pipelines are automatically triggered

## 🔗 Related Documentation

- [RepoSentry User Guide](../../../docs/en/user-guide-tekton.md)
- [Bootstrap Pipeline Architecture](../../../docs/en/bootstrap-pipeline-architecture.md)
- [Webhook Flow Architecture](../../../docs/en/webhook-flow-architecture.md)

## 🆘 Support

For issues and questions:
1. Check the troubleshooting section above
2. Run `./validate.sh --verbose` for detailed diagnostics
3. Check RepoSentry logs for integration issues
4. Consult the main documentation in `docs/` directory