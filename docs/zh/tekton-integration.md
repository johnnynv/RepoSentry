# 🚨 **重要变更通知** - RepoSentry Webhook Payload 标准化

## ⚠️ **破坏性变更警告**

**RepoSentry v2.0+ 已采用基于 CloudEvents 1.0 的标准化 webhook payload 格式**

### 🔄 **变更影响**
- **所有现有的 TriggerBinding 配置需要更新**
- **JSONPath 路径已标准化**
- **新格式提供更好的兼容性和扩展性**

### 📅 **迁移时间线**
- ✅ **v2.0+**: 新标准格式生效
- 🔄 **兼容期**: 提供迁移指导和工具
- ❌ **弃用旧格式**: 计划在下一个主版本移除

---

# Tekton 集成指南 - CloudEvents 标准格式

## 📋 **快速迁移检查表**

### ✅ **必须更新的配置**
- [ ] **TriggerBinding**: 更新 JSONPath 路径
- [ ] **TriggerTemplate**: 验证参数映射
- [ ] **Pipeline**: 确认参数接收正确
- [ ] **测试**: 验证端到端流程

### ✅ **推荐更新的配置**
- [ ] **监控**: 更新基于 CloudEvents 的监控
- [ ] **日志**: 利用标准化字段改进日志
- [ ] **标签**: 使用新的丰富元数据

---

## 🎯 **新 Payload 格式概览**

### **旧格式 (已弃用)**
```json
{
  "metadata": {
    "provider": "github",
    "organization": "johnnynv"
  },
  "repository": {...},
  "ref": "refs/heads/main"
}
```

### **新格式 (CloudEvents 标准)**
```json
{
  "specversion": "1.0",
  "type": "dev.reposentry.repository.branch_updated",
  "source": "reposentry/github", 
  "id": "event_37533c6d_20250822_073039",
  "time": "2025-08-22T07:30:39.306Z",
  "datacontenttype": "application/json",
  "data": {
    "repository": {
      "provider": "github",
      "organization": "johnnynv",
      "name": "TaaP_POC"
    },
    "branch": {
      "name": "main"
    },
    "commit": {
      "sha": "37533c6d...",
      "short_sha": "37533c6d"
    },
    "event": {
      "type": "branch_updated"
    }
  }
}
```

---

## 🔧 **JSONPath 路径对照表**

| 字段 | 旧路径 (已弃用) | 新路径 (CloudEvents) |
|------|----------------|---------------------|
| **Provider** | `$(body.metadata.provider)` | `$(body.data.repository.provider)` |
| **Organization** | `$(body.metadata.organization)` | `$(body.data.repository.organization)` |
| **Repository** | `$(body.metadata.repository_name)` | `$(body.data.repository.name)` |
| **Branch** | `$(body.metadata.branch)` | `$(body.data.branch.name)` |
| **Commit SHA** | `$(body.metadata.commit_sha)` | `$(body.data.commit.sha)` |
| **Short SHA** | `$(body.metadata.short_sha)` | `$(body.data.commit.short_sha)` |
| **Event Type** | `$(body.metadata.event_type)` | `$(body.data.event.type)` |
| **Event ID** | `$(body.event_id)` | `$(body.id)` |
| **Timestamp** | `$(body.metadata.detection_time)` | `$(body.time)` |

### 🆕 **新增的 CloudEvents 字段**
| 字段 | 路径 | 描述 |
|------|------|------|
| **Spec Version** | `$(body.specversion)` | CloudEvents 规范版本 |
| **Event Source** | `$(body.source)` | 事件源标识 |
| **Content Type** | `$(body.datacontenttype)` | 数据内容类型 |
| **Full Event Type** | `$(body.type)` | 完整的事件类型 |

---

## 📄 **完整模板文件**

### 1. **标准 TriggerBinding**

