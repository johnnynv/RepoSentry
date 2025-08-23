# RepoSentry API Usage Examples

RepoSentry provides complete RESTful API and Swagger UI online documentation for convenient use by developers and operations personnel.

## 🔗 **Quick Access**

- **Swagger UI**: `http://localhost:8080/swagger/`
- **API Documentation JSON**: `http://localhost:8080/swagger/doc.json`
- **API Basic Information**: `http://localhost:8080/api`

## 📊 **API Overview**

| Category | Endpoint | Description |
|----------|----------|-------------|
| **Health Check** | `/health`, `/healthz`, `/ready` | Service health status |
| **Service Information** | `/api/v1/status`, `/api/v1/version` | Runtime status and version |
| **Repository Management** | `/api/v1/repositories` | Repository list and details |
| **Event Query** | `/api/v1/events` | Event history and query |
| **Metrics** | `/api/v1/metrics` | Performance metrics |
| **Configuration** | `/api/v1/config` | Configuration management |

## 🏥 **Health Check APIs**

### Basic Health Check

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "success": true,
  "message": "Service is healthy",
  "data": {
    "healthy": true,
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Detailed Health Check

```bash
curl http://localhost:8080/healthz
```

**Response:**
```json
{
  "success": true,
  "data": {
    "healthy": true,
    "components": {
      "config": {"healthy": true, "message": "Configuration loaded"},
      "storage": {"healthy": true, "message": "Database connected"},
      "git_client": {"healthy": true, "message": "All clients ready"},
      "trigger": {"healthy": true, "message": "Tekton reachable"},
      "poller": {"healthy": true, "message": "Polling active"}
    }
  }
}
```

### Readiness Check

```bash
curl http://localhost:8080/ready
```

## 📊 **Service Information APIs**

### Service Status

```bash
curl http://localhost:8080/api/v1/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "uptime": "2h30m15s",
    "status": "running",
    "components": 5,
    "repositories": {
      "total": 10,
      "healthy": 9,
      "error": 1
    },
    "polling": {
      "active": true,
      "last_cycle": "2024-01-15T10:30:00Z",
      "next_cycle": "2024-01-15T10:35:00Z"
    }
  }
}
```

### Service Version

```bash
curl http://localhost:8080/api/v1/version
```

**Response:**
```json
{
  "success": true,
  "data": {
    "version": "v1.0.0",
    "build_time": "2024-01-15T08:00:00Z",
    "git_commit": "abc123def456",
    "go_version": "go1.21.5"
  }
}
```

## 📁 **Repository Management APIs**

### List All Repositories

```bash
curl http://localhost:8080/api/v1/repositories
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "name": "frontend-app",
      "url": "https://github.com/company/frontend-app",
      "provider": "github",
      "status": "healthy",
      "last_checked": "2024-01-15T10:25:00Z",
      "branch_count": 5,
      "polling_interval": "5m"
    },
    {
      "name": "backend-service",
      "url": "https://gitlab.example.com/team/backend",
      "provider": "gitlab",
      "status": "error",
      "last_checked": "2024-01-15T10:20:00Z",
      "error_message": "API rate limit exceeded"
    }
  ]
}
```

### Get Repository Details

```bash
curl http://localhost:8080/api/v1/repositories/frontend-app
```

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "frontend-app",
    "url": "https://github.com/company/frontend-app",
    "provider": "github",
    "branch_regex": "^(main|develop|release/.*)$",
    "polling_interval": "5m",
    "status": "healthy",
    "last_checked": "2024-01-15T10:25:00Z",
    "branches": [
      {
        "name": "main",
        "commit_sha": "abc123def456",
        "last_updated": "2024-01-15T09:15:00Z"
      },
      {
        "name": "develop",
        "commit_sha": "def456ghi789",
        "last_updated": "2024-01-15T10:00:00Z"
      }
    ],
    "metadata": {
      "team": "frontend",
      "environment": "production"
    }
  }
}
```

### Repository Status

```bash
curl http://localhost:8080/api/v1/repositories/frontend-app/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "repository": "frontend-app",
    "status": "healthy",
    "last_poll": "2024-01-15T10:25:00Z",
    "next_poll": "2024-01-15T10:30:00Z",
    "api_calls_remaining": 4950,
    "branch_count": 5,
    "last_event": {
      "id": "event-123",
      "type": "commit_pushed",
      "branch": "main",
      "timestamp": "2024-01-15T09:15:00Z"
    }
  }
}
```

## 📅 **Event Query APIs**

### Get All Events

```bash
curl http://localhost:8080/api/v1/events
```

**Query Parameters:**
- `limit`: Number of events (default: 50, max: 200)
- `offset`: Pagination offset (default: 0)
- `repository`: Filter by repository name
- `type`: Filter by event type
- `since`: Events since timestamp (ISO 8601)
- `until`: Events until timestamp (ISO 8601)

**Example with filters:**
```bash
curl "http://localhost:8080/api/v1/events?repository=frontend-app&type=commit_pushed&limit=10"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "event-123",
        "repository": "frontend-app",
        "type": "commit_pushed",
        "branch": "main",
        "commit_sha": "abc123def456",
        "status": "completed",
        "created_at": "2024-01-15T09:15:00Z",
        "processed_at": "2024-01-15T09:15:05Z",
        "metadata": {
          "author": "developer@example.com",
          "message": "Add new feature"
        }
      }
    ],
    "total": 156,
    "limit": 10,
    "offset": 0
  }
}
```

### Get Recent Events

```bash
curl http://localhost:8080/api/v1/events/recent
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "event-125",
      "repository": "backend-service",
      "type": "branch_created",
      "branch": "feature/new-api",
      "commit_sha": "def456ghi789",
      "status": "completed",
      "created_at": "2024-01-15T10:20:00Z",
      "processed_at": "2024-01-15T10:20:03Z"
    }
  ]
}
```

### Get Specific Event

```bash
curl http://localhost:8080/api/v1/events/event-123
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "event-123",
    "repository": "frontend-app",
    "type": "commit_pushed",
    "branch": "main",
    "commit_sha": "abc123def456",
    "status": "completed",
    "created_at": "2024-01-15T09:15:00Z",
    "processed_at": "2024-01-15T09:15:05Z",
    "webhook_url": "http://tekton-listener:8080",
    "webhook_response": {
      "status_code": 200,
      "body": "Event accepted"
    },
    "metadata": {
      "author": "developer@example.com",
      "message": "Add new feature",
      "files_changed": 3
    }
  }
}
```

## 📊 **Metrics APIs**

### Get Performance Metrics

```bash
curl http://localhost:8080/api/v1/metrics
```

**Response:**
```json
{
  "success": true,
  "data": {
    "uptime": "2h30m15s",
    "memory": {
      "allocated": "45.2MB",
      "total_allocated": "156.8MB",
      "gc_count": 23
    },
    "repositories": {
      "total": 10,
      "healthy": 9,
      "error": 1,
      "polling_active": true
    },
    "polling": {
      "cycles_completed": 156,
      "last_cycle_duration": "45s",
      "average_cycle_duration": "42s",
      "last_cycle": "2024-01-15T10:30:00Z",
      "next_cycle": "2024-01-15T10:35:00Z"
    },
    "events": {
      "total": 1250,
      "today": 89,
      "last_hour": 12,
      "pending": 2,
      "completed": 1245,
      "failed": 3
    },
    "api_calls": {
      "github": {
        "total": 2150,
        "remaining": 4950,
        "reset_time": "2024-01-15T11:00:00Z"
      },
      "gitlab": {
        "total": 856,
        "remaining": 1144,
        "reset_time": "2024-01-15T10:31:00Z"
      }
    },
    "webhooks": {
      "sent": 1248,
      "successful": 1245,
      "failed": 3,
      "average_response_time": "120ms"
    }
  }
}
```

## ⚙️ **Configuration Management APIs**

### Reload Configuration

```bash
curl -X POST http://localhost:8080/api/v1/config/reload
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration reloaded successfully",
  "data": {
    "reloaded_at": "2024-01-15T10:35:00Z",
    "changes": [
      "Updated polling interval from 5m to 3m",
      "Added new repository: mobile-app"
    ]
  }
}
```

### Validate Configuration

```bash
curl -X POST http://localhost:8080/api/v1/config/validate \
  -H "Content-Type: application/json" \
  -d @new-config.yaml
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration is valid",
  "data": {
    "validated_at": "2024-01-15T10:36:00Z",
    "repositories_count": 11,
    "warnings": [
      "Repository 'old-repo' has very long polling interval (30m)"
    ]
  }
}
```

## 🐍 **Python Examples**

### Basic Python Client

```python
import requests
import json
from datetime import datetime, timedelta

class RepoSentryClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        
    def get_health(self):
        """Get service health status"""
        response = requests.get(f"{self.base_url}/health")
        return response.json()
    
    def get_repositories(self):
        """Get all repositories"""
        response = requests.get(f"{self.base_url}/api/v1/repositories")
        return response.json()
    
    def get_repository(self, name):
        """Get specific repository details"""
        response = requests.get(f"{self.base_url}/api/v1/repositories/{name}")
        return response.json()
    
    def get_events(self, repository=None, since=None, limit=50):
        """Get events with optional filters"""
        params = {"limit": limit}
        if repository:
            params["repository"] = repository
        if since:
            params["since"] = since.isoformat()
            
        response = requests.get(
            f"{self.base_url}/api/v1/events",
            params=params
        )
        return response.json()
    
    def get_metrics(self):
        """Get performance metrics"""
        response = requests.get(f"{self.base_url}/api/v1/metrics")
        return response.json()
    
    def reload_config(self):
        """Reload configuration"""
        response = requests.post(f"{self.base_url}/api/v1/config/reload")
        return response.json()

# Usage example
client = RepoSentryClient()

# Check health
health = client.get_health()
print(f"Service healthy: {health['data']['healthy']}")

# Get repositories
repos = client.get_repositories()
print(f"Total repositories: {len(repos['data'])}")

