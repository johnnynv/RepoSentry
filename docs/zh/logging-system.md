# RepoSentry 企业级日志系统设计文档

## 📋 系统概述

RepoSentry 企业级日志系统是一个高性能、可扩展、结构化的日志解决方案，专为微服务架构和云原生环境设计。该系统提供统一的日志管理、上下文传播、性能监控和错误追踪功能。

## 🏗️ 架构设计

### 核心组件架构

```
Enterprise Logging System
├── Logger Manager (日志管理器)
│   ├── Root Logger (根日志器)
│   ├── Context Cache (上下文缓存)
│   ├── Hook System (钩子系统)
│   └── Business Operations (业务操作)
├── Context System (上下文系统)
│   ├── LogContext (日志上下文)
│   ├── Go Context Integration (Go上下文集成)
│   └── Context Propagation (上下文传播)
├── Business Logger (业务日志器)
│   ├── Repository Operations (仓库操作)
│   ├── Event Operations (事件操作)
│   ├── Trigger Operations (触发器操作)
│   ├── API Operations (API操作)
│   └── System Operations (系统操作)
└── Performance & Monitoring (性能监控)
    ├── Performance Hook (性能钩子)
    ├── Error Tracking Hook (错误追踪钩子)
    └── Metrics Collection (指标收集)
```

### 数据流设计

```
Application Start
    ↓
Logger Manager 初始化
    ↓
Business Logger 创建
    ↓
Context 传播
    ↓
Operation Logging
    ↓
Hook Processing
    ↓
Output (File/Console/Remote)
```

## 📊 核心特性

### 1. 结构化日志
- **JSON格式输出**：机器可读，便于解析和分析
- **标准化字段**：component, module, operation, repository, event_id等
- **自定义字段**：支持业务特定字段扩展

### 2. 上下文管理
- **Go Context集成**：与Go标准库无缝集成
- **跨组件传播**：请求ID、操作ID、用户ID等自动传播
- **分层上下文**：支持上下文继承和合并

### 3. 性能监控
- **操作耗时追踪**：自动记录操作执行时间
- **性能告警**：慢操作自动标记
- **资源使用监控**：内存、CPU使用情况

### 4. 错误追踪
- **自动错误捕获**：所有ERROR级别日志自动追踪
- **错误上下文**：保留错误发生时的完整上下文
- **错误统计**：错误频率和趋势分析

## 🎯 使用场景

### 场景1：仓库轮询操作
```go
// 开始业务操作
op := loggerManager.StartOperation(ctx, "poller", "repository", "poll")
op.WithRepository("my-repo", "github")

// 记录轮询开始
businessLogger.LogRepositoryPollStart(ctx, "my-repo", "github", "https://github.com/...")

// 处理轮询逻辑...
// ...

// 记录成功完成
businessLogger.LogRepositoryPollSuccess(ctx, "my-repo", 3, time.Since(start))
```

### 场景2：事件触发操作
```go
// 记录触发尝试
businessLogger.LogTriggerAttempt(ctx, "event-123", "my-repo")

// 执行触发逻辑...
result, err := trigger.SendEvent(ctx, event)

if err != nil {
    businessLogger.LogTriggerError(ctx, "event-123", "my-repo", err, 500)
} else {
    businessLogger.LogTriggerSuccess(ctx, "event-123", "my-repo", 200, duration)
}
```

### 场景3：API请求处理
```go
// 记录请求开始
businessLogger.LogAPIRequest(ctx, "GET", "/api/events", userAgent, remoteAddr)

// 处理请求...
// ...

// 记录响应
businessLogger.LogAPIResponse(ctx, "GET", "/api/events", 200, duration)
```

## 📈 性能指标

### 日志输出格式示例

```json
{
  "timestamp": "2025-08-22T10:30:45.123Z",
  "level": "info",
  "message": "Repository poll completed successfully",
  "component": "poller",
  "module": "repository",
  "operation": "poll_complete",
  "repository": "taap-poc-gitlab",
  "provider": "gitlab",
  "change_count": 3,
  "duration": "2.345s",
  "duration_ms": 2345,
  "success": true,
  "trace_id": "abc123def456",
  "request_id": "req-789"
}
```

### Hook系统日志增强
```json
{
  "timestamp": "2025-08-22T10:30:45.123Z",
  "level": "info",
  "message": "Event triggered successfully",
  "component": "trigger",
  "module": "tekton",
  "operation": "send_event",
  "event_id": "event-456",
  "repository": "my-repo",
  "status_code": 202,
  "duration": "1.234s",
  "duration_ms": 1234,
  "success": true,
  "performance_alert": null,
  "error_tracked": false
}
```