```yaml
# reposentry-basic-triggerbinding.yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerBinding
metadata:
  name: reposentry-basic-binding
  namespace: default
  labels:
    app.kubernetes.io/name: reposentry
    app.kubernetes.io/component: webhook
spec:
  params:
    # === 核心仓库信息 ===
    - name: provider
      value: $(body.data.repository.provider)
    - name: organization
      value: $(body.data.repository.organization)
    - name: repository-name
      value: $(body.data.repository.name)
    - name: repository-full-name
      value: $(body.data.repository.full_name)
    - name: repository-url
      value: $(body.data.repository.url)
    - name: repository-id
      value: $(body.data.repository.id)
    
    # === 分支和提交信息 ===
    - name: branch-name
      value: $(body.data.branch.name)
    - name: branch-ref
      value: $(body.data.branch.ref)
    - name: commit-sha
      value: $(body.data.commit.sha)
    - name: commit-short-sha
      value: $(body.data.commit.short_sha)
    - name: commit-message
      value: $(body.data.commit.message)
    
    # === 事件信息 ===
    - name: event-type
      value: $(body.data.event.type)
    - name: trigger-source
      value: $(body.data.event.trigger_source)
    - name: trigger-id
      value: $(body.data.event.trigger_id)
    
    # === CloudEvents 标准字段 ===
    - name: event-id
      value: $(body.id)
    - name: event-time
      value: $(body.time)
    - name: event-source
      value: $(body.source)
    - name: spec-version
      value: $(body.specversion)
    - name: content-type
      value: $(body.datacontenttype)
    - name: full-event-type
      value: $(body.type)
```

### 2. **标准 TriggerTemplate**

```yaml
# reposentry-basic-triggertemplate.yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
metadata:
  name: reposentry-basic-template
  namespace: default
  labels:
    app.kubernetes.io/name: reposentry
    app.kubernetes.io/component: webhook
spec:
  params:
    # === 核心参数 ===
    - name: provider
      description: "Git provider (github/gitlab)"
    - name: organization
      description: "Repository organization/owner"
    - name: repository-name
      description: "Repository name"
    - name: branch-name
      description: "Git branch name"
    - name: commit-sha
      description: "Full Git commit SHA"
    - name: commit-short-sha
      description: "Short Git commit SHA"
    - name: event-type
      description: "Event type (branch_updated/branch_created/branch_deleted)"
    
    # === 扩展参数 ===
    - name: repository-full-name
      description: "Full repository name (org/repo)"
    - name: repository-url
      description: "Repository URL"
    - name: repository-id
      description: "Repository unique identifier"
    - name: branch-ref
      description: "Full branch reference"
    - name: commit-message
      description: "Commit message"
    - name: trigger-source
      description: "Trigger source system"
    - name: trigger-id
      description: "Unique trigger identifier"
    - name: event-id
      description: "CloudEvents event ID"
    - name: event-time
      description: "CloudEvents event timestamp"
    - name: event-source
      description: "CloudEvents event source"
    
  resourcetemplates:
    - apiVersion: tekton.dev/v1beta1
      kind: PipelineRun
      metadata:
        generateName: "reposentry-$(tt.params.provider)-$(tt.params.organization)-"
        labels:
          # === RepoSentry 标准标签 ===
          reposentry.dev/provider: $(tt.params.provider)
          reposentry.dev/organization: $(tt.params.organization)
          reposentry.dev/repository: $(tt.params.repository-name)
          reposentry.dev/branch: $(tt.params.branch-name)
          reposentry.dev/event-type: $(tt.params.event-type)
          reposentry.dev/commit-sha: $(tt.params.commit-short-sha)
          reposentry.dev/trigger-source: $(tt.params.trigger-source)
          reposentry.dev/trigger-id: $(tt.params.trigger-id)
          
          # === CloudEvents 标准标签 ===
          cloudevents.io/event-id: $(tt.params.event-id)
          cloudevents.io/event-source: $(tt.params.event-source)
          cloudevents.io/spec-version: $(tt.params.spec-version)
          
          # === Tekton 标准标签 ===
          tekton.dev/pipeline: reposentry-demo-pipeline
          
        annotations:
          reposentry.dev/repository-url: $(tt.params.repository-url)
          reposentry.dev/commit-message: $(tt.params.commit-message)
          reposentry.dev/event-time: $(tt.params.event-time)
          
      spec:
        pipelineRef:
          name: reposentry-demo-pipeline
        params:
          - name: provider
            value: $(tt.params.provider)
          - name: organization
            value: $(tt.params.organization)
          - name: repository-name
            value: $(tt.params.repository-name)
          - name: repository-url
            value: $(tt.params.repository-url)
          - name: branch-name
            value: $(tt.params.branch-name)
          - name: commit-sha
            value: $(tt.params.commit-sha)
          - name: commit-message
            value: $(tt.params.commit-message)
          - name: event-type
            value: $(tt.params.event-type)
          - name: trigger-id
            value: $(tt.params.trigger-id)
```

### 3. **标准 Pipeline**

