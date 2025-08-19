package api

import (
	"runtime"
)

// Version information (should be set by build process)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// APIVersion represents API version information
type APIVersion struct {
	API     string `json:"api_version"`
	App     string `json:"app_version"`
	Build   string `json:"build_time"`
	Commit  string `json:"git_commit"`
	Runtime string `json:"go_version"`
}

// GetVersion returns version information
func GetVersion() APIVersion {
	return APIVersion{
		API:     "v1",
		App:     Version,
		Build:   BuildTime,
		Commit:  GitCommit,
		Runtime: runtime.Version(),
	}
}

// handleVersion function is defined in server.go with Swagger annotations
