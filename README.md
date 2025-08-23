# RepoSentry

A lightweight, cloud-native sentinel for monitoring GitLab and GitHub repositories. RepoSentry watches your Git repositories for changes and triggers Tekton pipelines via webhooks.

## 🚀 Quick Start

### Method 1: Interactive Setup (Recommended)
```bash
# Create a dedicated directory for your monitoring setup
mkdir repository-monitor
cd repository-monitor

# Download the RepoSentry binary
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry
chmod +x reposentry

# Run interactive configuration wizard
./reposentry setup interactive

# Start monitoring
./start.sh
```

### Method 2: Script-based Setup (Legacy)
```bash
# Run the main script to get everything set up
./examples/scripts/start.sh

# Choose option 7 for Quick Start (All Steps)
```

### Method 3: Manual Setup (Advanced)
```bash
# Copy an example configuration
cp examples/configs/basic.yaml config.yaml

# Edit with your repositories and Tekton webhook URL
vim config.yaml

# Start monitoring
./reposentry run --config config.yaml
```

## 🎯 Interactive Setup Details

The `./reposentry setup interactive` command creates a complete, self-contained monitoring environment:

**Generated Files:**
- `config.yaml` - Application configuration (polling, logging, Tekton)
- `repositories.yaml` - Repository definitions and monitoring settings
- `start.sh` / `stop.sh` - Control scripts with validation
- `.env` - Environment variables (your access tokens)
- `README.md` - Usage instructions and configuration guide

**Configuration Process:**
1. GitHub/GitLab access tokens
2. Repository URLs and branch patterns (supports regex)
3. Tekton EventListener URL
4. Polling interval settings

**Advantages:**
- ✅ Self-contained setup (all files in one directory)
- ✅ Separate configuration files for easy management
- ✅ Automatic validation and error checking
- ✅ Ready-to-use control scripts

### Access API Documentation
- **Swagger UI**: http://localhost:8080/swagger/
- **Health Check**: http://localhost:8080/health
- **API Info**: http://localhost:8080/api

## 🔧 Tekton Integration

RepoSentry provides two Tekton integration templates:

### Basic Version (`reposentry-basic-system.yaml`)
- **CloudEvents 1.0 Standard Compatible**
- **Simple Parameters**: provider, organization, repository-name, branch-name, commit-sha
- **Enterprise Ready**: Minimal configuration, maximum compatibility
- **Use Case**: Production environments requiring CloudEvents compliance

### Advanced Version (`reposentry-advanced-system.yaml`)
- **Rich Metadata Extraction**
- **Enhanced Parameters**: repository-id, trigger-source, reposentry-event-id, project-name
- **Development Friendly**: Detailed context for debugging and monitoring
- **Use Case**: Development teams needing comprehensive pipeline information

**Choose based on your needs**: Basic for production, Advanced for development.

## 📚 Documentation

For comprehensive documentation, see the [docs/](docs/) directory:
- **中文文档**: [docs/zh/](docs/zh/) - Chinese documentation
- **English Docs**: [docs/en/](docs/en/) - English documentation
- **Documentation Index**: [docs/README.md](docs/README.md) - Complete documentation guide

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
│   ├── tekton/             # Tekton integration templates
│   │   ├── basic/          # Basic CloudEvents-compatible system
│   │   └── advanced/       # Advanced system with rich metadata
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
