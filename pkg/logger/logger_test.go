package logger

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_NewLogger_WithValidConfig(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "json",
		Output: "stderr",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Equal(t, "info", logger.GetLevel().String())
}

func TestLogger_NewLogger_WithInvalidLevel(t *testing.T) {
	config := Config{
		Level:  "invalid_level",
		Format: "json",
		Output: "stderr",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err) // logrus handles invalid levels gracefully
	assert.NotNil(t, logger)
}

func TestLogger_NewLogger_WithInvalidFormat(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "invalid_format",
		Output: "stderr",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err) // logrus handles invalid formats gracefully
	assert.NotNil(t, logger)
}

func TestLogger_NewLogger_WithFileOutput(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "text",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)

	// Test writing to file
	logger.Info("test message")

	// Check if file was created and contains the message
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}

func TestLogger_NewLogger_WithStdoutOutput(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestLogger_NewLogger_WithStderrOutput(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "text",
		Output: "stderr",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestLogger_GetDefaultLogger(t *testing.T) {
	logger := GetDefaultLogger()
	assert.NotNil(t, logger)

	// Default logger should have reasonable defaults
	assert.Equal(t, "info", logger.GetLevel().String())
}

func TestLogger_Logging_AllLevels(t *testing.T) {
	// Use a temporary file for testing
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "debug",
		Format: "text",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test all log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Check file content
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_WithFields(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "json",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test with fields
	logger.WithField("key1", "value1").WithField("key2", "value2").Info("test message")

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
}

func TestLogger_WithFields_Chaining(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "json",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test field chaining
	logger.WithField("key1", "value1").
		WithField("key2", "value2").
		WithField("key3", "value3").
		Info("chained message")

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "chained message", logEntry["message"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "value3", logEntry["key3"])
}

func TestLogger_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Level:  "info",
				Format: "json",
				Output: "stderr",
			},
			wantErr: false,
		},
		{
			name: "empty level",
			config: Config{
				Level:  "",
				Format: "json",
				Output: "stderr",
			},
			wantErr: false, // logrus handles empty level gracefully
		},
		{
			name: "empty format",
			config: Config{
				Level:  "info",
				Format: "",
				Output: "stderr",
			},
			wantErr: false, // logrus handles empty format gracefully
		},
		{
			name: "empty output",
			config: Config{
				Level:  "info",
				Format: "json",
				Output: "",
			},
			wantErr: false, // logrus handles empty output gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestLogger_ErrorHandling(t *testing.T) {
	// Test with invalid file path
	config := Config{
		Level:  "info",
		Format: "text",
		Output: "/invalid/path/to/file.log",
	}

	logger, err := NewLogger(config)
	// This might succeed or fail depending on the system, but shouldn't panic
	if err != nil {
		t.Logf("Expected error with invalid file path: %v", err)
	} else {
		assert.NotNil(t, logger)
	}
}

func TestLogger_Concurrency(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "text",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.WithField("goroutine", id).Info("concurrent message")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)
	assert.Contains(t, output, "concurrent message")
}

func TestLogger_FieldTypes(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "json",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test different field types
	logger.WithField("string", "value").
		WithField("int", 42).
		WithField("float", 3.14).
		WithField("bool", true).
		WithField("time", time.Now()).
		Info("field types test")

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "value", logEntry["string"])
	assert.Equal(t, float64(42), logEntry["int"])
	assert.Equal(t, 3.14, logEntry["float"])
	assert.Equal(t, true, logEntry["bool"])
	assert.NotNil(t, logEntry["time"])
}

func TestLogger_LogRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := tempDir + "/test.log"

	config := Config{
		Level:  "info",
		Format: "text",
		Output: logFile,
		File: FileConfig{
			MaxSize:    1, // 1 MB
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)

	// Write some logs
	for i := 0; i < 1000; i++ {
		logger.Info("test message for rotation testing")
	}

	// Check if log file exists
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestLogger_LogLevels_Threshold(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "warn", // Set threshold to warn
		Format: "text",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Log at different levels
	logger.Debug("debug message") // Should not appear
	logger.Info("info message")   // Should not appear
	logger.Warn("warn message")   // Should appear
	logger.Error("error message") // Should appear

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_JSONFormat_Structured(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "json",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Log with structured data
	logger.WithField("user_id", 12345).
		WithField("action", "login").
		WithField("ip", "192.168.1.1").
		Info("user authentication")

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "user authentication", logEntry["message"])
	assert.Equal(t, float64(12345), logEntry["user_id"])
	assert.Equal(t, "login", logEntry["action"])
	assert.Equal(t, "192.168.1.1", logEntry["ip"])
	assert.NotNil(t, logEntry["timestamp"])
	assert.NotNil(t, logEntry["level"])
}

func TestLogger_TextFormat_Readable(t *testing.T) {
	tempFile := t.TempDir() + "/test.log"
	config := Config{
		Level:  "info",
		Format: "text",
		Output: tempFile,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Log a simple message
	logger.Info("simple text message")

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	output := string(content)

	// Text format should be human-readable
	assert.Contains(t, output, "simple text message")
	assert.Contains(t, output, "level=info")
	assert.Contains(t, output, "time=")
}
