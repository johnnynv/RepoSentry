# RepoSentry Technical Architecture

## 🎯 Overview

RepoSentry is a lightweight, cloud-native Git repository monitoring sentinel designed specifically for the Tekton ecosystem. It adopts a modular architecture with intelligent polling strategies, high availability, and scalability.

## 🏗️ System Architecture

### Overall Architecture

```mermaid
graph TB
    subgraph "External Systems"
        GH[GitHub API]
        GL[GitLab API]
        TK[Tekton EventListener]
    end
    
    subgraph "RepoSentry Core"
        subgraph "API Layer"
            API[REST API Server]
            SW[Swagger UI]
        end
        
        subgraph "Business Logic Layer"
            RT[Runtime Manager]
            PL[Poller]
            TR[Trigger]
            GC[Git Client]
        end
        
        subgraph "Infrastructure Layer"
            CF[Config Manager]
            ST[Storage]
            LG[Logger]
        end
    end
    
    subgraph "Data Storage"
        DB[(SQLite Database)]
        FS[File System]
    end
    
    subgraph "Deployment Environment"
        SY[Systemd]
        DK[Docker]
        K8[Kubernetes]
    end
    
    %% External connections
    PL --> GH
    PL --> GL
    TR --> TK
    
    %% Internal connections
    API --> RT
    RT --> PL
    RT --> TR
    RT --> GC
    RT --> CF
    RT --> ST
    
    PL --> GC
    TR --> ST
    CF --> ST
    
    %% Data storage
    ST --> DB
    LG --> FS
    
    %% Deployment
    RT --> SY
    RT --> DK
    RT --> K8
```

### Core Components

#### 1. Runtime Manager
- **Responsibility**: Component lifecycle management, service orchestration
- **Functions**: Start/stop, health checks, dependency injection
- **Interfaces**: `Runtime`, `Component`

#### 2. Poller
- **Responsibility**: Repository change detection, event generation
- **Functions**: Intelligent polling, branch filtering, state caching
- **Strategy**: API-first with git command fallback

#### 3. Trigger
- **Responsibility**: Event processing, external system triggering
- **Functions**: Tekton integration, retry mechanism, idempotency guarantee

#### 4. Git Client
- **Responsibility**: Git provider API abstraction
- **Functions**: GitHub/GitLab API, rate limiting, error handling

#### 5. Storage
- **Responsibility**: Data persistence, state management
- **Functions**: SQLite abstraction, database migration, transaction management

#### 6. Config Manager
- **Responsibility**: Configuration loading, validation, hot reload
- **Functions**: YAML parsing, environment variable expansion, configuration validation

## 🔄 Processing Flow

### Core Workflow

```mermaid
sequenceDiagram
    participant S as Scheduler
    participant P as Poller
    participant GC as GitClient
    participant ST as Storage
    participant T as Trigger
    participant TK as Tekton
    
    loop Every polling cycle
        S->>P: Trigger polling
        P->>ST: Get repository list
        ST-->>P: Return repository configuration
        
        loop Each repository
            P->>ST: Check cache status
            alt Cache valid
                ST-->>P: Skip polling
            else Cache expired
                P->>GC: Get branch list
                GC->>GH/GL: API call
                alt API success
                    GH/GL-->>GC: Branch data
                    GC-->>P: Return branches
                else API failure
                    GC->>Git: git ls-remote
                    Git-->>GC: Branch data
                    GC-->>P: Return branches
                end
                
                P->>P: Filter branches
                P->>ST: Compare state
                alt Has changes
                    P->>ST: Generate event
                    P->>T: Trigger processing
                    T->>TK: Send webhook
                    TK-->>T: Confirm receipt
                    T->>ST: Update event status
                end
                P->>ST: Update repository state
            end
        end
    end
```

## 🏗️ Component Design

### 1. Runtime Manager

#### Architecture Design

```go
type Runtime interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    GetStatus() *RuntimeStatus
    Health() error
}

type Component interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    Name() string
}
```

### 2. Poller Component

#### Multi-layer Architecture

```
┌─────────────────────────────────────┐
│           Scheduler                 │  ← Scheduler: Manage polling cycles
├─────────────────────────────────────┤
│        Branch Monitor               │  ← Branch monitoring: Handle single repository
├─────────────────────────────────────┤
│        Event Generator              │  ← Event generation: Change detection and event creation
├─────────────────────────────────────┤
│         Git Client                  │  ← Client: API calls and fallback
└─────────────────────────────────────┘
```

