# RepoSentry Tekton Integration Guide

## 🚨 Important Change Notice

**RepoSentry v2.0+ has adopted CloudEvents 1.0-based standardized webhook payload format**

### Change Impact
- All existing TriggerBinding configurations need updates
- JSONPath paths have been standardized
- New format provides better compatibility and extensibility

## 🔧 System Templates

### Basic Version (reposentry-basic-system.yaml)
- CloudEvents 1.0 Standard Compatible
- Simple Parameters: provider, organization, repository-name, branch-name, commit-sha
- Enterprise Ready: Minimal configuration, maximum compatibility

### Advanced Version (reposentry-advanced-system.yaml)
- Rich Metadata Extraction
- Enhanced Parameters: repository-id, trigger-source, reposentry-event-id, project-name
- Development Friendly: Detailed context for debugging and monitoring

## 📊 Webhook Payload Format

### CloudEvents Standard Format
```json
{
  "specversion": "1.0",
  "type": "com.github.push",
  "source": "https://github.com/org/repo",
  "data": {
    "provider": "github",
    "organization": "org-name",
    "repository-name": "repo-name",
    "branch-name": "main",
    "commit-sha": "abc123..."
  }
}
```

## 🚀 Deployment Commands

### Basic System
```bash
kubectl apply -f deployments/tekton/reposentry-basic-system.yaml
```

### Advanced System
```bash
kubectl apply -f deployments/tekton/reposentry-advanced-system.yaml
```

---
*This guide provides comprehensive information for integrating RepoSentry with Tekton using CloudEvents standard format.*
