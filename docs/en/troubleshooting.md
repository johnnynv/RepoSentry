# RepoSentry Troubleshooting Guide

## 🔍 Quick Diagnosis

### Health Check Checklist

When encountering issues, please check in the following order:

```bash
# 1. Check service status
reposentry status

# 2. Check configuration file
reposentry config validate config.yaml --check-env --check-connectivity

# 3. Check health interface
curl http://localhost:8080/health

# 4. View logs
tail -f /var/log/reposentry.log
# Or for systemd
sudo journalctl -u reposentry -f
```

## 🚨 Common Issues

### 1. Startup Failure

#### Symptom: Service cannot start
```bash
Error: failed to start RepoSentry: configuration validation failed
```

#### Troubleshooting Steps:

**Check configuration file syntax**
```bash
# Validate YAML syntax
reposentry config validate config.yaml

# Common errors: incorrect indentation, field name misspelling
# Use online YAML validator to check syntax
```

**Check required fields**
```bash
# Validate required fields
reposentry config validate config.yaml --verbose

# Ensure the following fields are configured:
# - tekton.event_listener_url
# - repositories[].name
# - repositories[].url  
# - repositories[].provider
# - repositories[].token
# - repositories[].branch_regex
```

**Check environment variables**
```bash
# Validate environment variables
echo $GITHUB_TOKEN
echo $GITLAB_TOKEN

# Check environment variable expansion
reposentry config show --config=config.yaml
```

#### Solutions:
1. Fix configuration file syntax errors
2. Add missing required fields
3. Set correct environment variables

### 2. Permission Issues

#### Symptom: API calls rejected
```
Error: failed to fetch branches: 401 Unauthorized
```

#### Troubleshooting Steps:

**Test GitHub Token**
```bash
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/user

# Successful response should contain user information
# Error response: {"message": "Bad credentials"}
```

**Test GitLab Token**
```bash
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/user

# Enterprise GitLab
curl -H "PRIVATE-TOKEN: $GITLAB_ENTERPRISE_TOKEN" \
  https://gitlab-master.nvidia.com/api/v4/user
```

**Check repository access permissions**
```bash
# GitHub repository permissions
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/owner/repository

# GitLab project permissions  
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/projects/owner%2Frepository
```

#### Solutions:
1. **Token expired**: Regenerate API Token
2. **Insufficient permissions**: Ensure token has repository read permissions
3. **Wrong token format**: Check token prefix (GitHub: ghp_, GitLab: glpat-)

### 3. Network Connection Issues

#### Symptom: Cannot connect to external services
```
Error: dial tcp: lookup github.com: no such host
Error: context deadline exceeded
```

#### Troubleshooting Steps:

**DNS resolution test**
```bash
# Test DNS resolution
nslookup github.com
nslookup gitlab.com
nslookup your-tekton-listener.com

# Test custom DNS
dig @8.8.8.8 github.com
```

**Network connection test**
```bash
# Test HTTPS connection
curl -I https://api.github.com
curl -I https://gitlab.com/api/v4

# Test Tekton EventListener
curl -X POST -H "Content-Type: application/json" \
  -d '{"test": true}' \
  $TEKTON_EVENTLISTENER_URL
```

**Firewall check**
```bash
# Check firewall status
sudo ufw status
sudo iptables -L

# Check open ports
sudo netstat -tulpn | grep :8080
ss -tulpn | grep :8080
```

#### Solutions:
1. **DNS issues**: Configure correct DNS servers
2. **Firewall blocking**: Open necessary outbound ports (80, 443, 8080)
3. **Proxy configuration**: Configure HTTP_PROXY and HTTPS_PROXY
4. **Network policies**: Check Kubernetes NetworkPolicy

### 4. Database Issues

#### Symptom: Database operation failed
```
Error: failed to initialize storage: database is locked
Error: no such table: repository_states
```

#### Troubleshooting Steps:

**Check database file**
```bash
# Check database file permissions
ls -la ./data/reposentry.db

# Check directory permissions
ls -la ./data/

# Check disk space
df -h ./data/
```

