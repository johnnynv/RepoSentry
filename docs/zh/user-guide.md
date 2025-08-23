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
- **智能轮询**: API 优先，Git 命令降级
- **多平台支持**: GitHub、GitLab（包括企业版）
- **灵活配置**: YAML 配置 + 环境变量
- **事件驱动**: 实时触发 Tekton 流水线
- **云原生**: 支持 Docker、Kubernetes 部署

## 🔧 安装

### 系统要求

- **操作系统**: Linux、macOS、Windows
- **内存**: 最小 128MB，推荐 512MB
- **存储**: 100MB （包含数据库）
- **网络**: 需要访问 Git 提供商 API 和 Tekton EventListener

### 安装方式

#### 1. 二进制安装（推荐）

```bash
# 下载最新版本
curl -L -o reposentry https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-linux-amd64

# 设置执行权限
chmod +x reposentry

# 移动到系统路径
sudo mv reposentry /usr/local/bin/
```

#### 2. 从源码构建

```bash
# 克隆仓库
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry

# 构建
make build

# 安装
sudo cp bin/reposentry /usr/local/bin/
```

#### 3. Docker 安装

```bash
# 拉取镜像
docker pull reposentry:latest

# 或从源码构建
docker build -t reposentry:latest .
```

## ⚙️ 配置详解

### 配置文件结构

RepoSentry 使用 YAML 格式的配置文件，主要包含以下部分：

```yaml
app:           # 应用程序配置
polling:       # 轮询配置
storage:       # 存储配置
tekton:        # Tekton 集成配置
repositories:  # 仓库列表配置
```

### 应用程序配置 (app)

```yaml
app:
  name: "reposentry"                    # 应用名称
  log_level: "info"                     # 日志级别：debug, info, warn, error
  log_format: "json"                    # 日志格式：json, text
  log_file: "/var/log/reposentry.log"   # 日志文件路径（可选）
  log_file_rotation:                    # 日志轮转配置（可选）
    max_size: 100                       # 最大文件大小（MB）
    max_backups: 5                      # 最大备份文件数
    max_age: 30                         # 最大保存天数
    compress: true                      # 是否压缩
  health_check_port: 8080               # 健康检查和 API 端口
  data_dir: "./data"                    # 数据目录
```

#### 重要字段说明

| 字段 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `log_level` | 否 | `info` | 生产环境建议 `info`，调试时使用 `debug` |
| `log_format` | 否 | `json` | JSON 格式便于日志聚合分析 |
| `health_check_port` | 否 | `8080` | REST API 和健康检查端口 |
| `data_dir` | 否 | `./data` | 数据库和日志文件存储目录 |

### 轮询配置 (polling)

```yaml
polling:
  interval: "5m"          # 全局轮询间隔
  timeout: "30s"          # API 请求超时时间
  max_workers: 5          # 最大并发工作协程数
  batch_size: 10          # 每批处理的仓库数量
  retry_attempts: 3       # 失败重试次数
  retry_backoff: "30s"    # 重试间隔
```

#### 性能调优指南

| 仓库数量 | 建议配置 | 说明 |
|----------|----------|------|
| 1-10 | `max_workers: 2, batch_size: 5` | 小规模部署 |
| 11-50 | `max_workers: 5, batch_size: 10` | 中等规模 |
| 51-200 | `max_workers: 10, batch_size: 20` | 大规模部署 |
| 200+ | `max_workers: 20, batch_size: 50` | 企业级部署 |

### 存储配置 (storage)

```yaml
storage:
  type: "sqlite"
  sqlite:
    path: "./data/reposentry.db"
    max_connections: 10
    connection_timeout: "30s"
    busy_timeout: "5s"
```

#### SQLite 配置说明

- **path**: 数据库文件路径，建议使用绝对路径
- **max_connections**: 连接池大小，一般不需要调整
- **connection_timeout**: 连接超时时间
- **busy_timeout**: 数据库锁等待时间

### Tekton 集成配置

```yaml
tekton:
  event_listener_url: "http://tekton-listener:8080"
  timeout: "10s"
  headers:
    Content-Type: "application/json"
    X-Custom-Header: "reposentry"
  retry_attempts: 3
  retry_backoff: "5s"
```

#### 必填字段

- **event_listener_url**: Tekton EventListener 的完整 URL
- 其他字段都是可选的，有合理的默认值

### 仓库配置 (repositories)

这是 RepoSentry 的核心配置部分：

