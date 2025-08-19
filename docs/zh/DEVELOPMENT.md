# RepoSentry 开发指南

## 🛠️ 开发环境设置

### 系统要求

- **Go**: 1.21 或更高版本
- **Git**: 2.0+ 
- **Make**: GNU Make 4.0+
- **Docker**: 20.10+ （可选，用于容器测试）
- **SQLite**: 3.35+ （通常系统自带）

### 环境准备

#### 1. 克隆仓库

```bash
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry
```

#### 2. 安装依赖

```bash
# 下载 Go 模块依赖
go mod download

# 安装开发工具
make dev-tools
```

#### 3. 设置开发环境变量

```bash
# 创建 .env 文件
cat > .env << EOF
# GitHub Token（用于测试）
export GITHUB_TOKEN="ghp_your_github_token_here"

# GitLab Token（用于测试）
export GITLAB_TOKEN="glpat-your_gitlab_token_here"

# 测试 Tekton EventListener URL
export TEKTON_TEST_URL="http://localhost:8080/test-webhook"
EOF

# 加载环境变量
source .env
```

#### 4. 验证环境

```bash
# 检查 Go 环境
go version

# 检查依赖
go mod verify

# 运行测试
make test

# 构建项目
make build
```

## 📁 项目结构详解

### 目录组织

```
RepoSentry/
├── cmd/reposentry/           # 应用入口点
│   ├── main.go              # 主函数
│   ├── root.go              # Cobra 根命令
│   ├── run.go               # 运行命令
│   ├── config.go            # 配置命令
│   └── ...                  # 其他 CLI 命令
├── internal/                 # 内部业务逻辑（不对外暴露）
│   ├── api/                 # REST API 服务器
│   │   ├── server.go        # HTTP 服务器
│   │   ├── router.go        # 路由配置
│   │   ├── middleware/      # 中间件
│   │   └── handlers/        # API 处理器
│   ├── config/              # 配置管理
│   │   ├── config.go        # 配置结构
│   │   ├── loader.go        # 配置加载器
│   │   └── validator.go     # 配置验证器
│   ├── gitclient/           # Git 客户端
│   │   ├── github.go        # GitHub API 客户端
│   │   ├── gitlab.go        # GitLab API 客户端
│   │   ├── fallback.go      # Git 命令降级
│   │   └── ratelimit.go     # 速率限制器
│   ├── poller/              # 轮询逻辑
│   │   ├── poller.go        # 轮询器接口
│   │   ├── scheduler.go     # 调度器
│   │   ├── monitor.go       # 分支监控器
│   │   └── events.go        # 事件生成器
│   ├── runtime/             # 运行时管理
│   │   ├── runtime.go       # 运行时接口
│   │   ├── manager.go       # 组件管理器
│   │   └── components.go    # 组件实现
│   ├── storage/             # 存储层
│   │   ├── storage.go       # 存储接口
│   │   ├── sqlite.go        # SQLite 实现
│   │   └── migrations.go    # 数据库迁移
│   └── trigger/             # 触发器
│       ├── trigger.go       # 触发器接口
│       ├── tekton.go        # Tekton 触发器
│       └── transformer.go   # 数据转换器
├── pkg/                      # 公共包（可对外暴露）
│   ├── logger/              # 日志组件
│   ├── types/               # 类型定义
│   └── utils/               # 工具函数
├── test/                     # 测试文件
│   ├── fixtures/            # 测试数据
│   └── integration/         # 集成测试
├── deployments/              # 部署配置
├── docs/                     # 文档
├── examples/                 # 示例配置
└── Makefile                  # 构建脚本
```

### 包设计原则

#### internal/ vs pkg/