**Database integrity check**
```bash
# SQLite integrity check
sqlite3 ./data/reposentry.db "PRAGMA integrity_check;"

# Check table structure
sqlite3 ./data/reposentry.db ".schema"

# Check migration status
sqlite3 ./data/reposentry.db "SELECT * FROM schema_migrations;"
```

#### Solutions:
1. **Permission issues**: `chmod 755 ./data && chmod 644 ./data/reposentry.db`
2. **Insufficient disk space**: Clean up disk space
3. **Database corruption**: Delete database file, reinitialize
4. **Multi-instance conflict**: Ensure only one instance accesses database

### 5. Polling Issues

#### Symptom: Polling not working or abnormal frequency
```
Warning: polling cycle took 5m30s, expected 5m
Error: no events generated in last 2 hours
```

#### Troubleshooting Steps:

**Check polling status**
```bash
# View polling metrics
curl http://localhost:8080/api/v1/metrics | jq '.data.polling'

# View repository status
reposentry repo list

# Check recent events
curl http://localhost:8080/api/v1/events/recent
```

**Analyze polling logs**
```bash
# Filter polling-related logs
grep '"component":"poller"' /var/log/reposentry.log | tail -20

# View error logs
grep '"level":"error"' /var/log/reposentry.log | grep poller
```

#### Solutions:
1. **API limits**: Increase polling interval, check API quota
2. **Performance issues**: Adjust `max_workers` and `batch_size`
3. **Branch filtering**: Check if `branch_regex` is correct
4. **Cache issues**: Clean database or restart service

### 6. Tekton Integration Issues

#### Symptom: Events not triggering Tekton pipelines
```
Error: failed to send webhook: connection refused
Warning: webhook sent but no pipeline triggered
```

#### Troubleshooting Steps:

**Test EventListener connection**
```bash
# Test EventListener health status
curl http://tekton-listener:8080/health

# Manually send test event
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Git-Source: github" \
  -d '{
    "repository": {"name": "test", "url": "https://github.com/test/test"},
    "ref": "refs/heads/main",
    "commits": [{"id": "abc123", "message": "test"}]
  }' \
  $TEKTON_EVENTLISTENER_URL
```

**Check Tekton configuration**
```bash
# Check EventListener
kubectl get eventlistener -A

# Check TriggerBinding
kubectl get triggerbinding -A

# Check TriggerTemplate  
kubectl get triggertemplate -A

# View EventListener logs
kubectl logs -l app=el-github-listener -n tekton-pipelines
```

#### Solutions:
1. **Wrong URL**: Check `tekton.event_listener_url` configuration
2. **Network disconnection**: Check Kubernetes network policies and service discovery
3. **Payload format**: Confirm expected payload format for Tekton
4. **Permission issues**: Check Tekton RBAC configuration

## 🛠️ Log Analysis

### Enable Detailed Logs

```yaml
# config.yaml
app:
  log_level: "debug"  # Enable detailed logs
  log_format: "json"  # Convenient for analysis
```

### Log Filtering Techniques

```bash
# Filter by component
grep '"component":"poller"' /var/log/reposentry.log
grep '"component":"trigger"' /var/log/reposentry.log  
grep '"component":"gitclient"' /var/log/reposentry.log

# Filter by log level
grep '"level":"error"' /var/log/reposentry.log
grep '"level":"warn"' /var/log/reposentry.log

# Filter by time range
grep '"timestamp":"2024-01-15T1[0-2]"' /var/log/reposentry.log

# Filter by operation
grep '"operation":"fetch_branches"' /var/log/reposentry.log
grep '"operation":"send_webhook"' /var/log/reposentry.log

# Filter by repository
grep '"repository":"my-repo"' /var/log/reposentry.log
```

### Key Log Patterns

```bash
# Success patterns
grep '"message":"Successfully"' /var/log/reposentry.log

# Error patterns
grep '"error"' /var/log/reposentry.log | jq -r '.error'

# Performance monitoring
grep '"duration"' /var/log/reposentry.log | jq '.duration'

# API call monitoring
grep '"api_rate_remaining"' /var/log/reposentry.log
```

## 🔧 Performance Issue Diagnosis

### High Memory Usage

**Check memory usage**
```bash
# System memory
free -h

# Process memory
ps aux | grep reposentry

# Container memory (Docker)
docker stats reposentry

# Pod memory (Kubernetes)
kubectl top pod -l app=reposentry
```

