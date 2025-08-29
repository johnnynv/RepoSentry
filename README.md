# RepoSentry

A lightweight, cloud-native sentinel for monitoring GitLab and GitHub repositories. RepoSentry watches your Git repositories for changes and triggers Tekton pipelines via webhooks.

## 🚀 Quick Start

### Method 1: Interactive Setup (Recommended)
```bash
# Create a dedicated directory for your monitoring setup
mkdir repository-monitor
cd repository-monitor

# Download the RepoSentry binary
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-v0.1.0.linux.x86_64
mv reposentry-v0.1.0.linux.x86_64 reposentry

chmod +x reposentry

# Run interactive configuration wizard
./reposentry setup interactive

# Start monitoring
./start.sh
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

RepoSentry provides seamless Tekton integration through a pre-deployed Bootstrap Pipeline architecture:

### Bootstrap Pipeline Architecture
- **Pre-deployed Infrastructure**: Static Bootstrap Pipeline deployed once to your cluster
- **Auto-Detection**: Automatically detects `.tekton/` directories in monitored repositories
- **CloudEvents Standard**: Uses CloudEvents 1.0 for triggering Bootstrap Pipeline
- **User Isolation**: Each repository gets its own namespace for secure execution
- **Direct YAML Application**: User Tekton resources applied directly without modification

### Quick Tekton Setup
1. **Deploy Bootstrap Pipeline**:
   ```bash
   # Download Bootstrap Pipeline files
   git clone https://github.com/johnnynv/RepoSentry.git
   cd RepoSentry/deployments/tekton/bootstrap/
   
   # One-click installation with auto-detection
   ./scripts/install.sh
   
   # Or customize your installation
   ./scripts/install.sh --ingress-class nginx --webhook-host webhook.example.com
   
   # Verify deployment
   ./scripts/validate.sh --verbose
   ```

2. **Advanced Configuration Options**:
   The install script now automatically detects and configures your Ingress Controller:
   
   ```bash
   # Auto-detection (recommended)
   ./scripts/install.sh                    # Detects nginx/traefik/istio automatically
   
   # Manual override options
   ./scripts/install.sh --ingress-class traefik \
                --webhook-host webhook.mycompany.com \
                --ssl-redirect true
   
   # Environment variable configuration
   export BOOTSTRAP_INGRESS_CLASS=nginx
   export BOOTSTRAP_WEBHOOK_HOST=webhook.10.0.0.100.nip.io
   ./scripts/install.sh
   
   # Disable auto-configuration
   ./scripts/install.sh --no-auto-configure
   ```

3. **Enable in RepoSentry**: During interactive setup, choose "Enable Tekton integration"

4. **Add Tekton resources to your repositories**: Place Pipeline/Task YAML files in `.tekton/` directory

**Architecture Benefits**: No dynamic generation, simplified deployment, better security isolation.

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

- **Swagger UI**: `http://localhost:8080/swagger/` (when running)

## 🚢 Deployment

### Docker
```bash
cd deployments/docker
docker-compose up -d
```

### Kubernetes
```bash
helm install reposentry ./deployments/helm/reposentry
```

### Systemd
```bash
sudo ./deployments/systemd/install.sh
```

## 🔧 Configuration

Create your configuration file with the following structure:

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

### Using Built-in Health Check
```bash
# Check service health
curl http://localhost:8080/health

# Monitor logs
./reposentry run --log-level debug
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
