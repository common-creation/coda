package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of log messages
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Fields represents structured data for logging
type Fields map[string]interface{}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Level      LogLevel  `json:"level"`
	Message    string    `json:"message"`
	Fields     Fields    `json:"fields,omitempty"`
	Caller     string    `json:"caller,omitempty"`
	StackTrace string    `json:"stack_trace,omitempty"`
}

// LogOutput defines the interface for log outputs
type LogOutput interface {
	Write(entry *LogEntry) error
	Close() error
}

// SamplingConfig controls log sampling behavior
type SamplingConfig struct {
	Enabled     bool
	Rate        float64 // 0.0 to 1.0
	BurstLimit  int
	BurstWindow time.Duration
}

// Logger provides structured logging functionality
type Logger struct {
	level      LogLevel
	outputs    []LogOutput
	fields     Fields
	sampling   SamplingConfig
	sanitizer  Sanitizer
	mu         sync.RWMutex
	skipCaller int
}

// Sanitizer interface for cleaning sensitive data
type Sanitizer interface {
	Sanitize(Fields) Fields
}

// DefaultSanitizer provides basic sanitization
type DefaultSanitizer struct {
	sensitiveKeys []string
}

// NewDefaultSanitizer creates a sanitizer with common sensitive keys
func NewDefaultSanitizer() *DefaultSanitizer {
	return &DefaultSanitizer{
		sensitiveKeys: []string{
			"api_key", "apikey", "token", "password", "secret",
			"authorization", "auth", "bearer", "key",
		},
	}
}

// Sanitize removes or masks sensitive information
func (s *DefaultSanitizer) Sanitize(fields Fields) Fields {
	if fields == nil {
		return nil
	}

	sanitized := make(Fields)
	for k, v := range fields {
		lowerKey := strings.ToLower(k)
		isSensitive := false

		for _, sensitive := range s.sensitiveKeys {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			switch vt := v.(type) {
			case string:
				if len(vt) > 4 {
					sanitized[k] = vt[:4] + "***"
				} else {
					sanitized[k] = "***"
				}
			default:
				sanitized[k] = "***"
			}
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:      level,
		outputs:    []LogOutput{NewConsoleOutput(true)}, // Default to console
		fields:     make(Fields),
		sanitizer:  NewDefaultSanitizer(),
		sampling:   SamplingConfig{Enabled: false},
		skipCaller: 1,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// AddOutput adds a log output
func (l *Logger) AddOutput(output LogOutput) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputs = append(l.outputs, output)
}

// SetSanitizer sets the field sanitizer
func (l *Logger) SetSanitizer(sanitizer Sanitizer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sanitizer = sanitizer
}

// With returns a new logger with additional fields
func (l *Logger) With(fields Fields) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:      l.level,
		outputs:    l.outputs,
		fields:     newFields,
		sampling:   l.sampling,
		sanitizer:  l.sanitizer,
		skipCaller: l.skipCaller + 1,
	}
}

// WithField returns a new logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.With(Fields{key: value})
}

// log writes a log entry at the specified level
func (l *Logger) log(level LogLevel, message string, fields Fields) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	// Merge fields
	allFields := make(Fields)
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}

	// Sanitize fields
	if l.sanitizer != nil {
		allFields = l.sanitizer.Sanitize(allFields)
	}

	// Get caller info
	caller := ""
	if level >= LevelWarn {
		_, file, line, ok := runtime.Caller(l.skipCaller + 1)
		if ok {
			caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	// Get stack trace for errors
	stackTrace := ""
	if level >= LevelError {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		stackTrace = string(buf[:n])
	}

	entry := &LogEntry{
		Timestamp:  time.Now().UTC(),
		Level:      level,
		Message:    message,
		Fields:     allFields,
		Caller:     caller,
		StackTrace: stackTrace,
	}

	// Write to all outputs
	for _, output := range l.outputs {
		if err := output.Write(entry); err != nil {
			// Fallback to stderr if output fails
			fmt.Fprintf(os.Stderr, "Logger output error: %v\n", err)
		}
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LevelDebug, message, nil)
}

// DebugWith logs a debug message with fields
func (l *Logger) DebugWith(message string, fields Fields) {
	l.log(LevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LevelInfo, message, nil)
}

// InfoWith logs an info message with fields
func (l *Logger) InfoWith(message string, fields Fields) {
	l.log(LevelInfo, message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LevelWarn, message, nil)
}

// WarnWith logs a warning message with fields
func (l *Logger) WarnWith(message string, fields Fields) {
	l.log(LevelWarn, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LevelError, message, nil)
}

// ErrorWith logs an error message with fields
func (l *Logger) ErrorWith(message string, fields Fields) {
	l.log(LevelError, message, fields)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(LevelFatal, message, nil)
	os.Exit(1)
}

// FatalWith logs a fatal message with fields and exits
func (l *Logger) FatalWith(message string, fields Fields) {
	l.log(LevelFatal, message, fields)
	os.Exit(1)
}

// Close closes all outputs
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []string
	for _, output := range l.outputs {
		if err := output.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing outputs: %s", strings.Join(errs, ", "))
	}
	return nil
}

// Context key for logger
type contextKey string

const loggerContextKey contextKey = "logger"

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext retrieves a logger from the context
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerContextKey).(*Logger); ok {
		return logger
	}
	return GetDefault() // Return default logger if none in context
}

// Global default logger
var defaultLogger *Logger
var defaultLoggerOnce sync.Once

// GetDefault returns the default global logger
func GetDefault() *Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLogger(LevelInfo)
	})
	return defaultLogger
}

// SetDefault sets the default global logger
func SetDefault(logger *Logger) {
	defaultLoggerOnce.Do(func() {})
	defaultLogger = logger
}

// Convenience functions using the default logger
func Debug(message string) {
	GetDefault().Debug(message)
}

func DebugWith(message string, fields Fields) {
	GetDefault().DebugWith(message, fields)
}

func Info(message string) {
	GetDefault().Info(message)
}

func InfoWith(message string, fields Fields) {
	GetDefault().InfoWith(message, fields)
}

func Warn(message string) {
	GetDefault().Warn(message)
}

func WarnWith(message string, fields Fields) {
	GetDefault().WarnWith(message, fields)
}

func Error(message string) {
	GetDefault().Error(message)
}

func ErrorWith(message string, fields Fields) {
	GetDefault().ErrorWith(message, fields)
}

func Fatal(message string) {
	GetDefault().Fatal(message)
}

func FatalWith(message string, fields Fields) {
	GetDefault().FatalWith(message, fields)
}

func With(fields Fields) *Logger {
	return GetDefault().With(fields)
}

func WithField(key string, value interface{}) *Logger {
	return GetDefault().WithField(key, value)
}
