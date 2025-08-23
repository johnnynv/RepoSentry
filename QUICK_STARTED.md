# RepoSentry - Getting Started Guide

Welcome to RepoSentry! This guide will help you set up repository monitoring in just a few minutes.

## 🚀 Quick Installation (5 minutes)

### Step 1: Create Your Monitoring Directory
```bash
mkdir repository-monitor
cd repository-monitor
```

### Step 2: Download RepoSentry
```bash
# Download the latest release
wget https://github.com/johnnynv/RepoSentry/releases/latest/download/reposentry-v0.1.0.linux.x86_64
mv reposentry-v0.1.0.linux.x86_64 reposentry

# Make it executable
chmod +x reposentry
```

### Step 3: Run Interactive Setup
```bash
./reposentry setup interactive
```

You'll be asked to provide:
- **GitHub Token** - Get one at [GitHub Settings > Personal Access Tokens](https://github.com/settings/tokens)
- **GitLab Token** - Get one at your GitLab instance: `/profile/personal_access_tokens`
- **Repository URLs** - The repositories you want to monitor
- **Branch Names** - Which branches to watch (supports regex like `feature/.*`)
- **Tekton URL** - Your Tekton EventListener webhook URL
- **Polling Interval** - How often to check for changes (recommended: 5 minutes)

### Step 4: Start Monitoring
```bash
./start.sh
```

That's it! 🎉

## 📁 What Gets Created

After setup, your `repository-monitor` directory contains:

```
repository-monitor/
├── reposentry              # The monitoring application
├── config.yaml            # Application settings
├── repositories.yaml      # Your repository definitions  
├── start.sh               # Start monitoring
├── stop.sh                # Stop monitoring
├── .env                   # Your access tokens (keep secure!)
├── README.md              # Detailed usage guide
└── logs/                  # Log files (created when running)
```

## 🔧 Managing Your Setup

### View Logs
```bash
tail -f logs/reposentry.log
```

### Stop Monitoring
```bash
./stop.sh
```

### Add More Repositories
Edit `repositories.yaml`:
```yaml
repositories:
  - name: "my-new-repo"
    provider: "github"  # or "gitlab"
    url: "https://github.com/user/repo"
    branches: ["main", "develop"]
    auth_token_env: "GITHUB_TOKEN"
```

### Change Polling Interval
Edit `config.yaml`:
```yaml
polling:
  interval: "10m"  # Check every 10 minutes
```

### Update Access Tokens
Edit `.env` file:
```bash
GITHUB_TOKEN=your_new_token
GITLAB_TOKEN=your_new_token
```

## 🔍 Troubleshooting

### Common Issues

**"Token authentication fails"**
- Verify your tokens in the `.env` file
- Ensure tokens have repository read permissions

**"Repository not found"**
- Check the repository URL format
- Verify your token has access to private repositories

**"Tekton connection failed"**
- Verify the EventListener URL is accessible
- Check network connectivity from your machine

### Get Help

- 📖 **Full Documentation**: [GitHub Wiki](https://github.com/johnnynv/RepoSentry/wiki)
- 🐛 **Report Issues**: [GitHub Issues](https://github.com/johnnynv/RepoSentry/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/johnnynv/RepoSentry/discussions)

## 🎯 Next Steps

Once RepoSentry is running:

1. **Monitor the logs** to see repository changes being detected
2. **Check your Tekton Dashboard** to see triggered pipelines
3. **Customize the configuration** for your specific needs
4. **Set up log rotation** for production use

For advanced configuration options, see the generated `README.md` in your monitoring directory.

---

**Need help?** Check out our [documentation](https://github.com/johnnynv/RepoSentry/wiki) or [open an issue](https://github.com/johnnynv/RepoSentry/issues).