### 3. Git Client

#### Client Architecture

```mermaid
classDiagram
    class ClientFactory {
        +CreateClient(provider string) GitClient
    }
    
    class GitClient {
        <<interface>>
        +GetBranches(repo Repository) []Branch
        +GetCommit(repo Repository, branch string) Commit
        +Health() error
    }
    
    class GitHubClient {
        -token string
        -rateLimiter RateLimiter
        +GetBranches() []Branch
    }
    
    class GitLabClient {
        -token string
        -baseURL string
        -rateLimiter RateLimiter
        +GetBranches() []Branch
    }
    
    class FallbackClient {
        -gitCommand GitCommand
        +GetBranches() []Branch
    }
    
    ClientFactory --> GitClient
    GitClient <|-- GitHubClient
    GitClient <|-- GitLabClient
    GitClient <|-- FallbackClient
```

## 🔧 Technology Stack

### Core Technologies

| Component | Technology Choice | Reason |
|-----------|-------------------|--------|
| **Language** | Go 1.21+ | High performance, concurrency support, cloud-native ecosystem |
| **Web Framework** | Gorilla Mux | Lightweight, standard library compatible, flexible routing |
| **Database** | SQLite | Zero dependencies, embedded, transaction support |
| **Configuration** | YAML + Viper | Human readable, strongly typed, environment variable support |
| **Logging** | Logrus | Structured logging, multiple output formats, excellent performance |
| **HTTP Client** | net/http | Standard library, controllable, context support |
| **Container** | Docker | Standardized, portable, easy deployment |
| **Orchestration** | Kubernetes | Cloud-native, auto-scaling, high availability |

### Performance Metrics

| Metric | Target | Current |
|--------|--------|---------|
| **Startup Time** | < 5s | ~2s |
| **Memory Usage** | < 512MB | ~128MB |
| **API Response Time** | < 100ms | ~50ms |
| **Polling Latency** | < 30s | ~15s |
| **Concurrent Processing** | 100+ repositories | Tested pass |

## 🔐 Security Architecture

### Security Layers

```mermaid
graph TB
    subgraph "Application Security"
        A1[Input Validation]
        A2[Output Encoding]
        A3[Error Handling]
    end
    
    subgraph "Authentication & Authorization"
        B1[API Token]
        B2[Permission Control]
        B3[Access Restriction]
    end
    
    subgraph "Transport Security"
        C1[HTTPS Only]
        C2[Certificate Verification]
        C3[Encrypted Communication]
    end
    
    subgraph "Data Security"
        D1[Sensitive Data Encryption]
        D2[Database Security]
        D3[Log Sanitization]
    end
    
    subgraph "Infrastructure Security"
        E1[Container Security]
        E2[Network Isolation]
        E3[Least Privilege]
    end
```

## 📊 Monitoring Architecture

### Observability Layers

```mermaid
mindmap
  root((Observability))
    Logging
      Structured Logging
      Log Aggregation
      Log Analysis
      Alert Rules
    Metrics
      Runtime Metrics
      Business Metrics
      Performance Metrics
      Resource Metrics
    Tracing
      Request Tracing
      Component Dependencies
      Performance Analysis
      Error Location
    Health Checks
      Component Status
      Dependency Checks
      Readiness Status
      Liveness Probes
```

## 🚀 Scalability Design

### Horizontal Scaling

```mermaid
graph LR
    subgraph "Load Balancer"
        LB[Load Balancer]
    end
    
    subgraph "RepoSentry Cluster"
        RS1[RepoSentry-1<br/>Poller + API]
        RS2[RepoSentry-2<br/>API Only]
        RS3[RepoSentry-3<br/>API Only]
    end
    
    subgraph "Shared Storage"
        DB[(Shared SQLite)]
        FS[Shared File System]
    end
    
    LB --> RS1
    LB --> RS2
    LB --> RS3
    
    RS1 --> DB
    RS2 --> DB
    RS3 --> DB
    
    RS1 --> FS
    RS2 --> FS
    RS3 --> FS
```

## 🔄 Deployment Architecture

### Multi-environment Deployment

