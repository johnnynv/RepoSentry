# RepoSentry API使用示例

RepoSentry提供完整的RESTful API和Swagger UI在线文档，方便开发者和运维人员使用。

## 🔗 **快速访问**

- **Swagger UI**: `http://localhost:8080/swagger/`
- **API文档JSON**: `http://localhost:8080/swagger/doc.json`
- **API基本信息**: `http://localhost:8080/api`

## 📋 **API概览**

### **Health & Status**
| 端点 | 方法 | 描述 |
|-----|------|------|
| `/health` | GET | 系统整体健康状态 |
| `/health/live` | GET | Kubernetes存活探针 |
| `/health/ready` | GET | Kubernetes就绪探针 |
| `/status` | GET | 运行时状态和组件信息 |

### **Repository Management**
| 端点 | 方法 | 描述 |
|-----|------|------|
| `/api/repositories` | GET | 列出所有监控的仓库 |
| `/api/repositories/{name}` | GET | 获取特定仓库详情 |

### **Event Management**
| 端点 | 方法 | 描述 |
|-----|------|------|
| `/api/events` | GET | 分页查询事件列表 |
| `/api/events/recent` | GET | 最近24小时事件 |
| `/api/events/{id}` | GET | 获取特定事件详情 |

### **System Information**
| 端点 | 方法 | 描述 |
|-----|------|------|
| `/metrics` | GET | 应用指标和统计 |
| `/version` | GET | 版本信息 |

## 💡 **使用示例**

### 1. **健康检查**

```bash
# 检查系统健康状态
curl -X GET "http://localhost:8080/health" \
  -H "accept: application/json"

# 响应示例
{
  "success": true,
  "data": {
    "healthy": true,
    "components": {
      "config": {
        "status": "healthy"
      },
      "storage": {
        "status": "healthy"
      },
      "git_client": {
        "status": "healthy"
      }
    }
  },
  "timestamp": "2023-12-01T10:00:00Z"
}
```

### 2. **查看监控的仓库**

```bash
# 获取所有仓库
curl -X GET "http://localhost:8080/api/repositories" \
  -H "accept: application/json"

# 响应示例
{
  "success": true,
  "data": {
    "total": 2,
    "repositories": [
      {
        "name": "example-repo",
        "url": "https://github.com/example/repo",
        "provider": "github",
        "branch_regex": "^(main|master)$",
        "polling_interval": "5m0s",
        "status": "active"
      }
    ]
  },
  "timestamp": "2023-12-01T10:00:00Z"
}
```

### 3. **查询事件**

```bash
# 分页查询事件
curl -X GET "http://localhost:8080/api/events?limit=10&offset=0" \
  -H "accept: application/json"

# 查询最近事件
curl -X GET "http://localhost:8080/api/events/recent" \
  -H "accept: application/json"

# 响应示例
{
  "success": true,
  "data": {
    "total": 25,
    "events": [
      {
        "id": "evt_123",
        "type": "push",
        "repository": "example-repo",
        "branch": "main",
        "commit_sha": "abc123def456",
        "status": "processed",
        "created_at": "2023-12-01T10:00:00Z",
        "updated_at": "2023-12-01T10:00:05Z"
      }
    ],
    "pagination": {
      "limit": 10,
      "offset": 0,
      "total": 25
    }
  },
  "timestamp": "2023-12-01T10:00:00Z"
}
```

### 4. **系统状态和指标**

```bash
# 获取系统状态
curl -X GET "http://localhost:8080/status" \
  -H "accept: application/json"

# 获取指标
curl -X GET "http://localhost:8080/metrics" \
  -H "accept: application/json"

# 获取版本信息
curl -X GET "http://localhost:8080/version" \
  -H "accept: application/json"
```

## 🔧 **参数说明**

### **事件查询参数**
- `limit`: 返回事件数量限制 (默认: 50, 最大: 1000)
- `offset`: 跳过的事件数量 (默认: 0)

### **响应格式**
所有API响应都遵循统一格式：

```json
{
  "success": true,           // 请求是否成功
  "data": {...},            // 响应数据
  "error": "error message", // 错误信息（仅在失败时）
  "timestamp": "2023-12-01T10:00:00Z" // 响应时间戳
}
```

## 🖥️ **使用Swagger UI**

1. **启动RepoSentry**:
   ```bash
   ./reposentry run --config=config.yaml
   ```

2. **访问Swagger UI**:
   打开浏览器访问: `http://localhost:8080/swagger/`

3. **交互式测试**:
   - 在Swagger UI中可以直接测试所有API端点
   - 查看详细的请求/响应schema
   - 查看API参数和示例

## 🔐 **认证 (未来功能)**

当前版本的API不需要认证，但未来版本将支持：
- API Key认证
- Bearer Token认证
- 基于角色的访问控制

## 📊 **监控集成**

RepoSentry API可以与各种监控工具集成：

### **Prometheus**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'reposentry'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### **健康检查脚本**
```bash
#!/bin/bash
# health_check.sh
response=$(curl -s http://localhost:8080/health)
healthy=$(echo $response | jq -r '.data.healthy')

if [ "$healthy" = "true" ]; then
  echo "RepoSentry is healthy"
  exit 0
else
  echo "RepoSentry is unhealthy"
  exit 1
fi
```

## 🛠️ **开发集成**

### **Python客户端示例**
```python
import requests

class RepoSentryClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def get_health(self):
        response = requests.get(f"{self.base_url}/health")
        return response.json()
    
    def get_repositories(self):
        response = requests.get(f"{self.base_url}/api/repositories")
        return response.json()
    
    def get_events(self, limit=50, offset=0):
        params = {"limit": limit, "offset": offset}
        response = requests.get(f"{self.base_url}/api/events", params=params)
        return response.json()

# 使用示例
client = RepoSentryClient()
health = client.get_health()
print(f"System healthy: {health['data']['healthy']}")
```

### **JavaScript/Node.js客户端示例**
```javascript
const axios = require('axios');

class RepoSentryClient {
  constructor(baseURL = 'http://localhost:8080') {
    this.client = axios.create({ baseURL });
  }

  async getHealth() {
    const response = await this.client.get('/health');
    return response.data;
  }

  async getRepositories() {
    const response = await this.client.get('/api/repositories');
    return response.data;
  }

  async getEvents(limit = 50, offset = 0) {
    const response = await this.client.get('/api/events', {
      params: { limit, offset }
    });
    return response.data;
  }
}

// 使用示例
const client = new RepoSentryClient();
client.getHealth().then(health => {
  console.log(`System healthy: ${health.data.healthy}`);
});
```

## 🚀 **最佳实践**

1. **错误处理**: 始终检查响应中的`success`字段
2. **分页**: 使用合适的`limit`和`offset`避免大量数据传输
3. **缓存**: 对于不频繁变化的数据（如仓库列表）可以适当缓存
4. **监控**: 定期调用`/health`端点监控服务状态
5. **日志**: 记录API调用日志便于调试和监控

## 📚 **更多资源**

- [RepoSentry GitHub](https://github.com/johnnynv/RepoSentry)
- [部署指南](../deployments/README.md)
- [配置文档](../configs/README.md)
- [故障排除](../docs/TROUBLESHOOTING.md)
