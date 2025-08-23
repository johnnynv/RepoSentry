# RepoSentry User Guide

## 🚀 Overview

RepoSentry is a lightweight cloud-native Git repository monitoring sentinel that supports monitoring GitHub and GitLab repository changes and triggering Tekton pipelines.

## ⚡ 5-Minute Quick Start

### Prerequisites
- Go 1.21+ (if building from source)
- Docker (if using container deployment)
- Kubernetes (if using Helm deployment)
- GitHub/GitLab API Token
- Tekton EventListener URL

### Step 1: Get RepoSentry
```bash
mkdir repository-monitor
cd repository-monitor
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry
chmod +x reposentry
```

### Step 2: Interactive Setup
```bash
./reposentry setup interactive
```

### Step 3: Start Monitoring
```bash
./start.sh
./reposentry status
```

## 📖 Configuration

### Generated Files
- config.yaml - Application configuration
- repositories.yaml - Repository definitions
- .env - Environment variables
- start.sh / stop.sh - Control scripts

## 🔧 CLI Commands
```bash
./reposentry status
./reposentry config validate
./reposentry run --config config.yaml
```

## 📊 Monitoring
- Health Check: GET /health
- Status: GET /api/v1/status
- Repositories: GET /api/v1/repositories
- Events: GET /api/v1/events

---
*This guide provides comprehensive information for using RepoSentry.*
