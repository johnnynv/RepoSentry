package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/storage"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
	// Note: httpSwagger is imported in router.go
)

// Server represents the HTTP API server
type Server struct {
	port          int
	server        *http.Server
	configManager *config.Manager
	storage       storage.Storage
	runtime       RuntimeProvider
	logger        *logger.Entry
}

// NewServer creates a new API server
func NewServer(port int, configManager *config.Manager, storage storage.Storage, parentLogger *logger.Entry) *Server {
	return &Server{
		port:          port,
		configManager: configManager,
		storage:       storage,
		runtime:       nil, // Will be set by SetRuntime
		logger: parentLogger.WithFields(logger.Fields{
			"component": "api",
			"module":    "server",
			"port":      port,
		}),
	}
}

// SetRuntime sets the runtime provider (called after creation)
func (s *Server) SetRuntime(runtime RuntimeProvider) {
	s.runtime = runtime
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Create router with all handlers
	router := s.setupRouter()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.WithFields(logger.Fields{
		"operation": "start",
		"addr":      s.server.Addr,
	}).Info("Starting API server")

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithFields(logger.Fields{
				"operation": "start",
				"error":     err.Error(),
			}).Error("API server failed to start")
		}
	}()

	s.logger.WithFields(logger.Fields{
		"operation": "start",
		"port":      s.port,
	}).Info("API server started successfully")

	return nil
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("Stopping API server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown API server: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"operation": "stop",
	}).Info("API server stopped successfully")

	return nil
}

// Health returns the server health status
func (s *Server) Health(ctx context.Context) error {
	// Simple health check - server is healthy if it's running
	return nil
}

// HTTP Handlers

// handleHealth returns overall system health
// @Summary Get system health
// @Description Returns the overall health status of all RepoSentry components
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "Healthy"
// @Success 503 {object} JSONResponse{data=object} "Unhealthy"
// @Router /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.runtime != nil {
		health := s.runtime.Health(ctx)
		response := NewJSONResponse(health)
		response.Write(w)
	} else {
		health := map[string]interface{}{
			"status": "healthy",
			"components": map[string]string{
				"api": "healthy",
			},
		}
		response := NewJSONResponse(health)
		response.Write(w)
	}
}

// handleLiveness returns liveness probe status
// @Summary Liveness probe
// @Description Kubernetes liveness probe endpoint - simple alive status
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "Alive"
// @Router /health/live [get]
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	response := NewJSONResponse(map[string]string{
		"status": "alive",
	})
	response.Write(w)
}

// handleReadiness returns readiness probe status
// @Summary Readiness probe
// @Description Kubernetes readiness probe endpoint - ready status
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "Ready"
// @Router /health/ready [get]
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	response := NewJSONResponse(map[string]string{
		"status": "ready",
	})
	response.Write(w)
}

// handleStatus returns system status and uptime
// @Summary Get system status
// @Description Returns runtime status and component information
// @Tags Status
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "System status"
// @Router /status [get]
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.runtime != nil {
		status := s.runtime.GetStatus()
		response := NewJSONResponse(*status)
		response.Write(w)
	} else {
		status := map[string]interface{}{
			"status":  "running",
			"message": "Runtime information not available",
		}
		response := NewJSONResponse(status)
		response.Write(w)
	}
}

// handleRepositories returns all configured repositories
// @Summary List repositories
// @Description Get all configured repositories with their monitoring status
// @Tags Repositories
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "List of repositories"
// @Router /api/repositories [get]
func (s *Server) handleRepositories(w http.ResponseWriter, r *http.Request) {
	repos := s.configManager.GetRepositories()

	// Convert to API format with time in seconds
	apiRepos := make([]map[string]interface{}, len(repos))
	for i, repo := range repos {
		apiRepos[i] = s.convertRepositoryToAPI(repo)
	}

	response := NewJSONResponse(map[string]interface{}{
		"total":        len(apiRepos),
		"repositories": apiRepos,
	})
	response.Write(w)
}

// handleRepository returns a specific repository by name
func (s *Server) handleRepository(w http.ResponseWriter, r *http.Request) {
	// Extract repository name from URL path
	path := r.URL.Path[len("/api/repositories/"):]
	if path == "" {
		response := NewErrorResponse("Repository name is required")
		response.WriteWithStatus(w, http.StatusBadRequest)
		return
	}

	repo, found := s.configManager.GetRepository(path)
	if !found {
		response := NewErrorResponse("Repository not found")
		response.WriteWithStatus(w, http.StatusNotFound)
		return
	}

	// Convert to API format with time in seconds
	apiRepo := s.convertRepositoryToAPI(*repo)

	response := NewJSONResponse(apiRepo)
	response.Write(w)
}

