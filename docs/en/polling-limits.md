# RepoSentry Polling Limits

## 📊 Polling Interval Restrictions

### Minimum Polling Interval: 1 Minute

RepoSentry enforces a **minimum polling interval of 1 minute** for important security and operational reasons.

## 🛡️ Why This Limit Exists

### 1. API Rate Limit Protection
- **GitHub API Limit**: 5,000 requests per hour
- **GitLab API Limit**: 10 requests per second
- **Frequent Polling Risk**: 30-second polling would consume 120 requests per hour, quickly exhausting quotas

### 2. Service Abuse Prevention
- Protects Git provider infrastructure
- Avoids being flagged for abusive behavior
- Prevents account restrictions or bans

### 3. Practical Usage Considerations
- Code changes typically don't happen every second
- 1-minute delay is acceptable for most CI/CD scenarios
- More frequent polling usually provides no practical benefit

## ⚙️ Configuration Examples

### ✅ Correct Configuration
```yaml
# Global polling configuration
polling:
  interval: "5m"  # Recommended: 5 minutes

# Repository-level configuration
repositories:
  - name: "my-repo"
    polling_interval: "1m"  # Minimum: 1 minute
```

### ❌ Invalid Configuration
```yaml
# These configurations will cause validation failures
polling:
  interval: "30s"  # ❌ Less than 1 minute

repositories:
  - name: "my-repo"
    polling_interval: "45s"  # ❌ Less than 1 minute
```

## 🔧 Configuration Recommendations

### Production Environment
- **Active Projects**: 1-2 minutes
- **Regular Projects**: 5 minutes
- **Archived Projects**: 15-30 minutes

### Development Environment
- **Minimum Interval**: 1 minute (for testing)
- **Recommended Interval**: 2-3 minutes

## 🚨 Error Message Explanation

When you see this error:
```
validation error for field 'repositories[0].polling_interval': 
polling interval cannot be less than 1 minute (to protect against API rate limits and avoid service abuse)
```

**Solutions**:
1. Set `polling_interval` to `1m` or higher
2. Remove `polling_interval` configuration to use global default
3. Consider if you really need such frequent polling

## 📈 Performance Optimization Suggestions

### Alternative Approaches
1. **Webhook Integration**: Use Git provider webhook features for real-time triggers
2. **Smart Polling**: Dynamically adjust polling intervals based on repository activity
3. **Batch Processing**: Increase `batch_size` to improve efficiency

### Monitoring and Tuning
```yaml
# Monitoring configuration
monitoring:
  metrics_enabled: true
  
# Rate limiting configuration
rate_limit:
  github:
    requests_per_hour: 4000  # Leave headroom
  gitlab:
    requests_per_second: 8   # Leave headroom
```

## 💡 Best Practices

1. **Start with Larger Intervals**: Begin with 5 minutes, adjust as needed
2. **Monitor API Usage**: Regularly check API quota consumption
3. **Consider Business Needs**: Evaluate if real-time monitoring is truly necessary
4. **Use Tiered Configuration**: Shorter intervals for critical repos, longer for others

## 🔗 Related Documentation

- [API Usage Limits](API_LIMITS.md)
- [Rate Limiting Configuration](RATE_LIMITING.md)
- [Performance Optimization Guide](PERFORMANCE.md)