## 🔧 配置管理

### 日志级别配置
```yaml
logging:
  level: "debug"          # trace, debug, info, warn, error
  format: "json"          # json, text
  output: "./logs/app.log" # file path or "stdout"
  rotation:
    max_size: 100         # MB
    max_backups: 10
    max_age: 30           # days
    compress: true
```

### 组件级别配置
```yaml
logging:
  components:
    poller:
      level: "debug"
      enabled: true
    trigger:
      level: "info"
      enabled: true
    api:
      level: "info"
      enabled: true
```

## 📋 最佳实践

### 1. 日志级别使用指南
- **TRACE**：详细的调试信息，仅开发环境使用
- **DEBUG**：调试信息，测试环境使用
- **INFO**：正常业务流程信息，生产环境标准级别
- **WARN**：警告信息，需要关注但不影响功能
- **ERROR**：错误信息，需要立即处理

### 2. 字段命名规范
- **component**：组件名称（poller, trigger, api, gitclient）
- **module**：模块名称（repository, scheduler, tekton）
- **operation**：操作名称（start, stop, poll, send_event）
- **repository**：仓库名称
- **provider**：提供商（github, gitlab）
- **duration**：操作耗时
- **success**：操作是否成功

### 3. 上下文传播
- 在函数间传递context.Context
- 使用LogContext添加业务字段
- 避免在日志中硬编码上下文信息

### 4. 性能考虑
- 使用结构化字段而非字符串拼接
- 避免在热路径中创建复杂对象
- 合理设置日志级别以控制输出量

## 🚀 扩展功能

### 1. 远程日志传输
```go
// 可扩展支持ELK、Fluentd等
type RemoteHook struct {
    endpoint string
    client   *http.Client
}

func (h *RemoteHook) Fire(entry *logrus.Entry) error {
    // 发送到远程日志系统
    return nil
}
```

### 2. 指标集成
```go
// 集成Prometheus指标
type MetricsHook struct {
    counter   prometheus.Counter
    histogram prometheus.Histogram
}
```

### 3. 分布式追踪
```go
// 集成OpenTelemetry
type TracingHook struct {
    tracer trace.Tracer
}
```

## 📊 监控告警

### 关键指标
- **日志错误率**：ERROR级别日志占比
- **慢操作数量**：超过阈值的操作数量
- **日志吞吐量**：每秒日志产生数量
- **存储使用量**：日志文件大小和增长趋势

### 告警规则
```yaml
alerts:
  - name: "high_error_rate"
    condition: "error_rate > 5%"
    action: "notify_ops_team"
  
  - name: "slow_operations"
    condition: "duration > 5s"
    action: "performance_alert"
  
  - name: "disk_usage"
    condition: "log_disk_usage > 80%"
    action: "cleanup_old_logs"
```

## 🔒 安全考虑

### 数据脱敏
- 自动检测和脱敏敏感信息（密码、令牌）
- 用户数据匿名化
- API密钥部分隐藏

### 访问控制
- 日志文件权限控制
- 审计日志访问记录
- 数据保留策略

## 📚 API参考

### Manager API
```go
// 创建日志管理器
manager, err := logger.NewManager(config)

// 获取组件日志器
componentLogger := manager.ForComponent("poller")

// 获取模块日志器
moduleLogger := manager.ForModule("poller", "scheduler")

// 创建业务操作
op := manager.StartOperation(ctx, "poller", "repository", "poll")
```

### BusinessLogger API
```go
// 创建业务日志器
businessLogger := logger.NewBusinessLogger(manager)

// 仓库操作日志
businessLogger.LogRepositoryPollStart(ctx, repo, provider, url)
businessLogger.LogRepositoryPollSuccess(ctx, repo, changeCount, duration)
businessLogger.LogRepositoryPollError(ctx, repo, err, duration)

// 事件操作日志
businessLogger.LogEventCreated(ctx, eventID, repo, branch, changeType)
businessLogger.LogTriggerSuccess(ctx, eventID, repo, statusCode, duration)
```

## 🔄 版本更新

### v1.0.0 (当前版本)
- ✅ 基础日志管理器
- ✅ 上下文系统
- ✅ 业务日志器
- ✅ 性能监控Hook
- ✅ 错误追踪Hook

### v1.1.0 (计划中)
- 🔄 远程日志传输
- 🔄 指标系统集成
- 🔄 分布式追踪支持
- 🔄 日志采样功能

### v1.2.0 (规划中)
- 📋 机器学习异常检测
- 📋 自动化日志分析
- 📋 智能告警优化
- 📋 性能自动优化

