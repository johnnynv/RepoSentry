package gitclient

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter defines interface for rate limiting
type RateLimiter interface {
	// Wait blocks until the rate limiter allows the request
	Wait(ctx context.Context) error

	// Allow returns true if the request can proceed immediately
	Allow() bool

	// GetLimit returns current rate limit info
	GetLimit() RateLimitInfo

	// UpdateLimit updates the rate limit based on API response
	UpdateLimit(limit, remaining int, resetTime time.Time)
}

// RateLimitInfo represents current rate limit status
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetTime time.Time `json:"reset_time"`
	Provider  string    `json:"provider"`
}

// GitHubRateLimiter implements rate limiting for GitHub API
type GitHubRateLimiter struct {
	limiter   *rate.Limiter
	mu        sync.RWMutex
	limit     int
	remaining int
	resetTime time.Time
}

// NewGitHubRateLimiter creates a GitHub rate limiter
func NewGitHubRateLimiter() *GitHubRateLimiter {
	// GitHub API allows 5000 requests per hour for authenticated users
	// We set it slightly lower for safety: 4000 requests/hour = ~1.11 requests/second
	return &GitHubRateLimiter{
		limiter:   rate.NewLimiter(rate.Limit(1.0), 10), // 1 req/sec, burst of 10
		limit:     5000,
		remaining: 5000,
		resetTime: time.Now().Add(time.Hour),
	}
}

// Wait blocks until the rate limiter allows the request
func (r *GitHubRateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}

// Allow returns true if the request can proceed immediately
func (r *GitHubRateLimiter) Allow() bool {
	return r.limiter.Allow()
}

// GetLimit returns current rate limit info
func (r *GitHubRateLimiter) GetLimit() RateLimitInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RateLimitInfo{
		Limit:     r.limit,
		Remaining: r.remaining,
		ResetTime: r.resetTime,
		Provider:  "github",
	}
}

// UpdateLimit updates the rate limit based on API response
func (r *GitHubRateLimiter) UpdateLimit(limit, remaining int, resetTime time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limit = limit
	r.remaining = remaining
	r.resetTime = resetTime

	// Adjust limiter based on remaining requests
	if remaining < 100 && time.Until(resetTime) > 10*time.Minute {
		// Slow down significantly if we're running low
		r.limiter.SetLimit(rate.Limit(0.1)) // 1 request per 10 seconds
	} else if remaining < 1000 {
		// Slow down moderately
		r.limiter.SetLimit(rate.Limit(0.5)) // 1 request per 2 seconds
	} else {
		// Normal rate
		r.limiter.SetLimit(rate.Limit(1.0)) // 1 request per second
	}
}

// GitLabRateLimiter implements rate limiting for GitLab API
type GitLabRateLimiter struct {
	limiter   *rate.Limiter
	mu        sync.RWMutex
	limit     int
	remaining int
	resetTime time.Time
}

// NewGitLabRateLimiter creates a GitLab rate limiter
func NewGitLabRateLimiter() *GitLabRateLimiter {
	// GitLab API allows 2000 requests per minute by default
	// We set it lower for safety: 8 requests/second with burst of 5
	return &GitLabRateLimiter{
		limiter:   rate.NewLimiter(rate.Limit(8.0), 5),
		limit:     2000,
		remaining: 2000,
		resetTime: time.Now().Add(time.Minute),
	}
}

// Wait blocks until the rate limiter allows the request
func (r *GitLabRateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}

// Allow returns true if the request can proceed immediately
func (r *GitLabRateLimiter) Allow() bool {
	return r.limiter.Allow()
}

// GetLimit returns current rate limit info
func (r *GitLabRateLimiter) GetLimit() RateLimitInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RateLimitInfo{
		Limit:     r.limit,
		Remaining: r.remaining,
		ResetTime: r.resetTime,
		Provider:  "gitlab",
	}
}

// UpdateLimit updates the rate limit based on API response
func (r *GitLabRateLimiter) UpdateLimit(limit, remaining int, resetTime time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limit = limit
	r.remaining = remaining
	r.resetTime = resetTime

	// Adjust limiter based on remaining requests
	if remaining < 50 && time.Until(resetTime) > 30*time.Second {
		// Slow down significantly if we're running low
		r.limiter.SetLimit(rate.Limit(0.5)) // 1 request per 2 seconds
	} else if remaining < 200 {
		// Slow down moderately
		r.limiter.SetLimit(rate.Limit(2.0)) // 2 requests per second
	} else {
		// Normal rate
		r.limiter.SetLimit(rate.Limit(8.0)) // 8 requests per second
	}
}

// NoOpRateLimiter is a no-operation rate limiter for testing
type NoOpRateLimiter struct{}

// NewNoOpRateLimiter creates a no-op rate limiter
func NewNoOpRateLimiter() *NoOpRateLimiter {
	return &NoOpRateLimiter{}
}

// Wait does nothing for no-op limiter
func (r *NoOpRateLimiter) Wait(ctx context.Context) error {
	return nil
}

// Allow always returns true for no-op limiter
func (r *NoOpRateLimiter) Allow() bool {
	return true
}

// GetLimit returns unlimited rate limit info
func (r *NoOpRateLimiter) GetLimit() RateLimitInfo {
	return RateLimitInfo{
		Limit:     999999,
		Remaining: 999999,
		ResetTime: time.Now().Add(time.Hour),
		Provider:  "noop",
	}
}

// UpdateLimit does nothing for no-op limiter
func (r *NoOpRateLimiter) UpdateLimit(limit, remaining int, resetTime time.Time) {
	// No-op
}
