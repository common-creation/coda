// Package ai provides error types and utilities for AI service interactions.
package ai

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the category of an AI error.
type ErrorType string

// Error type constants define different categories of errors.
const (
	// ErrTypeAuthentication indicates an authentication failure (invalid API key, expired token, etc.)
	ErrTypeAuthentication ErrorType = "authentication"

	// ErrTypeRateLimit indicates the rate limit has been exceeded
	ErrTypeRateLimit ErrorType = "rate_limit"

	// ErrTypeInvalidRequest indicates the request was malformed or invalid
	ErrTypeInvalidRequest ErrorType = "invalid_request"

	// ErrTypeNetwork indicates a network-related error (connection failure, DNS resolution, etc.)
	ErrTypeNetwork ErrorType = "network"

	// ErrTypeTimeout indicates the request timed out
	ErrTypeTimeout ErrorType = "timeout"

	// ErrTypeServerError indicates an error on the AI service side (5xx errors)
	ErrTypeServerError ErrorType = "server_error"

	// ErrTypeQuotaExceeded indicates the account quota has been exceeded
	ErrTypeQuotaExceeded ErrorType = "quota_exceeded"

	// ErrTypeModelNotFound indicates the requested model doesn't exist or isn't accessible
	ErrTypeModelNotFound ErrorType = "model_not_found"

	// ErrTypeContentFilter indicates the content was blocked by safety filters
	ErrTypeContentFilter ErrorType = "content_filter"

	// ErrTypeContextLength indicates the request exceeded the model's context length
	ErrTypeContextLength ErrorType = "context_length"

	// ErrTypeUnknown indicates an unknown or unexpected error
	ErrTypeUnknown ErrorType = "unknown"
)

// Error represents a structured error from the AI service.
// It provides detailed information about what went wrong and can be used
// for programmatic error handling.
type Error struct {
	// Type categorizes the error
	Type ErrorType

	// Message provides a human-readable error description
	Message string

	// Cause is the underlying error, if any
	Cause error

	// Details contains additional error information
	Details map[string]interface{}

	// StatusCode is the HTTP status code, if applicable
	StatusCode int

	// RequestID is the unique request identifier for debugging
	RequestID string

	// Stack contains the stack trace (only in debug mode)
	Stack []StackFrame
}

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// Error implements the error interface.
func (e *Error) Error() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("[%s] %s", e.Type, e.Message))

	if e.RequestID != "" {
		parts = append(parts, fmt.Sprintf("(request_id: %s)", e.RequestID))
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("caused by: %v", e.Cause))
	}

	return strings.Join(parts, " ")
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is interface for error comparison.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// NewError creates a new AI error with the given type and message.
func NewError(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// NewErrorf creates a new AI error with a formatted message.
func NewErrorf(errType ErrorType, format string, args ...interface{}) *Error {
	return &Error{
		Type:    errType,
		Message: fmt.Sprintf(format, args...),
		Details: make(map[string]interface{}),
	}
}

// WithCause adds an underlying cause to the error.
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithDetail adds a key-value detail to the error.
func (e *Error) WithDetail(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithStatusCode sets the HTTP status code.
func (e *Error) WithStatusCode(code int) *Error {
	e.StatusCode = code
	return e
}

// WithRequestID sets the request ID for debugging.
func (e *Error) WithRequestID(id string) *Error {
	e.RequestID = id
	return e
}

// WithStack captures the current stack trace.
func (e *Error) WithStack() *Error {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(2, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	e.Stack = make([]StackFrame, 0, n)

	for {
		frame, more := frames.Next()
		e.Stack = append(e.Stack, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}

	return e
}

// Error checking helpers

// IsRateLimitError checks if the error is a rate limit error.
//
// Example:
//
//	if IsRateLimitError(err) {
//	    // Wait before retrying
//	    time.Sleep(retryDelay)
//	}
func IsRateLimitError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type == ErrTypeRateLimit
	}
	return false
}

// IsAuthenticationError checks if the error is an authentication error.
//
// Example:
//
//	if IsAuthenticationError(err) {
//	    // Prompt user to check API key
//	    return fmt.Errorf("invalid API key: %w", err)
//	}
func IsAuthenticationError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type == ErrTypeAuthentication
	}
	return false
}

// IsRetryableError checks if the error is retryable.
// Network errors, timeouts, rate limits, and server errors are considered retryable.
//
// Example:
//
//	if IsRetryableError(err) {
//	    // Implement exponential backoff
//	    return retryWithBackoff(fn)
//	}
func IsRetryableError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		switch aiErr.Type {
		case ErrTypeNetwork, ErrTypeTimeout, ErrTypeRateLimit, ErrTypeServerError:
			return true
		}
	}
	return false
}

// IsQuotaError checks if the error is due to quota exhaustion.
func IsQuotaError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type == ErrTypeQuotaExceeded
	}
	return false
}

// IsContextLengthError checks if the error is due to context length exceeded.
func IsContextLengthError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type == ErrTypeContextLength
	}
	return false
}

// IsContentFilterError checks if the error is due to content filtering.
func IsContentFilterError(err error) bool {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type == ErrTypeContentFilter
	}
	return false
}

// GetErrorType returns the error type if it's an AI error, otherwise returns ErrTypeUnknown.
func GetErrorType(err error) ErrorType {
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr.Type
	}
	return ErrTypeUnknown
}

// WrapError wraps a generic error into an AI error with the appropriate type.
// It attempts to infer the error type from the error message or type.
func WrapError(err error, defaultType ErrorType) *Error {
	if err == nil {
		return nil
	}

	// If it's already an AI error, return it
	var aiErr *Error
	if errors.As(err, &aiErr) {
		return aiErr
	}

	// Try to infer error type from message
	errMsg := strings.ToLower(err.Error())
	errType := defaultType

	switch {
	case strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "invalid api key") ||
		strings.Contains(errMsg, "authentication"):
		errType = ErrTypeAuthentication

	case strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "too many requests"):
		errType = ErrTypeRateLimit

	case strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded"):
		errType = ErrTypeTimeout

	case strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "dial") ||
		strings.Contains(errMsg, "dns"):
		errType = ErrTypeNetwork

	case strings.Contains(errMsg, "invalid request") ||
		strings.Contains(errMsg, "bad request") ||
		strings.Contains(errMsg, "validation"):
		errType = ErrTypeInvalidRequest

	case strings.Contains(errMsg, "server error") ||
		strings.Contains(errMsg, "internal error") ||
		strings.Contains(errMsg, "500") ||
		strings.Contains(errMsg, "502") ||
		strings.Contains(errMsg, "503"):
		errType = ErrTypeServerError

	case strings.Contains(errMsg, "quota") ||
		strings.Contains(errMsg, "limit exceeded"):
		errType = ErrTypeQuotaExceeded

	case strings.Contains(errMsg, "model not found") ||
		strings.Contains(errMsg, "no such model"):
		errType = ErrTypeModelNotFound

	case strings.Contains(errMsg, "context length") ||
		strings.Contains(errMsg, "token limit"):
		errType = ErrTypeContextLength

	case strings.Contains(errMsg, "content policy") ||
		strings.Contains(errMsg, "safety"):
		errType = ErrTypeContentFilter
	}

	return NewError(errType, err.Error()).WithCause(err)
}
