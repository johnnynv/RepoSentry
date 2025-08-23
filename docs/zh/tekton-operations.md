# ⚡ RepoSentry CloudEvents 快速迁移命令

## 🚨 **一行命令完成迁移**

### **1. 备份现有配置**
```bash
# 备份现有Tekton配置 (推荐)
kubectl get eventlistener,triggerbinding,triggertemplate,pipeline -o yaml > backup-tekton-config.yaml
```

### **2. 部署新的CloudEvents标准系统**
```bash
# 一键部署CloudEvents标准配置
kubectl apply -f https://raw.githubusercontent.com/your-org/RepoSentry/main/deployments/tekton/reposentry-basic-system.yaml

# 或本地部署
kubectl apply -f deployments/tekton/reposentry-basic-system.yaml
```

### **3. 清理旧配置 (可选)**
```bash
# 删除旧的hello-*配置 (如果还存在)
kubectl delete eventlistener hello-event-listener --ignore-not-found
kubectl delete triggerbinding hello-trigger-binding --ignore-not-found  
kubectl delete triggertemplate hello-trigger-template --ignore-not-found
kubectl delete pipeline hello-pipeline --ignore-not-found

# 删除旧的reposentry配置 (如果使用了之前的版本)
kubectl delete eventlistener reposentry-webhook-handler --ignore-not-found
kubectl delete triggerbinding reposentry-advanced-binding --ignore-not-found
kubectl delete triggertemplate reposentry-advanced-template --ignore-not-found
```

### **4. 验证部署**
```bash
# 检查新配置状态
kubectl get eventlistener,triggerbinding,triggertemplate,pipeline -l reposentry.dev/format=cloudevents

# 检查EventListener是否就绪
kubectl get eventlistener reposentry-basic-eventlistener -o wide
```

### **5. 测试CloudEvents格式**
```bash
# 获取webhook URL
WEBHOOK_URL=$(kubectl get eventlistener reposentry-basic-eventlistener -o jsonpath='{.status.address.url}')

# 测试CloudEvents格式webhook
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "specversion": "1.0",
    "type": "dev.reposentry.repository.branch_updated", 
    "source": "reposentry/github",
    "id": "test-migration-123",
    "time": "2025-08-22T08:00:00Z",
    "datacontenttype": "application/json",
    "data": {
      "repository": {
        "provider": "github",
        "organization": "test-org",
        "name": "test-repo",
        "url": "https://github.com/test-org/test-repo"
      },
      "branch": {
        "name": "main"
      },
      "commit": {
        "sha": "abcd1234efgh5678",
        "message": "Test migration"
      },
      "event": {
        "type": "branch_updated",
        "trigger_id": "migration-test"
      }
    }
  }' \
  $WEBHOOK_URL
```

### **6. 验证结果**
```bash
# 查看最新PipelineRun
kubectl get pipelineruns --sort-by='.metadata.creationTimestamp' | tail -1

# 检查CloudEvents标签
kubectl get pipelinerun $(kubectl get pipelinerun --sort-by='.metadata.creationTimestamp' -o name | tail -1) -o yaml | grep -A 15 labels:
```

## 🔧 **故障排除快速命令**

### **检查EventListener状态**
```bash
# EventListener Pod日志
kubectl logs -l app.kubernetes.io/managed-by=EventListener

# EventListener服务状态  
kubectl get svc -l app.kubernetes.io/managed-by=EventListener
```

### **检查权限问题**
```bash
# 验证ServiceAccount
kubectl get serviceaccount tekton-triggers-serviceaccount

# 检查权限绑定
kubectl get clusterrolebinding | grep tekton-triggers
```

### **检查配置差异**
```bash
# 对比新旧TriggerBinding路径
echo "=== 旧格式路径 (已弃用) ==="
echo "$(body.metadata.provider) -> $(body.data.repository.provider)"
echo "$(body.metadata.organization) -> $(body.data.repository.organization)"

echo "=== 新格式路径 (CloudEvents) ==="
kubectl get triggerbinding reposentry-basic-binding -o yaml | grep -A 20 "params:"
```

## 📋 **迁移检查清单**

