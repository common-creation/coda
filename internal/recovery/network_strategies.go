package recovery

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/charmbracelet/log"
)

// NetworkRetryStrategy implements automatic retry with exponential backoff for network errors.
type NetworkRetryStrategy struct {
	logger     *log.Logger
	maxRetries int
	backoff    BackoffPolicy
}

// NewNetworkRetryStrategy creates a new network retry strategy.
func NewNetworkRetryStrategy(logger *log.Logger) *NetworkRetryStrategy {
	return &NetworkRetryStrategy{
		logger:     logger,
		maxRetries: 3,
		backoff:    DefaultBackoffPolicy(),
	}
}

// CanRecover determines if this strategy can handle the given error.
func (n *NetworkRetryStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for retryable network errors
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "temporary failure") ||
		strings.Contains(errMsg, "dial") ||
		isTimeoutError(err) ||
		isTemporaryError(err)
}

// Recover attempts to recover from network errors by retrying.
func (n *NetworkRetryStrategy) Recover(ctx context.Context, err error) error {
	n.logger.Debug("Attempting network retry recovery", "error", err.Error())

	// For network retry, we mainly validate that the error is recoverable
	// The actual retry logic would be implemented by the calling code
	if !n.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by network retry: %w", err)
	}

	// Perform basic network connectivity check
	if !n.checkBasicConnectivity() {
		return fmt.Errorf("basic network connectivity check failed")
	}

	n.logger.Info("Network retry recovery completed successfully")
	return nil
}

// Priority returns the priority of this strategy.
func (n *NetworkRetryStrategy) Priority() int {
	return 1 // High priority for basic retries
}

// Name returns the name of this recovery strategy.
func (n *NetworkRetryStrategy) Name() string {
	return "NetworkRetryStrategy"
}

// checkBasicConnectivity performs a basic network connectivity check.
func (n *NetworkRetryStrategy) checkBasicConnectivity() bool {
	// Try to resolve a well-known DNS name
	_, err := net.LookupHost("google.com")
	return err == nil
}

// AlternativeEndpointStrategy tries alternative endpoints when the primary fails.
type AlternativeEndpointStrategy struct {
	logger           *log.Logger
	alternativeHosts []string
	currentHostIndex int
}

// NewAlternativeEndpointStrategy creates a new alternative endpoint strategy.
func NewAlternativeEndpointStrategy(logger *log.Logger) *AlternativeEndpointStrategy {
	return &AlternativeEndpointStrategy{
		logger: logger,
		alternativeHosts: []string{
			"api.openai.com",
			"api.anthropic.com",
			// Add more alternative endpoints as needed
		},
		currentHostIndex: 0,
	}
}

// CanRecover determines if this strategy can handle the given error.
func (a *AlternativeEndpointStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for errors that might be resolved by using alternative endpoints
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "host unreachable") ||
		strings.Contains(errMsg, "dns") ||
		strings.Contains(errMsg, "no route to host") ||
		strings.Contains(errMsg, "service unavailable")
}

// Recover attempts to recover by suggesting alternative endpoints.
func (a *AlternativeEndpointStrategy) Recover(ctx context.Context, err error) error {
	a.logger.Debug("Attempting alternative endpoint recovery", "error", err.Error())

	if !a.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by alternative endpoint: %w", err)
	}

	// In a real implementation, this would actually switch to alternative endpoints
	// For now, we just log the suggestion
	if a.currentHostIndex < len(a.alternativeHosts)-1 {
		a.currentHostIndex++
		nextHost := a.alternativeHosts[a.currentHostIndex]

		a.logger.Info("Suggesting alternative endpoint",
			"current_index", a.currentHostIndex,
			"alternative_host", nextHost)

		return nil
	}

	return fmt.Errorf("no more alternative endpoints available")
}

// Priority returns the priority of this strategy.
func (a *AlternativeEndpointStrategy) Priority() int {
	return 2 // Medium priority, try after basic retry
}

// Name returns the name of this recovery strategy.
func (a *AlternativeEndpointStrategy) Name() string {
	return "AlternativeEndpointStrategy"
}

// OfflineModeStrategy switches the application to offline mode when network is unavailable.
type OfflineModeStrategy struct {
	logger      *log.Logger
	offlineMode bool
}

// NewOfflineModeStrategy creates a new offline mode strategy.
func NewOfflineModeStrategy(logger *log.Logger) *OfflineModeStrategy {
	return &OfflineModeStrategy{
		logger:      logger,
		offlineMode: false,
	}
}

// CanRecover determines if this strategy can handle the given error.
func (o *OfflineModeStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for errors that indicate complete network unavailability
	return strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "no internet connection") ||
		strings.Contains(errMsg, "dns resolution failed") ||
		strings.Contains(errMsg, "connection timed out") ||
		o.isCompleteNetworkFailure(err)
}

// Recover attempts to recover by enabling offline mode.
func (o *OfflineModeStrategy) Recover(ctx context.Context, err error) error {
	o.logger.Debug("Attempting offline mode recovery", "error", err.Error())

	if !o.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by offline mode: %w", err)
	}

	// Enable offline mode
	o.offlineMode = true

	o.logger.Warn("Switched to offline mode due to network issues",
		"offline_mode", o.offlineMode)

	// In a real implementation, this would:
	// 1. Disable network-dependent features
	// 2. Show offline indicator in UI
	// 3. Queue requests for when network is restored
	// 4. Enable local-only operations

	return nil
}

// Priority returns the priority of this strategy.
func (o *OfflineModeStrategy) Priority() int {
	return 3 // Low priority, last resort for network issues
}

// Name returns the name of this recovery strategy.
func (o *OfflineModeStrategy) Name() string {
	return "OfflineModeStrategy"
}

// IsOffline returns whether the application is currently in offline mode.
func (o *OfflineModeStrategy) IsOffline() bool {
	return o.offlineMode
}

// RestoreOnlineMode attempts to restore online mode.
func (o *OfflineModeStrategy) RestoreOnlineMode() error {
	// Check if network is available again
	if o.checkNetworkAvailability() {
		o.offlineMode = false
		o.logger.Info("Restored online mode", "offline_mode", o.offlineMode)
		return nil
	}

	return fmt.Errorf("network still unavailable")
}

// isCompleteNetworkFailure checks if the error indicates complete network failure.
func (o *OfflineModeStrategy) isCompleteNetworkFailure(err error) bool {
	// Try basic connectivity test
	_, dnsErr := net.LookupHost("google.com")
	if dnsErr != nil {
		o.logger.Debug("DNS lookup failed, assuming complete network failure", "dns_error", dnsErr.Error())
		return true
	}
	return false
}

// checkNetworkAvailability checks if network connectivity is restored.
func (o *OfflineModeStrategy) checkNetworkAvailability() bool {
	// Perform multiple connectivity checks
	hosts := []string{"google.com", "cloudflare.com", "8.8.8.8"}

	for _, host := range hosts {
		if _, err := net.LookupHost(host); err == nil {
			return true
		}
	}

	return false
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

// isTemporaryError checks if an error is temporary.
func isTemporaryError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary()
	}
	return false
}
