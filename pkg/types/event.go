package types

import (
	"time"
)

// EventType represents the type of Git event
type EventType string

const (
	EventTypeBranchUpdated  EventType = "branch_updated"
	EventTypeBranchCreated  EventType = "branch_created"
	EventTypeBranchDeleted  EventType = "branch_deleted"
	EventTypeTektonDetected EventType = "tekton_detected"
)

// Event represents a Git repository event
type Event struct {
	ID           string            `json:"id" db:"id"`
	Type         EventType         `json:"type" db:"type"`
	Repository   string            `json:"repository" db:"repository"`
	Branch       string            `json:"branch" db:"branch"`
	CommitSHA    string            `json:"commit_sha" db:"commit_sha"`
	PrevCommit   string            `json:"prev_commit,omitempty" db:"prev_commit"`
	Provider     string            `json:"provider" db:"provider"` // github, gitlab
	Timestamp    time.Time         `json:"timestamp" db:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty" db:"metadata"`
	Status       EventStatus       `json:"status" db:"status"`
	ErrorMessage string            `json:"error_message,omitempty" db:"error_message"` // Added for error tracking
	ProcessedAt  *time.Time        `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// EventStatus represents the processing status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusProcessed EventStatus = "processed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusRetrying  EventStatus = "retrying"
)

// TektonEvent represents the payload sent to Tekton EventListener
type TektonEvent struct {
	Source     string            `json:"source"`     // "reposentry"
	EventType  string            `json:"event_type"` // "push", "branch_created", etc.
	Repository TektonRepository  `json:"repository"`
	Branch     TektonBranch      `json:"branch"`
	Commit     TektonCommit      `json:"commit"`
	Provider   string            `json:"provider"` // "github", "gitlab"
	Timestamp  time.Time         `json:"timestamp"`
	EventID    string            `json:"event_id"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// TektonRepository represents repository information in Tekton event
type TektonRepository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	CloneURL string `json:"clone_url"`
	Owner    string `json:"owner,omitempty"`
}

// TektonBranch represents branch information in Tekton event
type TektonBranch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

// TektonCommit represents commit information in Tekton event
type TektonCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message,omitempty"`
	Author    string    `json:"author,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	URL       string    `json:"url,omitempty"`
}

// TektonDetectionEvent represents an event containing Tekton resource detection results
// This event is generated when .tekton/ directory resources are detected in a repository
type TektonDetectionEvent struct {
	// Standard event fields
	Source    string    `json:"source"`     // "reposentry"
	EventType string    `json:"event_type"` // "tekton_detected"
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`

	// Repository context
	Repository TektonRepository `json:"repository"`
	Branch     TektonBranch     `json:"branch"`
	Commit     TektonCommit     `json:"commit"`
	Provider   string           `json:"provider"`

	// Tekton detection results
	Detection TektonDetectionPayload `json:"detection"`

	// Event metadata
	Headers map[string]string `json:"headers,omitempty"`
}

// TektonDetectionPayload contains the actual detection results
type TektonDetectionPayload struct {
	// Detection metadata
	HasTektonDirectory bool      `json:"has_tekton_directory"`
	ScanPath           string    `json:"scan_path"`
	DetectedAt         time.Time `json:"detected_at"`

	// File summary
	TotalFiles int `json:"total_files"`
	ValidFiles int `json:"valid_files"`

	// Resource summary
	Resources      []TektonResourceSummary `json:"resources"`
	ResourceCounts map[string]int          `json:"resource_counts"` // "Pipeline": 2, "Task": 5, etc.

	// Action determination
	EstimatedAction string   `json:"estimated_action"` // "apply", "trigger", "validate", "skip"
	ActionReasons   []string `json:"action_reasons,omitempty"`

	// Processing results
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// TektonResourceSummary provides a summary of detected Tekton resources
type TektonResourceSummary struct {
	// Basic resource info
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`

	// Source info
	FilePath      string `json:"file_path"`
	ResourceIndex int    `json:"resource_index"`

	// Validation status
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`

	// Dependencies (referenced resources)
	Dependencies []string `json:"dependencies,omitempty"`
}