```bash
# 一键检查脚本
cat << 'EOF' > check-migration.sh
#!/bin/bash
echo "=== 🔍 RepoSentry CloudEvents 迁移检查 ==="
echo ""

echo "✅ 检查新配置部署状态:"
kubectl get eventlistener,triggerbinding,triggertemplate,pipeline -l reposentry.dev/format=cloudevents 2>/dev/null && echo "✅ 新配置已部署" || echo "❌ 新配置未部署"

echo ""
echo "✅ 检查EventListener就绪状态:"
kubectl get eventlistener reposentry-basic-eventlistener -o jsonpath='{.status.conditions[0].status}' 2>/dev/null | grep -q "True" && echo "✅ EventListener就绪" || echo "❌ EventListener未就绪"

echo ""
echo "✅ 检查最新PipelineRun格式:"
LATEST_PR=$(kubectl get pipelinerun --sort-by='.metadata.creationTimestamp' -o name 2>/dev/null | tail -1)
if [[ -n "$LATEST_PR" ]]; then
    kubectl get $LATEST_PR -o jsonpath='{.metadata.labels.reposentry\.dev/format}' 2>/dev/null | grep -q "cloudevents" && echo "✅ 最新PipelineRun使用CloudEvents格式" || echo "⚠️ 最新PipelineRun仍使用旧格式"
else
    echo "ℹ️ 暂无PipelineRun"
fi

echo ""
echo "✅ 检查旧配置清理状态:"
kubectl get eventlistener hello-event-listener 2>/dev/null >/dev/null && echo "⚠️ 发现旧hello配置，建议清理" || echo "✅ 旧hello配置已清理"

echo ""
echo "=== 🎯 迁移状态总结 ==="
echo "📖 完整指南: docs/zh/tekton-integration-guide.md"
echo "📦 模板文件: deployments/tekton/reposentry-basic-system.yaml"
EOF

chmod +x check-migration.sh
./check-migration.sh
```

## 🆘 **紧急回滚 (如果需要)**

```bash
# 恢复旧配置 (紧急情况)
kubectl apply -f backup-tekton-config.yaml

# 删除新配置
kubectl delete -f deployments/tekton/reposentry-basic-system.yaml
```

---

**🎉 恭喜！您已完成 CloudEvents 标准化迁移！**

📞 **需要帮助？** 查看详细指南：`docs/zh/tekton-integration-guide.md`


## 🔍 监控命令参考

```

### 2. 查看特定Pipeline的详细信息

```bash
# 查看Pipeline运行的详细信息
kubectl describe pipelinerun <pipeline-run-name> -n default

# 示例
kubectl describe pipelinerun hello-pipeline-run-2z6fl -n default
```

### 3. 查看Pipeline执行日志

```bash
# 查看特定Pipeline运行的日志
kubectl logs -l tekton.dev/pipelineRun=<pipeline-run-name> -n default

# 查看最新的日志（末尾几行）
kubectl logs -l tekton.dev/pipelineRun=<pipeline-run-name> -n default | tail -20

# 示例
kubectl logs -l tekton.dev/pipelineRun=hello-pipeline-run-2z6fl -n default
```

## 高级监控命令

### 4. 实时监控新的Pipeline运行

```bash
# 监控新创建的Pipeline运行
kubectl get pipelineruns -A -w

# 只监控特定namespace
kubectl get pipelineruns -n default -w
```

### 5. 查看Pipeline运行状态统计

```bash
# 查看成功/失败的Pipeline数量
kubectl get pipelineruns -A --no-headers | awk '{print $3}' | sort | uniq -c

# 查看最近24小时内的Pipeline运行
kubectl get pipelineruns -A --field-selector metadata.creationTimestamp>$(date -d '1 day ago' -u +%Y-%m-%dT%H:%M:%SZ)
```

### 6. 查看TaskRun详情

```bash
# 查看Pipeline中具体Task的执行情况
kubectl get taskruns -n default

# 查看特定TaskRun的详细信息
kubectl describe taskrun <taskrun-name> -n default
```

## 故障排查命令

### 7. 查看失败的Pipeline

```bash
# 查看失败的Pipeline运行
kubectl get pipelineruns -A --field-selector status.conditions[0].status=False

# 查看失败Pipeline的错误信息
kubectl describe pipelinerun <failed-pipeline-name> -n default
```

### 8. 查看EventListener状态

```bash
# 查看EventListener Pod状态
kubectl get pods -n default | grep eventlistener

# 查看EventListener日志
kubectl logs -l app.kubernetes.io/managed-by=EventListener -n default
```

## 快速检查脚本

### 检查最新Pipeline状态

```bash
#!/bin/bash
echo "=== 最新的Pipeline运行 ==="
kubectl get pipelineruns -A --sort-by=.metadata.creationTimestamp | tail -5

