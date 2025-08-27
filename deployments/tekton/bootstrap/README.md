# RepoSentry Bootstrap Pipeline

Generated at: 2025-08-27T02:46:05Z

## Overview

This directory contains the static Bootstrap Pipeline infrastructure for RepoSentry Tekton integration.

## Files

- **00-namespace.yaml**: System namespace
- **01-pipeline.yaml**: Bootstrap Pipeline definition
- **02-tasks.yaml**: Bootstrap Tasks
- **03-serviceaccount.yaml**: Service account for Bootstrap Pipeline
- **04-role.yaml**: RBAC role definition
- **05-rolebinding.yaml**: Role binding

## Deployment

To deploy the Bootstrap Pipeline infrastructure:

### 1. Apply all resources
```bash
kubectl apply -f .
```

### 2. Verify deployment
```bash
# Check namespace
kubectl get namespace reposentry-system

# Check pipeline
kubectl get pipeline -n reposentry-system

# Check tasks
kubectl get task -n reposentry-system

# Check RBAC
kubectl get serviceaccount,role,rolebinding -n reposentry-system
```

### 3. Configure RepoSentry
Ensure your RepoSentry configuration has Tekton enabled:

```yaml
tekton:
  enabled: true
  # Other Tekton configuration...
```

## Next Steps

1. Deploy these resources to your Kubernetes cluster
2. Configure RepoSentry with proper Tekton settings
3. Start RepoSentry - it will automatically trigger the Bootstrap Pipeline when detecting Tekton resources in monitored repositories

## Troubleshooting

- Ensure your cluster has Tekton Pipelines installed
- Verify RBAC permissions for the reposentry-bootstrap-sa service account
- Check Bootstrap Pipeline logs: `kubectl logs -n reposentry-system -l tekton.dev/pipeline=reposentry-bootstrap-pipeline`