```yaml
repositories:
  - name: "frontend-app"                              # 仓库唯一标识符
    url: "https://github.com/company/frontend-app"    # 仓库 HTTPS URL
    provider: "github"                                # 提供商：github 或 gitlab
    token: "${GITHUB_TOKEN}"                          # API Token（使用环境变量）
    branch_regex: "^(main|develop|release/.*)$"       # 分支过滤正则表达式
    polling_interval: "3m"                            # 仓库特定轮询间隔（可选）
    metadata:                                         # 自定义元数据（可选）
      team: "frontend"
      env: "production"
    
  - name: "backend-service"
    url: "https://gitlab-master.nvidia.com/team/backend"
    provider: "gitlab"
    token: "${GITLAB_ENTERPRISE_TOKEN}"
    branch_regex: "^(master|hotfix/.*)$"
    polling_interval: "10m"
```

#### 仓库配置字段详解

| 字段 | 必填 | 类型 | 说明 | 示例 |
|------|------|------|------|------|
| `name` | ✅ | string | 仓库唯一标识，不能重复 | `my-app` |
| `url` | ✅ | string | 仓库 HTTPS URL，不支持 SSH | `https://github.com/user/repo` |
| `provider` | ✅ | string | `github` 或 `gitlab` | `github` |
| `token` | ✅ | string | API 访问 Token，**必须**使用环境变量 | `${GITHUB_TOKEN}` |
| `branch_regex` | ✅ | string | 分支过滤正则表达式 | `^(main\|develop)$` |
| `polling_interval` | 否 | string | 覆盖全局轮询间隔 | `2m` |
| `metadata` | 否 | map | 自定义元数据，会传递给 Tekton | `team: frontend` |

#### 分支正则表达式示例

```yaml
# 只监控主分支
branch_regex: "^main$"

# 监控主分支和开发分支
branch_regex: "^(main|develop)$"

# 监控发布分支
branch_regex: "^release/.*$"

# 监控特定前缀
branch_regex: "^(feature|bugfix)/.*$"

# 监控多种模式
branch_regex: "^(main|develop|release/.*|hotfix/.*)$"
```

### 环境变量配置

RepoSentry 支持在配置文件中使用环境变量：

#### 支持的格式

```yaml
# 标准格式
token: "${GITHUB_TOKEN}"

# 带默认值
url: "${TEKTON_URL:-http://localhost:8080}"

# 复杂环境变量
token: "${GITLAB_ENTERPRISE_TOKEN}"
```

#### 环境变量白名单

出于安全考虑，只有以下模式的环境变量被允许：

- `*_TOKEN`
- `*_SECRET`
- `*_PASSWORD`
- `*_KEY`
- `*_URL`
- `*_HOST`
- `*_PORT`

## 🖥️ CLI 命令

### 主要命令

#### 1. 配置管理

```bash
# 生成配置文件
reposentry config init --type=basic > config.yaml
reposentry config init --type=minimal > minimal.yaml

# 验证配置
reposentry config validate config.yaml
reposentry config validate config.yaml --check-env
reposentry config validate config.yaml --check-connectivity

# 显示当前配置
reposentry config show --config=config.yaml
reposentry config show --config=config.yaml --hide-secrets
```

#### 2. 运行服务

```bash
# 前台运行
reposentry run --config=config.yaml

# 后台运行
reposentry run --config=config.yaml --daemon

# 指定日志级别
reposentry run --config=config.yaml --log-level=debug

# 自定义端口
reposentry run --config=config.yaml --port=9090
```

#### 3. 状态检查

```bash
# 检查服务状态
reposentry status

# 检查特定主机
reposentry status --host=remote-server --port=8080
```

#### 4. 仓库管理

```bash
# 列出所有仓库
reposentry repo list

# 显示仓库详情
reposentry repo show my-repo-name

# 测试仓库连接
reposentry repo test my-repo-name
```

#### 5. 其他工具命令

```bash
# 查看版本
reposentry version

# 测试 webhook
reposentry test-webhook --url=http://tekton:8080 --payload='{"test": true}'

# 查看帮助
reposentry --help
reposentry run --help
```

### CLI 配置文件查找顺序

RepoSentry 按以下顺序查找配置文件：

1. `--config` 参数指定的文件
2. `RS_CONFIG_PATH` 环境变量
3. `./config.yaml`
4. `./reposentry.yaml`
5. `~/.reposentry/config.yaml`
6. `/etc/reposentry/config.yaml`

## 🌐 API 接口

RepoSentry 提供完整的 RESTful API 接口。

### Swagger UI 文档

启动服务后，访问 Swagger 在线文档：

```
http://localhost:8080/swagger/
```