echo -e "\n=== 最新Pipeline的详细状态 ==="
LATEST_PIPELINE=$(kubectl get pipelineruns -n default --sort-by=.metadata.creationTimestamp --no-headers | tail -1 | awk '{print $1}')
if [ ! -z "$LATEST_PIPELINE" ]; then
    echo "检查Pipeline: $LATEST_PIPELINE"
    kubectl get pipelinerun $LATEST_PIPELINE -n default
    echo -e "\n=== Pipeline日志 ==="
    kubectl logs -l tekton.dev/pipelineRun=$LATEST_PIPELINE -n default | tail -10
fi
```

## RepoSentry集成监控

### 验证RepoSentry触发的Pipeline

```bash
# 查看由RepoSentry触发的Pipeline（通过labels识别）
kubectl get pipelineruns -n default -l triggers.tekton.dev/eventlistener=hello-event-listener

# 查看特定时间范围内的触发
kubectl get pipelineruns -n default --sort-by=.metadata.creationTimestamp | grep "$(date +%Y-%m-%d)"
```

### 完整的验证流程

```bash
# 1. 检查RepoSentry是否在运行
ps aux | grep reposentry

# 2. 检查最新的Pipeline运行
kubectl get pipelineruns -A --sort-by=.metadata.creationTimestamp | tail -3

# 3. 验证Pipeline参数（确认是否来自RepoSentry）
LATEST_PIPELINE=$(kubectl get pipelineruns -n default --sort-by=.metadata.creationTimestamp --no-headers | tail -1 | awk '{print $1}')
kubectl describe pipelinerun $LATEST_PIPELINE -n default | grep -A 10 "Params:"

# 4. 查看执行结果
kubectl logs -l tekton.dev/pipelineRun=$LATEST_PIPELINE -n default
```

## 注意事项

1. **权限要求**: 确保有足够的kubectl权限访问相关namespace
2. **命名空间**: 根据实际部署调整namespace（默认为`default`）
3. **时区**: 注意时间戳可能使用UTC时区
4. **资源清理**: 定期清理旧的Pipeline运行以避免资源占用

## Tekton Dashboard Web界面访问

### HTTPS访问（推荐）

```
URL: https://tekton.10.78.14.61.nip.io
用户名: admin
密码: admin123
```

### 访问说明

1. **使用HTTPS**: 支持SSL/TLS加密访问，端口443
2. **Basic Auth认证**: 需要输入用户名和密码
3. **如果Dashboard加载缓慢**: 使用本文档的命令行方式更可靠
4. **替代访问方式**: 
   - NodePort: `http://tekton.10.78.14.61.nip.io:30097` (不加密)
   - 命令行监控: 使用上述kubectl命令（最可靠）

### 浏览器访问步骤

1. 打开浏览器，访问 `https://tekton.10.78.14.61.nip.io`
2. 忽略SSL证书警告（如果出现）
3. 输入认证信息：
   - 用户名: `admin`
   - 密码: `admin123`
4. 进入Dashboard查看Pipeline运行状态

### Dashboard故障排查

如果Dashboard显示"Loading configuration..."且无法正常加载，可以使用以下方法：

#### 自动故障排查脚本

```bash
# 运行故障排查脚本
./scripts/dashboard-troubleshoot.sh
```

#### 手动修复步骤

```bash
# 1. 重启Dashboard Pod
kubectl rollout restart deployment/tekton-dashboard -n tekton-pipelines

# 2. 等待重启完成
kubectl rollout status deployment/tekton-dashboard -n tekton-pipelines

# 3. 测试访问
curl -k -u admin:admin123 -I https://tekton.10.78.14.61.nip.io
```

#### 常见问题和解决方案

1. **Dashboard卡在加载界面**
   - 原因：Dashboard Pod初始化问题
   - 解决：重启Dashboard deployment

2. **认证失败**
   - 检查用户名密码是否正确：`admin / admin123`
   - 检查Basic Auth配置：`kubectl get secret tekton-basic-auth -n tekton-pipelines`

3. **网络连接问题**
   - 检查Ingress状态：`kubectl get ingress -n tekton-pipelines`
   - 检查Nginx Controller：`kubectl get pods -n ingress-nginx`

## 相关文档

- [RepoSentry用户手册](user-manual.md)
- [故障排查指南](troubleshooting.md)
- [API使用示例](api-examples.md)
