# RepoSentry 快速开始指南

## 🚀 概述

RepoSentry 是一个轻量级的云原生 Git 仓库监控哨兵，支持监控 GitHub 和 GitLab 仓库的变更并触发 Tekton 流水线。

## ⚡ 5分钟快速开始

### 前置要求

- Go 1.21+ （如果从源码构建）
- Docker（如果使用容器部署）
- Kubernetes（如果使用 Helm 部署）
- GitHub/GitLab API Token
- Tekton EventListener URL

### 第1步：获取 RepoSentry

#### 方式1：下载预编译二进制（推荐）
```bash
# 下载最新版本（假设有发布版本）
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-linux-amd64
chmod +x reposentry-linux-amd64
sudo mv reposentry-linux-amd64 /usr/local/bin/reposentry
```

#### 方式2：从源码构建
```bash
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry
make build
sudo cp bin/reposentry /usr/local/bin/
```

### 第2步：准备配置文件

创建基础配置文件：

```bash
# 生成基础配置
reposentry config init --type=basic > config.yaml
```

**或者**手动创建 `config.yaml`：

```yaml
# 应用配置
app:
  name: "reposentry"
  log_level: "info"
  log_format: "json"
  health_check_port: 8080

# 轮询配置
polling:
  interval: "5m"
  timeout: "30s"
  max_workers: 5
  batch_size: 10

# 存储配置
storage:
  type: "sqlite"
  sqlite:
    path: "./data/reposentry.db"

# Tekton 集成
tekton:
  event_listener_url: "http://your-tekton-listener:8080"
  timeout: "10s"

# 监控的仓库列表
repositories:
  - name: "my-github-repo"
    url: "https://github.com/username/repository"
    provider: "github"
    token: "${GITHUB_TOKEN}"
    branch_regex: "^(main|develop|release/.*)$"
    
  - name: "my-gitlab-repo"
    url: "https://gitlab.example.com/group/project"
    provider: "gitlab"
    token: "${GITLAB_TOKEN}"
    branch_regex: "^(main|master|hotfix/.*)$"
```

### 第3步：设置环境变量

```bash
# GitHub Token
export GITHUB_TOKEN="ghp_your_github_token_here"

# GitLab Token
export GITLAB_TOKEN="glpat-your_gitlab_token_here"

# 企业版 GitLab（如果需要）
export GITLAB_ENTERPRISE_TOKEN="glpat-your_enterprise_token"
```

### 第4步：验证配置

```bash
# 验证配置文件语法
reposentry config validate config.yaml

# 验证环境变量和连接性
reposentry config validate config.yaml --check-env --check-connectivity
```

### 第5步：启动 RepoSentry

```bash
# 前台运行（用于测试）
reposentry run --config=config.yaml

# 后台运行
reposentry run --config=config.yaml --daemon
```

### 第6步：验证运行状态

```bash
# 检查健康状态
curl http://localhost:8080/health

# 查看运行状态
reposentry status

# 查看监控的仓库
reposentry repo list

# 查看事件历史
curl http://localhost:8080/api/v1/events
```

## 🐳 Docker 部署

### 快速启动

```bash
# 克隆仓库
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry/deployments/docker

# 编辑配置文件
cp ../../examples/configs/basic.yaml config.yaml
vim config.yaml  # 修改你的设置

# 设置环境变量
export GITHUB_TOKEN="your_github_token"
export GITLAB_TOKEN="your_gitlab_token"

# 启动服务
docker-compose up -d
```

### 查看日志

```bash
# 查看服务日志
docker-compose logs -f reposentry

# 查看健康状态
curl http://localhost:8080/health
```

### 停止服务

```bash
docker-compose down
```

## ☸️ Kubernetes (Helm) 部署

### 快速部署

```bash
# 添加必要的 Secret
kubectl create secret generic reposentry-tokens \
  --from-literal=github-token="your_github_token" \
  --from-literal=gitlab-token="your_gitlab_token"

# 使用示例配置部署
helm install reposentry ./deployments/helm/reposentry \
  -f examples/kubernetes/helm-values-prod.yaml
```

### 自定义部署

