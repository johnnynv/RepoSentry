# RepoSentry Tekton 集成用户指南

## 🎯 概述

RepoSentry 的 Tekton 集成功能允许您在自己的代码仓库中定义 Tekton 流水线，当代码发生变更时，这些流水线会自动执行。这个过程对您来说是完全透明的 - 您只需要在仓库中添加 `.tekton/` 目录和相关的 YAML 文件即可。

## 🚀 快速开始

### 第一步：在您的仓库中创建 Tekton 资源

在您的代码仓库根目录下创建 `.tekton/` 目录：

```bash
mkdir .tekton
cd .tekton
```

### 第二步：创建您的第一个 Pipeline

创建一个简单的构建和测试流水线：

```yaml
# .tekton/pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: my-app-ci
  labels:
    app: my-app
spec:
  params:
    - name: repository-url
      type: string
      description: "Git repository URL"
    - name: commit-sha
      type: string
      description: "Git commit SHA"
    - name: repository-name
      type: string
      description: "Repository name"
  
  workspaces:
    - name: source-code
    - name: docker-credentials
      optional: true
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    - name: run-tests
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: node:16
            workingDir: $(workspaces.source.path)
            script: |
              #!/bin/bash
              echo "🧪 Running tests for $(params.repository-name)..."
              
              # 检查是否存在 package.json
              if [ -f "package.json" ]; then
                npm install
                npm test
              fi
              
              # 检查是否存在 go.mod
              if [ -f "go.mod" ]; then
                go test ./...
              fi
              
              # 检查是否存在 pom.xml
              if [ -f "pom.xml" ]; then
                mvn test
              fi
              
              echo "✅ Tests completed!"
      workspaces:
        - name: source
          workspace: source-code
    
    - name: build-image
      runAfter: ["run-tests"]
      taskSpec:
        workspaces:
          - name: source
          - name: dockerconfig
            optional: true
        params:
          - name: image-name
            default: "$(params.repository-name):$(params.commit-sha)"
        steps:
          - name: build
            image: gcr.io/kaniko-project/executor:latest
            workingDir: $(workspaces.source.path)
            script: |
              #!/busybox/sh
              echo "🔨 Building container image..."
              
              # 检查是否存在 Dockerfile
              if [ -f "Dockerfile" ]; then
                echo "Found Dockerfile, building image: $(params.image-name)"
                /kaniko/executor \
                  --context $(workspaces.source.path) \
                  --dockerfile $(workspaces.source.path)/Dockerfile \
                  --destination $(params.image-name) \
                  --no-push
              else
                echo "⚠️  No Dockerfile found, skipping image build"
              fi
            env:
              - name: DOCKER_CONFIG
                value: $(workspaces.dockerconfig.path)
      workspaces:
        - name: source
          workspace: source-code
        - name: dockerconfig
          workspace: docker-credentials
```

### 第三步：提交代码

将您的 `.tekton/` 目录提交到 Git 仓库：

```bash
git add .tekton/
git commit -m "Add Tekton CI pipeline"
git push origin main
```

### 第四步：观察执行结果

提交代码后，RepoSentry 会自动检测到您的 Tekton 资源并执行 Pipeline。您可以通过以下方式查看执行状态：

```bash
# 查看您的命名空间中的 PipelineRun
kubectl get pipelineruns -n reposentry-user-{your-username}-{your-repo}

# 查看 Pipeline 执行日志
kubectl logs -f pipelinerun/{pipelinerun-name} -n reposentry-user-{your-username}-{your-repo}
```

## 📁 目录结构建议

推荐的 `.tekton/` 目录结构：

```
.tekton/
├── pipeline.yaml              # 主流水线定义
├── tasks/                     # 自定义任务
│   ├── build-task.yaml
│   ├── test-task.yaml
│   └── deploy-task.yaml
├── triggers/                  # 触发器配置（可选）
│   ├── binding.yaml
│   └── template.yaml
└── configs/                   # 配置文件
    ├── workspace-template.yaml
    └── secrets-template.yaml
```

## 🔧 常用 Tekton 资源示例

### 自定义 Task 示例

```yaml
# .tekton/tasks/build-task.yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: custom-build
spec:
  params:
    - name: project-type
      type: string
      default: "nodejs"
    - name: build-args
      type: string
      default: ""
  
  workspaces:
    - name: source
  
  steps:
    - name: detect-project-type
      image: alpine
      script: |
        #!/bin/sh
        cd $(workspaces.source.path)
        
        if [ -f "package.json" ]; then
          echo "nodejs" > /tmp/project-type
        elif [ -f "go.mod" ]; then
          echo "golang" > /tmp/project-type
        elif [ -f "pom.xml" ]; then
          echo "java" > /tmp/project-type
        elif [ -f "requirements.txt" ]; then
          echo "python" > /tmp/project-type
        else
          echo "unknown" > /tmp/project-type
        fi
    
    - name: build-project
      image: alpine
      script: |
        #!/bin/sh
        PROJECT_TYPE=$(cat /tmp/project-type)
        cd $(workspaces.source.path)
        
        echo "🔨 Building $PROJECT_TYPE project..."
        
        case $PROJECT_TYPE in
          "nodejs")
            npm install
            npm run build $(params.build-args)
            ;;
          "golang")
            go build $(params.build-args) ./...
            ;;
          "java")
            mvn compile $(params.build-args)
            ;;
          "python")
            pip install -r requirements.txt
            python setup.py build $(params.build-args)
            ;;
          *)
            echo "⚠️  Unknown project type, skipping build"
            ;;
        esac
        
        echo "✅ Build completed!"
```