```yaml
# reposentry-basic-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reposentry-demo-pipeline
  namespace: default
  labels:
    app.kubernetes.io/name: reposentry
    app.kubernetes.io/component: pipeline
spec:
  params:
    # === 必需参数 ===
    - name: provider
      type: string
      description: "Git provider (github/gitlab)"
    - name: organization
      type: string
      description: "Repository organization/owner"
    - name: repository-name
      type: string
      description: "Repository name"
    - name: repository-url
      type: string
      description: "Repository URL"
    - name: branch-name
      type: string
      description: "Git branch name"
    - name: commit-sha
      type: string
      description: "Git commit SHA"
    - name: commit-message
      type: string
      description: "Commit message"
    - name: event-type
      type: string
      description: "Event type"
    - name: trigger-id
      type: string
      description: "Trigger identifier"
    
  tasks:
    - name: display-event-info
      taskSpec:
        params:
          - name: provider
          - name: organization
          - name: repository-name
          - name: repository-url
          - name: branch-name
          - name: commit-sha
          - name: commit-message
          - name: event-type
          - name: trigger-id
        steps:
          - name: display-info
            image: alpine:latest
            script: |
              #!/bin/sh
              echo "=== 🚀 RepoSentry CloudEvents CI Pipeline ==="
              echo ""
              echo "📍 Repository Information:"
              echo "  Provider: $(params.provider)"
              echo "  Organization: $(params.organization)"
              echo "  Repository: $(params.repository-name)"
              echo "  URL: $(params.repository-url)"
              echo ""
              echo "🌿 Branch & Commit Information:"
              echo "  Branch: $(params.branch-name)"
              echo "  Commit SHA: $(params.commit-sha)"
              echo "  Commit Message: $(params.commit-message)"
              echo ""
              echo "⚡ Event Information:"
              echo "  Event Type: $(params.event-type)"
              echo "  Trigger ID: $(params.trigger-id)"
              echo ""
              echo "✅ CloudEvents standard format detected!"
              echo "✅ All parameters successfully extracted!"
              echo ""
              echo "🎯 Ready for CI/CD processing..."
      params:
        - name: provider
          value: $(params.provider)
        - name: organization
          value: $(params.organization)
        - name: repository-name
          value: $(params.repository-name)
        - name: repository-url
          value: $(params.repository-url)
        - name: branch-name
          value: $(params.branch-name)
        - name: commit-sha
          value: $(params.commit-sha)
        - name: commit-message
          value: $(params.commit-message)
        - name: event-type
          value: $(params.event-type)
        - name: trigger-id
          value: $(params.trigger-id)
    
    # 在这里添加您的自定义任务
    # - name: your-custom-task
    #   taskRef:
    #     name: your-task
    #   params:
    #     - name: repo-url
    #       value: $(params.repository-url)
```

### 4. **标准 EventListener**

```yaml
# reposentry-basic-eventlistener.yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: reposentry-basic-eventlistener
  namespace: default
  labels:
    app.kubernetes.io/name: reposentry
    app.kubernetes.io/component: webhook
spec:
  serviceAccountName: tekton-triggers-serviceaccount
  triggers:
    - name: reposentry-cloudevents-trigger
      bindings:
        - ref: reposentry-basic-binding
      template:
        ref: reposentry-basic-template
      interceptors:
        # 可选：添加验证拦截器
        - name: "validate-cloudevents"
          params:
            - name: "filter"
              value: "body.specversion == '1.0' && body.source.startsWith('reposentry/')"
```

---

## 🚀 **部署和测试指南**

### **1. 部署新配置**

```bash
# 应用所有标准配置
kubectl apply -f reposentry-basic-pipeline.yaml
kubectl apply -f reposentry-basic-triggerbinding.yaml  
kubectl apply -f reposentry-basic-triggertemplate.yaml
kubectl apply -f reposentry-basic-eventlistener.yaml

# 验证部署
kubectl get eventlistener,triggerbinding,triggertemplate,pipeline -l app.kubernetes.io/name=reposentry
```

### **2. 测试新格式**

```bash
# 测试 CloudEvents 格式的 webhook
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "specversion": "1.0",
    "type": "dev.reposentry.repository.branch_updated",
    "source": "reposentry/github",
    "id": "test-event-123",
    "time": "2025-08-22T07:30:39Z",
    "datacontenttype": "application/json",
    "data": {
      "repository": {
        "provider": "github",
        "organization": "test-org",
        "name": "test-repo",
        "full_name": "test-org/test-repo",
        "url": "https://github.com/test-org/test-repo"
      },
      "branch": {
        "name": "main",
        "ref": "refs/heads/main"
      },
      "commit": {
        "sha": "abc123def456",
        "short_sha": "abc123de"
      },
      "event": {
        "type": "branch_updated",
        "trigger_source": "reposentry"
      }
    }
  }' \
  http://your-eventlistener-url/
```