```bash
# 复制并编辑配置
cp examples/kubernetes/helm-values-prod.yaml my-values.yaml
vim my-values.yaml

# 部署
helm install reposentry ./deployments/helm/reposentry -f my-values.yaml
```

### 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -l app.kubernetes.io/name=reposentry

# 查看服务
kubectl get svc -l app.kubernetes.io/name=reposentry

# 端口转发测试
kubectl port-forward svc/reposentry 8080:8080

# 测试健康检查
curl http://localhost:8080/health
```

## 🔧 Systemd 部署

### 安装配置

```bash
# 复制二进制文件
sudo cp bin/reposentry /usr/local/bin/

# 创建配置目录
sudo mkdir -p /etc/reposentry

# 复制配置文件
sudo cp config.yaml /etc/reposentry/

# 创建数据目录
sudo mkdir -p /var/lib/reposentry
sudo chown reposentry:reposentry /var/lib/reposentry

# 安装 systemd 服务
sudo cp deployments/systemd/reposentry.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 设置环境变量

```bash
# 编辑服务文件添加环境变量
sudo systemctl edit reposentry

# 添加以下内容：
[Service]
Environment="GITHUB_TOKEN=your_github_token"
Environment="GITLAB_TOKEN=your_gitlab_token"
```

### 启动服务

```bash
# 启用并启动服务
sudo systemctl enable reposentry
sudo systemctl start reposentry

# 查看状态
sudo systemctl status reposentry

# 查看日志
sudo journalctl -u reposentry -f
```

## ⚙️ 必填配置字段

### 核心必填字段

| 字段路径 | 类型 | 说明 | 示例 |
|---------|------|------|------|
| `tekton.event_listener_url` | string | Tekton EventListener 的 URL | `http://tekton:8080` |
| `repositories[].name` | string | 仓库唯一标识 | `my-app` |
| `repositories[].url` | string | 仓库 HTTPS URL | `https://github.com/user/repo` |
| `repositories[].provider` | string | Git 提供商 | `github` 或 `gitlab` |
| `repositories[].token` | string | API 访问 Token | `${GITHUB_TOKEN}` |
| `repositories[].branch_regex` | string | 分支过滤正则表达式 | `^(main\|develop)$` |

### 可选但建议设置

| 字段路径 | 类型 | 默认值 | 说明 |
|---------|------|--------|------|
| `app.log_level` | string | `info` | 日志级别 |
| `app.health_check_port` | int | `8080` | 健康检查端口 |
| `polling.interval` | string | `5m` | 轮询间隔 |
| `storage.sqlite.path` | string | `./data/reposentry.db` | 数据库路径 |

## 🔍 验证清单

启动后请检查以下项目：

- [ ] ✅ 配置文件语法正确：`reposentry config validate config.yaml`
- [ ] ✅ 环境变量已设置：`reposentry config validate --check-env`
- [ ] ✅ 网络连接正常：`curl http://localhost:8080/health`
- [ ] ✅ 仓库访问正常：`reposentry repo list`
- [ ] ✅ Tekton 连接正常：检查 EventListener 日志
- [ ] ✅ 轮询工作正常：观察事件日志

## 🚨 常见问题

### 1. 配置验证失败
```bash
# 检查配置语法
reposentry config validate config.yaml

# 检查环境变量
echo $GITHUB_TOKEN
echo $GITLAB_TOKEN
```

### 2. 权限不足
```bash
# 检查 Token 权限
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# GitLab 检查
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/user
```

### 3. 网络连接问题
```bash
# 测试 Tekton 连接
curl -X POST $TEKTON_EVENTLISTENER_URL/health

# 检查防火墙设置
sudo ufw status
```

### 4. 数据库权限
```bash
# 检查数据目录权限
ls -la ./data/
chmod 755 ./data/
```

## 📖 下一步

- 阅读 [用户手册](USER_MANUAL.md) 了解详细配置
- 查看 [技术架构](ARCHITECTURE.md) 了解工作原理
- 访问 Swagger API 文档：`http://localhost:8080/swagger/`
- 查看 [部署指南](../deployments/README.md) 了解生产部署
