package api

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()

	if version.API != "v1" {
		t.Errorf("Expected API version v1, got %s", version.API)
	}

	if version.App == "" {
		t.Error("Expected App version to be set")
	}

	if version.Runtime == "" {
		t.Error("Expected Runtime version to be set")
	}

	// Build and Commit might be "unknown" in tests, but should be set
	if version.Build == "" {
		t.Error("Expected Build time to be set")
	}

	if version.Commit == "" {
		t.Error("Expected Git commit to be set")
	}
}

func TestVersionDefaults(t *testing.T) {
	// Test that default values are set correctly
	if Version == "" {
		t.Error("Expected Version to have a default value")
	}

	if BuildTime == "" {
		t.Error("Expected BuildTime to have a default value")
	}

	if GitCommit == "" {
		t.Error("Expected GitCommit to have a default value")
	}
}

func TestVersionConsistency(t *testing.T) {
	// Test that GetVersion returns consistent values
	version1 := GetVersion()
	version2 := GetVersion()

	if version1.API != version2.API {
		t.Error("GetVersion should return consistent API version")
	}

	if version1.App != version2.App {
		t.Error("GetVersion should return consistent App version")
	}

	if version1.Build != version2.Build {
		t.Error("GetVersion should return consistent Build time")
	}

	if version1.Commit != version2.Commit {
		t.Error("GetVersion should return consistent Git commit")
	}

	// Runtime version should be the same for the same Go binary
	if version1.Runtime != version2.Runtime {
		t.Error("GetVersion should return consistent Runtime version")
	}
}