// convertRepositoryToAPI converts a Repository to API format with time in seconds
func (s *Server) convertRepositoryToAPI(repo types.Repository) map[string]interface{} {
	apiRepo := map[string]interface{}{
		"name":         repo.Name,
		"url":          repo.URL,
		"provider":     repo.Provider,
		"branch_regex": repo.BranchRegex,
		"enabled":      repo.Enabled,
	}

	// Convert polling interval to seconds
	if repo.PollingInterval > 0 {
		seconds := int(repo.PollingInterval.Seconds())
		apiRepo["polling_interval_seconds"] = seconds
		apiRepo["polling_interval"] = fmt.Sprintf("%ds", seconds)
	}

	// Add API base URL if present
	if repo.APIBaseURL != "" {
		apiRepo["api_base_url"] = repo.APIBaseURL
	}

	return apiRepo
}

// handleEvents returns events with pagination
// @Summary List events
// @Description Get events with pagination support
// @Tags Events
// @Accept json
// @Produce json
// @Param limit query int false "Number of events to return (max 1000)" default(50)
// @Param offset query int false "Number of events to skip" default(0)
// @Success 200 {object} JSONResponse{data=object} "Paginated list of events"
// @Router /api/events [get]
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	events, err := s.storage.GetEvents(ctx, limit, offset)
	if err != nil {
		s.logger.WithFields(logger.Fields{
			"error":  err.Error(),
			"limit":  limit,
			"offset": offset,
		}).Error("Failed to get events")

		response := NewErrorResponse("Failed to retrieve events")
		response.WriteWithStatus(w, http.StatusInternalServerError)
		return
	}

	response := NewJSONResponse(map[string]interface{}{
		"total":  len(events),
		"limit":  limit,
		"offset": offset,
		"events": events,
	})
	response.Write(w)
}

// handleRecentEvents returns recent events (last 24 hours)
func (s *Server) handleRecentEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	since := time.Now().Add(-24 * time.Hour)
	events, err := s.storage.GetEventsSince(ctx, since)
	if err != nil {
		s.logger.WithFields(logger.Fields{
			"error": err.Error(),
			"since": since,
		}).Error("Failed to get recent events")

		response := NewErrorResponse("Failed to retrieve recent events")
		response.WriteWithStatus(w, http.StatusInternalServerError)
		return
	}

	response := NewJSONResponse(map[string]interface{}{
		"total":  len(events),
		"since":  since,
		"events": events,
	})
	response.Write(w)
}

// handleEvent returns a specific event by ID
func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract event ID from URL path
	path := r.URL.Path[len("/api/events/"):]
	if path == "" || path == "recent" {
		response := NewErrorResponse("Event ID is required")
		response.WriteWithStatus(w, http.StatusBadRequest)
		return
	}

	event, err := s.storage.GetEvent(ctx, path)
	if err != nil {
		s.logger.WithFields(logger.Fields{
			"error":    err.Error(),
			"event_id": path,
		}).Error("Failed to get event")

		response := NewErrorResponse("Event not found")
		response.WriteWithStatus(w, http.StatusNotFound)
		return
	}

	response := NewJSONResponse(event)
	response.Write(w)
}

// handleMetrics returns basic system metrics
// @Summary Get metrics
// @Description Returns application metrics and statistics
// @Tags Metrics
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "Application metrics"
// @Router /metrics [get]
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"system": map[string]interface{}{
			"name":    "RepoSentry",
			"version": GetVersion().App,
		},
		"runtime": map[string]interface{}{
			// TODO: Add real runtime metrics
			"goroutines": "N/A",
			"memory":     "N/A",
		},
		"api": map[string]interface{}{
			"requests_total": "N/A", // TODO: Add metrics collection
		},
	}

	response := NewJSONResponse(metrics)
	response.Write(w)
}

// handleVersion returns API and application version information
// @Summary Get version information
// @Description Returns API and application version details
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {object} JSONResponse{data=object} "Version information"
// @Router /version [get]
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := GetVersion()
	response := NewJSONResponse(version)
	response.Write(w)
}
