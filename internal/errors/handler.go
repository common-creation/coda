// Package errors provides a global error handling system for the application.
package errors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	ai_errors "github.com/common-creation/coda/internal/ai"
)

// ErrorCategory represents the type of error that occurred.
type ErrorCategory int

const (
	UserError      ErrorCategory = iota // ユーザー起因のエラー
	SystemError                         // システム内部エラー
	NetworkError                        // ネットワーク関連エラー
	ConfigError                         // 設定エラー
	SecurityError                       // セキュリティ違反
	AIServiceError                      // AI サービス関連エラー
)

// String returns a string representation of the error category.
func (c ErrorCategory) String() string {
	switch c {
	case UserError:
		return "user"
	case SystemError:
		return "system"
	case NetworkError:
		return "network"
	case ConfigError:
		return "config"
	case SecurityError:
		return "security"
	case AIServiceError:
		return "ai_service"
	default:
		return "unknown"
	}
}

// ErrorContext provides contextual information about when and where an error occurred.
type ErrorContext struct {
	SessionID   string                 `json:"session_id"`
	UserAction  string                 `json:"user_action"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	StackTrace  []StackFrame           `json:"stack_trace,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Component   string                 `json:"component"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
}

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// ErrorReporter defines the interface for error reporting systems.
type ErrorReporter interface {
	Report(ctx context.Context, category ErrorCategory, err error, context *ErrorContext) error
}

// FallbackHandler handles errors when the main error handling system fails.
type FallbackHandler interface {
	Handle(err error) error
}

// Logger defines the interface for logging systems.
type Logger interface {
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// loggerWrapper wraps charmbracelet/log.Logger to implement our Logger interface
type loggerWrapper struct {
	logger *log.Logger
}

func (l *loggerWrapper) Error(msg string, fields ...interface{}) {
	l.logger.Error(msg, fields...)
}

func (l *loggerWrapper) Warn(msg string, fields ...interface{}) {
	l.logger.Warn(msg, fields...)
}

func (l *loggerWrapper) Info(msg string, fields ...interface{}) {
	l.logger.Info(msg, fields...)
}

func (l *loggerWrapper) Debug(msg string, fields ...interface{}) {
	l.logger.Debug(msg, fields...)
}

// ErrorHandler is the global error handling system.
type ErrorHandler struct {
	logger      Logger
	reporters   []ErrorReporter
	fallback    FallbackHandler
	context     *ErrorContext
	mu          sync.RWMutex
	initialized bool
	logFile     *os.File
}

// Config holds configuration for the error handler.
type Config struct {
	LogLevel    string
	LogFile     string
	SessionID   string
	Component   string
	Version     string
	Environment string
	EnableStack bool
}

// globalHandler holds the singleton instance.
var (
	globalHandler *ErrorHandler
	once          sync.Once
)

// NewErrorHandler creates a new error handler with the given configuration.
func NewErrorHandler(config Config) *ErrorHandler {
	// Parse log level with error handling
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel // Default to Info if parsing fails
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           logLevel,
	})

	var logFile *os.File
	if config.LogFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(logDir, 0755); err == nil {
			if f, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				logFile = f
			}
		}
	}

	handler := &ErrorHandler{
		logger:      &loggerWrapper{logger: logger},
		reporters:   make([]ErrorReporter, 0),
		fallback:    &defaultFallbackHandler{},
		logFile:     logFile,
		initialized: true,
		context: &ErrorContext{
			SessionID:   config.SessionID,
			Component:   config.Component,
			Version:     config.Version,
			Environment: config.Environment,
			Metadata:    make(map[string]interface{}),
		},
	}

	return handler
}

// Init initializes the global error handler.
func Init(config Config) {
	once.Do(func() {
		globalHandler = NewErrorHandler(config)
	})
}

// Get returns the global error handler instance.
func Get() *ErrorHandler {
	if globalHandler == nil {
		// Fallback initialization with default config
		Init(Config{
			LogLevel:    "info",
			SessionID:   generateSessionID(),
			Component:   "coda",
			Version:     "unknown",
			Environment: "development",
		})
	}
	return globalHandler
}

// Handle processes an error through the global error handling system.
func Handle(err error) {
	Get().Handle(err)
}

// HandleWithContext processes an error with additional context.
func HandleWithContext(err error, userAction string, metadata map[string]interface{}) {
	Get().HandleWithContext(err, userAction, metadata)
}

// Handle processes an error through the error handling system.
func (h *ErrorHandler) Handle(err error) {
	h.HandleWithContext(err, "", nil)
}

