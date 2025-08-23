# RepoSentry Scripts

This directory contains scripts to help you set up and manage RepoSentry.

## 🚀 Quick Start

**Start here!** Run the main script to get everything set up:

```bash
./examples/scripts/start.sh
```

## 📋 Available Scripts

### 1. `start.sh` - Main Entry Point ⭐
**Run this first!** This is your one-stop script for setting up and managing RepoSentry.

**Features:**
- 🔨 Build RepoSentry binary
- ⚙️ Setup environment variables
- 📚 Manage repository configurations
- ✅ Validate and test configuration
- 🚀 Start RepoSentry monitoring

**Usage:**
```bash
./examples/scripts/start.sh
```

### 2. `setup_env.sh` - Environment Configuration
Sets up environment variables for GitHub, GitLab, and Tekton.

**Features:**
- Interactive token configuration
- Support for enterprise Git providers
- Automatic .env file generation

**Usage:**
```bash
./examples/scripts/setup_env.sh
```

### 3. `manage_repos.sh` - Repository Management
Interactive repository configuration management.

**Features:**
- Add new repositories
- Delete repositories
- List current repositories
- Validate configuration
- Test webhook connections

**Usage:**
```bash
./examples/scripts/manage_repos.sh
```

## 🎯 Recommended Workflow

1. **First Time Setup:**
   ```bash
   ./examples/scripts/start.sh
   # Choose option 7: Quick Start (All Steps)
   ```

2. **Add/Remove Repositories:**
   ```bash
   ./examples/scripts/start.sh
   # Choose option 3: Manage Repositories
   ```

3. **Validate Configuration:**
   ```bash
   ./examples/scripts/start.sh
   # Choose option 4: Validate & Test
   ```

4. **Start Monitoring:**
   ```bash
   ./examples/scripts/start.sh
   # Choose option 5: Start RepoSentry
   ```

## 🔧 Prerequisites

- Go 1.24+ installed
- kubectl (optional, for Kubernetes operations)
- Git repository access tokens
- Tekton webhook URL

## 📝 Environment Variables

The scripts will create a `.env` file with:

- `GITHUB_TOKEN` - GitHub personal access token
- `GITLAB_TOKEN` - GitLab personal access token
- `GITHUB_API_URL` - GitHub API base URL
- `GITLAB_API_URL` - GitLab API base URL
- `TEKTON_WEBHOOK_URL` - Tekton EventListener webhook URL

## 🚨 Important Notes

- **Security**: Keep your `.env` file secure and add it to `.gitignore`
- **Tokens**: Ensure your tokens have appropriate permissions (repo access)
- **Configuration**: The main script will guide you through each step
- **Validation**: Always validate configuration before starting RepoSentry

## 🆘 Troubleshooting

If you encounter issues:

1. Check prerequisites are met
2. Verify environment variables are set correctly
3. Validate configuration with `./bin/reposentry config validate config.yaml`
4. Test webhook connection with `./bin/reposentry test-webhook --config config.yaml`

## 📚 Next Steps

After running the scripts:

1. Review generated `config.yaml`
2. Customize repository settings if needed
3. Start RepoSentry monitoring
4. Check Tekton dashboard for pipeline triggers