### **3. 验证结果**

```bash
# 查看新创建的 PipelineRun
kubectl get pipelineruns --sort-by='.metadata.creationTimestamp' | tail -1

# 检查标签是否正确设置
kubectl get pipelinerun <newest-pipelinerun-name> -o yaml | grep -A 20 labels:

# 查看 Pipeline 执行日志
kubectl logs -l tekton.dev/pipelineRun=<newest-pipelinerun-name>
```

---

## 🔍 **故障排除**

### **常见问题**

#### **1. "JSONPath not found" 错误**
- **原因**: 使用了旧的路径格式
- **解决**: 检查上面的路径对照表，更新所有 `$(body.metadata.*)` 为 `$(body.data.*)`

#### **2. PipelineRun 未创建**
- **检查**: EventListener 日志中的错误信息
- **验证**: payload 格式是否符合 CloudEvents 标准

#### **3. 参数为空**
- **检查**: TriggerBinding 中的 JSONPath 是否正确
- **验证**: 发送的 payload 中是否包含所需字段

### **调试命令**

```bash
# 查看 EventListener 日志
kubectl logs -l app.kubernetes.io/managed-by=EventListener

# 查看最新 PipelineRun 详情  
kubectl describe pipelinerun $(kubectl get pipelinerun --sort-by='.metadata.creationTimestamp' -o name | tail -1)

# 检查 TriggerBinding 配置
kubectl get triggerbinding reposentry-basic-binding -o yaml
```

---

## 📞 **支持和反馈**

### **需要帮助？**
- 📧 **技术支持**: 联系 RepoSentry 团队
- 📖 **文档**: 参考 `docs/zh/webhook-payload-standard.md`
- 🐛 **Bug报告**: 通过 Issue 系统提交

### **迁移支持**
我们提供迁移支持工具和指导，帮助您从旧格式平滑过渡到新的 CloudEvents 标准格式。

---

**🎉 欢迎来到 CloudEvents 标准化的 RepoSentry 2.0+ 时代！**


