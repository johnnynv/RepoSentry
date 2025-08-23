# RepoSentry 故障排除指南

## 🔍 快速诊断

### 健康检查清单

在遇到问题时，请按以下顺序检查：

```bash
# 1. 检查服务状态
reposentry status

# 2. 检查配置文件
reposentry config validate config.yaml --check-env --check-connectivity

# 3. 检查健康接口
curl http://localhost:8080/health

# 4. 查看日志
tail -f /var/log/reposentry.log
# 或者对于 systemd
sudo journalctl -u reposentry -f
```

## 🚨 常见问题

### 1. 启动失败

#### 症状：服务无法启动
```bash
Error: failed to start RepoSentry: configuration validation failed
```

#### 排查步骤：

**检查配置文件语法**
```bash
# 验证 YAML 语法
reposentry config validate config.yaml

# 常见错误：缩进不正确、字段名拼写错误
# 使用 YAML 在线验证器检查语法
```

**检查必填字段**
```bash
# 验证必填字段
reposentry config validate config.yaml --verbose

# 确保以下字段已配置：
# - tekton.event_listener_url
# - repositories[].name
# - repositories[].url  
# - repositories[].provider
# - repositories[].token
# - repositories[].branch_regex
```

**检查环境变量**
```bash
# 验证环境变量
echo $GITHUB_TOKEN
echo $GITLAB_TOKEN

# 检查环境变量展开
reposentry config show --config=config.yaml
```

#### 解决方案：
1. 修复配置文件语法错误
2. 补充缺失的必填字段
3. 设置正确的环境变量

### 2. 权限问题

#### 症状：API 调用被拒绝
```
Error: failed to fetch branches: 401 Unauthorized
```

#### 排查步骤：

**测试 GitHub Token**
```bash
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/user

# 成功响应应包含用户信息
# 错误响应：{"message": "Bad credentials"}
```

**测试 GitLab Token**
```bash
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/user

# 企业版 GitLab
curl -H "PRIVATE-TOKEN: $GITLAB_ENTERPRISE_TOKEN" \
  https://gitlab-master.nvidia.com/api/v4/user
```

**检查仓库访问权限**
```bash
# GitHub 仓库权限
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/owner/repository

# GitLab 项目权限  
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/projects/owner%2Frepository
```

#### 解决方案：
1. **Token 过期**: 重新生成 API Token
2. **权限不足**: 确保 Token 有仓库读取权限
3. **Token 格式错误**: 检查 Token 前缀（GitHub: ghp_, GitLab: glpat-）

### 3. 网络连接问题

#### 症状：无法连接到外部服务
```
Error: dial tcp: lookup github.com: no such host
Error: context deadline exceeded
```

#### 排查步骤：

**DNS 解析测试**
```bash
# 测试 DNS 解析
nslookup github.com
nslookup gitlab.com
nslookup your-tekton-listener.com

# 测试自定义 DNS
dig @8.8.8.8 github.com
```

**网络连接测试**
```bash
# 测试 HTTPS 连接
curl -I https://api.github.com
curl -I https://gitlab.com/api/v4

# 测试 Tekton EventListener
curl -X POST -H "Content-Type: application/json" \
  -d '{"test": true}' \
  $TEKTON_EVENTLISTENER_URL
```

**防火墙检查**
```bash
# 检查防火墙状态
sudo ufw status
sudo iptables -L

# 检查端口开放
sudo netstat -tulpn | grep :8080
ss -tulpn | grep :8080
```

#### 解决方案：
1. **DNS 问题**: 配置正确的 DNS 服务器
2. **防火墙拦截**: 开放必要的出站端口 (80, 443, 8080)
3. **代理配置**: 配置 HTTP_PROXY 和 HTTPS_PROXY
4. **网络策略**: 检查 Kubernetes NetworkPolicy

### 4. 数据库问题

#### 症状：数据库操作失败
```
Error: failed to initialize storage: database is locked
Error: no such table: repository_states
```

#### 排查步骤：

**检查数据库文件**
```bash
# 检查数据库文件权限
ls -la ./data/reposentry.db

# 检查目录权限
ls -la ./data/

# 检查磁盘空间
df -h ./data/
```

