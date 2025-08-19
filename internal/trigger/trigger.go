package trigger

import (
	"context"
	"time"

	"github.com/johnnynv/RepoSentry/pkg/types"
)

// Trigger defines the interface for event triggers
type Trigger interface {
	// SendEvent sends an event to the trigger destination
	SendEvent(ctx context.Context, event types.Event) (*TriggerResult, error)
	
	// BatchSendEvents sends multiple events efficiently
	BatchSendEvents(ctx context.Context, events []types.Event) (*BatchTriggerResult, error)
	
	// ValidateConfig validates the trigger configuration
	ValidateConfig(config TriggerConfig) error
	
	// GetType returns the trigger type identifier
	GetType() string
	
	// HealthCheck checks if the trigger destination is available
	HealthCheck(ctx context.Context) error
	
	// GetMetrics returns trigger performance metrics
	GetMetrics() TriggerMetrics
	
	// Close closes the trigger and releases resources
	Close() error
}

// TriggerResult represents the result of a single trigger operation
type TriggerResult struct {
	EventID      string            `json:"event_id"`
	Success      bool              `json:"success"`
	StatusCode   int               `json:"status_code,omitempty"`
	ResponseBody string            `json:"response_body,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Error        error             `json:"error,omitempty"`
}

// BatchTriggerResult represents the result of batch trigger operations
type BatchTriggerResult struct {
	BatchID       string            `json:"batch_id"`
	TotalEvents   int               `json:"total_events"`
	SuccessCount  int               `json:"success_count"`
	FailureCount  int               `json:"failure_count"`
	Results       []TriggerResult   `json:"results"`
	Duration      time.Duration     `json:"duration"`
	Timestamp     time.Time         `json:"timestamp"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// TriggerConfig represents trigger configuration
type TriggerConfig struct {
	Type     string                 `yaml:"type" json:"type"`
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Tekton   TektonConfig          `yaml:"tekton,omitempty" json:"tekton,omitempty"`
	Webhook  WebhookConfig         `yaml:"webhook,omitempty" json:"webhook,omitempty"`
	Retry    RetryConfig           `yaml:"retry" json:"retry"`
	Timeout  time.Duration         `yaml:"timeout" json:"timeout"`
	Metadata map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// TektonConfig represents Tekton EventListener configuration
type TektonConfig struct {
	EventListenerURL string            `yaml:"event_listener_url" json:"event_listener_url"`
	Namespace        string            `yaml:"namespace" json:"namespace"`
	Headers          map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	AuthToken        string            `yaml:"auth_token,omitempty" json:"auth_token,omitempty"`
	TLSConfig        TLSConfig         `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// WebhookConfig represents generic webhook configuration
type WebhookConfig struct {
	URL       string            `yaml:"url" json:"url"`
	Method    string            `yaml:"method" json:"method"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	AuthToken string            `yaml:"auth_token,omitempty" json:"auth_token,omitempty"`
	TLSConfig TLSConfig         `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	CertFile          string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile           string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	CAFile            string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts   int           `yaml:"max_attempts" json:"max_attempts"`
	InitialDelay  time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay" json:"max_delay"`
	BackoffFactor float64       `yaml:"backoff_factor" json:"backoff_factor"`
	RetryableErrors []string    `yaml:"retryable_errors,omitempty" json:"retryable_errors,omitempty"`
}

// TriggerMetrics represents trigger performance metrics
type TriggerMetrics struct {
	TotalRequests    int64         `json:"total_requests"`
	SuccessfulSends  int64         `json:"successful_sends"`
	FailedSends      int64         `json:"failed_sends"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastSuccessTime  time.Time     `json:"last_success_time,omitempty"`
	LastFailureTime  time.Time     `json:"last_failure_time,omitempty"`
	ConsecutiveFails int64         `json:"consecutive_fails"`
	Uptime          time.Duration `json:"uptime"`
}

// EventTransformer transforms events to different payload formats
type EventTransformer interface {
	// TransformToGitHub transforms event to GitHub webhook format
	TransformToGitHub(event types.Event) (GitHubPayload, error)
	
	// TransformToTekton transforms event to Tekton EventListener format
	TransformToTekton(event types.Event) (TektonPayload, error)
	
	// TransformToGeneric transforms event to generic webhook format
	TransformToGeneric(event types.Event) (GenericPayload, error)
}

// GitHubPayload represents GitHub webhook-style payload
type GitHubPayload struct {
	Repository GitHubRepository `json:"repository"`
	After      string           `json:"after"`
	ShortSHA   string           `json:"short_sha"`
	Ref        string           `json:"ref"`
	Before     string           `json:"before,omitempty"`
	Pusher     GitHubUser       `json:"pusher,omitempty"`
	Commits    []GitHubCommit   `json:"commits,omitempty"`
	HeadCommit *GitHubCommit    `json:"head_commit,omitempty"`
}

// TektonPayload represents Tekton EventListener payload
type TektonPayload struct {
	GitHubPayload
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Source   string                 `json:"source"`
	EventID  string                 `json:"event_id"`
}

// GenericPayload represents generic webhook payload
type GenericPayload struct {
	Event      types.Event            `json:"event"`
	Repository map[string]interface{} `json:"repository"`
	Metadata   map[string]interface{} `json:"metadata"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
}

// GitHubRepository represents repository information in GitHub format
type GitHubRepository struct {
	ID       int64  `json:"id,omitempty"`
	Name     string `json:"name"`
	FullName string `json:"full_name,omitempty"`
	CloneURL string `json:"clone_url"`
	HTMLURL  string `json:"html_url,omitempty"`
	Private  bool   `json:"private,omitempty"`
}

// GitHubUser represents user information in GitHub format
type GitHubUser struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
}

// GitHubCommit represents commit information in GitHub format
type GitHubCommit struct {
	ID        string      `json:"id"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	URL       string      `json:"url,omitempty"`
	Author    GitHubUser  `json:"author,omitempty"`
	Added     []string    `json:"added,omitempty"`
	Removed   []string    `json:"removed,omitempty"`
	Modified  []string    `json:"modified,omitempty"`
}

// TriggerError represents trigger-specific errors
type TriggerError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

func (e *TriggerError) Error() string {
	return e.Message
}

// Common trigger error types
const (
	ErrorTypeConnection   = "connection_error"
	ErrorTypeTimeout     = "timeout_error"
	ErrorTypeAuth        = "auth_error"
	ErrorTypeValidation  = "validation_error"
	ErrorTypeServer      = "server_error"
	ErrorTypeClient      = "client_error"
	ErrorTypeUnknown     = "unknown_error"
)

// TriggerFactory creates trigger instances
type TriggerFactory struct{}

// NewTriggerFactory creates a new trigger factory
func NewTriggerFactory() *TriggerFactory {
	return &TriggerFactory{}
}

// Create creates a trigger instance based on configuration
func (f *TriggerFactory) Create(config TriggerConfig) (Trigger, error) {
	switch config.Type {
	case "tekton":
		return NewTektonTrigger(config)
	case "webhook":
		// TODO: Implement webhook trigger
		return nil, &TriggerError{
			Type:    ErrorTypeValidation,
			Message: "webhook trigger not implemented yet",
		}
	default:
		return nil, &TriggerError{
			Type:    ErrorTypeValidation,
			Message: "unsupported trigger type: " + config.Type,
		}
	}
}

// DefaultTriggerConfig returns default trigger configuration
func DefaultTriggerConfig() TriggerConfig {
	return TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Retry: RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		Timeout: 30 * time.Second,
	}
}
