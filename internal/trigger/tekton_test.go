package trigger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

func TestTektonTrigger_NewTektonTrigger(t *testing.T) {
	tests := []struct {
		name    string
		config  TriggerConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: TriggerConfig{
				Type:    "tekton",
				Enabled: true,
				Tekton: TektonConfig{
					EventListenerURL: "http://localhost:8080",
					Namespace:        "tekton-pipelines",
				},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Missing EventListener URL",
			config: TriggerConfig{
				Type:    "tekton",
				Enabled: true,
				Tekton: TektonConfig{
					Namespace: "tekton-pipelines",
				},
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid URL",
			config: TriggerConfig{
				Type:    "tekton",
				Enabled: true,
				Tekton: TektonConfig{
					EventListenerURL: "invalid-url",
					Namespace:        "tekton-pipelines",
				},
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero timeout",
			config: TriggerConfig{
				Type:    "tekton",
				Enabled: true,
				Tekton: TektonConfig{
					EventListenerURL: "http://localhost:8080",
					Namespace:        "tekton-pipelines",
				},
				Timeout: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewTektonTrigger(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if trigger == nil {
				t.Error("Expected trigger instance but got nil")
				return
			}

			if trigger.GetType() != "tekton" {
				t.Errorf("Expected type 'tekton', got %s", trigger.GetType())
			}
		})
	}
}

func TestTektonTrigger_SendEvent(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-GitHub-Event") != "push" {
			t.Errorf("Expected X-GitHub-Event push, got %s", r.Header.Get("X-GitHub-Event"))
		}

		// Parse request body
		var payload TektonPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify payload content
		if payload.Source != "reposentry" {
			t.Errorf("Expected source 'reposentry', got %s", payload.Source)
		}
		if payload.Repository.Name != "test-repo" {
			t.Errorf("Expected repository 'test-repo', got %s", payload.Repository.Name)
		}

		// Send success response
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status": "accepted"}`))
	}))
	defer server.Close()

	// Create trigger configuration
	config := TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Tekton: TektonConfig{
			EventListenerURL: server.URL,
			Namespace:        "tekton-pipelines",
		},
		Timeout: 30 * time.Second,
	}

	trigger, err := NewTektonTrigger(config)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	// Create test event
	event := types.Event{
		ID:         "test_event_123",
		Type:       types.EventTypeBranchUpdated,
		Repository: "test-repo",
		Branch:     "main",
		CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
		Provider:   "github",
		Timestamp:  time.Now(),
		Metadata: map[string]string{
			"repository_url": "https://github.com/owner/test-repo",
		},
	}

	// Send event
	ctx := context.Background()
	result, err := trigger.SendEvent(ctx, event)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}

	if result.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status code %d, got %d", http.StatusAccepted, result.StatusCode)
	}

	if result.EventID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, result.EventID)
	}
}

func TestTektonTrigger_SendEvent_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	config := TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Tekton: TektonConfig{
			EventListenerURL: server.URL,
			Namespace:        "tekton-pipelines",
		},
		Timeout: 30 * time.Second,
	}

	trigger, err := NewTektonTrigger(config)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	event := types.Event{
		ID:         "test_event_error",
		Type:       types.EventTypeBranchUpdated,
		Repository: "test-repo",
		Branch:     "main",
		CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
		Provider:   "github",
		Timestamp:  time.Now(),
		Metadata: map[string]string{
			"repository_url": "https://github.com/owner/test-repo",
		},
	}

	ctx := context.Background()
	result, err := trigger.SendEvent(ctx, event)

	if err == nil {
		t.Error("Expected error but got none")
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	if result.Success {
		t.Error("Expected failure but got success")
	}

	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, result.StatusCode)
	}

	// Check error type
	if triggerErr, ok := err.(*TriggerError); ok {
		if triggerErr.Type != ErrorTypeServer {
			t.Errorf("Expected error type %s, got %s", ErrorTypeServer, triggerErr.Type)
		}
	} else {
		t.Errorf("Expected TriggerError, got %T", err)
	}
}