// HandleWithContext processes an error with additional context information.
func (h *ErrorHandler) HandleWithContext(err error, userAction string, metadata map[string]interface{}) {
	if err == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 1. Capture error context
	ctx := h.captureContext(userAction, metadata)

	// 2. Classify error
	category := h.ClassifyError(err)

	// 3. Add context information
	ctx.Timestamp = time.Now()
	if category != UserError {
		ctx.StackTrace = h.captureStackTrace()
	}

	// 4. Log the error
	h.logError(category, err, ctx)

	// 5. Report to external systems
	h.reportError(category, err, ctx)

	// 6. Attempt recovery if possible
	h.attemptRecovery(category, err, ctx)
}

// UserMessage returns a user-friendly error message that hides technical details.
func (h *ErrorHandler) UserMessage(err error) string {
	if err == nil {
		return ""
	}

	category := h.ClassifyError(err)

	switch category {
	case UserError:
		return h.formatUserError(err)
	case NetworkError:
		return "ネットワーク接続に問題が発生しました。インターネット接続を確認してから再試行してください。"
	case ConfigError:
		return "設定に問題があります。設定ファイルを確認するか、デフォルト設定で再試行してください。"
	case SecurityError:
		return "セキュリティ上の問題が検出されました。操作が制限されています。"
	case AIServiceError:
		return h.formatAIServiceError(err)
	case SystemError:
		return "システムエラーが発生しました。しばらく待ってから再試行してください。問題が続く場合はサポートにお問い合わせください。"
	default:
		return "予期しないエラーが発生しました。しばらく待ってから再試行してください。"
	}
}

// ClassifyError determines the category of an error.
func (h *ErrorHandler) ClassifyError(err error) ErrorCategory {
	if err == nil {
		return SystemError
	}

	// Check for AI service errors first
	if ai_errors.IsAuthenticationError(err) ||
		ai_errors.IsRateLimitError(err) ||
		ai_errors.IsQuotaError(err) ||
		ai_errors.IsContextLengthError(err) ||
		ai_errors.IsContentFilterError(err) {
		return AIServiceError
	}

	if ai_errors.IsRetryableError(err) {
		errType := ai_errors.GetErrorType(err)
		switch errType {
		case ai_errors.ErrTypeNetwork, ai_errors.ErrTypeTimeout:
			return NetworkError
		default:
			return AIServiceError
		}
	}

	// Check error message for classification
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "access denied") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "forbidden"):
		return SecurityError

	case strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "dial") ||
		strings.Contains(errMsg, "dns"):
		return NetworkError

	case strings.Contains(errMsg, "config") ||
		strings.Contains(errMsg, "configuration") ||
		strings.Contains(errMsg, "invalid key") ||
		strings.Contains(errMsg, "missing required"):
		return ConfigError

	case strings.Contains(errMsg, "file not found") ||
		strings.Contains(errMsg, "no such file") ||
		strings.Contains(errMsg, "invalid argument") ||
		strings.Contains(errMsg, "invalid input"):
		return UserError

	default:
		return SystemError
	}
}

// formatUserError formats user-friendly messages for user errors.
func (h *ErrorHandler) formatUserError(err error) string {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "file not found") || strings.Contains(errMsg, "no such file"):
		return "指定されたファイルが見つかりません。ファイルパスを確認してください。"
	case strings.Contains(errMsg, "permission denied"):
		return "ファイルまたはディレクトリへのアクセス権限がありません。"
	case strings.Contains(errMsg, "invalid argument") || strings.Contains(errMsg, "invalid input"):
		return "入力された内容に問題があります。入力を確認してください。"
	default:
		return "入力や操作に問題があります。内容を確認してから再試行してください。"
	}
}

// formatAIServiceError formats user-friendly messages for AI service errors.
func (h *ErrorHandler) formatAIServiceError(err error) string {
	if ai_errors.IsAuthenticationError(err) {
		return "APIキーが無効です。設定を確認してください。"
	}
	if ai_errors.IsRateLimitError(err) {
		return "利用制限に達しました。しばらく待ってから再試行してください。"
	}
	if ai_errors.IsQuotaError(err) {
		return "利用量の上限に達しました。プランを確認してください。"
	}
	if ai_errors.IsContextLengthError(err) {
		return "入力が長すぎます。内容を短くしてください。"
	}
	if ai_errors.IsContentFilterError(err) {
		return "コンテンツポリシーに違反する内容が検出されました。内容を修正してください。"
	}
	return "AIサービスでエラーが発生しました。しばらく待ってから再試行してください。"
}

// captureContext creates an error context with current information.
func (h *ErrorHandler) captureContext(userAction string, metadata map[string]interface{}) *ErrorContext {
	ctx := &ErrorContext{
		SessionID:   h.context.SessionID,
		Component:   h.context.Component,
		Version:     h.context.Version,
		Environment: h.context.Environment,
		UserAction:  userAction,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Copy base metadata
	for k, v := range h.context.Metadata {
		ctx.Metadata[k] = v
	}

	// Add provided metadata
	for k, v := range metadata {
		ctx.Metadata[k] = v
	}

	return ctx
}

// captureStackTrace captures the current stack trace.
func (h *ErrorHandler) captureStackTrace() []StackFrame {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:]) // Skip captureStackTrace, HandleWithContext, Handle

	frames := runtime.CallersFrames(pcs[:n])
	stackTrace := make([]StackFrame, 0, n)

	for {
		frame, more := frames.Next()
		// Skip internal error handling frames
		if !strings.Contains(frame.Function, "internal/errors") {
			stackTrace = append(stackTrace, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
		}
		if !more {
			break
		}
	}

	return stackTrace
}

