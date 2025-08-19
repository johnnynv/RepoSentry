package types

import (
	"time"
)

// Repository represents a Git repository configuration
type Repository struct {
	Name             string        `yaml:"name" json:"name"`
	URL              string        `yaml:"url" json:"url"`
	Provider         string        `yaml:"provider" json:"provider"` // github, gitlab
	Token            string        `yaml:"token" json:"-"`           // Hidden in JSON output
	BranchRegex      string        `yaml:"branch_regex" json:"branch_regex"`
	Enabled          bool          `yaml:"enabled" json:"enabled"`
	PollingInterval  time.Duration `yaml:"polling_interval,omitempty" json:"polling_interval,omitempty"`
	APIBaseURL       string        `yaml:"api_base_url,omitempty" json:"api_base_url,omitempty"`
}

// Branch represents a Git branch
type Branch struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
	Protected bool   `json:"protected"`
}

// RepoState represents the stored state of a repository branch
type RepoState struct {
	ID          int64     `db:"id" json:"id"`
	Repository  string    `db:"repository" json:"repository"`
	Branch      string    `db:"branch" json:"branch"`
	CommitSHA   string    `db:"commit_sha" json:"commit_sha"`
	LastChecked time.Time `db:"last_checked" json:"last_checked"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// GitProvider defines the interface for Git providers
type GitProvider interface {
	GetBranches(repo Repository) ([]Branch, error)
	GetLatestCommit(repo Repository, branch string) (string, error)
	CheckPermissions(repo Repository) error
	GetRateLimit() RateLimit
}

// RateLimit represents API rate limit information
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}
