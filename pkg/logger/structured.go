package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// Entry wraps logrus.Entry to provide consistent interface
type Entry struct {
	*logrus.Entry
}

// WithFields adds multiple fields to log entries
func (l *Logger) WithFields(fields Fields) *Entry {
	logrusFields := make(logrus.Fields)
	for k, v := range fields {
		logrusFields[k] = v
	}
	return &Entry{l.Logger.WithFields(logrusFields)}
}

// WithField adds a single field to log entries
func (l *Logger) WithField(key string, value interface{}) *Entry {
	return &Entry{l.Logger.WithField(key, value)}
}

// WithComponent adds component field to log entries
func (l *Logger) WithComponent(component string) *Entry {
	return l.WithField("component", component)
}

// WithRepository adds repository field to log entries
func (l *Logger) WithRepository(repo string) *Entry {
	return l.WithField("repository", repo)
}

// WithProvider adds provider field to log entries
func (l *Logger) WithProvider(provider string) *Entry {
	return l.WithField("provider", provider)
}

// WithBranch adds branch field to log entries
func (l *Logger) WithBranch(branch string) *Entry {
	return l.WithField("branch", branch)
}

// WithEventID adds event_id field to log entries
func (l *Logger) WithEventID(eventID string) *Entry {
	return l.WithField("event_id", eventID)
}

// WithError adds error field to log entries
func (l *Logger) WithError(err error) *Entry {
	return l.WithField("error", err.Error())
}

// WithRequestID adds request_id field to log entries
func (l *Logger) WithRequestID(requestID string) *Entry {
	return l.WithField("request_id", requestID)
}

// WithDuration adds duration field to log entries (for performance logging)
func (l *Logger) WithDuration(duration string) *Entry {
	return l.WithField("duration", duration)
}

// WithHTTPStatus adds http_status field to log entries
func (l *Logger) WithHTTPStatus(status int) *Entry {
	return l.WithField("http_status", status)
}

// WithURL adds url field to log entries
func (l *Logger) WithURL(url string) *Entry {
	return l.WithField("url", url)
}

// Entry methods for chaining additional fields
func (e *Entry) WithField(key string, value interface{}) *Entry {
	return &Entry{e.Entry.WithField(key, value)}
}

func (e *Entry) WithFields(fields Fields) *Entry {
	logrusFields := make(logrus.Fields)
	for k, v := range fields {
		logrusFields[k] = v
	}
	return &Entry{e.Entry.WithFields(logrusFields)}
}

func (e *Entry) WithComponent(component string) *Entry {
	return e.WithField("component", component)
}

func (e *Entry) WithOperation(operation string) *Entry {
	return e.WithField("operation", operation)
}

func (e *Entry) WithModule(module string) *Entry {
	return e.WithField("module", module)
}

func (e *Entry) WithRepository(repo string) *Entry {
	return e.WithField("repository", repo)
}

func (e *Entry) WithProvider(provider string) *Entry {
	return e.WithField("provider", provider)
}

func (e *Entry) WithBranch(branch string) *Entry {
	return e.WithField("branch", branch)
}

func (e *Entry) WithError(err error) *Entry {
	return e.WithField("error", err.Error())
}

// createFileWriter creates a file writer with rotation support
func createFileWriter(config Config) (io.Writer, error) {
	if !strings.HasPrefix(config.Output, "/") && config.Output != "stdout" && config.Output != "stderr" {
		// Relative path, make it absolute
		absPath, err := filepath.Abs(config.Output)
		if err != nil {
			return nil, err
		}
		config.Output = absPath
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(config.Output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &lumberjack.Logger{
		Filename:   config.Output,
		MaxSize:    config.File.MaxSize,
		MaxBackups: config.File.MaxBackups,
		MaxAge:     config.File.MaxAge,
		Compress:   config.File.Compress,
	}, nil
}

// getWriter returns the appropriate writer based on configuration
func getWriter(config Config) (io.Writer, error) {
	switch config.Output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// File path
		return createFileWriter(config)
	}
}
