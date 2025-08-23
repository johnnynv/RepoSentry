# RepoSentry Documentation

Welcome to the RepoSentry documentation! This directory contains comprehensive documentation for RepoSentry in both Chinese and English.

## 📁 Documentation Structure

### 中文文档 (Chinese Documentation) - `zh/`

| 文档 | 描述 | 适用人群 |
|------|------|----------|
| [快速开始](zh/QUICKSTART.md) | 5分钟快速部署指南 | 所有用户 |
| [用户手册](zh/USER_MANUAL.md) | 详细的配置和使用说明 | 运维人员、开发者 |
| [技术架构](zh/ARCHITECTURE.md) | 系统架构和设计原理 | 架构师、高级开发者 |
| [故障排除](zh/TROUBLESHOOTING.md) | 常见问题和解决方案 | 运维人员 |
| [开发指南](zh/DEVELOPMENT.md) | 开发环境搭建和贡献指南 | 开发者 |
| [API 示例](zh/API_EXAMPLES.md) | REST API 使用示例 | 集成开发者 |

### English Documentation - `en/`

| Document | Description | Target Audience |
|----------|-------------|-----------------|
| [Quick Start](en/QUICKSTART.md) | 5-minute deployment guide | All users |
| [User Manual](en/USER_MANUAL.md) | Detailed configuration and usage | Operators, Developers |
| [Technical Architecture](en/ARCHITECTURE.md) | System architecture and design principles | Architects, Senior developers |
| [Troubleshooting](en/TROUBLESHOOTING.md) | Common issues and solutions | Operators |
| [Development Guide](en/DEVELOPMENT.md) | Development setup and contribution guide | Developers |
| [API Examples](en/API_EXAMPLES.md) | REST API usage examples | Integration developers |

## 🚀 Getting Started

### For New Users
1. **Start Here**: [Quick Start Guide](en/QUICKSTART.md) / [快速开始指南](zh/QUICKSTART.md)
2. **Basic Setup**: Follow the 5-minute deployment guide
3. **Configuration**: Read the [User Manual](en/USER_MANUAL.md) for detailed configuration

### For Operators
1. **Deployment**: [Quick Start Guide](en/QUICKSTART.md) for deployment options
2. **Configuration**: [User Manual](en/USER_MANUAL.md) for detailed configuration
3. **Monitoring**: [API Examples](en/API_EXAMPLES.md) for monitoring endpoints
4. **Troubleshooting**: [Troubleshooting Guide](en/TROUBLESHOOTING.md) for common issues

### For Developers
1. **Understanding**: [Technical Architecture](en/ARCHITECTURE.md) to understand the system
2. **Development**: [Development Guide](en/DEVELOPMENT.md) for contribution guidelines
3. **Integration**: [API Examples](en/API_EXAMPLES.md) for API integration
4. **Contributing**: See [Development Guide](en/DEVELOPMENT.md) for contribution process

### For Architects
1. **Architecture**: [Technical Architecture](en/ARCHITECTURE.md) for system design
2. **Scalability**: Review scaling strategies in the architecture document
3. **Security**: Security considerations in [User Manual](en/USER_MANUAL.md)

## 📊 API Documentation

### Interactive Documentation
- **Swagger UI**: `http://localhost:8080/swagger/` (when service is running)
- **API Examples**: [English](en/API_EXAMPLES.md) / [中文](zh/API_EXAMPLES.md)

### Quick API Reference

| Endpoint | Purpose | Documentation |
|----------|---------|---------------|
| `/health` | Health check | [API Examples](en/API_EXAMPLES.md#health-check-apis) |
| `/api/v1/status` | Service status | [API Examples](en/API_EXAMPLES.md#service-information-apis) |
| `/api/v1/repositories` | Repository management | [API Examples](en/API_EXAMPLES.md#repository-management-apis) |
| `/api/v1/events` | Event history | [API Examples](en/API_EXAMPLES.md#event-query-apis) |
| `/api/v1/metrics` | Performance metrics | [API Examples](en/API_EXAMPLES.md#metrics-apis) |

## 🛠️ Configuration Reference

### Required Configuration
```yaml
# Minimum required fields
tekton:
  event_listener_url: "http://your-tekton-listener:8080"

repositories:
  - name: "my-repo"
    url: "https://github.com/user/repo"
    provider: "github"
    token: "${GITHUB_TOKEN}"
    branch_regex: "^(main|develop)$"
```

### Environment Variables
```bash
# Required tokens
export GITHUB_TOKEN="ghp_your_github_token"
export GITLAB_TOKEN="glpat-your_gitlab_token"
```

For detailed configuration options, see [User Manual](en/USER_MANUAL.md#configuration-details).

## 🚢 Deployment Options

| Method | Documentation | Use Case |
|--------|---------------|----------|
| **Binary** | [Quick Start](en/QUICKSTART.md#step-1-get-reposentry) | Development, small deployments |
| **Docker** | [Quick Start](en/QUICKSTART.md#docker-deployment) | Container environments |
| **Kubernetes** | [Quick Start](en/QUICKSTART.md#kubernetes-helm-deployment) | Production, cloud-native |

---

**Note**: For project overview, quick start, and project structure, see the main [README.md](../README.md) in the project root.
