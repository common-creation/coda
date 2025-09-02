package recovery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	ai_errors "github.com/common-creation/coda/internal/ai"
)

// RateLimitStrategy implements rate limiting compliance for API limit errors.
type RateLimitStrategy struct {
	logger               *log.Logger
	lastRequestTime      time.Time
	requestCount         int
	resetTime            time.Time
	mu                   sync.Mutex
	rateLimitWindow      time.Duration
	maxRequestsPerWindow int
}

// NewRateLimitStrategy creates a new rate limit strategy.
func NewRateLimitStrategy(logger *log.Logger) *RateLimitStrategy {
	return &RateLimitStrategy{
		logger:               logger,
		rateLimitWindow:      time.Minute,
		maxRequestsPerWindow: 60, // Default: 60 requests per minute
	}
}

// CanRecover determines if this strategy can handle the given error.
func (r *RateLimitStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	// Check for rate limit specific errors
	if ai_errors.IsRateLimitError(err) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "too many requests") ||
		strings.Contains(errMsg, "quota exceeded") ||
		strings.Contains(errMsg, "429") ||
		strings.Contains(errMsg, "throttled")
}

// Recover attempts to recover from rate limit errors by implementing proper rate limiting.
func (r *RateLimitStrategy) Recover(ctx context.Context, err error) error {
	r.logger.Debug("Attempting rate limit recovery", "error", err.Error())

	if !r.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by rate limiting: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Check if we need to reset the rate limit window
	if now.Sub(r.resetTime) >= r.rateLimitWindow {
		r.requestCount = 0
		r.resetTime = now
	}

	// Calculate wait time based on rate limit
	waitTime := r.calculateWaitTime(err)

	r.logger.Info("Applying rate limit delay",
		"wait_time", waitTime.String(),
		"request_count", r.requestCount,
		"max_requests", r.maxRequestsPerWindow)

	// Wait for the calculated time
	select {
	case <-time.After(waitTime):
		r.lastRequestTime = time.Now()
		r.requestCount++
		r.logger.Info("Rate limit recovery completed")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Priority returns the priority of this strategy.
func (r *RateLimitStrategy) Priority() int {
	return 1 // High priority for rate limit errors
}

// Name returns the name of this recovery strategy.
func (r *RateLimitStrategy) Name() string {
	return "RateLimitStrategy"
}

// calculateWaitTime calculates how long to wait based on the error and current state.
func (r *RateLimitStrategy) calculateWaitTime(err error) time.Duration {
	// Try to extract wait time from error message if available
	if waitTime := r.extractWaitTimeFromError(err); waitTime > 0 {
		return waitTime
	}

	// Calculate based on current request rate
	if r.requestCount >= r.maxRequestsPerWindow {
		// Wait until the rate limit window resets
		return r.rateLimitWindow - time.Since(r.resetTime)
	}

	// Default exponential backoff
	baseDelay := 1 * time.Second
	return time.Duration(r.requestCount+1) * baseDelay
}

// extractWaitTimeFromError attempts to extract retry-after time from error message.
func (r *RateLimitStrategy) extractWaitTimeFromError(err error) time.Duration {
	// This is a simplified implementation
	// In a real implementation, you would parse headers like "Retry-After"
	errMsg := err.Error()

	if strings.Contains(errMsg, "retry after") {
		// Try to extract time from message
		// This is a placeholder - actual implementation would parse the time
		return 30 * time.Second
	}

	return 0
}

// RequestQueueStrategy queues requests when rate limits are hit.
type RequestQueueStrategy struct {
	logger       *log.Logger
	requestQueue []QueuedRequest
	processing   bool
	mu           sync.Mutex
	maxQueueSize int
}

// QueuedRequest represents a queued API request.
type QueuedRequest struct {
	ID        string
	Request   interface{}
	Timestamp time.Time
	Priority  int
	Retries   int
}

// NewRequestQueueStrategy creates a new request queue strategy.
func NewRequestQueueStrategy(logger *log.Logger) *RequestQueueStrategy {
	return &RequestQueueStrategy{
		logger:       logger,
		requestQueue: make([]QueuedRequest, 0),
		maxQueueSize: 100,
	}
}

// CanRecover determines if this strategy can handle the given error.
func (q *RequestQueueStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	// This strategy can handle any rate limit or quota error
	if ai_errors.IsRateLimitError(err) || ai_errors.IsQuotaError(err) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "quota") ||
		strings.Contains(errMsg, "too many requests") ||
		strings.Contains(errMsg, "service unavailable")
}

// Recover attempts to recover by queuing the request for later processing.
func (q *RequestQueueStrategy) Recover(ctx context.Context, err error) error {
	q.logger.Debug("Attempting request queue recovery", "error", err.Error())

	if !q.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by request queuing: %w", err)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check queue capacity
	if len(q.requestQueue) >= q.maxQueueSize {
		// Remove oldest request to make room
		q.requestQueue = q.requestQueue[1:]
		q.logger.Warn("Request queue full, removed oldest request")
	}

	// Add current request to queue (placeholder)
	queuedReq := QueuedRequest{
		ID:        fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Request:   "placeholder_request", // In real implementation, this would be the actual request
		Timestamp: time.Now(),
		Priority:  1,
		Retries:   0,
	}

	q.requestQueue = append(q.requestQueue, queuedReq)

	q.logger.Info("Request queued for later processing",
		"queue_size", len(q.requestQueue),
		"request_id", queuedReq.ID)

	// Start processing queue if not already running
	if !q.processing {
		go q.processQueue()
	}

	return nil
}

// Priority returns the priority of this strategy.
func (q *RequestQueueStrategy) Priority() int {
	return 2 // Medium priority
}

// Name returns the name of this recovery strategy.
func (q *RequestQueueStrategy) Name() string {
	return "RequestQueueStrategy"
}

// processQueue processes queued requests with proper rate limiting.
func (q *RequestQueueStrategy) processQueue() {
	q.mu.Lock()
	q.processing = true
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		q.processing = false
		q.mu.Unlock()
	}()

	for {
		q.mu.Lock()
		if len(q.requestQueue) == 0 {
			q.mu.Unlock()
			break
		}

		// Get next request
		req := q.requestQueue[0]
		q.requestQueue = q.requestQueue[1:]
		q.mu.Unlock()

		q.logger.Debug("Processing queued request", "request_id", req.ID)

		// In a real implementation, this would actually process the request
		// For now, just simulate processing with a delay
		time.Sleep(1 * time.Second)

		q.logger.Info("Processed queued request", "request_id", req.ID)
	}
}