# Get recent events
since = datetime.now() - timedelta(hours=1)
events = client.get_events(since=since)
print(f"Events in last hour: {len(events['data']['events'])}")

# Get metrics
metrics = client.get_metrics()
print(f"Service uptime: {metrics['data']['uptime']}")
```

### Repository Status Monitor

```python
import time
import requests
from datetime import datetime

def monitor_repositories(client, check_interval=300):
    """Monitor repository status and alert on issues"""
    
    while True:
        try:
            repos = client.get_repositories()
            
            for repo in repos['data']:
                if repo['status'] != 'healthy':
                    print(f"🚨 ALERT: Repository '{repo['name']}' is {repo['status']}")
                    if 'error_message' in repo:
                        print(f"   Error: {repo['error_message']}")
                else:
                    print(f"✅ Repository '{repo['name']}' is healthy")
            
            # Check overall metrics
            metrics = client.get_metrics()
            error_count = metrics['data']['repositories']['error']
            if error_count > 0:
                print(f"⚠️  Warning: {error_count} repositories have errors")
            
            print(f"Next check in {check_interval} seconds...\n")
            time.sleep(check_interval)
            
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to connect to RepoSentry: {e}")
            time.sleep(60)  # Wait longer on connection errors

# Run monitor
client = RepoSentryClient()
monitor_repositories(client, check_interval=300)  # Check every 5 minutes
```

## 🌐 **JavaScript/Node.js Examples**

### Basic Node.js Client

```javascript
const axios = require('axios');

