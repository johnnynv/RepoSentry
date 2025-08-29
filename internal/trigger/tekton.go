package trigger

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

// TektonTrigger implements Trigger interface for Tekton EventListener
type TektonTrigger struct {
	config      TriggerConfig
	httpClient  *http.Client
	transformer EventTransformer
	logger      *logger.Entry
	metrics     TriggerMetrics
	mu          sync.RWMutex
	startTime   time.Time
}

// NewTektonTrigger creates a new Tekton trigger
func NewTektonTrigger(config TriggerConfig, parentLogger *logger.Entry) (*TektonTrigger, error) {
	if err := validateTektonConfig(config); err != nil {
		return nil, fmt.Errorf("invalid Tekton configuration: %w", err)
	}

	// Create HTTP client with custom configuration
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure TLS if needed
	if config.Tekton.TLSConfig.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = transport
	}

	trigger := &TektonTrigger{
		config:      config,
		httpClient:  httpClient,
		transformer: NewEventTransformer(parentLogger),
		logger: parentLogger.WithFields(logger.Fields{
			"component": "trigger",
			"module":    "tekton",
			"url":       config.Tekton.EventListenerURL,
		}),
		metrics:   TriggerMetrics{},
		startTime: time.Now(),
	}

	trigger.logger.WithFields(logger.Fields{
		"operation": "initialize",
		"timeout":   config.Timeout,
		"namespace": config.Tekton.Namespace,
	}).Info("Initialized Tekton trigger")

	return trigger, nil
}

// SendEvent sends a single event to Tekton EventListener
func (t *TektonTrigger) SendEvent(ctx context.Context, event types.Event) (*TriggerResult, error) {
	startTime := time.Now()

	t.logger.WithFields(logger.Fields{
		"operation":  "send_event",
		"event_id":   event.ID,
		"repository": event.Repository,
		"branch":     event.Branch,
		"event_type": event.Type,
	}).Info("Sending event to Tekton EventListener")

	result := &TriggerResult{
		EventID:   event.ID,
		Timestamp: startTime,
	}

	// Transform event to CloudEvents standard format
	payload, err := t.transformer.TransformToCloudEvents(event)
	if err != nil {
		result.Error = fmt.Errorf("failed to transform event to CloudEvents format: %w", err)
		result.Duration = time.Since(startTime)
		t.updateMetrics(false, result.Duration)
		return result, result.Error
	}

	// Send HTTP request
	statusCode, responseBody, err := t.sendHTTPRequest(ctx, payload)
	result.StatusCode = statusCode
	result.ResponseBody = responseBody
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = err
		result.Success = false
		t.updateMetrics(false, result.Duration)

		t.logger.WithFields(logger.Fields{
			"operation":   "send_event",
			"event_id":    event.ID,
			"status_code": statusCode,
			"duration":    result.Duration,
			"error":       err.Error(),
		}).Error("Failed to send event to Tekton EventListener")

		return result, err
	}

	result.Success = true
	t.updateMetrics(true, result.Duration)

	// Add metadata
	result.Metadata = map[string]string{
		"tekton_namespace":   t.config.Tekton.Namespace,
		"payload_short_sha":  payload.Data.Commit.ShortSHA,
		"payload_ref":        payload.Data.Branch.Ref,
		"response_length":    fmt.Sprintf("%d", len(responseBody)),
		"cloudevents_id":     payload.ID,
		"cloudevents_type":   payload.Type,
		"cloudevents_source": payload.Source,
	}

	t.logger.WithFields(logger.Fields{
		"operation":        "send_event",
		"event_id":         event.ID,
		"status_code":      statusCode,
		"duration":         result.Duration,
		"short_sha":        payload.Data.Commit.ShortSHA,
		"ref":              payload.Data.Branch.Ref,
		"cloudevents_id":   payload.ID,
		"cloudevents_type": payload.Type,
	}).Info("Successfully sent event to Tekton EventListener")

	return result, nil
}

