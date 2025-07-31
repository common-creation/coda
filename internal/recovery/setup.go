package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/common-creation/coda/internal/errors"
)

// SetupConfig holds configuration for the recovery system setup.
type SetupConfig struct {
	// Logger for the recovery system
	Logger *log.Logger

	// Enable health monitoring
	EnableMonitoring bool

	// Monitoring intervals
	QuickCheckInterval time.Duration
	FullCheckInterval  time.Duration

	// Recovery settings
	MaxRetries    int
	BackoffPolicy BackoffPolicy

	// Crash log settings
	CrashLogDir  string
	MaxCrashLogs int

	// Integration with error handler
	IntegrateWithErrorHandler bool
}

// DefaultSetupConfig returns a default setup configuration.
func DefaultSetupConfig() SetupConfig {
	return SetupConfig{
		Logger:                    log.Default(),
		EnableMonitoring:          true,
		QuickCheckInterval:        30 * time.Second,
		FullCheckInterval:         5 * time.Minute,
		MaxRetries:                3,
		BackoffPolicy:             DefaultBackoffPolicy(),
		MaxCrashLogs:              10,
		IntegrateWithErrorHandler: true,
	}
}

// SetupRecoverySystem sets up the complete recovery system with the given configuration.
func SetupRecoverySystem(ctx context.Context, config SetupConfig) error {
	// Initialize the recovery system
	InitializeRecoverySystem(config.Logger)

	integration := GetRecoverySystem()
	if !integration.initialized {
		return fmt.Errorf("failed to initialize recovery system")
	}

	// Configure recovery manager
	if config.MaxRetries > 0 {
		integration.recoveryManager.maxRetries = config.MaxRetries
	}
	if config.BackoffPolicy.InitialDelay > 0 {
		integration.recoveryManager.backoff = config.BackoffPolicy
	}

	// Configure system diagnostics
	if config.QuickCheckInterval > 0 {
		integration.systemDiagnostics.quickCheckInterval = config.QuickCheckInterval
	}
	if config.FullCheckInterval > 0 {
		integration.systemDiagnostics.fullCheckInterval = config.FullCheckInterval
	}

	// Start health monitoring if enabled
	if config.EnableMonitoring {
		integration.systemDiagnostics.StartMonitoring(ctx)
	}

	// Integrate with error handler if requested
	if config.IntegrateWithErrorHandler {
		if err := setupErrorHandlerIntegration(config.Logger); err != nil {
			config.Logger.Warn("Failed to integrate with error handler", "error", err.Error())
		}
	}

	// Register custom recovery strategies based on configuration
	registerConfigBasedStrategies(config)

	config.Logger.Info("Recovery system setup completed",
		"monitoring_enabled", config.EnableMonitoring,
		"error_handler_integration", config.IntegrateWithErrorHandler,
		"max_retries", config.MaxRetries)

	return nil
}

// setupErrorHandlerIntegration integrates the recovery system with the global error handler.
func setupErrorHandlerIntegration(logger *log.Logger) error {
	// Get the global error handler
	errorHandler := errors.Get()
	if errorHandler == nil {
		return fmt.Errorf("global error handler not available")
	}

	// Create and add the recovery error reporter
	recoveryReporter := NewRecoveryErrorReporter(logger)
	errorHandler.AddReporter(recoveryReporter)

	logger.Debug("Integrated recovery system with error handler")
	return nil
}

// registerConfigBasedStrategies registers recovery strategies based on configuration.
func registerConfigBasedStrategies(config SetupConfig) {
	// integration := GetRecoverySystem() // TODO: Use this when custom strategies are implemented

	// Register additional strategies based on config
	// This is where you would add custom strategies based on configuration

	// Example: Register a custom cache strategy if cache directory is configured
	if config.CrashLogDir != "" {
		// Custom strategy could be registered here
		config.Logger.Debug("Custom strategies could be registered based on config")
	}
}

// QuickSetup provides a simple way to set up the recovery system with sensible defaults.
func QuickSetup(ctx context.Context) error {
	config := DefaultSetupConfig()
	return SetupRecoverySystem(ctx, config)
}

// SetupWithLogger sets up the recovery system with a custom logger.
func SetupWithLogger(ctx context.Context, logger *log.Logger) error {
	config := DefaultSetupConfig()
	config.Logger = logger
	return SetupRecoverySystem(ctx, config)
}

// SetupMinimal sets up the recovery system without health monitoring (useful for testing).
func SetupMinimal(logger *log.Logger) error {
	config := DefaultSetupConfig()
	config.Logger = logger
	config.EnableMonitoring = false
	config.IntegrateWithErrorHandler = false

	return SetupRecoverySystem(context.Background(), config)
}

// Shutdown gracefully shuts down the recovery system.
func Shutdown() {
	integration := GetRecoverySystem()
	if integration != nil && integration.initialized {
		// Stop health monitoring
		integration.systemDiagnostics.StopMonitoring()

		// Log final recovery statistics
		stats := integration.recoveryManager.GetRecoveryStats()
		integration.logger.Info("Recovery system shutdown", "final_stats", stats)
	}
}

// Example usage functions for common scenarios

// HandlePanicWithRecovery is a helper function for panic recovery in goroutines.
func HandlePanicWithRecovery(logger log.Logger) {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic occurred: %v", r)

		// Create context with timeout for recovery
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt recovery
		if recoveryErr := RecoverFromError(ctx, err); recoveryErr != nil {
			logger.Error("Failed to recover from panic",
				"panic", r,
				"recovery_error", recoveryErr.Error())
		} else {
			logger.Info("Successfully recovered from panic", "panic", r)
		}
	}
}

// SafeGoRoutine runs a function in a goroutine with panic recovery.
func SafeGoRoutine(fn func(), logger log.Logger) {
	go func() {
		defer HandlePanicWithRecovery(logger)
		fn()
	}()
}

// WithRecovery wraps a function with error recovery.
func WithRecovery(ctx context.Context, fn func() error, logger log.Logger) error {
	err := fn()
	if err != nil {
		// Attempt recovery
		if recoveryErr := RecoverFromError(ctx, err); recoveryErr != nil {
			logger.Warn("Recovery failed",
				"original_error", err.Error(),
				"recovery_error", recoveryErr.Error())
			return err // Return original error if recovery fails
		}

		logger.Info("Successfully recovered from error", "error", err.Error())
		return nil // Return nil if recovery succeeds
	}

	return nil
}

// GetSystemHealth returns the current system health status.
func GetSystemHealth(ctx context.Context) (*DiagnosticReport, error) {
	return RunSystemDiagnostics(ctx)
}

// IsRecoverySystemHealthy performs a quick health check of the recovery system itself.
func IsRecoverySystemHealthy() bool {
	integration := GetRecoverySystem()
	if integration == nil || !integration.initialized {
		return false
	}

	// Check if recovery manager is operational
	if integration.recoveryManager == nil {
		return false
	}

	// Check if system diagnostics is operational
	if integration.systemDiagnostics == nil {
		return false
	}

	return true
}