## 📋 Webhook Payload 标准

    // 事件具体数据
  }
}
```

### Data 字段详细结构

```json
{
  "data": {
    "repository": {
      "provider": "github",
      "organization": "johnnynv",
      "name": "TaaP_POC",
      "full_name": "johnnynv/TaaP_POC",
      "url": "https://github.com/johnnynv/TaaP_POC",
      "id": "github-johnnynv-taap-poc"
    },
    "branch": {
      "name": "main",
      "previous_commit": "abc123",
      "current_commit": "def456"
    },
    "commit": {
      "sha": "def456",
      "short_sha": "def456",
      "message": "Update documentation",
      "author": {
        "name": "Developer Name",
        "email": "dev@example.com"
      },
      "timestamp": "2023-12-01T09:55:00Z"
    },
    "event": {
      "type": "branch_updated",
      "trigger": "polling",
      "detected_at": "2023-12-01T10:00:00Z"
    }
  }
}
```

## 字段说明

### CloudEvents 标准字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `specversion` | string | ✅ | CloudEvents 规范版本 (固定为 "1.0") |
| `type` | string | ✅ | 事件类型 (com.reposentry.repository.branch.updated) |
| `source` | string | ✅ | 事件源 (固定为 "reposentry") |
| `id` | string | ✅ | 事件唯一标识符 |
| `time` | string | ✅ | 事件发生时间 (RFC3339 格式) |
| `datacontenttype` | string | ✅ | 数据内容类型 (application/json) |

### Data 字段说明

#### Repository 对象
| 字段 | 类型 | 说明 |
|------|------|------|
| `provider` | string | Git 提供商 (github/gitlab) |
| `organization` | string | 组织/用户名 |
| `name` | string | 仓库名称 |
| `full_name` | string | 完整仓库名 (organization/name) |
| `url` | string | 仓库完整 URL |
| `id` | string | 仓库唯一标识符 |

#### Branch 对象
| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 分支名称 |
| `previous_commit` | string | 上一次提交 SHA |
| `current_commit` | string | 当前提交 SHA |

#### Commit 对象
| 字段 | 类型 | 说明 |
|------|------|------|
| `sha` | string | 完整提交 SHA |
| `short_sha` | string | 短提交 SHA |
| `message` | string | 提交消息 |
| `author.name` | string | 作者姓名 |
| `author.email` | string | 作者邮箱 |
| `timestamp` | string | 提交时间 |

#### Event 对象
| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 事件类型 (branch_updated) |
| `trigger` | string | 触发方式 (polling/webhook) |
| `detected_at` | string | 检测时间 |

## 示例 Payload

### GitHub 仓库更新事件

```json
{
  "specversion": "1.0",
  "type": "com.reposentry.repository.branch.updated",
  "source": "reposentry",
  "id": "evt_2023120110001234",
  "time": "2023-12-01T10:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "repository": {
      "provider": "github",
      "organization": "johnnynv",
      "name": "TaaP_POC",
      "full_name": "johnnynv/TaaP_POC",
      "url": "https://github.com/johnnynv/TaaP_POC",
      "id": "github-johnnynv-taap-poc"
    },
    "branch": {
      "name": "main",
      "previous_commit": "a1b2c3d4e5f6",
      "current_commit": "f6e5d4c3b2a1"
    },
    "commit": {
      "sha": "f6e5d4c3b2a1",
      "short_sha": "f6e5d4c",
      "message": "feat: add new feature implementation",
      "author": {
        "name": "John Developer",
        "email": "john@example.com"
      },
      "timestamp": "2023-12-01T09:55:00Z"
    },
    "event": {
      "type": "branch_updated",
      "trigger": "polling",
      "detected_at": "2023-12-01T10:00:00Z"
    }
  }
}
```

### GitLab 仓库更新事件

```json
{
  "specversion": "1.0",
  "type": "com.reposentry.repository.branch.updated",
  "source": "reposentry",
  "id": "evt_2023120110001235",
  "time": "2023-12-01T10:05:00Z",
  "datacontenttype": "application/json",
  "data": {
    "repository": {
      "provider": "gitlab",
      "organization": "johnnyj",
      "name": "taap_poc_gitlab",
      "full_name": "johnnyj/taap_poc_gitlab",
      "url": "https://gitlab-master.nvidia.com/johnnyj/taap_poc_gitlab",
      "id": "gitlab-johnnyj-taap-poc-gitlab"
    },
    "branch": {
      "name": "main",
      "previous_commit": "x9y8z7w6v5u4",
      "current_commit": "u4v5w6x7y8z9"
    },
    "commit": {
      "sha": "u4v5w6x7y8z9",
      "short_sha": "u4v5w6x",
      "message": "fix: resolve critical bug in authentication",
      "author": {
        "name": "Jane Developer",
        "email": "jane@nvidia.com"
      },
      "timestamp": "2023-12-01T10:00:00Z"
    },
    "event": {
      "type": "branch_updated",
      "trigger": "polling",
      "detected_at": "2023-12-01T10:05:00Z"
    }
  }
}
```

## 优势特性

### 1. 标准化
- 基于 CloudEvents 1.0 国际标准
- 与其他云原生工具兼容
- 支持事件溯源和审计

### 2. 结构化
- 清晰的数据层次结构
- 类型安全的字段定义
- 易于解析和处理

### 3. 可扩展性
- 支持自定义扩展字段
- 向后兼容
- 易于集成第三方系统

### 4. 工具支持
- 支持 JSONPath 查询
- 支持标准 JSON Schema 验证
- 丰富的开发工具生态

## 迁移指南

### 从旧格式迁移
如果您之前使用的是非标准格式，请参考 [Tekton 集成指南](./tekton-integration-guide.md) 进行迁移。

### 关键变化
1. **统一的根级别字段**：所有 CloudEvents 标准字段
2. **嵌套的 data 结构**：所有业务数据放在 `data` 字段下
3. **标准化的字段命名**：使用 snake_case 命名约定
4. **丰富的元数据**：提供更完整的仓库、分支、提交信息

## 最佳实践

### 1. JSONPath 查询
```yaml
# 获取仓库名称
$(body.data.repository.name)

# 获取分支名称
$(body.data.branch.name)

# 获取提交 SHA
$(body.data.commit.sha)

# 获取作者信息
$(body.data.commit.author.name)
```

### 2. 条件处理
```yaml
# 仅处理特定提供商
$(body.data.repository.provider == 'github')

# 仅处理主分支
$(body.data.branch.name == 'main')
```

### 3. 错误处理
```yaml
# 提供默认值
$(body.data.commit.message || 'No commit message')

# 安全的字段访问
$(body.data.repository.organization || 'unknown')
```

## 相关文档
- [Tekton 集成指南](./tekton-integration-guide.md)
- [快速迁移命令](./quick-migration-commands.md)
- [CloudEvents 官方规范](https://cloudevents.io/)