**数据库完整性检查**
```bash
# SQLite 完整性检查
sqlite3 ./data/reposentry.db "PRAGMA integrity_check;"

# 检查表结构
sqlite3 ./data/reposentry.db ".schema"

# 检查迁移状态
sqlite3 ./data/reposentry.db "SELECT * FROM schema_migrations;"
```

#### 解决方案：
1. **权限问题**: `chmod 755 ./data && chmod 644 ./data/reposentry.db`
2. **磁盘空间不足**: 清理磁盘空间
3. **数据库损坏**: 删除数据库文件，重新初始化
4. **多实例冲突**: 确保只有一个实例访问数据库

### 5. 轮询问题

#### 症状：轮询不工作或频率异常
```
Warning: polling cycle took 5m30s, expected 5m
Error: no events generated in last 2 hours
```

#### 排查步骤：

**检查轮询状态**
```bash
# 查看轮询指标
curl http://localhost:8080/api/v1/metrics | jq '.data.polling'

# 查看仓库状态
reposentry repo list

# 检查最近事件
curl http://localhost:8080/api/v1/events/recent
```

**分析轮询日志**
```bash
# 过滤轮询相关日志
grep '"component":"poller"' /var/log/reposentry.log | tail -20

# 查看错误日志
grep '"level":"error"' /var/log/reposentry.log | grep poller
```

#### 解决方案：
1. **API 限制**: 增加轮询间隔，检查 API 配额
2. **性能问题**: 调整 `max_workers` 和 `batch_size`
3. **分支过滤**: 检查 `branch_regex` 是否正确
4. **缓存问题**: 清理数据库或重启服务

### 6. Tekton 集成问题

#### 症状：事件未触发 Tekton 流水线
```
Error: failed to send webhook: connection refused
Warning: webhook sent but no pipeline triggered
```

#### 排查步骤：

**测试 EventListener 连接**
```bash
# 测试 EventListener 健康状态
curl http://tekton-listener:8080/health

# 手动发送测试事件
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

**检查 Tekton 配置**
```bash
# 检查 EventListener
kubectl get eventlistener -A

# 检查 TriggerBinding
kubectl get triggerbinding -A

# 检查 TriggerTemplate  
kubectl get triggertemplate -A

# 查看 EventListener 日志
kubectl logs -l app=el-github-listener -n tekton-pipelines
```

#### 解决方案：
1. **URL 错误**: 检查 `tekton.event_listener_url` 配置
2. **网络不通**: 检查 Kubernetes 网络策略和服务发现
3. **Payload 格式**: 确认 Tekton 期望的 payload 格式
4. **权限问题**: 检查 Tekton 的 RBAC 配置

## 🛠️ 日志分析

### 启用详细日志

```yaml
# config.yaml
app:
  log_level: "debug"  # 启用详细日志
  log_format: "json"  # 便于分析
```

### 日志过滤技巧

```bash
# 按组件过滤
grep '"component":"poller"' /var/log/reposentry.log
grep '"component":"trigger"' /var/log/reposentry.log  
grep '"component":"gitclient"' /var/log/reposentry.log

# 按日志级别过滤
grep '"level":"error"' /var/log/reposentry.log
grep '"level":"warn"' /var/log/reposentry.log

# 按时间范围过滤
grep '"timestamp":"2024-01-15T1[0-2]"' /var/log/reposentry.log

# 按操作过滤
grep '"operation":"fetch_branches"' /var/log/reposentry.log
grep '"operation":"send_webhook"' /var/log/reposentry.log

# 按仓库过滤
grep '"repository":"my-repo"' /var/log/reposentry.log
```

### 关键日志模式

```bash
# 成功模式
grep '"message":"Successfully"' /var/log/reposentry.log

# 错误模式
grep '"error"' /var/log/reposentry.log | jq -r '.error'

# 性能监控
grep '"duration"' /var/log/reposentry.log | jq '.duration'

# API 调用监控
grep '"api_rate_remaining"' /var/log/reposentry.log
```

## 🔧 性能问题诊断

### 内存使用过高

**检查内存使用**
```bash
# 系统内存
free -h

# 进程内存
ps aux | grep reposentry

# 容器内存（Docker）
docker stats reposentry

