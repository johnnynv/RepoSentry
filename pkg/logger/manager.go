package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Manager is the enterprise-grade logger manager
type Manager struct {
	rootLogger     *Logger
	config         Config
	contexts       map[string]*Entry
	hooks          []logrus.Hook
	rotatingWriter io.WriteCloser // 支持轮转的写入器
	mu             sync.RWMutex
}

// NewManager creates a new logger manager with enterprise features
func NewManager(config Config) (*Manager, error) {
	rootLogger, err := NewLogger(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create root logger: %w", err)
	}

	manager := &Manager{
		rootLogger: rootLogger,
		config:     config,
		contexts:   make(map[string]*Entry),
	}

	// 设置日志轮转（如果输出到文件）
	if err := manager.setupLogRotation(); err != nil {
		return nil, fmt.Errorf("failed to setup log rotation: %w", err)
	}

	// Add default hooks for enterprise features
	manager.addDefaultHooks()

	return manager, nil
}

// setupLogRotation 设置日志轮转
func (m *Manager) setupLogRotation() error {
	// 仅当输出到文件时才设置轮转
	if m.config.Output == "stdout" || m.config.Output == "stderr" {
		return nil
	}

	// 创建lumberjack轮转写入器
	rotatingWriter := &lumberjack.Logger{
		Filename:   m.config.Output,
		MaxSize:    m.config.File.MaxSize,    // MB
		MaxAge:     m.config.File.MaxAge,     // days
		MaxBackups: m.config.File.MaxBackups, // number of backups
		Compress:   m.config.File.Compress,   // compress rotated files
	}

	m.rotatingWriter = rotatingWriter

	// 将轮转写入器设置给root logger
	m.rootLogger.SetOutput(rotatingWriter)

	return nil
}

// addDefaultHooks adds enterprise-grade hooks
func (m *Manager) addDefaultHooks() {
	// Add performance monitoring hook
	m.rootLogger.AddHook(&PerformanceHook{})

	// Add error tracking hook
	m.rootLogger.AddHook(&ErrorTrackingHook{})
}

// GetRootLogger returns the root logger
func (m *Manager) GetRootLogger() *Logger {
	return m.rootLogger
}

// ForComponent creates a logger for a specific component
func (m *Manager) ForComponent(component string) *Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("component:%s", component)
	if entry, exists := m.contexts[key]; exists {
		return entry
	}

	entry := m.rootLogger.WithField("component", component)
	m.contexts[key] = entry
	return entry
}

// ForModule creates a logger for a component module
func (m *Manager) ForModule(component, module string) *Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("component:%s:module:%s", component, module)
	if entry, exists := m.contexts[key]; exists {
		return entry
	}

	entry := m.rootLogger.WithFields(Fields{
		"component": component,
		"module":    module,
	})
	m.contexts[key] = entry
	return entry
}

// ForOperation creates a logger for a specific operation
func (m *Manager) ForOperation(component, module, operation string) *Entry {
	return m.rootLogger.WithFields(Fields{
		"component": component,
		"module":    module,
		"operation": operation,
	})
}

// WithContext creates a logger with full context
func (m *Manager) WithContext(logCtx LogContext) *Entry {
	return m.rootLogger.WithFields(logCtx.ToFields())
}

// WithGoContext creates a logger from Go context
func (m *Manager) WithGoContext(ctx context.Context) *Entry {
	logCtx := FromContext(ctx)
	return m.WithContext(logCtx)
}

// PerformanceHook tracks performance metrics
type PerformanceHook struct{}

func (h *PerformanceHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
	}
}

func (h *PerformanceHook) Fire(entry *logrus.Entry) error {
	// Track performance metrics here
	if duration, exists := entry.Data["duration"]; exists {
		if d, ok := duration.(time.Duration); ok {
			// Could send to metrics system
			if d > 5*time.Second {
				entry.Data["performance_alert"] = "slow_operation"
			}
		}
	}
	return nil
}

// ErrorTrackingHook tracks errors
type ErrorTrackingHook struct{}

