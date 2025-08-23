package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// EnvExpander handles environment variable expansion in configuration values
type EnvExpander struct {
	allowedVars []string
	pattern     *regexp.Regexp
}

// NewEnvExpander creates a new environment variable expander
func NewEnvExpander(allowedVars []string) *EnvExpander {
	// Pattern to match ${VAR_NAME} syntax
	pattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	return &EnvExpander{
		allowedVars: allowedVars,
		pattern:     pattern,
	}
}

// ExpandString expands environment variables in a string
func (e *EnvExpander) ExpandString(s string) (string, error) {
	return e.pattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := e.pattern.FindStringSubmatch(match)[1]

		// Check if variable is allowed
		if !e.isVarAllowed(varName) {
			// Return original string if not allowed (security)
			return match
		}

		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			// Return original string if environment variable is not set
			return match
		}

		return value
	}), nil
}

// ExpandMap expands environment variables in all string values of a map
func (e *EnvExpander) ExpandMap(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range m {
		expandedValue, err := e.expandValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to expand value for key %s: %w", key, err)
		}
		result[key] = expandedValue
	}

	return result, nil
}

// expandValue recursively expands environment variables in various value types
func (e *EnvExpander) expandValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return e.ExpandString(v)
	case map[string]interface{}:
		return e.ExpandMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			expandedItem, err := e.expandValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = expandedItem
		}
		return result, nil
	default:
		// Return unchanged for non-string types
		return value, nil
	}
}

// isVarAllowed checks if an environment variable is allowed for expansion
func (e *EnvExpander) isVarAllowed(varName string) bool {
	for _, allowed := range e.allowedVars {
		if e.matchPattern(allowed, varName) {
			return true
		}
	}
	return false
}

// matchPattern checks if a variable name matches an allowed pattern
// Supports wildcards like "TOKEN_*" and "*_TOKEN"
func (e *EnvExpander) matchPattern(pattern, varName string) bool {
	if pattern == varName {
		return true
	}

	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(varName, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(varName, suffix)
	}

	return false
}

// ValidateRequiredEnvVars checks if required environment variables are set
func ValidateRequiredEnvVars(required []string) []string {
	var missing []string

	for _, varName := range required {
		if os.Getenv(varName) == "" {
			missing = append(missing, varName)
		}
	}

	return missing
}

// GetEnvWithDefault returns environment variable value or default if not set
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
