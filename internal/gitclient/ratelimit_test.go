package gitclient

import (
	"context"
	"testing"
	"time"
)

func TestGitHubRateLimiter(t *testing.T) {
	limiter := NewGitHubRateLimiter()

	// Test initial state
	info := limiter.GetLimit()
	if info.Provider != "github" {
		t.Errorf("Expected provider 'github', got %s", info.Provider)
	}
	if info.Limit != 5000 {
		t.Errorf("Expected limit 5000, got %d", info.Limit)
	}
	if info.Remaining != 5000 {
		t.Errorf("Expected remaining 5000, got %d", info.Remaining)
	}

	// Test Allow (should be true initially)
	if !limiter.Allow() {
		t.Error("Expected Allow() to return true initially")
	}

	// Test UpdateLimit
	resetTime := time.Now().Add(time.Hour)
	limiter.UpdateLimit(5000, 100, resetTime)

	info = limiter.GetLimit()
	if info.Remaining != 100 {
		t.Errorf("Expected remaining 100, got %d", info.Remaining)
	}
	if !info.ResetTime.Equal(resetTime) {
		t.Errorf("Expected reset time %v, got %v", resetTime, info.ResetTime)
	}

	// Test Wait (should not block for short duration)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait() failed: %v", err)
	}

	// Should take some time due to rate limiting but not timeout
	if duration > 90*time.Millisecond {
		t.Errorf("Wait() took too long: %v", duration)
	}
}

func TestGitLabRateLimiter(t *testing.T) {
	limiter := NewGitLabRateLimiter()

	// Test initial state
	info := limiter.GetLimit()
	if info.Provider != "gitlab" {
		t.Errorf("Expected provider 'gitlab', got %s", info.Provider)
	}
	if info.Limit != 2000 {
		t.Errorf("Expected limit 2000, got %d", info.Limit)
	}
	if info.Remaining != 2000 {
		t.Errorf("Expected remaining 2000, got %d", info.Remaining)
	}

	// Test Allow (should be true initially)
	if !limiter.Allow() {
		t.Error("Expected Allow() to return true initially")
	}

	// Test UpdateLimit
	resetTime := time.Now().Add(time.Minute)
	limiter.UpdateLimit(2000, 50, resetTime)

	info = limiter.GetLimit()
	if info.Remaining != 50 {
		t.Errorf("Expected remaining 50, got %d", info.Remaining)
	}
	if !info.ResetTime.Equal(resetTime) {
		t.Errorf("Expected reset time %v, got %v", resetTime, info.ResetTime)
	}
}

func TestNoOpRateLimiter(t *testing.T) {
	limiter := NewNoOpRateLimiter()

	// Test initial state
	info := limiter.GetLimit()
	if info.Provider != "noop" {
		t.Errorf("Expected provider 'noop', got %s", info.Provider)
	}
	if info.Limit != 999999 {
		t.Errorf("Expected unlimited limit, got %d", info.Limit)
	}
	if info.Remaining != 999999 {
		t.Errorf("Expected unlimited remaining, got %d", info.Remaining)
	}

	// Test Allow (should always be true)
	if !limiter.Allow() {
		t.Error("Expected Allow() to always return true for no-op limiter")
	}

	// Test Wait (should not block)
	ctx := context.Background()
	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait() failed: %v", err)
	}

	// Should return immediately
	if duration > 10*time.Millisecond {
		t.Errorf("Wait() took too long for no-op limiter: %v", duration)
	}

	// Test UpdateLimit (should be no-op)
	limiter.UpdateLimit(100, 50, time.Now())

	// Should still be unlimited
	info = limiter.GetLimit()
	if info.Limit != 999999 {
		t.Errorf("Expected limit to remain unlimited after update, got %d", info.Limit)
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	// Use no-op limiter for concurrency test to avoid blocking
	limiter := NewNoOpRateLimiter()
	ctx := context.Background()

	// Test concurrent access
	const numGoroutines = 10
	const numRequests = 5

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequests; j++ {
				if err := limiter.Wait(ctx); err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Errorf("Concurrent access failed: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Test timed out")
		}
	}
}

func TestRateLimiterAdaptiveRates(t *testing.T) {
	limiter := NewGitHubRateLimiter()

	// Test high remaining requests (should use normal rate)
	limiter.UpdateLimit(5000, 4000, time.Now().Add(time.Hour))
	// Note: We can't easily test the internal rate limiter adjustment
	// without accessing private fields, but we can test that it doesn't crash

	// Test low remaining requests (should slow down)
	limiter.UpdateLimit(5000, 50, time.Now().Add(time.Hour))

	// Test very low remaining requests (should slow down significantly)
	limiter.UpdateLimit(5000, 10, time.Now().Add(time.Hour))

	// Verify the limiter still works
	info := limiter.GetLimit()
	if info.Remaining != 10 {
		t.Errorf("Expected remaining 10, got %d", info.Remaining)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	limiter := NewGitHubRateLimiter()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Wait should return context error
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}