---

*该文档描述了RepoSentry企业级日志系统的完整设计和实现。如有疑问或建议，请联系开发团队。*



## 📚 实施指南

2. 创建新的核心组件
   - ✅ pkg/logger/context.go - 上下文管理
   - ✅ pkg/logger/manager.go - 日志管理器
   - ✅ pkg/logger/business.go - 业务日志接口
```

### 第二阶段：应用集成 🔄
```
1. 修改应用启动流程
   - 🔄 cmd/reposentry/run.go - 集成Logger Manager
   - 📋 删除旧的logger初始化代码
   - 📋 传递LoggerManager到Runtime

2. 更新Runtime Manager
   - 📋 internal/runtime/manager.go - 使用新日志系统
   - 📋 移除GetDefaultLogger()调用
   - 📋 传递logger到所有组件
```

### 第三阶段：组件改造 📋
```
1. Poller组件
   - 📋 使用BusinessLogger记录轮询操作
   - 📋 添加详细的仓库轮询日志
   - 📋 记录分支变化检测过程
   - 📋 追踪事件生成流程

2. Trigger组件
   - 📋 记录Tekton触发详情
   - 📋 添加HTTP请求/响应日志
   - 📋 错误重试机制日志

3. API组件
   - 📋 请求/响应中间件日志
   - 📋 性能监控集成
   - 📋 错误处理增强

4. Git客户端
   - 📋 API调用详情记录
   - 📋 认证和权限日志
   - 📋 网络错误追踪
```

### 第四阶段：验证测试 📋
```
1. 功能验证
   - 📋 日志输出格式正确性
   - 📋 上下文传播完整性
   - 📋 性能指标准确性

2. 性能测试
   - 📋 日志系统性能影响
   - 📋 内存使用情况
   - 📋 磁盘I/O优化

3. 集成测试
   - 📋 端到端业务流程日志
   - 📋 错误场景日志验证
   - 📋 高并发情况测试
```

## 🔧 技术实施细节

### 1. 应用启动改造

**当前问题：**
```go
// 旧的方式 - 分散的logger初始化
appLogger, err := initializeLogger()
configManager := config.NewManager(appLogger)
```

**新的架构：**
```go
// 企业级方式 - 统一的logger管理
loggerManager, err := logger.NewManager(loggerConfig)
businessLogger := logger.NewBusinessLogger(loggerManager)
configManager := config.NewManager(loggerManager.GetRootLogger())
```

### 2. 运行时管理器改造

**当前问题：**
```go
// 各组件独立创建logger
func NewRuntimeManager(cfg *types.Config) (*RuntimeManager, error) {
    runtimeLogger := logger.GetDefaultLogger().WithFields(...)
}
```

**新的架构：**
```go
// 统一的logger传递
func NewRuntimeManager(cfg *types.Config, loggerManager *logger.Manager) (*RuntimeManager, error) {
    runtimeLogger := loggerManager.ForComponent("runtime")
    businessLogger := logger.NewBusinessLogger(loggerManager)
}
```

### 3. 组件构造函数标准化

**标准模式：**
```go
// 所有组件构造函数统一接受logger参数
func NewPoller(config PollerConfig, storage storage.Storage, 
               trigger trigger.Trigger, logger *logger.Entry) *PollerImpl

func NewTektonTrigger(config TriggerConfig, logger *logger.Entry) (*TektonTrigger, error)

func NewAPIServer(port int, configManager *config.Manager, 
                  storage storage.Storage, logger *logger.Entry) *Server
```

### 4. 业务操作日志记录

**轮询操作示例：**
```go
func (p *PollerImpl) PollRepository(ctx context.Context, repo types.Repository) (*PollResult, error) {
    // 开始业务操作
    op := p.loggerManager.StartOperation(ctx, "poller", "repository", "poll")
    op.WithRepository(repo.Name, repo.Provider)
    
    // 记录开始
    p.businessLogger.LogRepositoryPollStart(ctx, repo.Name, repo.Provider, repo.URL)
    
    // 执行业务逻辑...
    changes, err := p.branchMonitor.CheckBranches(ctx, repo)
    if err != nil {
        p.businessLogger.LogRepositoryPollError(ctx, repo.Name, err, time.Since(start))
        return nil, err
    }
    
    // 记录成功
    p.businessLogger.LogRepositoryPollSuccess(ctx, repo.Name, len(changes), time.Since(start))
    return result, nil
}
```

## 📊 预期效果

### 日志输出对比

**改造前：**
```
2025-08-22T10:30:45Z INFO Starting repository poll repository=my-repo
2025-08-22T10:30:47Z INFO Poll completed repository=my-repo
```

**改造后：**
```json
{
  "timestamp": "2025-08-22T10:30:45.123Z",
  "level": "info",
  "message": "Starting repository poll",
  "component": "poller",
  "module": "repository", 
  "operation": "poll_start",
  "repository": "my-repo",
  "provider": "github",
  "url": "https://github.com/user/my-repo.git",
  "trace_id": "abc123",
  "request_id": "req-456"
}