- **internal/**: 内部业务逻辑，不允许外部导入
- **pkg/**: 公共库，可以被其他项目导入

#### 分层架构

```
┌─────────────────────────────────┐
│          CLI Layer              │  ← cmd/reposentry/
├─────────────────────────────────┤
│          API Layer              │  ← internal/api/
├─────────────────────────────────┤
│       Business Logic Layer      │  ← internal/poller/, internal/trigger/
├─────────────────────────────────┤
│        Service Layer            │  ← internal/gitclient/, internal/storage/
├─────────────────────────────────┤
│       Foundation Layer          │  ← pkg/logger/, pkg/types/, pkg/utils/
└─────────────────────────────────┘
```

## 🏗️ 代码架构

### 设计模式

#### 1. 依赖注入

```go
// 接口定义
type Storage interface {
    Store(event Event) error
    GetEvents(filter Filter) ([]Event, error)
}

// 依赖注入
type Poller struct {
    storage   Storage      // 注入存储接口
    gitClient GitClient    // 注入 Git 客户端接口
    logger    *Logger      // 注入日志器
}

func NewPoller(storage Storage, gitClient GitClient, logger *Logger) *Poller {
    return &Poller{
        storage:   storage,
        gitClient: gitClient,
        logger:    logger,
    }
}
```

#### 2. 工厂模式

```go
// 客户端工厂
type ClientFactory struct {
    logger *Logger
}

func (f *ClientFactory) CreateClient(provider string, config ClientConfig) (GitClient, error) {
    switch provider {
    case "github":
        return NewGitHubClient(config.Token, f.logger), nil
    case "gitlab":
        return NewGitLabClient(config.Token, config.BaseURL, f.logger), nil
    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}
```

#### 3. 策略模式

```go
// 轮询策略接口
type PollingStrategy interface {
    ShouldPoll(repo Repository, lastCheck time.Time) bool
    NextPollTime(repo Repository) time.Time
}

// 固定间隔策略
type FixedIntervalStrategy struct {
    interval time.Duration
}

// 自适应策略
type AdaptiveStrategy struct {
    baseInterval time.Duration
    maxInterval  time.Duration
}
```

### 错误处理

#### 错误类型设计

```go
// 自定义错误类型
type RepoSentryError struct {
    Code      string      `json:"code"`
    Message   string      `json:"message"`
    Details   interface{} `json:"details,omitempty"`
    Cause     error       `json:"-"`
}

func (e *RepoSentryError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

// 预定义错误
var (
    ErrConfigValidation = &RepoSentryError{
        Code:    "CONFIG_VALIDATION_FAILED",
        Message: "configuration validation failed",
    }
    
    ErrRepositoryNotFound = &RepoSentryError{
        Code:    "REPOSITORY_NOT_FOUND", 
        Message: "repository not found",
    }
)
```

#### 错误处理模式

```go
// 包装错误
func (p *Poller) pollRepository(repo Repository) error {
    branches, err := p.gitClient.GetBranches(repo)
    if err != nil {
        return fmt.Errorf("failed to fetch branches for %s: %w", repo.Name, err)
    }
    
    // 处理逻辑...
    return nil
}

// 记录并处理错误
func (p *Poller) handleError(repo Repository, err error) {
    p.logger.WithField("repository", repo.Name).
             WithError(err).
             Error("polling failed")
    
    // 记录错误事件
    errorEvent := Event{
        Repository:   repo.Name,
        Type:        EventTypeError,
        ErrorMessage: err.Error(),
    }
    p.storage.Store(errorEvent)
}
```

### 并发模式

#### Worker Pool

```go
type WorkerPool struct {
    workers    int
    taskQueue  chan Task
    resultChan chan Result
    wg         sync.WaitGroup
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workers; i++ {
        wp.wg.Add(1)
        go wp.worker()
    }
}

func (wp *WorkerPool) worker() {
    defer wp.wg.Done()
    for task := range wp.taskQueue {
        result := task.Execute()
        wp.resultChan <- result
    }
}
```

#### Context 使用

```go
func (p *Poller) Start(ctx context.Context) error {
    ticker := time.NewTicker(p.config.Interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := p.pollAll(ctx); err != nil {
                p.logger.WithError(err).Error("polling cycle failed")
            }
        }
    }
}
```

## 🧪 测试策略

### 测试层次

#### 1. 单元测试

```go
// poller_test.go
func TestPoller_ShouldPollRepository(t *testing.T) {
    tests := []struct {
        name        string
        repo        Repository
        lastCheck   time.Time
        expected    bool
    }{
        {
            name: "should poll when last check is old",
            repo: Repository{Name: "test", PollingInterval: 5 * time.Minute},
            lastCheck: time.Now().Add(-10 * time.Minute),
            expected: true,
        },
        {
            name: "should not poll when last check is recent",
            repo: Repository{Name: "test", PollingInterval: 5 * time.Minute},
            lastCheck: time.Now().Add(-2 * time.Minute),
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            poller := NewPoller(nil, nil, nil)
            result := poller.shouldPoll(tt.repo, tt.lastCheck)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### 2. 集成测试

```go
// integration_test.go
func TestGitHubClientIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        t.Skip("GITHUB_TOKEN not set")
    }
    
    client := NewGitHubClient(token, logger.NewTestLogger())
    repo := Repository{
        URL: "https://github.com/octocat/Hello-World",
        Provider: "github",
    }
    
    branches, err := client.GetBranches(repo)
    require.NoError(t, err)
    assert.NotEmpty(t, branches)
}
```

#### 3. Mock 测试

```go
// 使用 testify/mock
type MockGitClient struct {
    mock.Mock
}

func (m *MockGitClient) GetBranches(repo Repository) ([]Branch, error) {
    args := m.Called(repo)
    return args.Get(0).([]Branch), args.Error(1)
}

func TestPoller_WithMockClient(t *testing.T) {
    mockClient := new(MockGitClient)
    mockStorage := new(MockStorage)
    
    // 设置 mock 期望
    expectedBranches := []Branch{
        {Name: "main", CommitSHA: "abc123"},
    }
    mockClient.On("GetBranches", mock.Anything).Return(expectedBranches, nil)
    
    poller := NewPoller(mockStorage, mockClient, logger.NewTestLogger())
    
    err := poller.PollRepository(testRepo)
    require.NoError(t, err)
    
    // 验证 mock 调用
    mockClient.AssertExpectations(t)
}
```

### 测试工具

#### 测试辅助函数

```go
// test/helpers.go
package test

func CreateTestConfig() *Config {
    return &Config{
        App: AppConfig{
            LogLevel: "debug",
            LogFormat: "text",
        },
        Polling: PollingConfig{
            Interval: 1 * time.Minute,
        },
        Storage: StorageConfig{
            Type: "sqlite",
            SQLite: SQLiteConfig{
                Path: ":memory:",
            },
        },
    }
}

func CreateTestRepository() Repository {
    return Repository{
        Name:        "test-repo",
        URL:         "https://github.com/test/repo",
        Provider:    "github",
        Token:       "test-token",
        BranchRegex: ".*",
    }
}
```

#### 测试数据管理

```go
// test/fixtures/
├── configs/
│   ├── valid-config.yaml
│   ├── invalid-config.yaml
│   └── minimal-config.yaml
├── responses/
│   ├── github-branches.json
│   ├── gitlab-projects.json
│   └── tekton-webhook.json
└── databases/
    └── test-data.sql
```

### 测试执行

```bash
# 运行所有测试
make test

# 运行单元测试
go test ./...

# 运行特定包的测试
go test ./internal/poller/

# 运行集成测试
go test -tags=integration ./...

# 生成覆盖率报告
make test-coverage

# 运行基准测试
go test -bench=. ./...

# 运行竞态检测
go test -race ./...
```

## 🔧 开发工作流

### 分支策略

我们使用 GitHub Flow 模式：

```
main (稳定分支)
  ↑
feature/add-new-provider    # 功能分支
feature/improve-logging     # 功能分支
bugfix/fix-memory-leak      # 修复分支
```

### 提交规范

使用 Conventional Commits 规范：

```bash
# 功能添加
git commit -m "feat(poller): add adaptive polling strategy"

# 修复 bug
git commit -m "fix(storage): resolve database lock issue"

# 文档更新
git commit -m "docs: update API documentation"

# 重构代码
git commit -m "refactor(gitclient): extract rate limiter interface"

# 性能优化
git commit -m "perf(poller): optimize branch filtering algorithm"

# 测试相关
git commit -m "test(trigger): add integration tests for Tekton"
```

### Code Review 检查清单

#### 代码质量
- [ ] 代码符合 Go 语言规范
- [ ] 函数和变量命名清晰
- [ ] 添加了必要的注释
- [ ] 错误处理恰当
- [ ] 没有硬编码值

#### 测试覆盖
- [ ] 添加了单元测试
- [ ] 测试覆盖关键路径
- [ ] 测试用例有代表性
- [ ] Mock 使用恰当

#### 性能考虑
- [ ] 没有明显的性能问题
- [ ] 正确使用 context
- [ ] 避免内存泄漏
- [ ] 数据库查询优化

#### 安全性
- [ ] 输入验证充分
- [ ] 敏感信息不在代码中
- [ ] SQL 注入防护
- [ ] 访问控制正确

### 开发命令

#### 代码生成

```bash
# 生成 mock 文件
go generate ./...

# 生成 Swagger 文档
make swagger

# 生成协议缓冲区文件（如果使用）
make protoc
```

#### 代码检查

```bash
# 代码格式化
make fmt

# 代码检查
make lint

# 导入整理
make imports

# 静态分析
make vet

# 安全检查
make security
```

#### 本地测试

```bash
# 启动本地环境
make dev-up

# 停止本地环境
make dev-down

# 重新构建并启动
make dev-restart

# 查看日志
make dev-logs
```

## 🚀 调试技巧

### 日志调试

```go
// 添加详细日志
logger.WithFields(logrus.Fields{
    "repository": repo.Name,
    "branch":     branch.Name,
    "operation":  "fetch_commit",
}).Debug("fetching commit information")

// 性能调试
start := time.Now()
result, err := operation()
logger.WithField("duration", time.Since(start)).
       Debug("operation completed")
```

### 性能分析

```bash
# 启用性能分析
go run cmd/reposentry/main.go run --config=config.yaml --pprof

# 分析 CPU 使用
go tool pprof http://localhost:8080/debug/pprof/profile

# 分析内存使用
go tool pprof http://localhost:8080/debug/pprof/heap

# 分析协程
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### 断点调试

#### 使用 Delve

```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试程序
dlv debug cmd/reposentry/main.go -- run --config=config.yaml

# 在 VS Code 中调试
# 使用 launch.json 配置
```

#### 调试配置示例

```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug RepoSentry",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/reposentry",
            "args": ["run", "--config=test-config.yaml"],
            "env": {
                "GITHUB_TOKEN": "your_token_here"
            }
        }
    ]
}
```

## 📦 发布流程

### 版本管理

使用语义化版本：

- **主版本号**: 不兼容的 API 修改
- **次版本号**: 向下兼容的功能性新增
- **修订版本号**: 向下兼容的问题修正

### 发布检查清单

#### 发布前检查
- [ ] 所有测试通过
- [ ] 代码质量检查通过
- [ ] 文档已更新
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新

#### 构建和测试
- [ ] 多平台构建成功
- [ ] Docker 镜像构建成功
- [ ] Helm Chart 测试通过
- [ ] 集成测试通过

#### 发布执行
```bash
# 创建发布标签
git tag -a v1.2.3 -m "Release version 1.2.3"

# 推送标签（触发 CI/CD）
git push origin v1.2.3

# 构建发布包
make release

# 发布 Docker 镜像
make docker-publish

# 发布 Helm Chart
make helm-publish
```

## 🤝 贡献指南

### 贡献流程

1. **Fork 仓库**
2. **创建功能分支**: `git checkout -b feature/amazing-feature`
3. **提交更改**: `git commit -m 'feat: add amazing feature'`
4. **推送分支**: `git push origin feature/amazing-feature`
5. **创建 Pull Request**

### PR 模板

```markdown
## 变更描述
简要描述本次 PR 的变更内容

## 变更类型
- [ ] 新功能
- [ ] Bug 修复
- [ ] 文档更新
- [ ] 重构
- [ ] 性能优化
- [ ] 其他

## 测试
- [ ] 添加了单元测试
- [ ] 添加了集成测试
- [ ] 手动测试通过

## 检查清单
- [ ] 代码符合项目规范
- [ ] 添加了必要的文档
- [ ] 更新了 CHANGELOG
- [ ] 所有测试通过
```

### 本地开发环境

```bash
# 设置开发环境
make dev-setup

# 安装 pre-commit 钩子
make install-hooks

# 运行开发服务器
make dev-server

# 重新加载配置
make dev-reload
```

---

希望这个开发指南能帮助你更好地参与 RepoSentry 项目的开发！如果有任何问题，请查看其他文档或提交 Issue。