```mermaid
graph TB
    subgraph "Development Environment"
        DEV[Local Development]
        DEV_DB[(SQLite)]
        DEV --> DEV_DB
    end
    
    subgraph "Testing Environment"
        TEST[Docker Compose]
        TEST_DB[(SQLite Volume)]
        TEST --> TEST_DB
    end
    
    subgraph "Pre-production Environment"
        STAGE[Kubernetes]
        STAGE_DB[(PVC SQLite)]
        STAGE --> STAGE_DB
    end
    
    subgraph "Production Environment"
        PROD1[RepoSentry Pod 1]
        PROD2[RepoSentry Pod 2]
        PROD3[RepoSentry Pod 3]
        PROD_DB[(Shared Storage)]
        
        PROD1 --> PROD_DB
        PROD2 --> PROD_DB
        PROD3 --> PROD_DB
    end
```

## 🛠️ Development Architecture

### Code Organization

```
RepoSentry/
├── cmd/reposentry/           # CLI entry point
├── internal/                 # Internal packages
│   ├── api/                 # REST API server
│   ├── config/              # Configuration management
│   ├── gitclient/           # Git client
│   ├── poller/              # Polling logic
│   ├── runtime/             # Runtime management
│   ├── storage/             # Storage layer
│   └── trigger/             # Trigger system
├── pkg/                      # Public packages
│   ├── logger/              # Logging component
│   ├── types/               # Type definitions
│   └── utils/               # Utility functions
├── deployments/              # Deployment configurations
├── docs/                     # Documentation
├── examples/                 # Example configurations
└── test/                     # Test files
```

### Design Principles

#### 1. SOLID Principles
- **Single Responsibility**: Each component handles one functionality
- **Open/Closed**: Support extension, refuse modification
- **Liskov Substitution**: Interfaces can replace implementations
- **Interface Segregation**: Fine-grained interface design
- **Dependency Inversion**: Depend on abstractions, not concrete implementations

#### 2. 12-Factor App
- **Configuration**: Environment variables and configuration file separation
- **Dependencies**: Explicitly declare and isolate dependencies
- **Config**: Store configuration in environment
- **Backing Services**: Services as attached resources
- **Logs**: Logs as event streams

#### 3. Cloud-native Principles
- **Stateless**: Application layer stateless design
- **Observable**: Health checks, metrics, logs
- **Scalable**: Horizontal scaling support
- **Fault-tolerant**: Graceful degradation and error recovery

## 📈 Performance Optimization

### Polling Optimization

```go
// Adaptive polling interval
type AdaptivePoller struct {
    baseInterval    time.Duration
    maxInterval     time.Duration
    backoffFactor   float64
    activityWindow  time.Duration
}

func (p *AdaptivePoller) NextInterval(repo *Repository) time.Duration {
    // Adjust polling interval based on repository activity
    activity := p.getRecentActivity(repo)
    if activity > 0.8 {
        return p.baseInterval // High activity, frequent polling
    } else if activity < 0.2 {
        return p.maxInterval // Low activity, reduce frequency
    }
    return time.Duration(float64(p.baseInterval) * (1 + activity))
}
```

### Caching Strategy

```go
type CacheStrategy interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    InvalidatePattern(pattern string)
}

// Branch cache
type BranchCache struct {
    cache    map[string]CacheEntry
    ttl      time.Duration
    maxSize  int
}
```

## 🔮 Future Architecture Evolution

### Short-term Goals (3-6 months)

1. **Multi-database Support**: PostgreSQL, MySQL
2. **Message Queue**: Redis, RabbitMQ integration
3. **Configuration Hot Reload**: Real-time configuration changes
4. **Metrics Monitoring**: Prometheus integration

### Medium-term Goals (6-12 months)

1. **Distributed Architecture**: Multi-node deployment
2. **Plugin System**: Custom triggers
3. **Web UI**: Management interface
4. **Alert System**: Multi-channel notifications

### Long-term Goals (12+ months)

1. **AI Intelligence**: Intelligent polling strategies
2. **Multi-cloud Support**: AWS, Azure, GCP
3. **GraphQL API**: Flexible query interface
4. **Microservice Architecture**: Service decomposition

---

## 📚 Related Documentation

- [Quick Start Guide](QUICKSTART.md)
- [User Manual](USER_MANUAL.md)
- [Deployment Guide](../../deployments/README.md)
- [Development Guide](DEVELOPMENT.md)
- [API Documentation](../API_EXAMPLES.md)