### 主要接口

#### 1. 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 详细健康检查
curl http://localhost:8080/healthz

# 就绪检查
curl http://localhost:8080/ready
```

#### 2. 服务状态

```bash
# 获取运行时状态
curl http://localhost:8080/api/v1/status

# 获取服务版本
curl http://localhost:8080/api/v1/version

# 获取指标信息
curl http://localhost:8080/api/v1/metrics
```

#### 3. 仓库管理

```bash
# 列出所有仓库
curl http://localhost:8080/api/v1/repositories

# 获取特定仓库信息
curl http://localhost:8080/api/v1/repositories/my-repo

# 获取仓库状态
curl http://localhost:8080/api/v1/repositories/my-repo/status
```

#### 4. 事件查询

```bash
# 获取所有事件
curl http://localhost:8080/api/v1/events

# 获取最近事件
curl http://localhost:8080/api/v1/events/recent

# 获取特定事件
curl http://localhost:8080/api/v1/events/{event-id}

# 按仓库过滤
curl "http://localhost:8080/api/v1/events?repository=my-repo"

# 按时间范围过滤
curl "http://localhost:8080/api/v1/events?since=2024-01-01T00:00:00Z"
```

### API 认证

当前版本的 API 不需要认证，但建议在生产环境中通过防火墙或反向代理限制访问。

### API 响应格式

所有 API 响应都遵循统一格式：

```json
{
  "success": true,
  "message": "操作成功",
  "data": {
    // 响应数据
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

错误响应：

```json
{
  "success": false,
  "message": "错误描述",
  "error": "详细错误信息",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🔄 配置热更新

RepoSentry 支持运行时配置热更新，无需重启服务。

### 触发热更新

#### 方法1: 发送信号（Linux/macOS）

```bash
# 发送 SIGHUP 信号
sudo kill -HUP $(pgrep reposentry)

# 或使用 systemctl（如果使用 systemd）
sudo systemctl reload reposentry
```

#### 方法2: API 接口

```bash
# 重新加载配置
curl -X POST http://localhost:8080/api/v1/config/reload
```

#### 方法3: CLI 命令

```bash
# 重新加载配置
reposentry config reload --host=localhost --port=8080
```

### 热更新注意事项

#### ✅ 支持热更新的配置

- 仓库列表 (`repositories`)
- 轮询间隔 (`polling.interval`)
- 日志级别 (`app.log_level`)
- Tekton 配置 (`tekton`)

#### ❌ 不支持热更新的配置

- 端口配置 (`app.health_check_port`)
- 存储配置 (`storage`)
- 数据目录 (`app.data_dir`)

这些配置需要重启服务才能生效。

### 验证热更新

```bash
# 1. 修改配置文件
vim config.yaml

# 2. 触发重新加载
curl -X POST http://localhost:8080/api/v1/config/reload

# 3. 检查配置是否生效
reposentry config show --host=localhost --port=8080
```

## 📊 监控和日志

### 日志配置

#### 日志级别

- **debug**: 详细调试信息，包含所有操作细节
- **info**: 一般信息，生产环境推荐
- **warn**: 警告信息，需要关注但不影响运行
- **error**: 错误信息，需要立即处理

#### 日志格式

```yaml
# JSON 格式（推荐用于生产环境）
app:
  log_format: "json"

# 文本格式（适合开发和调试）
app:
  log_format: "text"
```

#### 日志文件

```yaml
app:
  log_file: "/var/log/reposentry.log"
  log_file_rotation:
    max_size: 100      # 100MB
    max_backups: 5     # 保留5个备份
    max_age: 30        # 保留30天
    compress: true     # 压缩旧日志
```

### 关键日志字段

JSON 格式日志包含以下关键字段：

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "poller",
  "module": "github_client",
  "operation": "fetch_branches",
  "repository": "my-repo",
  "duration": 1250,
  "message": "Successfully fetched branches",
  "metadata": {
    "branch_count": 5,
    "api_rate_remaining": 4999
  }
}
```

### 监控指标

通过 API 获取运行时指标：

```bash
curl http://localhost:8080/api/v1/metrics | jq
```

响应示例：

```json
{
  "success": true,
  "data": {
    "uptime": "2h30m15s",
    "repositories": {
      "total": 10,
      "healthy": 9,
      "error": 1
    },
    "polling": {
      "last_cycle": "2024-01-15T10:30:00Z",
      "next_cycle": "2024-01-15T10:35:00Z",
      "cycle_duration": "45s"
    },
    "events": {
      "total": 156,
      "today": 23,
      "last_hour": 3
    },
    "api_calls": {
      "github_remaining": 4950,
      "gitlab_remaining": 1850
    }
  }
}
```

### 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 详细组件健康状态
curl http://localhost:8080/healthz
```

健康检查响应：

```json
{
  "success": true,
  "data": {
    "healthy": true,
    "components": {
      "config": {"healthy": true, "message": "OK"},
      "storage": {"healthy": true, "message": "Database connected"},
      "git_client": {"healthy": true, "message": "All clients ready"},
      "trigger": {"healthy": true, "message": "Tekton reachable"},
      "poller": {"healthy": true, "message": "Polling active"}
    }
  }
}
```

## 🔐 安全最佳实践

### 1. 敏感信息管理

#### ✅ 正确做法

```yaml
repositories:
  - name: "my-repo"
    token: "${GITHUB_TOKEN}"  # 使用环境变量
```

#### ❌ 错误做法

```yaml
repositories:
  - name: "my-repo"
    token: "ghp_xxxxxxxxxxxx"  # 硬编码 Token
```

### 2. Token 权限控制

#### GitHub Token 权限

- **公开仓库**: `public_repo` 权限
- **私有仓库**: `repo` 权限
- **组织仓库**: 需要组织授权

#### GitLab Token 权限

- **项目访问**: `read_repository` 权限
- **API 访问**: `read_api` 权限
- **企业版**: 可能需要额外的访问权限

### 3. 网络安全

```yaml
# 限制监听地址（生产环境）
app:
  health_check_bind: "127.0.0.1:8080"  # 仅本地访问

# 使用 HTTPS（通过反向代理）
tekton:
  event_listener_url: "https://tekton.example.com:8080"
```

### 4. 文件权限

```bash
# 配置文件权限
chmod 600 config.yaml
chown reposentry:reposentry config.yaml

# 数据目录权限
chmod 750 ./data
chown reposentry:reposentry ./data
```

### 5. 容器安全

```yaml
# docker-compose.yml 安全配置
services:
  reposentry:
    user: "1000:1000"      # 非 root 用户
    read_only: true        # 只读文件系统
    cap_drop:
      - ALL                # 移除所有权限
    cap_add:
      - NET_BIND_SERVICE   # 仅保留必要权限
```

## 🔧 故障排除

### 常见问题

#### 1. 配置文件问题

**症状**: 启动时配置验证失败

```bash
# 排查步骤
# 1. 检查 YAML 语法
reposentry config validate config.yaml

# 2. 检查环境变量
reposentry config validate config.yaml --check-env

# 3. 检查网络连接
reposentry config validate config.yaml --check-connectivity
```

#### 2. API Token 问题

**症状**: 仓库访问被拒绝

```bash
# GitHub Token 测试
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/owner/repo

# GitLab Token 测试
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  https://gitlab.com/api/v4/projects/owner%2Frepo
```

#### 3. 网络连接问题

**症状**: 无法连接到 Tekton EventListener

```bash
# 测试连接
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"test": true}' \
  $TEKTON_EVENTLISTENER_URL

# 检查 DNS 解析
nslookup tekton-listener.example.com

# 检查端口连通性
telnet tekton-listener.example.com 8080
```

#### 4. 权限问题

**症状**: 数据库创建失败

```bash
# 检查目录权限
ls -la ./data/

# 修复权限
mkdir -p ./data
chmod 755 ./data
chown $USER:$USER ./data
```

#### 5. 性能问题

**症状**: 轮询速度慢

```bash
# 调优配置
polling:
  max_workers: 10        # 增加并发数
  batch_size: 20         # 增加批处理大小
  timeout: "60s"         # 增加超时时间
```

### 日志分析

#### 开启详细日志

```yaml
app:
  log_level: "debug"
```

#### 关键日志模式

```bash
# 过滤错误日志
grep '"level":"error"' /var/log/reposentry.log

# 查看轮询状态
grep '"component":"poller"' /var/log/reposentry.log

# 监控 API 调用
grep '"operation":"api_call"' /var/log/reposentry.log
```

### 数据库恢复

#### 备份数据库

```bash
# 停止服务
sudo systemctl stop reposentry

# 备份数据库
cp ./data/reposentry.db ./data/reposentry.db.backup

# 重启服务
sudo systemctl start reposentry
```

#### 重置数据库

```bash
# 停止服务
sudo systemctl stop reposentry

# 删除数据库（所有历史数据丢失）
rm ./data/reposentry.db

# 重启服务（会自动创建新数据库）
sudo systemctl start reposentry
```

## 🚀 高级用法

### 1. 多环境部署

#### 开发环境配置

```yaml
app:
  log_level: "debug"
  log_format: "text"

polling:
  interval: "1m"         # 频繁轮询用于测试
  
repositories:
  - name: "test-repo"
    url: "https://github.com/user/test-repo"
    provider: "github"
    token: "${GITHUB_TOKEN}"
    branch_regex: ".*"   # 监控所有分支
```

#### 生产环境配置

```yaml
app:
  log_level: "info"
  log_format: "json"
  log_file: "/var/log/reposentry.log"
  log_file_rotation:
    max_size: 100
    max_backups: 10
    max_age: 90

polling:
  interval: "10m"        # 较长间隔减少 API 调用
  max_workers: 20
  
repositories:
  - name: "prod-app"
    url: "https://github.com/company/prod-app"
    provider: "github"
    token: "${GITHUB_PROD_TOKEN}"
    branch_regex: "^(main|release/.*)$"  # 仅生产分支
```

### 2. 企业级 GitLab 集成

```yaml
repositories:
  - name: "enterprise-project"
    url: "https://gitlab-master.nvidia.com/ai/chat-bot"
    provider: "gitlab"
    token: "${GITLAB_ENTERPRISE_TOKEN}"
    branch_regex: "^(master|develop|feature/.*)$"
    polling_interval: "15m"
    metadata:
      team: "ai-research"
      priority: "high"
      environment: "production"
```

### 3. 分支策略模式

#### Git Flow 模式

```yaml
repositories:
  - name: "gitflow-repo"
    branch_regex: "^(master|develop|release/.*|hotfix/.*)$"
```

#### GitHub Flow 模式

```yaml
repositories:
  - name: "githubflow-repo"
    branch_regex: "^(main|feature/.*)$"
```

#### 自定义模式

```yaml
repositories:
  - name: "custom-repo"
    branch_regex: "^(main|staging|prod|feature/.*|bugfix/.*|hotfix/.*)$"
```

### 4. 监控集成

#### Prometheus 指标

虽然 RepoSentry 不直接支持 Prometheus，但可以通过脚本定期采集指标：

```bash
#!/bin/bash
# prometheus-exporter.sh

metrics=$(curl -s http://localhost:8080/api/v1/metrics)
echo "reposentry_uptime_seconds $(echo $metrics | jq -r '.data.uptime_seconds')"
echo "reposentry_repositories_total $(echo $metrics | jq -r '.data.repositories.total')"
echo "reposentry_events_total $(echo $metrics | jq -r '.data.events.total')"
```

#### 日志聚合

使用 ELK Stack 或类似工具聚合日志：

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  paths:
    - /var/log/reposentry.log
  json.keys_under_root: true
  json.add_error_key: true
  
output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "reposentry-%{+yyyy.MM.dd}"
```

### 5. 高可用部署

#### 主从模式（数据库共享）

```yaml
# 主节点 - 启用轮询
polling:
  enabled: true
  interval: "5m"

# 从节点 - 仅 API 服务
polling:
  enabled: false
```

#### 负载均衡

```nginx
# nginx.conf
upstream reposentry {
    server 192.168.1.10:8080;
    server 192.168.1.11:8080;
    server 192.168.1.12:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://reposentry;
    }
}
```

### 6. 自动化运维

#### 健康检查脚本

```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/health.json $HEALTH_URL)

if [ "$RESPONSE" != "200" ]; then
    echo "RepoSentry unhealthy, restarting..."
    sudo systemctl restart reposentry
    
    # 发送告警
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"RepoSentry service restarted"}' \
        $SLACK_WEBHOOK_URL
fi
```

#### 配置同步脚本

```bash
#!/bin/bash
# sync-config.sh

# 从 Git 仓库拉取最新配置
cd /etc/reposentry/
git pull origin main

# 验证配置
if reposentry config validate config.yaml; then
    # 重新加载配置
    curl -X POST http://localhost:8080/api/v1/config/reload
    echo "Configuration updated successfully"
else
    echo "Configuration validation failed"
    exit 1
fi
```

## 📝 参考资料

- [快速开始指南](QUICKSTART.md)
- [技术架构文档](ARCHITECTURE.md)
- [部署指南](../deployments/README.md)
- [API 示例](../API_EXAMPLES.md)
- [配置示例](../examples/README.md)
- [故障排除指南](TROUBLESHOOTING.md)

---

**提示**: 如果遇到问题，请优先查看日志文件或使用 `reposentry status` 命令诊断问题。
