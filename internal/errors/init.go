// Package errors provides initialization utilities for the global error handling system.
package errors

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InitGlobalErrorHandler initializes the global error handler with the given configuration.
func InitGlobalErrorHandler(config Config) (*ErrorHandler, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid error handler configuration: %w", err)
	}

	// Initialize the global handler
	Init(config)
	handler := Get()

	// Add default reporters based on configuration
	if err := setupDefaultReporters(handler, config); err != nil {
		return nil, fmt.Errorf("failed to setup error reporters: %w", err)
	}

	return handler, nil
}

// validateConfig validates the error handler configuration.
func validateConfig(config Config) error {
	if config.SessionID == "" {
		config.SessionID = generateSessionID()
	}

	if config.Component == "" {
		config.Component = "coda"
	}

	if config.Version == "" {
		config.Version = "unknown"
	}

	if config.Environment == "" {
		config.Environment = "development"
	}

	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	return nil
}

// setupDefaultReporters sets up default error reporters based on configuration.
func setupDefaultReporters(handler *ErrorHandler, config Config) error {
	// Setup file reporter for persistent logging
	if config.LogFile != "" {
		logDir := filepath.Dir(config.LogFile)
		fileReporter := NewFileReporter(logDir, 10*1024*1024, 10) // 10MB, 10 files
		handler.AddReporter(fileReporter)
	}

	// Setup debug reporter for development environment
	if config.Environment == "development" {
		debugDir := filepath.Join(os.TempDir(), "coda", "debug")
		debugReporter := NewDebugReporter(debugDir, true, true)
		handler.AddReporter(debugReporter)
	}

	// Setup crash dump reporter for critical errors
	crashDir := filepath.Join(os.TempDir(), "coda", "crashes")
	crashReporter := NewCrashDumpReporter(crashDir, SystemError)
	handler.AddReporter(crashReporter)

	// Setup statistics reporter
	statsFile := filepath.Join(os.TempDir(), "coda", "error_stats.json")
	statsReporter := NewStatisticsReporter(statsFile)
	handler.AddReporter(statsReporter)

	return nil
}

// CreateDefaultConfig creates a default error handler configuration.
func CreateDefaultConfig(appVersion, environment string) Config {
	logDir := filepath.Join(os.TempDir(), "coda", "logs")
	os.MkdirAll(logDir, 0755)

	return Config{
		LogLevel:    "info",
		LogFile:     filepath.Join(logDir, "coda.log"),
		SessionID:   generateSessionID(),
		Component:   "coda",
		Version:     appVersion,
		Environment: environment,
		EnableStack: environment == "development",
	}
}

// CreateProductionConfig creates a production-ready error handler configuration.
func CreateProductionConfig(appVersion string) Config {
	return Config{
		LogLevel:    "warn",
		LogFile:     "/var/log/coda/coda.log",
		SessionID:   generateSessionID(),
		Component:   "coda",
		Version:     appVersion,
		Environment: "production",
		EnableStack: false,
	}
}

// CreateDevelopmentConfig creates a development-friendly error handler configuration.
func CreateDevelopmentConfig(appVersion string) Config {
	return CreateDefaultConfig(appVersion, "development")
}

// SetupErrorHandlerForTesting sets up a minimal error handler for testing purposes.
func SetupErrorHandlerForTesting() *ErrorHandler {
	config := Config{
		LogLevel:    "debug",
		SessionID:   "test-session",
		Component:   "coda-test",
		Version:     "test",
		Environment: "test",
		EnableStack: true,
	}

	handler := NewErrorHandler(config)
	return handler
}

// Global error handling helpers

// HandleError is a convenience function to handle errors through the global handler.
func HandleError(err error, userAction string, metadata ...map[string]interface{}) {
	if err == nil {
		return
	}

	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}

	HandleWithContext(err, userAction, meta)
}

// HandlePanic recovers from panics and processes them through the error handler.
func HandlePanic() {
	if r := recover(); r != nil {
		var err error
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}

		// Log the panic with stack trace
		HandleWithContext(err, "panic_recovery", map[string]interface{}{
			"panic_value": r,
			"recovered":   true,
		})
	}
}

// WithErrorHandling wraps a function with error handling.
func WithErrorHandling(fn func() error, userAction string) error {
	defer HandlePanic()

	if err := fn(); err != nil {
		HandleError(err, userAction)
		return err
	}

	return nil
}

// UserFriendlyError returns a user-friendly error message.
func UserFriendlyError(err error) string {
	if err == nil {
		return ""
	}

	handler := Get()
	return handler.UserMessage(err)
}

// IsRetryableError checks if an error should be retried.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	handler := Get()
	category := handler.ClassifyError(err)

	switch category {
	case NetworkError, AIServiceError:
		return true
	case UserError, ConfigError, SecurityError, SystemError:
		return false
	default:
		return false
	}
}

// Cleanup cleans up the global error handler resources.
func Cleanup() error {
	handler := Get()
	if handler != nil {
		return handler.Close()
	}
	return nil
}

// GetErrorStats returns error statistics from the global handler.
func GetErrorStats() map[string]interface{} {
	// This would return comprehensive error statistics
	// For now, return basic information
	stats := make(map[string]interface{})
	stats["handler_initialized"] = globalHandler != nil
	stats["timestamp"] = time.Now().Format(time.RFC3339)

	return stats
}

// ExportErrorReport exports a detailed error report for debugging.
func ExportErrorReport(outputPath string) error {
	// This would generate a comprehensive error report
	// Including logs, statistics, and diagnostic information
	return fmt.Errorf("error report export not yet implemented")
}

// GetLogLocation returns the current log file location
func GetLogLocation() string {
	if globalHandler != nil && globalHandler.logFile != nil {
		return globalHandler.logFile.Name()
	}
	return ""
}