func TestTektonTrigger_BatchSendEvents(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status": "accepted"}`))
	}))
	defer server.Close()

	config := TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Tekton: TektonConfig{
			EventListenerURL: server.URL,
			Namespace:        "tekton-pipelines",
		},
		Timeout: 30 * time.Second,
	}

	trigger, err := NewTektonTrigger(config)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	// Create multiple test events
	events := []types.Event{
		{
			ID:         "batch_event_1",
			Type:       types.EventTypeBranchUpdated,
			Repository: "test-repo-1",
			Branch:     "main",
			CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
			Provider:   "github",
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"repository_url": "https://github.com/owner/test-repo-1",
			},
		},
		{
			ID:         "batch_event_2",
			Type:       types.EventTypeBranchCreated,
			Repository: "test-repo-2",
			Branch:     "develop",
			CommitSHA:  "efgh5678567890efgh5678567890efgh56785678",
			Provider:   "gitlab",
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"repository_url": "https://gitlab.com/owner/test-repo-2",
			},
		},
	}

	ctx := context.Background()
	result, err := trigger.BatchSendEvents(ctx, events)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	if result.TotalEvents != len(events) {
		t.Errorf("Expected total events %d, got %d", len(events), result.TotalEvents)
	}

	if result.SuccessCount != len(events) {
		t.Errorf("Expected success count %d, got %d", len(events), result.SuccessCount)
	}

	if result.FailureCount != 0 {
		t.Errorf("Expected failure count 0, got %d", result.FailureCount)
	}

	if len(result.Results) != len(events) {
		t.Errorf("Expected %d results, got %d", len(events), len(result.Results))
	}

	// Verify all requests were made
	if requestCount != len(events) {
		t.Errorf("Expected %d requests to server, got %d", len(events), requestCount)
	}
}

func TestTektonTrigger_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "Healthy server",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "healthy"}`))
			},
			expectError: false,
		},
		{
			name: "Server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "server error"}`))
			},
			expectError: true,
		},
		{
			name: "Client error (should not fail health check)",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "not found"}`))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			config := TriggerConfig{
				Type:    "tekton",
				Enabled: true,
				Tekton: TektonConfig{
					EventListenerURL: server.URL,
					Namespace:        "tekton-pipelines",
				},
				Timeout: 30 * time.Second,
			}

			trigger, err := NewTektonTrigger(config)
			if err != nil {
				t.Fatalf("Failed to create trigger: %v", err)
			}

			ctx := context.Background()
			err = trigger.HealthCheck(ctx)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestTektonTrigger_GetMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	config := TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Tekton: TektonConfig{
			EventListenerURL: server.URL,
			Namespace:        "tekton-pipelines",
		},
		Timeout: 30 * time.Second,
	}

	trigger, err := NewTektonTrigger(config)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	// Initial metrics should be zero
	metrics := trigger.GetMetrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Expected initial total requests 0, got %d", metrics.TotalRequests)
	}

	// Send an event to update metrics
	event := types.Event{
		ID:         "metrics_test",
		Type:       types.EventTypeBranchUpdated,
		Repository: "test-repo",
		Branch:     "main",
		CommitSHA:  "abcd1234567890abcdef1234567890abcdef1234",
		Provider:   "github",
		Timestamp:  time.Now(),
		Metadata: map[string]string{
			"repository_url": "https://github.com/owner/test-repo",
		},
	}

	ctx := context.Background()
	_, err = trigger.SendEvent(ctx, event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check updated metrics
	metrics = trigger.GetMetrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("Expected total requests 1, got %d", metrics.TotalRequests)
	}
	if metrics.SuccessfulSends != 1 {
		t.Errorf("Expected successful sends 1, got %d", metrics.SuccessfulSends)
	}
	if metrics.FailedSends != 0 {
		t.Errorf("Expected failed sends 0, got %d", metrics.FailedSends)
	}
	if metrics.Uptime <= 0 {
		t.Errorf("Expected positive uptime, got %v", metrics.Uptime)
	}
}

func TestTektonTrigger_Close(t *testing.T) {
	config := TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Tekton: TektonConfig{
			EventListenerURL: "http://localhost:8080",
			Namespace:        "tekton-pipelines",
		},
		Timeout: 30 * time.Second,
	}

	trigger, err := NewTektonTrigger(config)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	err = trigger.Close()
	if err != nil {
		t.Errorf("Unexpected error during close: %v", err)
	}
}