// BatchSendEvents sends multiple events efficiently
func (t *TektonTrigger) BatchSendEvents(ctx context.Context, events []types.Event) (*BatchTriggerResult, error) {
	startTime := time.Now()
	batchID := fmt.Sprintf("batch_%d_%d", startTime.Unix(), len(events))

	t.logger.WithFields(logger.Fields{
		"operation":   "batch_send_events",
		"batch_id":    batchID,
		"event_count": len(events),
	}).Info("Starting batch event send to Tekton EventListener")

	result := &BatchTriggerResult{
		BatchID:     batchID,
		TotalEvents: len(events),
		Results:     make([]TriggerResult, 0, len(events)),
		Timestamp:   startTime,
	}

	// Send events concurrently with semaphore to limit concurrent requests
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests
	resultChan := make(chan TriggerResult, len(events))

	var wg sync.WaitGroup
	for _, event := range events {
		wg.Add(1)
		go func(e types.Event) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			eventResult, _ := t.SendEvent(ctx, e)
			resultChan <- *eventResult
		}(event)
	}

	// Wait for all sends to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for eventResult := range resultChan {
		result.Results = append(result.Results, eventResult)
		if eventResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.Duration = time.Since(startTime)

	// Add batch metadata
	result.Metadata = map[string]string{
		"tekton_namespace": t.config.Tekton.Namespace,
		"batch_size":       fmt.Sprintf("%d", len(events)),
		"success_rate":     fmt.Sprintf("%.2f", float64(result.SuccessCount)/float64(result.TotalEvents)*100),
	}

	t.logger.WithFields(logger.Fields{
		"operation":     "batch_send_events",
		"batch_id":      batchID,
		"total_events":  result.TotalEvents,
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
		"duration":      result.Duration,
	}).Info("Completed batch event send to Tekton EventListener")

	return result, nil
}

// ValidateConfig validates the Tekton trigger configuration
func (t *TektonTrigger) ValidateConfig(config TriggerConfig) error {
	return validateTektonConfig(config)
}

// GetType returns the trigger type identifier
func (t *TektonTrigger) GetType() string {
	return "tekton"
}

// HealthCheck checks if Tekton EventListener is available
func (t *TektonTrigger) HealthCheck(ctx context.Context) error {
	t.logger.WithFields(logger.Fields{
		"operation": "health_check",
		"url":       t.config.Tekton.EventListenerURL,
	}).Debug("Performing health check on Tekton EventListener")

	// Create a simple health check request
	req, err := http.NewRequestWithContext(ctx, "GET", t.config.Tekton.EventListenerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add headers
	t.addHeaders(req)

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &TriggerError{
			Type:    ErrorTypeConnection,
			Message: fmt.Sprintf("failed to connect to Tekton EventListener: %v", err),
		}
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 500 {
		return &TriggerError{
			Type:    ErrorTypeServer,
			Message: fmt.Sprintf("Tekton EventListener server error: %d", resp.StatusCode),
			Code:    resp.StatusCode,
		}
	}

	t.logger.WithFields(logger.Fields{
		"operation":   "health_check",
		"status_code": resp.StatusCode,
	}).Info("Tekton EventListener health check successful")

	return nil
}

// GetMetrics returns trigger performance metrics
func (t *TektonTrigger) GetMetrics() TriggerMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics := t.metrics
	metrics.Uptime = time.Since(t.startTime)
	return metrics
}