// logError logs the error with appropriate level and detail.
func (h *ErrorHandler) logError(category ErrorCategory, err error, ctx *ErrorContext) {
	fields := []interface{}{
		"category", category.String(),
		"session_id", ctx.SessionID,
		"component", ctx.Component,
		"user_action", ctx.UserAction,
		"timestamp", ctx.Timestamp.Format(time.RFC3339),
	}

	if ctx.RequestID != "" {
		fields = append(fields, "request_id", ctx.RequestID)
	}

	// Add metadata as fields
	for k, v := range ctx.Metadata {
		fields = append(fields, k, v)
	}

	switch category {
	case SecurityError:
		h.logger.Error("Security error occurred", append(fields, "error", err.Error())...)
	case SystemError:
		h.logger.Error("System error occurred", append(fields, "error", err.Error())...)
	case NetworkError, AIServiceError:
		h.logger.Warn("Service error occurred", append(fields, "error", err.Error())...)
	case ConfigError:
		h.logger.Warn("Configuration error occurred", append(fields, "error", err.Error())...)
	case UserError:
		h.logger.Info("User error occurred", append(fields, "error", err.Error())...)
	default:
		h.logger.Error("Unknown error occurred", append(fields, "error", err.Error())...)
	}

	// Write to log file if available
	if h.logFile != nil {
		logLine := fmt.Sprintf("[%s] [%s] %s - %s\n",
			ctx.Timestamp.Format(time.RFC3339),
			category.String(),
			ctx.UserAction,
			err.Error(),
		)
		h.logFile.WriteString(logLine)
		h.logFile.Sync()
	}
}

// reportError reports the error to external systems.
func (h *ErrorHandler) reportError(category ErrorCategory, err error, ctx *ErrorContext) {
	for _, reporter := range h.reporters {
		if reportErr := reporter.Report(context.Background(), category, err, ctx); reportErr != nil {
			// Log reporter failure but don't fail the main error handling
			h.logger.Debug("Error reporter failed", "reporter_error", reportErr.Error())
		}
	}
}

// attemptRecovery tries to recover from certain types of errors.
func (h *ErrorHandler) attemptRecovery(category ErrorCategory, err error, ctx *ErrorContext) {
	switch category {
	case AIServiceError:
		h.attemptAIServiceRecovery(err, ctx)
	case NetworkError:
		h.attemptNetworkRecovery(err, ctx)
	case ConfigError:
		h.attemptConfigRecovery(err, ctx)
	}
}

// attemptAIServiceRecovery attempts recovery for AI service errors.
func (h *ErrorHandler) attemptAIServiceRecovery(err error, ctx *ErrorContext) {
	if ai_errors.IsRateLimitError(err) {
		// Could implement exponential backoff here
		h.logger.Debug("Rate limit detected, consider implementing backoff", "session_id", ctx.SessionID)
	}
}

// attemptNetworkRecovery attempts recovery for network errors.
func (h *ErrorHandler) attemptNetworkRecovery(err error, ctx *ErrorContext) {
	// Could implement network diagnostics or retry logic
	h.logger.Debug("Network error detected, consider retry logic", "session_id", ctx.SessionID)
}

// attemptConfigRecovery attempts recovery for configuration errors.
func (h *ErrorHandler) attemptConfigRecovery(err error, ctx *ErrorContext) {
	// Could attempt to load default configuration
	h.logger.Debug("Config error detected, consider fallback config", "session_id", ctx.SessionID)
}

// AddReporter adds an error reporter to the handler.
func (h *ErrorHandler) AddReporter(reporter ErrorReporter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reporters = append(h.reporters, reporter)
}

// SetFallbackHandler sets the fallback error handler.
func (h *ErrorHandler) SetFallbackHandler(handler FallbackHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fallback = handler
}

// UpdateContext updates the global context information.
func (h *ErrorHandler) UpdateContext(key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.context.Metadata == nil {
		h.context.Metadata = make(map[string]interface{})
	}
	h.context.Metadata[key] = value
}

// Close cleans up resources used by the error handler.
func (h *ErrorHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.logFile != nil {
		return h.logFile.Close()
	}
	return nil
}

// defaultFallbackHandler is the default fallback error handler.
type defaultFallbackHandler struct{}

func (d *defaultFallbackHandler) Handle(err error) error {
	// Last resort: print to stderr
	fmt.Fprintf(os.Stderr, "FATAL ERROR: %v\n", err)
	return err
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
