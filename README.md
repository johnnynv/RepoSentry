# RepoSentry

A lightweight, cloud-native sentinel for monitoring GitLab and GitHub repositories. RepoSentry watches your Git repositories for changes and triggers Tekton pipelines via webhooks.

## 🚀 Quick Start

### 1. Setup Environment
```bash
# Run the setup script to configure environment variables
./examples/scripts/setup_env.sh

# Or manually set tokens
export GITHUB_TOKEN="your_github_token"
export GITLAB_TOKEN="your_gitlab_token"
```

### 2. Configure RepoSentry
```bash
# Copy an example configuration
cp examples/configs/basic.yaml config.yaml

# Edit with your repositories and Tekton webhook URL
vim config.yaml
```

### 3. Run RepoSentry
```bash
# Validate configuration
./reposentry config validate config.yaml

# Start monitoring
./reposentry run --config config.yaml
```

### 4. Access API Documentation
- **Swagger UI**: http://localhost:8080/swagger/
- **Health Check**: http://localhost:8080/health
- **API Info**: http://localhost:8080/api

## 📁 Project Structure

```
├── cmd/reposentry/          # CLI application
├── internal/                # Internal packages
│   ├── api/                # REST API and Swagger docs
│   ├── config/             # Configuration management
│   ├── gitclient/          # Git provider clients
│   ├── poller/             # Repository polling logic
│   ├── runtime/            # Application runtime
│   ├── storage/            # Data persistence
│   └── trigger/            # Webhook triggers
├── pkg/                    # Public packages
│   ├── logger/             # Structured logging
│   ├── types/              # Common types
│   └── utils/              # Utilities
├── examples/               # Configuration examples
│   ├── configs/            # Example configurations
│   ├── docker/             # Docker examples
│   ├── kubernetes/         # Kubernetes examples
│   └── scripts/            # Utility scripts
├── deployments/            # Deployment configurations
│   ├── docker/             # Docker deployment
│   ├── helm/               # Helm charts
│   └── systemd/            # Systemd service
├── docs/                   # Documentation
└── test/                   # Test files
```

## 🛠️ Development

### Build
```bash
make build          # Build binary
make build-linux    # Build for Linux
make swagger        # Generate API docs
```

### Test
```bash
make test           # Unit tests
make test-integration  # Integration tests
make check          # All checks (fmt, vet, lint, test)
```

### Release
```bash
make release        # Create release build
```

## 📖 Documentation

### 📖 Complete Documentation
- **[Documentation Hub](docs/README.md)** - Complete documentation index
- **Quick Start**: [English](docs/en/QUICKSTART.md) | [中文](docs/zh/QUICKSTART.md)
- **User Manual**: [English](docs/en/USER_MANUAL.md) | [中文](docs/zh/USER_MANUAL.md)
- **Technical Architecture**: [English](docs/en/ARCHITECTURE.md) | [中文](docs/zh/ARCHITECTURE.md)

### 🛠️ For Developers
- **[Development Guide](docs/en/DEVELOPMENT.md)** - Setup and contribution guide
- **[API Examples](docs/en/API_EXAMPLES.md)** - REST API usage examples
- **[Troubleshooting](docs/en/TROUBLESHOOTING.md)** - Common issues and solutions

### 🚀 For Operations
- **[Deployment Guide](deployments/README.md)** - Production deployment
- **[Configuration Examples](examples/README.md)** - Sample configurations
- **Swagger UI**: `http://localhost:8080/swagger/` (when running)

## 🚢 Deployment

### Docker
```bash
cd deployments/docker
docker-compose up -d
```

### Kubernetes
```bash
helm install reposentry ./deployments/helm/reposentry \
  -f examples/kubernetes/helm-values-prod.yaml
```

### Systemd
```bash
sudo ./deployments/systemd/install.sh
```

## 🔧 Configuration

See [examples/configs/](examples/configs/) for configuration templates:
- `basic.yaml` - Standard configuration
- `minimal.yaml` - Minimal setup
- `development.yaml` - Development environment
- `production.yaml` - Production ready

## 🔗 Features

- **Multi-Provider Support**: GitHub and GitLab (including enterprise)
- **Intelligent Polling**: API-first with git fallback
- **Flexible Triggers**: Tekton EventListener webhooks
- **RESTful API**: Complete API with Swagger documentation
- **Cloud Native**: Kubernetes-ready with Helm charts
- **Monitoring**: Health checks and metrics
- **Security**: Environment variable injection, HTTPS validation

## 📊 Monitoring

RepoSentry provides comprehensive monitoring capabilities:

### Health Checks
```bash
# System health
curl http://localhost:8080/health

# Component status
curl http://localhost:8080/status

# Application metrics
curl http://localhost:8080/metrics
```

### Using Scripts
```bash
# Automated health check
./examples/scripts/health_check.sh

# Setup monitoring environment
./examples/scripts/setup_env.sh
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
RepoSentry is a lightweight, cloud-native sentinel that keeps an independent watch over your GitLab and GitHub repositories.
