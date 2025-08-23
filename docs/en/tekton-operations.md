# RepoSentry Tekton Operations Guide

## 🚨 One-Line Migration Commands

### 1. Backup Existing Configuration
```bash
kubectl get eventlistener,triggerbinding,triggertemplate,pipeline -o yaml > backup-tekton-config.yaml
```

### 2. Deploy New CloudEvents Standard System
```bash
kubectl apply -f deployments/tekton/reposentry-basic-system.yaml
```

### 3. Clean Up Old Configuration
```bash
kubectl delete eventlistener hello-event-listener --ignore-not-found
kubectl delete triggerbinding hello-trigger-binding --ignore-not-found
```

## 🔄 Migration Checklist

### Pre-Migration
- [ ] Backup existing Tekton resources
- [ ] Verify RepoSentry webhook endpoint
- [ ] Check current pipeline compatibility

### Migration Steps
- [ ] Deploy new CloudEvents system
- [ ] Update webhook URL in RepoSentry
- [ ] Test webhook delivery
- [ ] Verify pipeline execution

## 🔍 Monitoring Commands Reference

### View Pipeline Runs
```bash
kubectl get pipelineruns -A --sort-by=.metadata.creationTimestamp
kubectl describe pipelinerun <name> -n default
```

### View Logs
```bash
kubectl logs -f pipelinerun <name> -n default
kubectl logs -f taskrun <name> -n default
```

## 📊 Migration Validation

### Webhook Testing
```bash
curl -X POST http://your-reposentry:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "payload"}'
```

### Pipeline Verification
```bash
kubectl get pipelineruns
kubectl logs -f pipelinerun/<run-name>
```

## 🔧 Troubleshooting

### Common Issues
- Webhook not delivered: Check EventListener service
- Pipeline not triggered: Verify TriggerBinding parameters
- Parameter mismatch: Check webhook payload format

### Debug Commands
```bash
kubectl get eventlistener -o wide
kubectl logs -f deployment/reposentry-basic-eventlistener
```

---
*This guide provides quick migration commands and monitoring operations for RepoSentry Tekton integration.*
