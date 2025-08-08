// Package recovery provides system recovery mechanisms for handling errors and abnormal states.
package recovery

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/common-creation/coda/internal/errors"
)

// ErrorType represents different types of errors that can be recovered from.
type ErrorType int

const (
	NetworkErrorType ErrorType = iota
	APILimitErrorType
	MemoryErrorType
	PanicErrorType
	SystemErrorType
)

// String returns a string representation of the error type.
func (e ErrorType) String() string {
	switch e {
	case NetworkErrorType:
		return "network"
	case APILimitErrorType:
		return "api_limit"
	case MemoryErrorType:
		return "memory"
	case PanicErrorType:
		return "panic"
	case SystemErrorType:
		return "system"
	default:
		return "unknown"
	}
}

// BackoffPolicy defines the policy for retry backoff.
type BackoffPolicy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	MaxRetries   int
}

// DefaultBackoffPolicy returns a default backoff policy.
func DefaultBackoffPolicy() BackoffPolicy {
	return BackoffPolicy{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		MaxRetries:   3,
	}
}

// RecoveryStrategy defines the interface for error recovery strategies.
type RecoveryStrategy interface {
	// CanRecover determines if this strategy can handle the given error
	CanRecover(error) bool

	// Recover attempts to recover from the error
	Recover(context.Context, error) error

	// Priority returns the priority of this strategy (lower numbers = higher priority)
	Priority() int

	// Name returns the name of this recovery strategy
	Name() string
}

// ApplicationState holds the current state of the application.
type ApplicationState struct {
	Sessions   map[string]SessionData `json:"sessions"`
	UIState    UIState                `json:"ui_state"`
	Config     map[string]interface{} `json:"config"`
	LastUpdate time.Time              `json:"last_update"`
	mu         sync.RWMutex
}

// SessionData represents session-specific data.
type SessionData struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
	Context     map[string]interface{} `json:"context"`
	ChatHistory []ChatMessage          `json:"chat_history"`
}

// ChatMessage represents a single chat message.
type ChatMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// UIState represents the state of the user interface.
type UIState struct {
	ActiveView      string                 `json:"active_view"`
	WindowSize      WindowSize             `json:"window_size"`
	Settings        map[string]interface{} `json:"settings"`
	LastInteraction time.Time              `json:"last_interaction"`
}

// WindowSize represents the size of the application window.
type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// StateSnapshot represents a snapshot of the application state.
type StateSnapshot struct {
	Session   SessionData `json:"session"`
	UI        UIState     `json:"ui"`
	Timestamp time.Time   `json:"timestamp"`
	Checksum  string      `json:"checksum"`
	Version   string      `json:"version"`
}