{
  "timestamp": "2025-08-22T10:30:47.456Z",
  "level": "info",
  "message": "Repository poll completed successfully",
  "component": "poller",
  "module": "repository",
  "operation": "poll_complete", 
  "repository": "my-repo",
  "provider": "github",
  "change_count": 3,
  "duration": "2.333s",
  "duration_ms": 2333,
  "success": true,
  "trace_id": "abc123",
  "request_id": "req-456"
}
```

### 业务流程可视化

改造后的日志系统将支持完整的业务流程追踪：

```
Request ID: req-456
├── Repository Poll Started (poller.repository.poll_start)
├── Branch Changes Detected (poller.branch_monitor.detect_changes) 
├── Events Generated (poller.event_generator.generate_events)
├── Event Stored (storage.event.create)
├── Trigger Attempted (trigger.tekton.send_event)
└── Trigger Successful (trigger.tekton.send_event_complete)
```

## 🎯 成功指标

### 技术指标
- **日志结构化率**: 100% JSON格式
- **上下文传播率**: 所有业务操作包含完整上下文
- **性能开销**: < 2% CPU和内存增长
- **存储效率**: 相比文本日志减少30%存储空间

### 业务指标  
- **问题诊断时间**: 减少80%
- **性能监控覆盖**: 100%关键操作
- **错误追踪率**: 100%错误包含完整上下文
- **运维效率**: 自动化分析和告警

## 🚀 下一步行动

1. **立即开始** - 完成应用启动流程改造
2. **并行进行** - Runtime Manager和主要组件改造
3. **逐步验证** - 每个组件改造后立即测试
4. **持续优化** - 根据实际使用情况调整配置

---

*这个实施指南将指导我们将RepoSentry升级为具有企业级日志能力的现代应用。*



## 🔧 快速参考


### 基本日志记录
```go
// 组件级日志
componentLogger := loggerManager.ForComponent("poller")
componentLogger.Info("Component started")

// 模块级日志
moduleLogger := loggerManager.ForModule("poller", "scheduler")  
moduleLogger.Debug("Scheduler processing")

// 业务操作日志
op := loggerManager.StartOperation(ctx, "poller", "repository", "poll")
op.WithRepository("my-repo", "github").Success("Poll completed")
```

## 📋 标准字段规范

### 必须字段
```go
Fields{
    "component": "poller",      // 组件名称
    "module":    "repository",  // 模块名称  
    "operation": "poll",        // 操作名称
}
```

### 业务字段
```go
Fields{
    "repository":  "my-repo",           // 仓库名称
    "provider":    "github",            // 提供商
    "branch":      "main",              // 分支名称
    "event_id":    "evt-123",           // 事件ID
    "request_id":  "req-456",           // 请求ID
    "duration":    time.Duration,       // 操作耗时
    "duration_ms": int64,               // 毫秒耗时
    "success":     true,                // 是否成功
}
```

### 性能字段
```go
Fields{
    "start_time":        time.Time,     // 开始时间
    "duration":          time.Duration, // 总耗时
    "duration_ms":       int64,         // 毫秒
    "duration_ns":       int64,         // 纳秒
    "performance_alert": "slow_op",     // 性能告警
}
```

## 🎯 业务日志API

### 仓库操作
```go
// 开始轮询
businessLogger.LogRepositoryPollStart(ctx, "my-repo", "github", "https://...")

// 轮询成功
businessLogger.LogRepositoryPollSuccess(ctx, "my-repo", 3, duration)

// 轮询失败
businessLogger.LogRepositoryPollError(ctx, "my-repo", err, duration)
```

### 分支操作
```go
// 分支变化
businessLogger.LogBranchChange(ctx, "my-repo", "main", "updated", 
                              "old-sha", "new-sha", false)

// 变化检测
businessLogger.LogBranchChangesDetected(ctx, "my-repo", 3)
```

### 事件操作
```go
// 事件创建
businessLogger.LogEventCreated(ctx, "evt-123", "my-repo", "main", "updated")

// 事件生成
businessLogger.LogEventGeneration(ctx, "my-repo", 3, duration)