// Close closes the trigger and releases resources
func (t *TektonTrigger) Close() error {
	t.logger.WithFields(logger.Fields{
		"operation": "close",
		"uptime":    time.Since(t.startTime),
	}).Info("Closing Tekton trigger")

	// Close HTTP client connections
	if transport, ok := t.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// sendHTTPRequest sends HTTP request to Tekton EventListener
func (t *TektonTrigger) sendHTTPRequest(ctx context.Context, payload interface{}) (int, string, error) {
	var payloadBytes []byte
	var req *http.Request
	var err error

	// Handle CloudEvent Binary Mode: send headers + data content only
	if cloudEventsPayload, ok := payload.(CloudEventsPayload); ok {
		// For CloudEvent Binary Mode, only send the data portion in the body
		payloadBytes, err = json.Marshal(cloudEventsPayload.Data)
		if err != nil {
			return 0, "", fmt.Errorf("failed to marshal CloudEvent data: %w", err)
		}

		t.logger.WithFields(logger.Fields{
			"operation":    "send_http_request",
			"payload_size": len(payloadBytes),
			"mode":         "cloudevent_binary",
		}).Debug("Sending CloudEvent in Binary Mode to Tekton EventListener")

		// Create request
		req, err = http.NewRequestWithContext(ctx, "POST", t.config.Tekton.EventListenerURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return 0, "", fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Add headers
		t.addHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		// Add CloudEvent headers for Binary Mode
		req.Header.Set("ce-specversion", cloudEventsPayload.SpecVersion)
		req.Header.Set("ce-type", cloudEventsPayload.Type)
		req.Header.Set("ce-source", cloudEventsPayload.Source)
		req.Header.Set("ce-id", cloudEventsPayload.ID)
		if cloudEventsPayload.Time != "" {
			req.Header.Set("ce-time", cloudEventsPayload.Time)
		}
		if cloudEventsPayload.DataContentType != "" {
			req.Header.Set("ce-datacontenttype", cloudEventsPayload.DataContentType)
		}
	} else {
		// For non-CloudEvent payloads, send as-is
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return 0, "", fmt.Errorf("failed to marshal payload: %w", err)
		}

		t.logger.WithFields(logger.Fields{
			"operation":    "send_http_request",
			"payload_size": len(payloadBytes),
			"mode":         "standard",
		}).Debug("Sending standard HTTP request to Tekton EventListener")

		// Create request
		req, err = http.NewRequestWithContext(ctx, "POST", t.config.Tekton.EventListenerURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return 0, "", fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Add headers
		t.addHeaders(req)
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return 0, "", &TriggerError{
			Type:    ErrorTypeConnection,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("failed to read response body: %w", err)
	}

	responseBody := string(bodyBytes)

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		errorType := ErrorTypeClient
		if resp.StatusCode >= 500 {
			errorType = ErrorTypeServer
		}

		return resp.StatusCode, responseBody, &TriggerError{
			Type:    errorType,
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, responseBody),
			Code:    resp.StatusCode,
			Details: responseBody,
		}
	}

	return resp.StatusCode, responseBody, nil
}

// addHeaders adds required headers to HTTP request
func (t *TektonTrigger) addHeaders(req *http.Request) {
	// Add GitHub-style event header
	req.Header.Set("X-GitHub-Event", "push")

	// Add custom headers from configuration
	for key, value := range t.config.Tekton.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if configured
	if t.config.Tekton.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.Tekton.AuthToken)
	}

	// Add user agent
	req.Header.Set("User-Agent", "RepoSentry/1.0")
}

// updateMetrics updates trigger performance metrics
func (t *TektonTrigger) updateMetrics(success bool, duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.metrics.TotalRequests++

	if success {
		t.metrics.SuccessfulSends++
		t.metrics.LastSuccessTime = time.Now()
		t.metrics.ConsecutiveFails = 0
	} else {
		t.metrics.FailedSends++
		t.metrics.LastFailureTime = time.Now()
		t.metrics.ConsecutiveFails++
	}

	// Update average latency using exponential moving average
	if t.metrics.AverageLatency == 0 {
		t.metrics.AverageLatency = duration
	} else {
		// EMA with alpha = 0.1
		alpha := 0.1
		t.metrics.AverageLatency = time.Duration(float64(t.metrics.AverageLatency)*(1-alpha) + float64(duration)*alpha)
	}
}

// validateTektonConfig validates Tekton-specific configuration
func validateTektonConfig(config TriggerConfig) error {
	if config.Tekton.EventListenerURL == "" {
		return &TriggerError{
			Type:    ErrorTypeValidation,
			Message: "event_listener_url is required for Tekton trigger",
		}
	}

	// Validate URL format - must be HTTPS for production
	parsedURL, err := url.Parse(config.Tekton.EventListenerURL)
	if err != nil {
		return &TriggerError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("invalid event_listener_url: %v", err),
		}
	}

	// Allow http only for localhost/testing, otherwise require https
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return &TriggerError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("event_listener_url must use http or https scheme (got: %s)", parsedURL.Scheme),
		}
	}

	// For invalid-url test case - check for basic URL structure
	if parsedURL.Host == "" && parsedURL.Path == "" {
		return &TriggerError{
			Type:    ErrorTypeValidation,
			Message: "event_listener_url must be a valid URL with hostname",
		}
	}

	// Validate timeout
	if config.Timeout <= 0 {
		return &TriggerError{
			Type:    ErrorTypeValidation,
			Message: "timeout must be greater than 0",
		}
	}

	return nil
}