### 多环境部署 Pipeline

```yaml
# .tekton/pipeline-deploy.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: my-app-deploy
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: repository-name
    - name: target-environment
      default: "development"
  
  workspaces:
    - name: source-code
    - name: docker-credentials
  
  tasks:
    - name: clone-source
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    - name: build-and-push
      runAfter: ["clone-source"]
      taskSpec:
        workspaces:
          - name: source
          - name: dockerconfig
        params:
          - name: image-name
            default: "my-registry/$(params.repository-name):$(params.commit-sha)"
        steps:
          - name: build-and-push
            image: gcr.io/kaniko-project/executor:latest
            script: |
              #!/busybox/sh
              /kaniko/executor \
                --context $(workspaces.source.path) \
                --dockerfile $(workspaces.source.path)/Dockerfile \
                --destination $(params.image-name)
            env:
              - name: DOCKER_CONFIG
                value: $(workspaces.dockerconfig.path)
      workspaces:
        - name: source
          workspace: source-code
        - name: dockerconfig
          workspace: docker-credentials
    
    - name: deploy-to-environment
      runAfter: ["build-and-push"]
      taskSpec:
        params:
          - name: environment
          - name: image
          - name: app-name
        steps:
          - name: deploy
            image: bitnami/kubectl
            script: |
              #!/bin/bash
              echo "🚀 Deploying to $(params.environment) environment..."
              
              # 根据环境选择命名空间
              case $(params.environment) in
                "development")
                  NAMESPACE="dev-$(params.app-name)"
                  ;;
                "staging")
                  NAMESPACE="staging-$(params.app-name)"
                  ;;
                "production")
                  NAMESPACE="prod-$(params.app-name)"
                  ;;
                *)
                  echo "❌ Unknown environment: $(params.environment)"
                  exit 1
                  ;;
              esac
              
              # 创建命名空间（如果不存在）
              kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
              
              # 部署应用
              cat <<EOF | kubectl apply -f -
              apiVersion: apps/v1
              kind: Deployment
              metadata:
                name: $(params.app-name)
                namespace: $NAMESPACE
              spec:
                replicas: 1
                selector:
                  matchLabels:
                    app: $(params.app-name)
                template:
                  metadata:
                    labels:
                      app: $(params.app-name)
                  spec:
                    containers:
                    - name: app
                      image: $(params.image)
                      ports:
                      - containerPort: 8080
              EOF
              
              echo "✅ Deployment completed!"
      params:
        - name: environment
          value: $(params.target-environment)
        - name: image
          value: "my-registry/$(params.repository-name):$(params.commit-sha)"
        - name: app-name
          value: $(params.repository-name)
```

## 🔧 高级配置

### 条件执行

根据分支或文件变更执行不同的任务：

```yaml
# .tekton/conditional-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: conditional-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: branch-name
  
  workspaces:
    - name: source-code
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    # 只在 main 分支运行部署
    - name: deploy-to-production
      when:
        - input: "$(params.branch-name)"
          operator: in
          values: ["main", "master"]
      runAfter: ["git-clone"]
      taskSpec:
        steps:
          - name: deploy
            image: alpine
            script: |
              echo "🚀 Deploying to production (branch: $(params.branch-name))..."
              # 部署逻辑...
    
    # 只在非 main 分支运行测试
    - name: run-dev-tests
      when:
        - input: "$(params.branch-name)"
          operator: notin
          values: ["main", "master"]
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: alpine
            script: |
              echo "🧪 Running development tests (branch: $(params.branch-name))..."
              # 测试逻辑...
      workspaces:
        - name: source
          workspace: source-code
```

### 并行任务执行

```yaml
# .tekton/parallel-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: parallel-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
  
  workspaces:
    - name: source-code
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    # 并行执行的任务
    - name: lint-code
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: lint
            image: alpine
            script: |
              echo "🔍 Running code linting..."
              # 代码检查逻辑...
      workspaces:
        - name: source
          workspace: source-code
    
    - name: security-scan
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: scan
            image: alpine
            script: |
              echo "🔒 Running security scan..."
              # 安全扫描逻辑...
      workspaces:
        - name: source
          workspace: source-code
    
    - name: unit-tests
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: alpine
            script: |
              echo "🧪 Running unit tests..."
              # 单元测试逻辑...
      workspaces:
        - name: source
          workspace: source-code
    
    # 等待所有并行任务完成后执行
    - name: build-application
      runAfter: ["lint-code", "security-scan", "unit-tests"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: build
            image: alpine
            script: |
              echo "🔨 Building application..."
              # 构建逻辑...
      workspaces:
        - name: source
          workspace: source-code
```