func (h *ErrorTrackingHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

func (h *ErrorTrackingHook) Fire(entry *logrus.Entry) error {
	// Track errors here - could send to error tracking service
	entry.Data["error_tracked"] = true
	entry.Data["error_timestamp"] = time.Now()
	return nil
}

// BusinessOperation represents a business operation for logging
type BusinessOperation struct {
	Component  string
	Module     string
	Operation  string
	Repository string
	EventID    string
	StartTime  time.Time
	logger     *Entry
	ctx        context.Context
}

// StartOperation starts a business operation with logging
func (m *Manager) StartOperation(ctx context.Context, component, module, operation string) *BusinessOperation {
	startTime := time.Now()

	logCtx := LogContext{
		Component: component,
		Module:    module,
		Operation: operation,
		StartTime: startTime,
	}

	// Merge with existing context
	existingCtx := FromContext(ctx)
	logCtx = existingCtx.Merge(logCtx)

	logger := m.WithContext(logCtx)

	// Create enhanced context
	enhancedCtx := WithContext(ctx, logCtx)

	op := &BusinessOperation{
		Component: component,
		Module:    module,
		Operation: operation,
		StartTime: startTime,
		logger:    logger,
		ctx:       enhancedCtx,
	}

	op.logger.Info("Operation started")
	return op
}

// WithRepository adds repository context
func (bo *BusinessOperation) WithRepository(repository, provider string) *BusinessOperation {
	bo.Repository = repository
	bo.logger = bo.logger.WithFields(Fields{
		"repository": repository,
		"provider":   provider,
	})
	return bo
}

// WithEvent adds event context
func (bo *BusinessOperation) WithEvent(eventID string) *BusinessOperation {
	bo.EventID = eventID
	bo.logger = bo.logger.WithField("event_id", eventID)
	return bo
}

// Info logs info message
func (bo *BusinessOperation) Info(message string, fields ...Fields) {
	if len(fields) > 0 {
		bo.logger.WithFields(fields[0]).Info(message)
	} else {
		bo.logger.Info(message)
	}
}

// Error logs error message
func (bo *BusinessOperation) Error(message string, err error, fields ...Fields) {
	logFields := Fields{"error": err.Error()}
	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}
	bo.logger.WithFields(logFields).Error(message)
}

// Success logs successful completion
func (bo *BusinessOperation) Success(message string, fields ...Fields) {
	duration := time.Since(bo.StartTime)
	logFields := Fields{
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
		"success":     true,
	}
	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}
	bo.logger.WithFields(logFields).Info(message)
}

// Close 关闭Manager及其资源
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rotatingWriter != nil {
		return m.rotatingWriter.Close()
	}
	return nil
}

// RotateLog 手动触发日志轮转
func (m *Manager) RotateLog() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if lj, ok := m.rotatingWriter.(*lumberjack.Logger); ok {
		return lj.Rotate()
	}
	return fmt.Errorf("log rotation not available")
}

// GetLogStats 获取日志统计信息
func (m *Manager) GetLogStats() (*LogStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if lj, ok := m.rotatingWriter.(*lumberjack.Logger); ok {
		stats := &LogStats{
			CurrentFile: lj.Filename,
			MaxSize:     lj.MaxSize,
			MaxAge:      lj.MaxAge,
			MaxBackups:  lj.MaxBackups,
			Compress:    lj.Compress,
		}

		// 获取当前文件信息
		if info, err := os.Stat(lj.Filename); err == nil {
			stats.CurrentSize = info.Size()
			stats.LastModified = info.ModTime()
		}

		return stats, nil
	}

	return nil, fmt.Errorf("log rotation not available")
}

// LogStats 日志统计信息
type LogStats struct {
	CurrentFile  string    `json:"current_file"`
	CurrentSize  int64     `json:"current_size"`
	LastModified time.Time `json:"last_modified"`
	MaxSize      int       `json:"max_size"`
	MaxAge       int       `json:"max_age"`
	MaxBackups   int       `json:"max_backups"`
	Compress     bool      `json:"compress"`
}

// FormatSize 格式化文件大小显示
func (ls *LogStats) FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// String 返回统计信息的字符串表示
func (ls *LogStats) String() string {
	return fmt.Sprintf(
		"File: %s, Size: %s, MaxSize: %dMB, MaxAge: %dd, MaxBackups: %d, Compress: %t",
		ls.CurrentFile,
		ls.FormatSize(ls.CurrentSize),
		ls.MaxSize,
		ls.MaxAge,
		ls.MaxBackups,
		ls.Compress,
	)
}

// Fail logs operation failure
func (bo *BusinessOperation) Fail(message string, err error, fields ...Fields) {
	duration := time.Since(bo.StartTime)
	logFields := Fields{
		"duration":    duration,
		"duration_ms": duration.Milliseconds(),
		"success":     false,
		"error":       err.Error(),
	}
	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}
	bo.logger.WithFields(logFields).Error(message)
}

// GetContext returns the enhanced context
func (bo *BusinessOperation) GetContext() context.Context {
	return bo.ctx
}

// GetLogger returns the operation logger
func (bo *BusinessOperation) GetLogger() *Entry {
	return bo.logger
}
