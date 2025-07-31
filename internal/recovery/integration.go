package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/common-creation/coda/internal/errors"
)

// Integration provides integration between the recovery system and error handler.
type Integration struct {
	recoveryManager   *RecoveryManager
	systemDiagnostics *SystemDiagnostics
	logger            *log.Logger
	mu                sync.RWMutex
	initialized       bool
}

var (
	globalIntegration *Integration
	integrationOnce   sync.Once
)

// InitializeRecoverySystem initializes the global recovery system.
func InitializeRecoverySystem(logger *log.Logger) {
	integrationOnce.Do(func() {
		// Create application state
		appState := &ApplicationState{
			Sessions:   make(map[string]SessionData),
			UIState:    UIState{ActiveView: "chat", WindowSize: WindowSize{Width: 800, Height: 600}},
			Config:     make(map[string]interface{}),
			LastUpdate: time.Now(),
		}

		// Create recovery manager
		recoveryMgr := NewRecoveryManager(appState, logger)

		// Create system diagnostics
		sysDiagnostics := NewSystemDiagnostics(recoveryMgr, logger)

		globalIntegration = &Integration{
			recoveryManager:   recoveryMgr,
			systemDiagnostics: sysDiagnostics,
			logger:            logger,
			initialized:       true,
		}

		logger.Info("Recovery system initialized")
	})
}

// GetRecoverySystem returns the global recovery system integration.
func GetRecoverySystem() *Integration {
	if globalIntegration == nil {
		// Initialize with default logger if not already initialized
		InitializeRecoverySystem(log.Default())
	}
	return globalIntegration
}

// RecoverFromError attempts to recover from an error using the recovery system.
func RecoverFromError(ctx context.Context, err error) error {
	integration := GetRecoverySystem()
	if !integration.initialized {
		return err // Return original error if not initialized
	}

	return integration.recoveryManager.Recover(ctx, err)
}

// StartHealthMonitoring starts the health monitoring system.
func StartHealthMonitoring(ctx context.Context) {
	integration := GetRecoverySystem()
	if integration.initialized {
		integration.systemDiagnostics.StartMonitoring(ctx)
	}
}

// StopHealthMonitoring stops the health monitoring system.
func StopHealthMonitoring() {
	integration := GetRecoverySystem()
	if integration.initialized {
		integration.systemDiagnostics.StopMonitoring()
	}
}

// RunSystemDiagnostics runs a full system diagnostic.
func RunSystemDiagnostics(ctx context.Context) (*DiagnosticReport, error) {
	integration := GetRecoverySystem()
	if !integration.initialized {
		return nil, fmt.Errorf("recovery system not initialized")
	}

	return integration.systemDiagnostics.RunDiagnostics(ctx)
}

// RecoveryErrorReporter integrates recovery system with the error reporting system.
type RecoveryErrorReporter struct {
	logger *log.Logger
}

// NewRecoveryErrorReporter creates a new recovery error reporter.
func NewRecoveryErrorReporter(logger *log.Logger) *RecoveryErrorReporter {
	return &RecoveryErrorReporter{
		logger: logger,
	}
}

// Report handles error reporting and triggers recovery if needed.
func (r *RecoveryErrorReporter) Report(ctx context.Context, category errors.ErrorCategory, err error, errorCtx *errors.ErrorContext) error {
	r.logger.Debug("Recovery error reporter triggered",
		"category", category.String(),
		"error", err.Error(),
		"session_id", errorCtx.SessionID)

	// Attempt recovery for certain error types
	if r.shouldAttemptRecovery(category, err) {
		if recoveryErr := RecoverFromError(ctx, err); recoveryErr != nil {
			r.logger.Warn("Recovery attempt failed",
				"original_error", err.Error(),
				"recovery_error", recoveryErr.Error())
		} else {
			r.logger.Info("Recovery successful",
				"category", category.String(),
				"session_id", errorCtx.SessionID)
		}
	}

	return nil
}

// shouldAttemptRecovery determines if recovery should be attempted for the given error.
func (r *RecoveryErrorReporter) shouldAttemptRecovery(category errors.ErrorCategory, err error) bool {
	// Attempt recovery for these error categories
	switch category {
	case errors.NetworkError:
		return true
	case errors.AIServiceError:
		return true
	case errors.SystemError:
		// Only attempt recovery for memory-related system errors
		return isMemoryError(err)
	default:
		return false
	}
}

// Enhanced error handler functions that integrate with recovery system

// HandleWithRecovery handles an error and attempts recovery if appropriate.
func HandleWithRecovery(ctx context.Context, err error, userAction string, metadata map[string]interface{}) error {
	// Use the existing error handler for logging and reporting
	errors.HandleWithContext(err, userAction, metadata)

	// Also attempt recovery
	return RecoverFromError(ctx, err)
}

// CreateSnapshot creates a snapshot of the current application state.
func CreateSnapshot() (*StateSnapshot, error) {
	integration := GetRecoverySystem()
	if !integration.initialized {
		return nil, fmt.Errorf("recovery system not initialized")
	}

	return integration.recoveryManager.CreateSnapshot()
}

// RestoreFromSnapshot restores the application state from a snapshot.
func RestoreFromSnapshot(snapshot *StateSnapshot) error {
	integration := GetRecoverySystem()
	if !integration.initialized {
		return fmt.Errorf("recovery system not initialized")
	}

	return integration.recoveryManager.RestoreFromSnapshot(snapshot)
}

// GetRecoveryStats returns recovery statistics.
func GetRecoveryStats() map[string]interface{} {
	integration := GetRecoverySystem()
	if !integration.initialized {
		return make(map[string]interface{})
	}

	return integration.recoveryManager.GetRecoveryStats()
}

// RegisterRecoveryStrategy registers a custom recovery strategy.
func RegisterRecoveryStrategy(errorType ErrorType, strategy RecoveryStrategy) {
	integration := GetRecoverySystem()
	if integration.initialized {
		integration.recoveryManager.RegisterStrategy(errorType, strategy)
	}
}

// RegisterHealthChecker registers a custom health checker.
func RegisterHealthChecker(checker HealthChecker) {
	integration := GetRecoverySystem()
	if integration.initialized {
		integration.systemDiagnostics.RegisterHealthChecker(checker)
	}
}

// GetApplicationState returns the current application state.
func GetApplicationState() *ApplicationState {
	integration := GetRecoverySystem()
	if integration.initialized {
		return integration.recoveryManager.state
	}
	return nil
}

// UpdateApplicationState updates the application state.
func UpdateApplicationState(updateFunc func(*ApplicationState)) {
	integration := GetRecoverySystem()
	if integration.initialized && integration.recoveryManager.state != nil {
		integration.recoveryManager.state.mu.Lock()
		defer integration.recoveryManager.state.mu.Unlock()
		updateFunc(integration.recoveryManager.state)
		integration.recoveryManager.state.LastUpdate = time.Now()
	}
}
