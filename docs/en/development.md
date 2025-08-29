# RepoSentry Development Guide

## 🛠️ Development Environment Setup

### System Requirements

- **Go**: 1.21 or higher
- **Git**: 2.0+ 
- **Make**: GNU Make 4.0+
- **Docker**: 20.10+ (optional, for container testing)
- **SQLite**: 3.35+ (usually system built-in)

### Environment Preparation

#### 1. Clone Repository

```bash
git clone https://github.com/johnnynv/RepoSentry.git
cd RepoSentry
```

#### 2. Install Dependencies

```bash
# Download Go module dependencies
go mod download

# Install development tools
make dev-tools
```

#### 3. Setup Development Environment Variables

```bash
# Create .env file
cat > .env << EOF
# GitHub Token (for testing)
export GITHUB_TOKEN="ghp_your_github_token_here"

# GitLab Token (for testing)
export GITLAB_TOKEN="glpat-your_gitlab_token_here"

# Test Tekton EventListener URL
export TEKTON_TEST_URL="http://localhost:8080/test-webhook"
EOF

# Load environment variables
source .env
```

#### 4. Verify Environment

```bash
# Check Go environment
go version

# Check dependencies
go mod verify

# Run tests
make test

# Build project
make build
```

## 📁 Project Structure Details

### Directory Organization

```
RepoSentry/
├── cmd/reposentry/           # Application entry point
│   ├── main.go              # Main function
│   ├── root.go              # Cobra root command
│   ├── run.go               # Run command
│   ├── config.go            # Config command
│   └── ...                  # Other CLI commands
├── internal/                 # Internal business logic (not exposed externally)
│   ├── api/                 # REST API server
│   │   ├── server.go        # HTTP server
│   │   ├── router.go        # Route configuration
│   │   ├── middleware/      # Middleware
│   │   └── handlers/        # API handlers
│   ├── config/              # Configuration management
│   │   ├── config.go        # Configuration structure
│   │   ├── loader.go        # Configuration loader
│   │   └── validator.go     # Configuration validator
│   ├── gitclient/           # Git client
│   │   ├── github.go        # GitHub API client
│   │   ├── gitlab.go        # GitLab API client
│   │   ├── fallback.go      # Git command fallback
│   │   └── ratelimit.go     # Rate limiter
│   ├── poller/              # Polling logic
│   │   ├── poller.go        # Poller interface
│   │   ├── scheduler.go     # Scheduler
│   │   ├── monitor.go       # Branch monitor
│   │   └── events.go        # Event generator
│   ├── runtime/             # Runtime management
│   │   ├── runtime.go       # Runtime interface
│   │   ├── manager.go       # Component manager
│   │   └── components.go    # Component implementation
│   ├── storage/             # Storage layer
│   │   ├── storage.go       # Storage interface
│   │   ├── sqlite.go        # SQLite implementation
│   │   └── migrations.go    # Database migration
│   └── trigger/             # Trigger system
│       ├── trigger.go       # Trigger interface
│       ├── tekton.go        # Tekton trigger
│       └── transformer.go   # Data transformer
├── pkg/                      # Public packages (can be exposed externally)
│   ├── logger/              # Logging component
│   ├── types/               # Type definitions
│   └── utils/               # Utility functions
├── test/                     # Test files
│   ├── fixtures/            # Test data
│   └── integration/         # Integration tests
├── deployments/              # Deployment configurations
├── docs/                     # Documentation

└── Makefile                  # Build scripts
```

### Package Design Principles

#### internal/ vs pkg/

