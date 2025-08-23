package api

import "time"

// JSONResponse represents the standard API response format
// @Description Standard API response wrapper
type JSONResponse struct {
	Success   bool        `json:"success" example:"true"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp" example:"2023-12-01T10:00:00Z"`
} // @name JSONResponse

// HealthStatus represents system health status
// @Description System health check response
type HealthStatus struct {
	Healthy    bool                             `json:"healthy" example:"true"`
	Components map[string]ComponentHealthStatus `json:"components"`
} // @name HealthStatus

// ComponentHealthStatus represents individual component health
// @Description Health status of a single component
type ComponentHealthStatus struct {
	Status      string `json:"status" example:"healthy"`
	Message     string `json:"message,omitempty" example:"Component is running normally"`
	LastChecked string `json:"last_checked,omitempty" example:"2023-12-01T10:00:00Z"`
} // @name ComponentHealthStatus

// RepositoryInfo represents repository configuration
// @Description Repository monitoring configuration
type RepositoryInfo struct {
	Name            string `json:"name" example:"my-repo"`
	URL             string `json:"url" example:"https://github.com/org/repo"`
	Provider        string `json:"provider" example:"github"`
	BranchRegex     string `json:"branch_regex" example:"^(main|develop)$"`
	PollingInterval string `json:"polling_interval" example:"5m"`
	Status          string `json:"status" example:"active"`
	LastChecked     string `json:"last_checked,omitempty" example:"2023-12-01T10:00:00Z"`
} // @name RepositoryInfo

// EventInfo represents a monitoring event
// @Description Repository monitoring event
type EventInfo struct {
	ID         string                 `json:"id" example:"evt_123"`
	Type       string                 `json:"type" example:"push"`
	Repository string                 `json:"repository" example:"my-repo"`
	Branch     string                 `json:"branch" example:"main"`
	CommitSHA  string                 `json:"commit_sha" example:"abc123def456"`
	Status     string                 `json:"status" example:"processed"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at" example:"2023-12-01T10:00:00Z"`
	UpdatedAt  time.Time              `json:"updated_at" example:"2023-12-01T10:00:00Z"`
} // @name EventInfo

// SystemMetrics represents application metrics
// @Description Application performance metrics
type SystemMetrics struct {
	System map[string]interface{} `json:"system"`
	API    map[string]interface{} `json:"api"`
} // @name SystemMetrics

// VersionInfo represents version information
// @Description Application and API version information
type VersionInfo struct {
	App       string `json:"app" example:"1.0.0"`
	API       string `json:"api" example:"v1"`
	BuildTime string `json:"build_time" example:"2023-12-01T10:00:00Z"`
	GitCommit string `json:"git_commit" example:"abc123d"`
} // @name VersionInfo

// ErrorResponse represents an error response
// @Description Standard error response format
type ErrorResponse struct {
	Success   bool      `json:"success" example:"false"`
	Error     string    `json:"error" example:"Resource not found"`
	Timestamp time.Time `json:"timestamp" example:"2023-12-01T10:00:00Z"`
} // @name ErrorResponse
