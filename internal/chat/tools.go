package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/security"
	"github.com/common-creation/coda/internal/tools"
)

// ToolExecutor manages tool execution from AI responses
type ToolExecutor struct {
	manager     *tools.Manager
	validator   security.SecurityValidator
	approver    ApprovalHandler
	concurrency int
	timeout     time.Duration
	retryPolicy RetryPolicy
}

// ApprovalHandler defines the interface for tool execution approval
type ApprovalHandler interface {
	RequestApproval(ctx context.Context, tool string, params map[string]interface{}) (bool, error)
	IsAutoApproved(tool string) bool
}

// RetryPolicy defines retry behavior for tool execution
type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
	BackoffRate float64
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string                 `json:"tool_call_id"`
	ToolName   string                 `json:"tool_name"`
	Result     interface{}            `json:"result"`
	Error      error                  `json:"error,omitempty"`
	ExecutedAt time.Time              `json:"executed_at"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutorOption is a function that configures a ToolExecutor
type ToolExecutorOption func(*ToolExecutor)

// WithConcurrency sets the maximum concurrent tool executions
func WithConcurrency(n int) ToolExecutorOption {
	return func(e *ToolExecutor) {
		if n > 0 {
			e.concurrency = n
		}
	}
}

// WithTimeout sets the timeout for tool execution
func WithTimeout(timeout time.Duration) ToolExecutorOption {
	return func(e *ToolExecutor) {
		e.timeout = timeout
	}
}

// WithRetryPolicy sets the retry policy
func WithRetryPolicy(policy RetryPolicy) ToolExecutorOption {
	return func(e *ToolExecutor) {
		e.retryPolicy = policy
	}
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(manager *tools.Manager, validator security.SecurityValidator, approver ApprovalHandler, opts ...ToolExecutorOption) *ToolExecutor {
	e := &ToolExecutor{
		manager:     manager,
		validator:   validator,
		approver:    approver,
		concurrency: 5, // Default concurrency
		timeout:     30 * time.Second,
		retryPolicy: RetryPolicy{
			MaxAttempts: 3,
			Delay:       1 * time.Second,
			BackoffRate: 2.0,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ExecuteToolCalls processes tool calls from an AI response
func (e *ToolExecutor) ExecuteToolCalls(ctx context.Context, toolCalls []ai.ToolCall) ([]ToolResult, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Prepare execution groups
	groups := e.groupToolCalls(toolCalls)
	results := make([]ToolResult, 0, len(toolCalls))
	resultsChan := make(chan ToolResult, len(toolCalls))
	errorsChan := make(chan error, len(toolCalls))

	// Execute groups sequentially, tools within groups in parallel
	for _, group := range groups {
		groupResults, err := e.executeGroup(execCtx, group, resultsChan, errorsChan)
		if err != nil {
			return results, err
		}
		results = append(results, groupResults...)
	}

	return results, nil
}

// groupToolCalls groups tool calls for execution based on dependencies
func (e *ToolExecutor) groupToolCalls(toolCalls []ai.ToolCall) [][]ai.ToolCall {
	// Simple implementation: all independent tools in one group
	// Advanced implementation would analyze dependencies
	return [][]ai.ToolCall{toolCalls}
}

// executeGroup executes a group of tool calls in parallel
func (e *ToolExecutor) executeGroup(ctx context.Context, group []ai.ToolCall, resultsChan chan<- ToolResult, errorsChan chan<- error) ([]ToolResult, error) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.concurrency)
	results := make([]ToolResult, 0, len(group))
	mu := sync.Mutex{}

	for _, toolCall := range group {
		wg.Add(1)
		go func(tc ai.ToolCall) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := e.executeSingleTool(ctx, tc)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			if result.Error != nil {
				select {
				case errorsChan <- result.Error:
				default:
				}
			}
		}(toolCall)
	}

	wg.Wait()
	return results, nil
}

// executeSingleTool executes a single tool call with retry logic
func (e *ToolExecutor) executeSingleTool(ctx context.Context, toolCall ai.ToolCall) ToolResult {
	startTime := time.Now()
	result := ToolResult{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Function.Name,
		ExecutedAt: startTime,
		Metadata:   make(map[string]interface{}),
	}

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		result.Error = fmt.Errorf("failed to parse arguments: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Security validation
	if err := e.validateToolCall(toolCall.Function.Name, args); err != nil {
		result.Error = fmt.Errorf("security validation failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Request approval if needed
	approved, err := e.requestApproval(ctx, toolCall.Function.Name, args)
	if err != nil {
		result.Error = fmt.Errorf("approval request failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}
	if !approved {
		result.Error = fmt.Errorf("tool execution not approved by user")
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute with retry
	var execErr error
	for attempt := 1; attempt <= e.retryPolicy.MaxAttempts; attempt++ {
		execResult, err := e.manager.Execute(ctx, toolCall.Function.Name, args)
		if err == nil {
			result.Result = execResult
			result.Duration = time.Since(startTime)
			return result
		}

		execErr = err
		if attempt < e.retryPolicy.MaxAttempts {
			delay := time.Duration(float64(e.retryPolicy.Delay) * float64(attempt-1) * e.retryPolicy.BackoffRate)
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Duration = time.Since(startTime)
				return result
			}
		}
	}

	result.Error = fmt.Errorf("failed after %d attempts: %w", e.retryPolicy.MaxAttempts, execErr)
	result.Duration = time.Since(startTime)
	return result
}

// validateToolCall performs security validation on a tool call
func (e *ToolExecutor) validateToolCall(toolName string, args map[string]interface{}) error {
	// Check if tool exists
	_, err := e.manager.Get(toolName)
	if err != nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	// Validate based on tool type
	switch toolName {
	case "read_file", "write_file", "edit_file", "list_files":
		// Validate file paths
		if pathArg, ok := args["path"].(string); ok {
			if err := e.validator.ValidatePath(pathArg); err != nil {
				return err
			}
		}
		if pathArg, ok := args["file_path"].(string); ok {
			if err := e.validator.ValidatePath(pathArg); err != nil {
				return err
			}
		}
	case "search_files":
		// Validate search directory
		if dirArg, ok := args["directory"].(string); ok {
			if err := e.validator.ValidatePath(dirArg); err != nil {
				return err
			}
		}
	}

	return nil
}

// requestApproval requests user approval for tool execution
func (e *ToolExecutor) requestApproval(ctx context.Context, toolName string, args map[string]interface{}) (bool, error) {
	// Check if auto-approved
	if e.approver.IsAutoApproved(toolName) {
		return true, nil
	}

	// Request approval
	return e.approver.RequestApproval(ctx, toolName, args)
}

// FormatToolResults formats tool results for AI consumption
func (e *ToolExecutor) FormatToolResults(results []ToolResult) []ai.Message {
	messages := make([]ai.Message, 0, len(results))

	for _, result := range results {
		content := e.formatSingleResult(result)
		messages = append(messages, ai.Message{
			Role:       ai.RoleTool,
			Content:    content,
			ToolCallID: result.ToolCallID,
		})
	}

	return messages
}

// formatSingleResult formats a single tool result
func (e *ToolExecutor) formatSingleResult(result ToolResult) string {
	if result.Error != nil {
		return fmt.Sprintf("Error executing %s: %v", result.ToolName, result.Error)
	}

	// Format based on result type
	switch v := result.Result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return fmt.Sprintf("Error: %v", v)
	default:
		// Try JSON encoding for complex types
		if data, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// ToolCallAnalyzer analyzes tool calls for patterns and optimization
type ToolCallAnalyzer struct {
	history []ToolResult
	mu      sync.Mutex
}

// NewToolCallAnalyzer creates a new analyzer
func NewToolCallAnalyzer() *ToolCallAnalyzer {
	return &ToolCallAnalyzer{
		history: make([]ToolResult, 0),
	}
}

// Record records a tool execution result
func (a *ToolCallAnalyzer) Record(result ToolResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = append(a.history, result)
}

// GetStatistics returns execution statistics
func (a *ToolCallAnalyzer) GetStatistics() map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	stats := map[string]interface{}{
		"total_executions": len(a.history),
		"tool_usage":       make(map[string]int),
		"error_count":      0,
		"avg_duration":     time.Duration(0),
	}

	toolUsage := stats["tool_usage"].(map[string]int)
	var totalDuration time.Duration

	for _, result := range a.history {
		toolUsage[result.ToolName]++
		if result.Error != nil {
			stats["error_count"] = stats["error_count"].(int) + 1
		}
		totalDuration += result.Duration
	}

	if len(a.history) > 0 {
		stats["avg_duration"] = totalDuration / time.Duration(len(a.history))
	}

	return stats
}

// DefaultApprovalHandler provides a simple approval implementation
type DefaultApprovalHandler struct {
	autoApproved map[string]bool
	mu           sync.RWMutex
}

// NewDefaultApprovalHandler creates a new default approval handler
func NewDefaultApprovalHandler() *DefaultApprovalHandler {
	return &DefaultApprovalHandler{
		autoApproved: map[string]bool{
			"read_file":    false,
			"list_files":   true,
			"search_files": true,
			"write_file":   false,
			"edit_file":    false,
		},
	}
}

// RequestApproval implements ApprovalHandler
func (h *DefaultApprovalHandler) RequestApproval(ctx context.Context, tool string, params map[string]interface{}) (bool, error) {
	// This is a placeholder - actual implementation would interact with UI
	fmt.Printf("\n[Approval Required] Tool: %s\n", tool)
	fmt.Printf("Parameters: %v\n", params)
	fmt.Print("Approve? (y/n): ")

	// For now, return true for testing
	// Real implementation would wait for user input
	return true, nil
}

// IsAutoApproved implements ApprovalHandler
func (h *DefaultApprovalHandler) IsAutoApproved(tool string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.autoApproved[tool]
}

// SetAutoApproval sets auto-approval for a tool
func (h *DefaultApprovalHandler) SetAutoApproval(tool string, approved bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.autoApproved[tool] = approved
}