- **internal/**: Internal business logic, not allowed external import
- **pkg/**: Public libraries, can be imported by other projects

#### Layered Architecture

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

## 🏗️ Code Architecture

### Design Patterns

#### 1. Dependency Injection

```go
// Interface definition
type Storage interface {
    Store(event Event) error
    GetEvents(filter Filter) ([]Event, error)
}

// Dependency injection
type Poller struct {
    storage   Storage      // Inject storage interface
    gitClient GitClient    // Inject Git client interface
    logger    *Logger      // Inject logger
}

func NewPoller(storage Storage, gitClient GitClient, logger *Logger) *Poller {
    return &Poller{
        storage:   storage,
        gitClient: gitClient,
        logger:    logger,
    }
}
```

#### 2. Factory Pattern

```go
// Client factory
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

#### 3. Strategy Pattern

```go
// Polling strategy interface
type PollingStrategy interface {
    ShouldPoll(repo Repository, lastCheck time.Time) bool
    NextPollTime(repo Repository) time.Time
}

// Fixed interval strategy
type FixedIntervalStrategy struct {
    interval time.Duration
}

// Adaptive strategy
type AdaptiveStrategy struct {
    baseInterval time.Duration
    maxInterval  time.Duration
}
```

### Error Handling

#### Error Type Design

```go
// Custom error type
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

// Predefined errors
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

#### Error Handling Patterns

```go
// Wrap errors
func (p *Poller) pollRepository(repo Repository) error {
    branches, err := p.gitClient.GetBranches(repo)
    if err != nil {
        return fmt.Errorf("failed to fetch branches for %s: %w", repo.Name, err)
    }
    
    // Processing logic...
    return nil
}

// Log and handle errors
func (p *Poller) handleError(repo Repository, err error) {
    p.logger.WithField("repository", repo.Name).
             WithError(err).
             Error("polling failed")
    
    // Record error event
    errorEvent := Event{
        Repository:   repo.Name,
        Type:        EventTypeError,
        ErrorMessage: err.Error(),
    }
    p.storage.Store(errorEvent)
}
```

### Concurrency Patterns

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

#### Context Usage

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

## 🧪 Testing Strategy

### Test Layers

#### 1. Unit Tests

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

#### 2. Integration Tests

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

#### 3. Mock Tests

```go
// Using testify/mock
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
    
    // Setup mock expectations
    expectedBranches := []Branch{
        {Name: "main", CommitSHA: "abc123"},
    }
    mockClient.On("GetBranches", mock.Anything).Return(expectedBranches, nil)
    
    poller := NewPoller(mockStorage, mockClient, logger.NewTestLogger())
    
    err := poller.PollRepository(testRepo)
    require.NoError(t, err)
    
    // Verify mock calls
    mockClient.AssertExpectations(t)
}
```

### Test Execution

```bash
# Run all tests
make test

# Run unit tests
go test ./...

# Run specific package tests
go test ./internal/poller/

# Run integration tests
go test -tags=integration ./...

# Generate coverage report
make test-coverage

# Run benchmark tests
go test -bench=. ./...

# Run race detection
go test -race ./...
```

## 🔧 Development Workflow

### Branch Strategy

We use GitHub Flow pattern:

```
main (stable branch)
  ↑
feature/add-new-provider    # Feature branch
feature/improve-logging     # Feature branch
bugfix/fix-memory-leak      # Fix branch
```

### Commit Convention

Use Conventional Commits specification:

```bash
# Feature addition
git commit -m "feat(poller): add adaptive polling strategy"

# Bug fix
git commit -m "fix(storage): resolve database lock issue"

# Documentation update
git commit -m "docs: update API documentation"

# Code refactoring
git commit -m "refactor(gitclient): extract rate limiter interface"

# Performance optimization
git commit -m "perf(poller): optimize branch filtering algorithm"

# Test related
git commit -m "test(trigger): add integration tests for Tekton"
```

### Code Review Checklist

#### Code Quality
- [ ] Code follows Go language conventions
- [ ] Functions and variables are clearly named
- [ ] Necessary comments added
- [ ] Proper error handling
- [ ] No hard-coded values

#### Test Coverage
- [ ] Unit tests added
- [ ] Tests cover key paths
- [ ] Test cases are representative
- [ ] Proper mock usage

#### Performance Considerations
- [ ] No obvious performance issues
- [ ] Proper context usage
- [ ] Avoid memory leaks
- [ ] Database query optimization

#### Security
- [ ] Adequate input validation
- [ ] Sensitive information not in code
- [ ] SQL injection protection
- [ ] Correct access control

### Development Commands

#### Code Generation

```bash
# Generate mock files
go generate ./...

# Generate Swagger documentation
make swagger

# Generate protocol buffer files (if used)
make protoc
```

#### Code Checking

```bash
# Code formatting
make fmt

# Code checking
make lint

# Import organization
make imports

# Static analysis
make vet

# Security check
make security
```

#### Local Testing

```bash
# Start local environment
make dev-up

# Stop local environment
make dev-down

# Rebuild and start
make dev-restart

# View logs
make dev-logs
```

## 🚀 Debugging Techniques

### Log Debugging

```go
// Add detailed logs
logger.WithFields(logrus.Fields{
    "repository": repo.Name,
    "branch":     branch.Name,
    "operation":  "fetch_commit",
}).Debug("fetching commit information")

// Performance debugging
start := time.Now()
result, err := operation()
logger.WithField("duration", time.Since(start)).
       Debug("operation completed")
```

### Performance Analysis

```bash
# Enable performance analysis
go run cmd/reposentry/main.go run --config=config.yaml --pprof

# Analyze CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile

# Analyze memory usage
go tool pprof http://localhost:8080/debug/pprof/heap

# Analyze goroutines
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### Breakpoint Debugging

#### Using Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug program
dlv debug cmd/reposentry/main.go -- run --config=config.yaml

# Debug in VS Code
# Use launch.json configuration
```

#### Debug Configuration Example

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

## 📦 Release Process

### Version Management

Use semantic versioning:

- **Major version**: Incompatible API changes
- **Minor version**: Backward-compatible functionality additions
- **Patch version**: Backward-compatible bug fixes

### Release Checklist

#### Pre-release Check
- [ ] All tests pass
- [ ] Code quality checks pass
- [ ] Documentation updated
- [ ] CHANGELOG updated
- [ ] Version number updated

#### Build and Test
- [ ] Multi-platform build successful
- [ ] Docker image build successful
- [ ] Helm Chart test pass
- [ ] Integration tests pass

#### Release Execution
```bash
# Create release tag
git tag -a v1.2.3 -m "Release version 1.2.3"

# Push tag (trigger CI/CD)
git push origin v1.2.3

# Build release package
make release

# Publish Docker image
make docker-publish

# Publish Helm Chart
make helm-publish
```

## 🤝 Contributing Guidelines

### Contribution Process

1. **Fork Repository**
2. **Create Feature Branch**: `git checkout -b feature/amazing-feature`
3. **Commit Changes**: `git commit -m 'feat: add amazing feature'`
4. **Push Branch**: `git push origin feature/amazing-feature`
5. **Create Pull Request**

### PR Template

```markdown
## Change Description
Brief description of changes in this PR

## Change Type
- [ ] New feature
- [ ] Bug fix
- [ ] Documentation update
- [ ] Refactoring
- [ ] Performance optimization
- [ ] Other

## Testing
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] Manual testing passed

## Checklist
- [ ] Code follows project conventions
- [ ] Necessary documentation added
- [ ] CHANGELOG updated
- [ ] All tests pass
```

### Local Development Environment

```bash
# Setup development environment
make dev-setup

# Install pre-commit hooks
make install-hooks

# Run development server
make dev-server

# Reload configuration
make dev-reload
```

---

Hope this development guide helps you better participate in RepoSentry project development! If you have any questions, please check other documentation or submit an Issue.
