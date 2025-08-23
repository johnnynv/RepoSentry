# RepoSentry 轮询限制说明

## 📊 轮询间隔限制

### 最小轮询间隔：1分钟

RepoSentry 强制要求**轮询间隔不能小于1分钟**，这是出于以下重要原因：

## 🛡️ 为什么有这个限制？

### 1. API 速率限制保护
- **GitHub API 限制**：每小时 5,000 次请求
- **GitLab API 限制**：每秒 10 次请求
- **频繁轮询风险**：30秒轮询会在1小时内消耗 120 次请求，很快耗尽配额

### 2. 避免服务滥用
- 保护 Git 服务提供商的基础设施
- 避免被标记为滥用行为
- 防止账号被限制或封禁

### 3. 实际使用场景考虑
- 代码变更通常不会每秒发生
- 1分钟延迟对大多数 CI/CD 场景是可接受的
- 更频繁的轮询通常没有实际价值

## ⚙️ 配置示例

### ✅ 正确配置
```yaml
# 全局轮询配置
polling:
  interval: "5m"  # 推荐：5分钟

# 仓库级别配置
repositories:
  - name: "my-repo"
    polling_interval: "1m"  # 最小：1分钟
```

### ❌ 错误配置
```yaml
# 这些配置会导致验证失败
polling:
  interval: "30s"  # ❌ 小于1分钟

repositories:
  - name: "my-repo"
    polling_interval: "45s"  # ❌ 小于1分钟
```

## 🔧 配置建议

### 生产环境推荐配置
- **活跃项目**：1-2分钟
- **普通项目**：5分钟
- **归档项目**：15-30分钟

### 开发环境
- **最小间隔**：1分钟（测试用）
- **推荐间隔**：2-3分钟

## 🚨 错误消息说明

当您看到以下错误时：
```
validation error for field 'repositories[0].polling_interval': 
polling interval cannot be less than 1 minute (to protect against API rate limits and avoid service abuse)
```

**解决方法**：
1. 将 `polling_interval` 设置为 `1m` 或更大值
2. 删除 `polling_interval` 配置，使用全局默认值
3. 考虑实际需求，是否真的需要如此频繁的轮询

## 📈 性能优化建议

### 替代方案
1. **Webhook 集成**：使用 Git 提供商的 webhook 功能实现实时触发
2. **智能轮询**：根据仓库活跃度动态调整轮询间隔
3. **批量处理**：增加 `batch_size` 来提高效率

### 监控和调优
```yaml
# 监控配置
monitoring:
  metrics_enabled: true
  
# 速率限制配置
rate_limit:
  github:
    requests_per_hour: 4000  # 留有余量
  gitlab:
    requests_per_second: 8   # 留有余量
```

## 💡 最佳实践

1. **从较大间隔开始**：先使用5分钟，根据需要调整
2. **监控 API 使用量**：定期检查 API 配额使用情况
3. **考虑业务需求**：评估是否真的需要实时监控
4. **使用分层配置**：重要仓库用较短间隔，其他仓库用较长间隔

## 🔗 相关文档

- [API 使用限制](API_LIMITS.md)
- [速率限制配置](RATE_LIMITING.md)
- [性能优化指南](PERFORMANCE.md)


