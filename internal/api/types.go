package api

import (
	"context"
	"time"
)

// RuntimeHealthStatus represents runtime health status
type RuntimeHealthStatus struct {
	Healthy    bool                       `json:"healthy"`
	Components map[string]ComponentHealth `json:"components"`
	Checks     []HealthCheck             `json:"checks"`
}

// ComponentHealth represents individual component health
type ComponentHealth struct {
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration,omitempty"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
}

// RuntimeStatus represents runtime status
type RuntimeStatus struct {
	State      string                         `json:"state"`
	StartedAt  time.Time                     `json:"started_at"`
	Uptime     time.Duration                 `json:"uptime"`
	Version    string                        `json:"version"`
	Components map[string]ComponentStatus    `json:"components"`
}

// ComponentStatus represents individual component status
type ComponentStatus struct {
	Name      string        `json:"name"`
	State     string        `json:"state"`
	StartedAt time.Time     `json:"started_at"`
	Uptime    time.Duration `json:"uptime"`
	Health    string        `json:"health"`
}

// RuntimeProvider interface for runtime operations
type RuntimeProvider interface {
	Health(ctx context.Context) RuntimeHealthStatus
	GetStatus() *RuntimeStatus
}
