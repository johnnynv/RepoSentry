package logger

import (
	"context"
	"time"
)

// Context keys for logger context
type loggerContextKey string

const (
	ComponentKey   loggerContextKey = "component"
	OperationKey   loggerContextKey = "operation"
	RepositoryKey  loggerContextKey = "repository"
	EventIDKey     loggerContextKey = "event_id"
	RequestIDKey   loggerContextKey = "request_id"
	CorrelationKey loggerContextKey = "correlation_id"
)

// LogContext represents structured logging context
type LogContext struct {
	Component  string                 `json:"component,omitempty"`
	Module     string                 `json:"module,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	Provider   string                 `json:"provider,omitempty"`
	Branch     string                 `json:"branch,omitempty"`
	EventID    string                 `json:"event_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	StartTime  time.Time              `json:"start_time,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// ToFields converts LogContext to logger Fields
func (lc LogContext) ToFields() Fields {
	fields := Fields{}

	if lc.Component != "" {
		fields["component"] = lc.Component
	}
	if lc.Module != "" {
		fields["module"] = lc.Module
	}
	if lc.Operation != "" {
		fields["operation"] = lc.Operation
	}
	if lc.Repository != "" {
		fields["repository"] = lc.Repository
	}
	if lc.Provider != "" {
		fields["provider"] = lc.Provider
	}
	if lc.Branch != "" {
		fields["branch"] = lc.Branch
	}
	if lc.EventID != "" {
		fields["event_id"] = lc.EventID
	}
	if lc.RequestID != "" {
		fields["request_id"] = lc.RequestID
	}
	if lc.UserID != "" {
		fields["user_id"] = lc.UserID
	}
	if lc.SessionID != "" {
		fields["session_id"] = lc.SessionID
	}
	if lc.TraceID != "" {
		fields["trace_id"] = lc.TraceID
	}
	if lc.SpanID != "" {
		fields["span_id"] = lc.SpanID
	}
	if !lc.StartTime.IsZero() {
		fields["start_time"] = lc.StartTime
	}
	if lc.Duration > 0 {
		fields["duration"] = lc.Duration
		fields["duration_ms"] = lc.Duration.Milliseconds()
		fields["duration_ns"] = lc.Duration.Nanoseconds()
	}

	// Add custom fields
	for k, v := range lc.Custom {
		fields[k] = v
	}

	return fields
}

// WithContext adds logging context to Go context
func WithContext(ctx context.Context, logCtx LogContext) context.Context {
	for k, v := range logCtx.ToFields() {
		if v != nil && v != "" {
			ctx = context.WithValue(ctx, loggerContextKey(k), v)
		}
	}
	return ctx
}

// FromContext extracts logging context from Go context
func FromContext(ctx context.Context) LogContext {
	logCtx := LogContext{
		Custom: make(map[string]interface{}),
	}

	if v := ctx.Value(ComponentKey); v != nil {
		if s, ok := v.(string); ok {
			logCtx.Component = s
		}
	}
	if v := ctx.Value(OperationKey); v != nil {
		if s, ok := v.(string); ok {
			logCtx.Operation = s
		}
	}
	if v := ctx.Value(RepositoryKey); v != nil {
		if s, ok := v.(string); ok {
			logCtx.Repository = s
		}
	}
	if v := ctx.Value(EventIDKey); v != nil {
		if s, ok := v.(string); ok {
			logCtx.EventID = s
		}
	}
	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			logCtx.RequestID = s
		}
	}

	return logCtx
}

// Merge merges two LogContext objects
func (lc LogContext) Merge(other LogContext) LogContext {
	result := lc

	if other.Component != "" {
		result.Component = other.Component
	}
	if other.Module != "" {
		result.Module = other.Module
	}
	if other.Operation != "" {
		result.Operation = other.Operation
	}
	if other.Repository != "" {
		result.Repository = other.Repository
	}
	if other.Provider != "" {
		result.Provider = other.Provider
	}
	if other.Branch != "" {
		result.Branch = other.Branch
	}
	if other.EventID != "" {
		result.EventID = other.EventID
	}
	if other.RequestID != "" {
		result.RequestID = other.RequestID
	}
	if other.UserID != "" {
		result.UserID = other.UserID
	}
	if other.SessionID != "" {
		result.SessionID = other.SessionID
	}
	if other.TraceID != "" {
		result.TraceID = other.TraceID
	}
	if other.SpanID != "" {
		result.SpanID = other.SpanID
	}
	if !other.StartTime.IsZero() {
		result.StartTime = other.StartTime
	}
	if other.Duration > 0 {
		result.Duration = other.Duration
	}

	// Merge custom fields
	if result.Custom == nil {
		result.Custom = make(map[string]interface{})
	}
	for k, v := range other.Custom {
		result.Custom[k] = v
	}

	return result
}