// RecoveryManager manages the recovery strategies and coordinates recovery attempts.
type RecoveryManager struct {
	strategies map[ErrorType][]RecoveryStrategy
	state      *ApplicationState
	maxRetries int
	backoff    BackoffPolicy
	logger     *log.Logger
	mu         sync.RWMutex

	// Recovery statistics
	recoveryAttempts map[ErrorType]int
	recoverySuccess  map[ErrorType]int
	lastRecovery     time.Time
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager(state *ApplicationState, logger *log.Logger) *RecoveryManager {
	rm := &RecoveryManager{
		strategies:       make(map[ErrorType][]RecoveryStrategy),
		state:            state,
		maxRetries:       3,
		backoff:          DefaultBackoffPolicy(),
		logger:           logger,
		recoveryAttempts: make(map[ErrorType]int),
		recoverySuccess:  make(map[ErrorType]int),
	}

	// Register default recovery strategies
	rm.registerDefaultStrategies()

	return rm
}

// RegisterStrategy registers a recovery strategy for a specific error type.
func (rm *RecoveryManager) RegisterStrategy(errorType ErrorType, strategy RecoveryStrategy) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.strategies[errorType] == nil {
		rm.strategies[errorType] = make([]RecoveryStrategy, 0)
	}

	// Insert strategy in priority order (lower priority number = higher priority)
	strategies := rm.strategies[errorType]
	inserted := false

	for i, existing := range strategies {
		if strategy.Priority() < existing.Priority() {
			// Insert at position i
			strategies = append(strategies[:i], append([]RecoveryStrategy{strategy}, strategies[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		strategies = append(strategies, strategy)
	}

	rm.strategies[errorType] = strategies
	rm.logger.Debug("Registered recovery strategy", "type", errorType.String(), "strategy", strategy.Name())
}

// Recover attempts to recover from the given error using appropriate strategies.
func (rm *RecoveryManager) Recover(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Determine error type
	errorType := rm.classifyError(err)
	rm.recoveryAttempts[errorType]++

	rm.logger.Info("Starting recovery attempt",
		"error_type", errorType.String(),
		"error", err.Error(),
		"attempt", rm.recoveryAttempts[errorType])

	// Get strategies for this error type
	strategies, exists := rm.strategies[errorType]
	if !exists || len(strategies) == 0 {
		rm.logger.Warn("No recovery strategies available", "error_type", errorType.String())
		return fmt.Errorf("no recovery strategies available for error type: %s", errorType.String())
	}

	// Try each strategy in priority order
	var lastErr error
	for _, strategy := range strategies {
		if !strategy.CanRecover(err) {
			continue
		}

		rm.logger.Debug("Attempting recovery with strategy",
			"strategy", strategy.Name(),
			"error_type", errorType.String())

		// Apply backoff if this is a retry
		if rm.recoveryAttempts[errorType] > 1 {
			delay := rm.calculateBackoffDelay(rm.recoveryAttempts[errorType] - 1)
			rm.logger.Debug("Applying backoff delay", "delay", delay.String())

			select {
			case <-time.After(delay):
				// Continue with recovery
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Attempt recovery
		if recoveryErr := strategy.Recover(ctx, err); recoveryErr == nil {
			rm.recoverySuccess[errorType]++
			rm.lastRecovery = time.Now()

			rm.logger.Info("Recovery successful",
				"strategy", strategy.Name(),
				"error_type", errorType.String())

			// Integrate with global error handler
			rm.notifyErrorHandler(errorType, err, strategy.Name(), true)

			return nil
		} else {
			lastErr = recoveryErr
			rm.logger.Warn("Recovery strategy failed",
				"strategy", strategy.Name(),
				"error", recoveryErr.Error())
		}
	}

	rm.logger.Error("All recovery strategies failed",
		"error_type", errorType.String(),
		"attempts", rm.recoveryAttempts[errorType])

	// Integrate with global error handler
	rm.notifyErrorHandler(errorType, err, "", false)

	return fmt.Errorf("recovery failed after trying %d strategies: %w", len(strategies), lastErr)
}

// CreateSnapshot creates a snapshot of the current application state.
func (rm *RecoveryManager) CreateSnapshot() (*StateSnapshot, error) {
	rm.state.mu.RLock()
	defer rm.state.mu.RUnlock()

	// For now, create a snapshot with limited data
	// In a real implementation, this would capture the full state
	snapshot := &StateSnapshot{
		UI:        rm.state.UIState,
		Timestamp: time.Now(),
		Version:   "1.0.0", // This should come from build info
	}

	// Calculate checksum for integrity verification
	snapshot.Checksum = rm.calculateChecksum(snapshot)

	rm.logger.Debug("Created state snapshot", "timestamp", snapshot.Timestamp)

	return snapshot, nil
}

// RestoreFromSnapshot restores the application state from a snapshot.
func (rm *RecoveryManager) RestoreFromSnapshot(snapshot *StateSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	// Verify checksum
	expectedChecksum := rm.calculateChecksum(snapshot)
	if snapshot.Checksum != expectedChecksum {
		return fmt.Errorf("snapshot checksum verification failed")
	}

	rm.state.mu.Lock()
	defer rm.state.mu.Unlock()

	// Restore UI state
	rm.state.UIState = snapshot.UI
	rm.state.LastUpdate = time.Now()

	rm.logger.Info("Restored state from snapshot", "timestamp", snapshot.Timestamp)

	return nil
}

// GetRecoveryStats returns statistics about recovery attempts.
func (rm *RecoveryManager) GetRecoveryStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["last_recovery"] = rm.lastRecovery
	stats["attempts"] = rm.recoveryAttempts
	stats["success"] = rm.recoverySuccess

	// Calculate success rates
	successRates := make(map[string]float64)
	for errorType, attempts := range rm.recoveryAttempts {
		if attempts > 0 {
			success := rm.recoverySuccess[errorType]
			successRates[errorType.String()] = float64(success) / float64(attempts) * 100
		}
	}
	stats["success_rates"] = successRates

	return stats
}

// registerDefaultStrategies registers the default recovery strategies.
func (rm *RecoveryManager) registerDefaultStrategies() {
	// Network error recovery strategies
	rm.RegisterStrategy(NetworkErrorType, NewNetworkRetryStrategy(rm.logger))
	rm.RegisterStrategy(NetworkErrorType, NewAlternativeEndpointStrategy(rm.logger))
	rm.RegisterStrategy(NetworkErrorType, NewOfflineModeStrategy(rm.logger))

	// API limit error recovery strategies
	rm.RegisterStrategy(APILimitErrorType, NewRateLimitStrategy(rm.logger))
	rm.RegisterStrategy(APILimitErrorType, NewRequestQueueStrategy(rm.logger))
	rm.RegisterStrategy(APILimitErrorType, NewAlternativeModelStrategy(rm.logger))

	// Memory error recovery strategies
	rm.RegisterStrategy(MemoryErrorType, NewSessionCleanupStrategy(rm.state, rm.logger))
	rm.RegisterStrategy(MemoryErrorType, NewCacheClearStrategy(rm.logger))
	rm.RegisterStrategy(MemoryErrorType, NewGCForceStrategy(rm.logger))

	// Panic recovery strategies
	rm.RegisterStrategy(PanicErrorType, NewPanicRecoveryStrategy(rm.state, rm.logger))
	rm.RegisterStrategy(PanicErrorType, NewSafeModeStrategy(rm.state, rm.logger))
}

// classifyError determines the error type based on the error content.
func (rm *RecoveryManager) classifyError(err error) ErrorType {
	// Use the existing error handler classification
	handler := errors.Get()
	category := handler.ClassifyError(err)

	switch category {
	case errors.NetworkError:
		return NetworkErrorType
	case errors.AIServiceError:
		return APILimitErrorType
	case errors.SystemError:
		// Check if it's a memory-related system error
		if isMemoryError(err) {
			return MemoryErrorType
		}
		return SystemErrorType
	default:
		return SystemErrorType
	}
}

// calculateBackoffDelay calculates the delay for retry attempts.
func (rm *RecoveryManager) calculateBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rm.backoff.InitialDelay
	}

	delay := time.Duration(float64(rm.backoff.InitialDelay) *
		float64(int(1)<<uint(attempt-1)) * rm.backoff.Multiplier)

	if delay > rm.backoff.MaxDelay {
		delay = rm.backoff.MaxDelay
	}

	return delay
}

// calculateChecksum calculates a checksum for state snapshot integrity.
func (rm *RecoveryManager) calculateChecksum(snapshot *StateSnapshot) string {
	// This is a simplified checksum calculation
	// In a real implementation, you would use a proper hash function
	return fmt.Sprintf("checksum_%d", snapshot.Timestamp.UnixNano())
}

// notifyErrorHandler notifies the global error handler about recovery attempts.
func (rm *RecoveryManager) notifyErrorHandler(errorType ErrorType, err error, strategy string, success bool) {
	handler := errors.Get()

	metadata := map[string]interface{}{
		"recovery_attempted": true,
		"recovery_strategy":  strategy,
		"recovery_success":   success,
		"error_type":         errorType.String(),
	}

	if success {
		handler.UpdateContext("last_successful_recovery", time.Now())
	}

	// Log the recovery attempt with metadata
	handler.HandleWithContext(err, "recovery_attempt", metadata)

	handler.UpdateContext("recovery_stats", rm.GetRecoveryStats())
}

// isMemoryError checks if an error is related to memory issues.
func isMemoryError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return fmt.Sprintf("%s", errMsg) != errMsg || // This is a simple check
		runtime.NumGoroutine() > 10000 // High goroutine count might indicate memory issues
}
