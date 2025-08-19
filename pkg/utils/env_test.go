package utils

import (
	"os"
	"testing"
)

func TestEnvExpander_ExpandString(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_TOKEN", "secret123")
	os.Setenv("ALLOWED_VAR", "allowed_value")
	defer func() {
		os.Unsetenv("TEST_TOKEN")
		os.Unsetenv("ALLOWED_VAR")
	}()

	allowedVars := []string{"TEST_TOKEN", "ALLOWED_VAR", "*_TOKEN"}
	expander := NewEnvExpander(allowedVars)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple variable expansion",
			input:    "${TEST_TOKEN}",
			expected: "secret123",
		},
		{
			name:     "Variable in string",
			input:    "token=${TEST_TOKEN}",
			expected: "token=secret123",
		},
		{
			name:     "Multiple variables",
			input:    "${TEST_TOKEN}-${ALLOWED_VAR}",
			expected: "secret123-allowed_value",
		},
		{
			name:     "Non-existent variable",
			input:    "${NONEXISTENT}",
			expected: "${NONEXISTENT}", // Should remain unchanged
		},
		{
			name:     "No variables",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expander.ExpandString(tt.input)
			if err != nil {
				t.Errorf("ExpandString() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEnvExpander_isVarAllowed(t *testing.T) {
	allowedVars := []string{"GITHUB_TOKEN", "GITLAB_TOKEN", "*_TOKEN", "CUSTOM_*"}
	expander := NewEnvExpander(allowedVars)

	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		{
			name:     "Exact match",
			varName:  "GITHUB_TOKEN",
			expected: true,
		},
		{
			name:     "Wildcard suffix match",
			varName:  "MY_TOKEN",
			expected: true,
		},
		{
			name:     "Wildcard prefix match",
			varName:  "CUSTOM_VAR",
			expected: true,
		},
		{
			name:     "Not allowed",
			varName:  "SECRET_KEY",
			expected: false,
		},
		{
			name:     "Case sensitive",
			varName:  "github_token",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.isVarAllowed(tt.varName)
			if result != tt.expected {
				t.Errorf("isVarAllowed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateRequiredEnvVars(t *testing.T) {
	// Set some test environment variables
	os.Setenv("EXISTING_VAR", "value")
	defer os.Unsetenv("EXISTING_VAR")

	required := []string{"EXISTING_VAR", "MISSING_VAR", "ANOTHER_MISSING"}
	missing := ValidateRequiredEnvVars(required)

	if len(missing) != 2 {
		t.Errorf("Expected 2 missing variables, got %d", len(missing))
	}

	expectedMissing := map[string]bool{
		"MISSING_VAR":     true,
		"ANOTHER_MISSING": true,
	}

	for _, varName := range missing {
		if !expectedMissing[varName] {
			t.Errorf("Unexpected missing variable: %s", varName)
		}
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "Existing variable",
			key:          "TEST_VAR",
			defaultValue: "default",
			expected:     "test_value",
		},
		{
			name:         "Non-existing variable",
			key:          "NONEXISTENT",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "Empty default",
			key:          "NONEXISTENT",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnvWithDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
