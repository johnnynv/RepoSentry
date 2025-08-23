# RepoSentry Tools

## Swagger UI Static Server

This tool allows you to view Swagger UI without running the full RepoSentry application.

### Usage

```bash
# From project root directory
make swagger-ui

# Or run directly
go run tools/swagger-static-server.go
```

### Access

- **Swagger UI**: http://localhost:8081/swagger
- **YAML file**: http://localhost:8081/swagger.yaml  
- **JSON file**: http://localhost:8081/swagger.json

### Use Cases

- View API documentation when webhook connections fail
- Offline API documentation browsing
- Development reference without full service startup
- API design validation

### Note

This server only provides static documentation viewing. API calls will not work since there's no backend service running.