# Pod 内存（Kubernetes）
kubectl top pod -l app=reposentry
```

**优化配置**
```yaml
polling:
  max_workers: 5      # 降低并发数
  batch_size: 10      # 减少批处理大小
  interval: "10m"     # 增加轮询间隔
```

### CPU 使用过高

**分析 CPU 使用**
```bash
# 系统 CPU
top -p $(pgrep reposentry)

# 详细 CPU 分析
pidstat -p $(pgrep reposentry) 1

# Go 性能分析
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

**优化策略**
1. 增加轮询间隔
2. 减少并发协程数
3. 优化分支正则表达式
4. 启用缓存机制

### 磁盘 I/O 过高

**检查磁盘使用**
```bash
# 磁盘 I/O
iotop -p $(pgrep reposentry)

# 数据库大小
du -sh ./data/reposentry.db

# 日志文件大小
du -sh /var/log/reposentry.log
```

**优化配置**
```yaml
app:
  log_file_rotation:
    max_size: 50        # 减少日志文件大小
    max_backups: 3      # 减少备份文件数
```

## 🚀 恢复程序

### 服务恢复

```bash
# 1. 停止服务
sudo systemctl stop reposentry

# 2. 备份当前配置和数据
cp config.yaml config.yaml.backup
cp -r ./data ./data.backup

# 3. 重置配置（如果需要）
reposentry config init --type=basic > config.yaml.new

# 4. 验证配置
reposentry config validate config.yaml.new

# 5. 重启服务
sudo systemctl start reposentry

# 6. 验证运行状态
reposentry status
```

### 数据库恢复

```bash
# 1. 停止服务
sudo systemctl stop reposentry

# 2. 备份损坏的数据库
mv ./data/reposentry.db ./data/reposentry.db.corrupted

# 3. 如果有备份，恢复备份
cp ./data/reposentry.db.backup ./data/reposentry.db

# 4. 如果没有备份，重新初始化
rm -f ./data/reposentry.db

# 5. 重启服务（会自动创建新数据库）
sudo systemctl start reposentry

# 6. 验证数据库
sqlite3 ./data/reposentry.db ".tables"
```

### 完全重置

```bash
# 警告：这将删除所有数据和配置

# 1. 停止服务
sudo systemctl stop reposentry

# 2. 备份重要配置
cp config.yaml config.yaml.emergency.backup

# 3. 删除所有数据
rm -rf ./data
rm -f /var/log/reposentry.log*

# 4. 重新生成配置
reposentry config init --type=basic > config.yaml

# 5. 编辑配置文件
vim config.yaml

# 6. 设置环境变量
export GITHUB_TOKEN="your_token"
export GITLAB_TOKEN="your_token"

# 7. 验证配置
reposentry config validate config.yaml --check-env

# 8. 启动服务
sudo systemctl start reposentry
```

## 📞 获取帮助

### 社区支持

1. **GitHub Issues**: https://github.com/johnnynv/RepoSentry/issues
2. **讨论区**: https://github.com/johnnynv/RepoSentry/discussions
3. **文档站点**: https://reposentry.docs.example.com

### 报告问题

提交问题时请包含：

1. **RepoSentry 版本**: `reposentry version`
2. **操作系统**: `uname -a`
3. **配置文件**: 脱敏后的配置文件
4. **错误日志**: 相关的错误日志
5. **复现步骤**: 详细的复现步骤

### 日志收集脚本

```bash
#!/bin/bash
# 生成诊断报告

echo "=== RepoSentry 诊断报告 ===" > diagnostic.txt
echo "生成时间: $(date)" >> diagnostic.txt
echo "" >> diagnostic.txt

echo "=== 版本信息 ===" >> diagnostic.txt
reposentry version >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== 系统信息 ===" >> diagnostic.txt
uname -a >> diagnostic.txt
cat /etc/os-release >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== 配置验证 ===" >> diagnostic.txt
reposentry config validate config.yaml >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== 健康检查 ===" >> diagnostic.txt
curl -s http://localhost:8080/health >> diagnostic.txt 2>&1
echo "" >> diagnostic.txt

echo "=== 最近日志 ===" >> diagnostic.txt
tail -50 /var/log/reposentry.log >> diagnostic.txt 2>&1

echo "诊断报告已生成: diagnostic.txt"
```

---

如果以上方法都无法解决问题，请提交详细的问题报告到 GitHub Issues。
