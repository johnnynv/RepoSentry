package runtime

import (
	"context"
	
	"github.com/johnnynv/RepoSentry/internal/api"
)

// runtimeAPIAdapter adapts Runtime interface to api.RuntimeProvider
type runtimeAPIAdapter struct {
	runtime Runtime
}

// newRuntimeAPIAdapter creates a new adapter
func newRuntimeAPIAdapter(runtime Runtime) api.RuntimeProvider {
	return &runtimeAPIAdapter{runtime: runtime}
}

// Health implements api.RuntimeProvider.Health
func (a *runtimeAPIAdapter) Health(ctx context.Context) api.RuntimeHealthStatus {
	health, err := a.runtime.Health(ctx)
	if err != nil || health == nil {
		return api.RuntimeHealthStatus{
			Healthy:    false,
			Components: make(map[string]api.ComponentHealth),
			Checks:     []api.HealthCheck{},
		}
	}

	// Convert components
	components := make(map[string]api.ComponentHealth)
	for name, state := range health.Components {
		components[name] = api.ComponentHealth{
			Status: string(state),
		}
	}

	// Convert health checks
	checks := make([]api.HealthCheck, len(health.Checks))
	for i, check := range health.Checks {
		checks[i] = api.HealthCheck{
			Name:     check.Name,
			Status:   string(check.Status),
			Duration: check.Duration,
		}
	}

	return api.RuntimeHealthStatus{
		Healthy:    health.Status == HealthStateHealthy,
		Components: components,
		Checks:     checks,
	}
}

// GetStatus implements api.RuntimeProvider.GetStatus
func (a *runtimeAPIAdapter) GetStatus() *api.RuntimeStatus {
	status := a.runtime.GetStatus()
	if status == nil {
		return &api.RuntimeStatus{
			State:      "unknown",
			Components: make(map[string]api.ComponentStatus),
		}
	}

	// Convert components
	components := make(map[string]api.ComponentStatus)
	for name, comp := range status.Components {
		components[name] = api.ComponentStatus{
			Name:      comp.Name,
			State:     string(comp.State),
			StartedAt: comp.StartedAt,
			Uptime:    comp.Uptime,
			Health:    string(comp.Health),
		}
	}

	return &api.RuntimeStatus{
		State:      string(status.State),
		StartedAt:  status.StartedAt,
		Uptime:     status.Uptime,
		Version:    status.Version,
		Components: components,
	}
}