**Optimization configuration**
```yaml
polling:
  max_workers: 5      # Reduce concurrency
  batch_size: 10      # Reduce batch size
  interval: "10m"     # Increase polling interval
```

### High CPU Usage

**Analyze CPU usage**
```bash
# System CPU
top -p $(pgrep reposentry)

# Detailed CPU analysis
pidstat -p $(pgrep reposentry) 1

# Go performance analysis
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

**Optimization strategies**
1. Increase polling interval
2. Reduce concurrent goroutines
3. Optimize branch regex
4. Enable caching mechanism

### High Disk I/O

**Check disk usage**
```bash
# Disk I/O
iotop -p $(pgrep reposentry)

# Database size
du -sh ./data/reposentry.db

# Log file size
du -sh /var/log/reposentry.log
```

**Optimization configuration**
```yaml
app:
  log_file_rotation:
    max_size: 50        # Reduce log file size
    max_backups: 3      # Reduce backup files count
```

## 🚀 Recovery Procedures

### Service Recovery

```bash
# 1. Stop service
sudo systemctl stop reposentry

# 2. Backup current configuration and data
cp config.yaml config.yaml.backup
cp -r ./data ./data.backup

# 3. Reset configuration (if needed)
reposentry config init --type=basic > config.yaml.new

# 4. Validate configuration
reposentry config validate config.yaml.new

# 5. Restart service
sudo systemctl start reposentry

# 6. Verify running status
reposentry status
```

### Database Recovery

```bash
# 1. Stop service
sudo systemctl stop reposentry

# 2. Backup corrupted database
mv ./data/reposentry.db ./data/reposentry.db.corrupted

# 3. If backup exists, restore backup
cp ./data/reposentry.db.backup ./data/reposentry.db

# 4. If no backup, reinitialize
rm -f ./data/reposentry.db

# 5. Restart service (will automatically create new database)
sudo systemctl start reposentry

# 6. Verify database
sqlite3 ./data/reposentry.db ".tables"
```

### Complete Reset

```bash
# Warning: This will delete all data and configuration

# 1. Stop service
sudo systemctl stop reposentry

# 2. Backup important configuration
cp config.yaml config.yaml.emergency.backup

# 3. Delete all data
rm -rf ./data
rm -f /var/log/reposentry.log*

# 4. Regenerate configuration
reposentry config init --type=basic > config.yaml

# 5. Edit configuration file
vim config.yaml

# 6. Set environment variables
export GITHUB_TOKEN="your_token"
export GITLAB_TOKEN="your_token"

# 7. Validate configuration
reposentry config validate config.yaml --check-env

# 8. Start service
sudo systemctl start reposentry
```

## 📞 Getting Help

### Community Support

1. **GitHub Issues**: https://github.com/johnnynv/RepoSentry/issues
2. **Discussions**: https://github.com/johnnynv/RepoSentry/discussions
3. **Documentation Site**: https://reposentry.docs.example.com

### Reporting Issues

When submitting issues, please include:

1. **RepoSentry version**: `reposentry version`
2. **Operating system**: `uname -a`
3. **Configuration file**: Sanitized configuration file
4. **Error logs**: Relevant error logs
5. **Reproduction steps**: Detailed reproduction steps

### Log Collection Script

```bash
#!/bin/bash
# Generate diagnostic report

echo "=== RepoSentry Diagnostic Report ===" > diagnostic.txt
echo "Generated at: $(date)" >> diagnostic.txt
echo "" >> diagnostic.txt

echo "=== Version Information ===" >> diagnostic.txt
reposentry version >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== System Information ===" >> diagnostic.txt
uname -a >> diagnostic.txt
cat /etc/os-release >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== Configuration Validation ===" >> diagnostic.txt
reposentry config validate config.yaml >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== Health Check ===" >> diagnostic.txt
curl -s http://localhost:8080/health >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== Recent Logs ===" >> diagnostic.txt
tail -50 /var/log/reposentry.log >> diagnostic.txt 2>&1

echo "Diagnostic report generated: diagnostic.txt"
```

---

If all the above methods cannot solve the problem, please submit a detailed issue report to GitHub Issues.