## 🔍 调试和故障排除

### 查看执行日志

```bash
# 列出您的命名空间中的所有 PipelineRun
kubectl get pipelineruns -n reposentry-user-{username}-{repo}

# 查看特定 PipelineRun 的详细信息
kubectl describe pipelinerun {pipelinerun-name} -n reposentry-user-{username}-{repo}

# 查看实时日志
kubectl logs -f pipelinerun/{pipelinerun-name} -n reposentry-user-{username}-{repo}

# 查看特定任务的日志
kubectl logs -f pipelinerun/{pipelinerun-name} -c step-{step-name} -n reposentry-user-{username}-{repo}
```

### 常见问题解决

#### 1. Pipeline 没有自动触发

**可能原因**：
- `.tekton/` 目录不存在或为空
- YAML 文件格式错误
- RepoSentry 没有检测到变更

**解决方法**：
```bash
# 检查 .tekton 目录结构
ls -la .tekton/

# 验证 YAML 文件格式
yamllint .tekton/*.yaml

# 手动触发检测（如果有权限）
curl -X POST http://reposentry-api/api/v1/repositories/{repo}/trigger
```

#### 2. 任务执行失败

**常见错误**：
```yaml
# 错误的镜像引用
steps:
  - name: build
    image: node:16-invalid  # 镜像不存在
    
# 错误的工作目录
steps:
  - name: test
    workingDir: /nonexistent/path  # 路径不存在
    
# 权限不足
steps:
  - name: deploy
    script: |
      kubectl apply -f deployment.yaml  # 可能没有权限
```

**解决方法**：
- 使用有效的镜像标签
- 确保工作目录存在
- 检查所需的权限和 RBAC 配置

#### 3. 资源配额超限

**错误信息**：
```
Error: pods "my-task-pod" is forbidden: exceeded quota
```

**解决方法**：
- 减少并行任务数量
- 优化资源请求和限制
- 联系管理员调整配额

## 📚 最佳实践

### 1. 资源优化

```yaml
# 为任务设置合适的资源限制
taskSpec:
  stepTemplate:
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "200m"
  steps:
    - name: build
      # ... 其他配置
```

### 2. 镜像选择

```yaml
# 使用轻量级镜像
steps:
  - name: test
    image: alpine:3.18  # 而不是 ubuntu:latest
    
  # 使用特定版本标签
  - name: build
    image: node:16.20.0-alpine  # 而不是 node:latest
```

### 3. 安全实践

```yaml
# 不要在 YAML 中硬编码敏感信息
steps:
  - name: deploy
    env:
      - name: API_KEY
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: api-key
    script: |
      # 使用环境变量
      curl -H "Authorization: Bearer $API_KEY" ...
```

### 4. 错误处理

```yaml
steps:
  - name: robust-task
    image: alpine
    script: |
      #!/bin/bash
      set -euo pipefail  # 严格错误处理
      
      # 检查必要的文件
      if [ ! -f "required-file.txt" ]; then
        echo "❌ Required file not found"
        exit 1
      fi
      
      # 执行操作并检查结果
      if ! some-command; then
        echo "❌ Command failed"
        exit 1
      fi
      
      echo "✅ Task completed successfully"
```

### 5. 可重用性

```yaml
# 使用参数使 Pipeline 更灵活
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reusable-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: build-image
      default: "node:16"
    - name: test-command
      default: "npm test"
    - name: build-command
      default: "npm run build"
  
  tasks:
    - name: flexible-build
      taskSpec:
        params:
          - name: build-image
          - name: test-cmd
          - name: build-cmd
        steps:
          - name: test
            image: $(params.build-image)
            script: $(params.test-cmd)
          - name: build
            image: $(params.build-image)
            script: $(params.build-cmd)
      params:
        - name: build-image
          value: $(params.build-image)
        - name: test-cmd
          value: $(params.test-command)
        - name: build-cmd
          value: $(params.build-command)
```

## 🔗 相关资源

- [Tekton Pipelines 官方文档](https://tekton.dev/docs/pipelines/)
- [Tekton Tasks Catalog](https://hub.tekton.dev/)
- [Kubernetes 资源管理](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [YAML 语法指南](https://yaml.org/spec/)

## 💬 获取帮助

如果您在使用过程中遇到问题：

1. **查看日志**：首先检查 PipelineRun 的执行日志
2. **验证 YAML**：确保您的 Tekton 资源格式正确
3. **检查权限**：确认您的 Pipeline 有足够的权限执行所需操作
4. **参考示例**：查看本指南中的示例和最佳实践
5. **联系支持**：如果问题仍然存在，请联系您的平台管理员

---

**注意**：RepoSentry 的 Tekton 集成功能完全透明，您无需配置任何 Webhook 或进行额外设置。只需在仓库中添加 `.tekton/` 目录和相关 YAML 文件，系统会自动检测并执行您的 Pipeline。