// 生成失败
businessLogger.LogEventGenerationError(ctx, "my-repo", err)
```

### 触发操作
```go
// 触发尝试
businessLogger.LogTriggerAttempt(ctx, "evt-123", "my-repo")

// 触发成功
businessLogger.LogTriggerSuccess(ctx, "evt-123", "my-repo", 202, duration)

// 触发失败
businessLogger.LogTriggerError(ctx, "evt-123", "my-repo", err, 500)
```

### API操作
```go
// 请求开始
businessLogger.LogAPIRequest(ctx, "GET", "/api/events", userAgent, remoteAddr)

// 响应完成
businessLogger.LogAPIResponse(ctx, "GET", "/api/events", 200, duration)

// 请求错误
businessLogger.LogAPIError(ctx, "GET", "/api/events", err, 500)
```

### 系统操作
```go
// 组件启动
businessLogger.LogComponentStart(ctx, "poller", "scheduler", config)

// 组件停止
businessLogger.LogComponentStop(ctx, "poller", "scheduler", uptime)

// 组件错误
businessLogger.LogComponentError(ctx, "poller", "scheduler", err)

// 健康检查
businessLogger.LogComponentHealth(ctx, "poller", true, checks)
```

## 🔧 高级用法

### 业务操作模式
```go
// 开始复杂业务操作
op := loggerManager.StartOperation(ctx, "poller", "repository", "full_poll")
op.WithRepository("my-repo", "github")
op.WithEvent("evt-123")

// 记录进度
op.Info("Checking branches")
op.Info("Generating events", Fields{"event_count": 3})

// 完成操作
op.Success("Poll completed successfully", Fields{
    "total_changes": 3,
    "events_created": 3,
    "triggers_sent": 3,
})
```

### 上下文传播
```go
// 创建带上下文的Context
ctx := logger.WithContext(context.Background(), logger.LogContext{
    Component:  "poller",
    Repository: "my-repo", 
    RequestID:  "req-123",
})

// 传递给其他函数
func processRepository(ctx context.Context) {
    // 自动获取上下文信息
    logger := loggerManager.WithGoContext(ctx)
    logger.Info("Processing repository") // 自动包含上下文
}
```

### 错误处理模式
```go
// 带上下文的错误记录
op := loggerManager.StartOperation(ctx, "trigger", "tekton", "send_event")
result, err := sendToTekton(event)

if err != nil {
    op.Fail("Failed to send event to Tekton", err, Fields{
        "event_id": event.ID,
        "attempt":  1,
        "retryable": isRetryable(err),
    })
    return err
}

op.Success("Event sent successfully", Fields{
    "status_code": result.StatusCode,
    "response_time": result.Duration,
})
```

## 📊 日志级别指南

```go
// TRACE - 详细调试信息（开发环境）
logger.Trace("Detailed execution flow")

// DEBUG - 调试信息（测试环境）  
logger.Debug("Variable values and state")

// INFO - 正常业务流程（生产环境标准）
logger.Info("Operation completed successfully")

// WARN - 警告但不影响功能
logger.Warn("Deprecated API used")

// ERROR - 错误需要关注
logger.Error("Operation failed", err)
```

## ⚡ 性能优化

### 避免昂贵操作
```go
// ❌ 避免字符串拼接
logger.Info("Processing repo: " + repo.Name)

// ✅ 使用结构化字段
logger.WithField("repository", repo.Name).Info("Processing repository")

// ❌ 避免复杂对象序列化
logger.WithField("config", complexObject).Info("Starting")

// ✅ 选择关键字段
logger.WithFields(Fields{
    "timeout": config.Timeout,
    "retries": config.MaxRetries,
}).Info("Starting")
```

### 条件日志
```go
// 昂贵的debug日志使用条件检查
if logger.Level <= logrus.DebugLevel {
    expensiveData := computeExpensiveData()
    logger.WithField("data", expensiveData).Debug("Debug info")
}
```

## 🎛️ 配置示例

### 基本配置
```go
config := logger.Config{
    Level:  "info",
    Format: "json", 
    Output: "./logs/app.log",
    File: logger.FileConfig{
        MaxSize:    100,  // MB
        MaxBackups: 10,
        MaxAge:     30,   // days
        Compress:   true,
    },
}
```

### 生产环境配置
```go
config := logger.Config{
    Level:  "info",
    Format: "json",
    Output: "./logs/reposentry.log",
    File: logger.FileConfig{
        MaxSize:    500,  // MB
        MaxBackups: 20,
        MaxAge:     90,   // days
        Compress:   true,
    },
}
```

---

*这个快速参考将帮助开发团队快速掌握新的企业级日志系统。*

