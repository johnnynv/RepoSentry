package api

import (
	"net/http"

	"github.com/johnnynv/RepoSentry/internal/api/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	// Import generated docs
	_ "github.com/johnnynv/RepoSentry/docs"
)

// setupRouter configures all API routes
func (s *Server) setupRouter() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)

	// Business API endpoints
	mux.HandleFunc("/api/repositories", s.handleRepositories)
	mux.HandleFunc("/api/repositories/", s.handleRepository) // with ID

	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/events/recent", s.handleRecentEvents)
	mux.HandleFunc("/api/events/", s.handleEvent) // with ID

	// System endpoints
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// API documentation and version
	mux.HandleFunc("/api", s.handleAPIDocumentation)
	mux.HandleFunc("/version", s.handleVersion)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	// Apply middleware
	handler := middleware.RequestLogger(s.logger)(mux)
	handler = middleware.CORS()(handler)
	handler = middleware.Recovery(s.logger)(handler)

	return handler
}

// handleAPIDocumentation provides comprehensive API documentation
func (s *Server) handleAPIDocumentation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	version := GetVersion()

	apiDoc := map[string]interface{}{
		"name":        "RepoSentry API",
		"version":     version.API,
		"app_version": version.App,
		"description": "Git Repository Monitoring and Event Tracking API",
		"base_url":    r.Host,
		"endpoints": map[string]interface{}{
			"health": map[string]interface{}{
				"GET /health": map[string]string{
					"description": "Overall health status of all components",
					"returns":     "JSON with component health details",
				},
				"GET /health/live": map[string]string{
					"description": "Kubernetes liveness probe endpoint",
					"returns":     "Simple alive status",
				},
				"GET /health/ready": map[string]string{
					"description": "Kubernetes readiness probe endpoint",
					"returns":     "Ready status",
				},
			},
			"repositories": map[string]interface{}{
				"GET /api/repositories": map[string]string{
					"description": "List all configured repositories",
					"returns":     "Array of repository configurations",
				},
				"GET /api/repositories/{name}": map[string]string{
					"description": "Get specific repository by name",
					"parameters":  "name: repository name",
					"returns":     "Single repository configuration",
				},
			},
			"events": map[string]interface{}{
				"GET /api/events": map[string]string{
					"description": "List events with pagination",
					"parameters":  "limit (max 1000), offset",
					"returns":     "Paginated list of events",
				},
				"GET /api/events/recent": map[string]string{
					"description": "List events from last 24 hours",
					"returns":     "Array of recent events",
				},
				"GET /api/events/{id}": map[string]string{
					"description": "Get specific event by ID",
					"parameters":  "id: event ID",
					"returns":     "Single event details",
				},
			},
			"system": map[string]interface{}{
				"GET /status": map[string]string{
					"description": "Runtime status and component uptime",
					"returns":     "System status and metrics",
				},
				"GET /metrics": map[string]string{
					"description": "Basic system metrics",
					"returns":     "Runtime and performance metrics",
				},
				"GET /version": map[string]string{
					"description": "API and application version information",
					"returns":     "Version details",
				},
			},
		},
		"response_format": map[string]interface{}{
			"success": map[string]interface{}{
				"success":   true,
				"data":      "response payload",
				"timestamp": "ISO8601 timestamp",
			},
			"error": map[string]interface{}{
				"success":   false,
				"error":     "error message",
				"timestamp": "ISO8601 timestamp",
			},
		},
		"middleware": []string{
			"Request logging",
			"CORS headers",
			"Panic recovery",
		},
	}

	w.WriteHeader(http.StatusOK)
	response := NewJSONResponse(apiDoc)
	response.Write(w)
}