// GetQueueSize returns the current size of the request queue.
func (q *RequestQueueStrategy) GetQueueSize() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.requestQueue)
}

// AlternativeModelStrategy switches to alternative AI models when primary model is rate limited.
type AlternativeModelStrategy struct {
	logger            *log.Logger
	alternativeModels []string
	currentModelIndex int
	mu                sync.Mutex
}

// NewAlternativeModelStrategy creates a new alternative model strategy.
func NewAlternativeModelStrategy(logger *log.Logger) *AlternativeModelStrategy {
	return &AlternativeModelStrategy{
		logger: logger,
		alternativeModels: []string{
			"gpt-3.5-turbo",
			"o3",
			"claude-3-haiku",
			"claude-3-sonnet",
			// Add more alternative models as needed
		},
		currentModelIndex: 0,
	}
}

// CanRecover determines if this strategy can handle the given error.
func (a *AlternativeModelStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	// Check for errors that might be model-specific
	if ai_errors.IsRateLimitError(err) ||
		ai_errors.IsQuotaError(err) ||
		ai_errors.IsContextLengthError(err) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "model") ||
		strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "quota") ||
		strings.Contains(errMsg, "unavailable") ||
		strings.Contains(errMsg, "overloaded")
}

// Recover attempts to recover by switching to an alternative model.
func (a *AlternativeModelStrategy) Recover(ctx context.Context, err error) error {
	a.logger.Debug("Attempting alternative model recovery", "error", err.Error())

	if !a.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by alternative model: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Try next model in the list
	if a.currentModelIndex < len(a.alternativeModels)-1 {
		a.currentModelIndex++
		newModel := a.alternativeModels[a.currentModelIndex]

		a.logger.Info("Switching to alternative model",
			"previous_index", a.currentModelIndex-1,
			"new_index", a.currentModelIndex,
			"new_model", newModel)

		// In a real implementation, this would actually switch the model configuration
		return nil
	}

	return fmt.Errorf("no more alternative models available")
}

// Priority returns the priority of this strategy.
func (a *AlternativeModelStrategy) Priority() int {
	return 3 // Lower priority, try after rate limiting and queuing
}

// Name returns the name of this recovery strategy.
func (a *AlternativeModelStrategy) Name() string {
	return "AlternativeModelStrategy"
}

// GetCurrentModel returns the currently selected model.
func (a *AlternativeModelStrategy) GetCurrentModel() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentModelIndex < len(a.alternativeModels) {
		return a.alternativeModels[a.currentModelIndex]
	}
	return "unknown"
}

// ResetToDefaultModel resets to the default (first) model.
func (a *AlternativeModelStrategy) ResetToDefaultModel() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.currentModelIndex = 0
	a.logger.Info("Reset to default model", "model", a.alternativeModels[0])
}