class RepoSentryClient {
    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.client = axios.create({
            baseURL: baseUrl,
            timeout: 10000,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    async getHealth() {
        const response = await this.client.get('/health');
        return response.data;
    }

    async getRepositories() {
        const response = await this.client.get('/api/v1/repositories');
        return response.data;
    }

    async getRepository(name) {
        const response = await this.client.get(`/api/v1/repositories/${name}`);
        return response.data;
    }

    async getEvents(options = {}) {
        const params = {
            limit: options.limit || 50,
            ...(options.repository && { repository: options.repository }),
            ...(options.since && { since: options.since }),
            ...(options.type && { type: options.type })
        };
        
        const response = await this.client.get('/api/v1/events', { params });
        return response.data;
    }

    async getMetrics() {
        const response = await this.client.get('/api/v1/metrics');
        return response.data;
    }

    async reloadConfig() {
        const response = await this.client.post('/api/v1/config/reload');
        return response.data;
    }
}

// Usage example
async function main() {
    const client = new RepoSentryClient();
    
    try {
        // Check health
        const health = await client.getHealth();
        console.log(`Service healthy: ${health.data.healthy}`);
        
        // Get repositories
        const repos = await client.getRepositories();
        console.log(`Total repositories: ${repos.data.length}`);
        
        // Get recent events
        const events = await client.getEvents({
            since: new Date(Date.now() - 3600000).toISOString(), // Last hour
            limit: 10
        });
        console.log(`Recent events: ${events.data.events.length}`);
        
        // Get metrics
        const metrics = await client.getMetrics();
        console.log(`Service uptime: ${metrics.data.uptime}`);
        
    } catch (error) {
        console.error('API Error:', error.response?.data || error.message);
    }
}

main();
```

### Real-time Event Monitoring

```javascript
const WebSocket = require('ws');
const EventEmitter = require('events');

class RepoSentryMonitor extends EventEmitter {
    constructor(apiUrl = 'http://localhost:8080') {
        super();
        this.apiUrl = apiUrl;
        this.client = new RepoSentryClient(apiUrl);
        this.lastEventId = null;
        this.polling = false;
    }

    async startMonitoring(pollInterval = 30000) {
        this.polling = true;
        console.log('🔍 Starting RepoSentry monitoring...');
        
        while (this.polling) {
            try {
                await this.checkForNewEvents();
                await this.sleep(pollInterval);
            } catch (error) {
                console.error('Monitoring error:', error.message);
                this.emit('error', error);
                await this.sleep(60000); // Wait longer on errors
            }
        }
    }

    async checkForNewEvents() {
        const events = await this.client.getEvents({
            limit: 20,
            ...(this.lastEventId && { since: this.lastEventId })
        });

        const newEvents = events.data.events;
        if (newEvents.length > 0) {
            // Update last event ID
            this.lastEventId = newEvents[0].created_at;
            
            // Process new events
            for (const event of newEvents.reverse()) {
                this.emit('event', event);
                this.handleEvent(event);
            }
        }
    }

    handleEvent(event) {
        console.log(`📅 New event: ${event.type} in ${event.repository}/${event.branch}`);
        
        switch (event.type) {
            case 'commit_pushed':
                console.log(`   📝 Commit: ${event.commit_sha.substring(0, 8)}`);
                break;
            case 'branch_created':
                console.log(`   🌿 New branch: ${event.branch}`);
                break;
            case 'branch_deleted':
                console.log(`   🗑️  Deleted branch: ${event.branch}`);
                break;
        }
        
        if (event.status === 'failed') {
            console.log(`   ❌ Event processing failed: ${event.error_message}`);
            this.emit('eventFailed', event);
        }
    }

    stopMonitoring() {
        this.polling = false;
        console.log('⏹️  Monitoring stopped');
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Usage
const monitor = new RepoSentryMonitor();

monitor.on('event', (event) => {
    // Custom event handling
    if (event.repository === 'critical-app' && event.branch === 'main') {
        console.log('🚨 Critical repository updated!');
        // Send alert, trigger deployment, etc.
    }
});

monitor.on('eventFailed', (event) => {
    console.log(`🔥 Failed event requires attention: ${event.id}`);
    // Send notification to ops team
});

monitor.startMonitoring(30000); // Check every 30 seconds
```

## 🔧 **cURL Advanced Examples**

### Batch Repository Status Check

```bash
#!/bin/bash
# check_all_repos.sh

API_BASE="http://localhost:8080/api/v1"

echo "🔍 Checking all repository statuses..."

# Get repository list
repos=$(curl -s "$API_BASE/repositories" | jq -r '.data[].name')

for repo in $repos; do
    echo "Checking $repo..."
    
    status=$(curl -s "$API_BASE/repositories/$repo/status" | jq -r '.data.status')
    last_poll=$(curl -s "$API_BASE/repositories/$repo/status" | jq -r '.data.last_poll')
    
    if [ "$status" = "healthy" ]; then
        echo "  ✅ $repo: $status (last poll: $last_poll)"
    else
        echo "  ❌ $repo: $status"
        # Get error details
        error=$(curl -s "$API_BASE/repositories/$repo" | jq -r '.data.error_message // "No error message"')
        echo "     Error: $error"
    fi
done
```

### Event Analysis Script

```bash
#!/bin/bash
# analyze_events.sh

API_BASE="http://localhost:8080/api/v1"

echo "📊 Event Analysis Report"
echo "======================="

# Get events from last 24 hours
since=$(date -d "24 hours ago" -Iseconds)
events=$(curl -s "$API_BASE/events?since=$since&limit=200")

total=$(echo "$events" | jq '.data.total')
echo "Total events (24h): $total"

# Event type breakdown
echo -e "\n📊 Event Types:"
echo "$events" | jq -r '.data.events[].type' | sort | uniq -c | sort -nr

# Repository activity
echo -e "\n🏢 Repository Activity:"
echo "$events" | jq -r '.data.events[].repository' | sort | uniq -c | sort -nr

# Failed events
failed_count=$(echo "$events" | jq '[.data.events[] | select(.status == "failed")] | length')
echo -e "\n❌ Failed Events: $failed_count"

if [ "$failed_count" -gt 0 ]; then
    echo "Failed event details:"
    echo "$events" | jq -r '.data.events[] | select(.status == "failed") | "  - \(.repository)/\(.branch): \(.error_message // "Unknown error")"'
fi
```

## 🔒 **Authentication Examples**

While the current version doesn't require authentication, here are examples for when authentication is added:

### API Key Authentication

```bash
# Future API key authentication
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/v1/repositories
```

### Bearer Token Authentication

```bash
# Future JWT token authentication
curl -H "Authorization: Bearer your-jwt-token" \
  http://localhost:8080/api/v1/events
```

### Python with Authentication

```python
class AuthenticatedRepoSentryClient(RepoSentryClient):
    def __init__(self, base_url="http://localhost:8080", api_key=None, token=None):
        super().__init__(base_url)
        self.headers = {}
        
        if api_key:
            self.headers['X-API-Key'] = api_key
        elif token:
            self.headers['Authorization'] = f'Bearer {token}'
    
    def _request(self, method, endpoint, **kwargs):
        kwargs.setdefault('headers', {}).update(self.headers)
        return requests.request(method, f"{self.base_url}{endpoint}", **kwargs)
```

## 📚 **Additional Resources**

- **Swagger UI**: Access interactive API documentation at `http://localhost:8080/swagger/`
- **Postman Collection**: [Download collection file](./postman/RepoSentry.postman_collection.json)
- **API Schema**: [OpenAPI 3.0 specification](./api-schema.yaml)
- **Rate Limits**: Current implementation has no rate limits, but monitoring is recommended
- **Webhook Format**: Events sent to Tekton follow GitHub webhook format for compatibility

---

For more detailed information, please refer to the [User Manual](USER_MANUAL.md) and [Technical Architecture](ARCHITECTURE.md) documentation.
