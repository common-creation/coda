package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ApprovalMode defines the approval behavior
type ApprovalMode int

const (
	// ApproveAll automatically approves all operations
	ApproveAll ApprovalMode = iota
	// ApproveWrite requires approval for write operations
	ApproveWrite
	// ApproveNone rejects all operations
	ApproveNone
	// Interactive asks for user approval
	Interactive
)

// ApprovalRecord represents a single approval decision
type ApprovalRecord struct {
	Timestamp   time.Time              `json:"timestamp"`
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Approved    bool                   `json:"approved"`
	Mode        ApprovalMode           `json:"mode"`
	Reason      string                 `json:"reason,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	AutoApprove bool                   `json:"auto_approve"`
}

// InteractiveApprovalHandler implements interactive approval
type InteractiveApprovalHandler struct {
	mode         ApprovalMode
	history      []ApprovalRecord
	autoRules    map[string]bool
	pathRules    map[string]bool
	sessionRules map[string]map[string]bool
	mu           sync.RWMutex
	input        *bufio.Reader
	outputFunc   func(string)
	historyLimit int
}

// NewInteractiveApprovalHandler creates a new interactive approval handler
func NewInteractiveApprovalHandler(mode ApprovalMode) *InteractiveApprovalHandler {
	return &InteractiveApprovalHandler{
		mode:         mode,
		history:      make([]ApprovalRecord, 0),
		autoRules:    make(map[string]bool),
		pathRules:    make(map[string]bool),
		sessionRules: make(map[string]map[string]bool),
		input:        bufio.NewReader(os.Stdin),
		outputFunc:   func(s string) { fmt.Print(s) },
		historyLimit: 1000,
	}
}

// RequestApproval implements ApprovalHandler interface
func (h *InteractiveApprovalHandler) RequestApproval(ctx context.Context, tool string, params map[string]interface{}) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check approval mode
	switch h.mode {
	case ApproveAll:
		h.recordApproval(tool, params, true, "auto-approved (mode: all)")
		return true, nil
	case ApproveNone:
		h.recordApproval(tool, params, false, "auto-rejected (mode: none)")
		return false, nil
	case ApproveWrite:
		if h.isWriteOperation(tool) {
			return h.interactiveApproval(ctx, tool, params)
		}
		h.recordApproval(tool, params, true, "auto-approved (read operation)")
		return true, nil
	case Interactive:
		return h.interactiveApproval(ctx, tool, params)
	}

	return false, fmt.Errorf("unknown approval mode: %v", h.mode)
}

// IsAutoApproved implements ApprovalHandler interface
func (h *InteractiveApprovalHandler) IsAutoApproved(tool string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check global auto rules
	if approved, exists := h.autoRules[tool]; exists {
		return approved
	}

	// Check if it's a safe operation
	return h.isSafeOperation(tool)
}

// SetApprovalMode sets the approval mode
func (h *InteractiveApprovalHandler) SetApprovalMode(mode ApprovalMode) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mode = mode
}

// GetApprovalMode returns the current approval mode
func (h *InteractiveApprovalHandler) GetApprovalMode() ApprovalMode {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.mode
}

// GetHistory returns the approval history
func (h *InteractiveApprovalHandler) GetHistory() []ApprovalRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]ApprovalRecord, len(h.history))
	copy(history, h.history)
	return history
}

// SetAutoRule sets an auto-approval rule for a tool
func (h *InteractiveApprovalHandler) SetAutoRule(tool string, approve bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.autoRules[tool] = approve
}

// SetPathRule sets an auto-approval rule for a path pattern
func (h *InteractiveApprovalHandler) SetPathRule(pathPattern string, approve bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pathRules[pathPattern] = approve
}

// SetSessionRule sets a session-specific approval rule
func (h *InteractiveApprovalHandler) SetSessionRule(sessionID, tool string, approve bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessionRules[sessionID] == nil {
		h.sessionRules[sessionID] = make(map[string]bool)
	}
	h.sessionRules[sessionID][tool] = approve
}

// ClearHistory clears the approval history
func (h *InteractiveApprovalHandler) ClearHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = make([]ApprovalRecord, 0)
}

// SetOutputFunc sets a custom output function
func (h *InteractiveApprovalHandler) SetOutputFunc(f func(string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.outputFunc = f
}

// Private methods

func (h *InteractiveApprovalHandler) interactiveApproval(ctx context.Context, tool string, params map[string]interface{}) (bool, error) {
	// Check session rules first
	sessionID := h.getSessionID(ctx)
	if sessionID != "" && h.sessionRules[sessionID] != nil {
		if approved, exists := h.sessionRules[sessionID][tool]; exists {
			h.recordApproval(tool, params, approved, fmt.Sprintf("session rule (%s)", sessionID))
			return approved, nil
		}
	}

	// Check path rules
	if path := h.extractPath(params); path != "" {
		for pattern, approved := range h.pathRules {
			if h.matchPath(path, pattern) {
				h.recordApproval(tool, params, approved, fmt.Sprintf("path rule (%s)", pattern))
				return approved, nil
			}
		}
	}

	// Display approval request
	h.displayApprovalRequest(tool, params)

	// Read user input
	response, err := h.readUserResponse()
	if err != nil {
		return false, err
	}

	// Process response
	switch strings.ToLower(response) {
	case "y", "yes":
		h.recordApproval(tool, params, true, "user approved")
		return true, nil
	case "n", "no":
		h.recordApproval(tool, params, false, "user rejected")
		return false, nil
	case "a", "always":
		h.autoRules[tool] = true
		h.recordApproval(tool, params, true, "user approved (always)")
		return true, nil
	case "never":
		h.autoRules[tool] = false
		h.recordApproval(tool, params, false, "user rejected (never)")
		return false, nil
	case "session":
		if sessionID != "" {
			h.SetSessionRule(sessionID, tool, true)
			h.recordApproval(tool, params, true, "user approved (session)")
			return true, nil
		}
		h.outputFunc("No active session for session-specific approval\n")
		return h.interactiveApproval(ctx, tool, params)
	default:
		h.outputFunc("Invalid response. Please enter y/n/a/never/session\n")
		return h.interactiveApproval(ctx, tool, params)
	}
}

func (h *InteractiveApprovalHandler) displayApprovalRequest(tool string, params map[string]interface{}) {
	h.outputFunc("\n" + strings.Repeat("=", 60) + "\n")
	h.outputFunc(fmt.Sprintf("APPROVAL REQUIRED: %s\n", tool))
	h.outputFunc(strings.Repeat("-", 60) + "\n")

	// Display operation details
	h.outputFunc("Operation Details:\n")
	h.outputFunc(fmt.Sprintf("  Tool: %s\n", tool))
	h.outputFunc(fmt.Sprintf("  Risk Level: %s\n", h.getRiskLevel(tool)))

	// Display parameters
	if len(params) > 0 {
		h.outputFunc("\nParameters:\n")
		for key, value := range params {
			h.outputFunc(fmt.Sprintf("  %s: %v\n", key, h.formatValue(value)))
		}
	}

	// Display potential impact
	impact := h.getOperationImpact(tool, params)
	if impact != "" {
		h.outputFunc(fmt.Sprintf("\nPotential Impact:\n%s\n", impact))
	}

	h.outputFunc(strings.Repeat("-", 60) + "\n")
	h.outputFunc("Options: [y]es / [n]o / [a]lways / never / session\n")
	h.outputFunc("Choice: ")
}

func (h *InteractiveApprovalHandler) readUserResponse() (string, error) {
	response, err := h.input.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}

func (h *InteractiveApprovalHandler) recordApproval(tool string, params map[string]interface{}, approved bool, reason string) {
	record := ApprovalRecord{
		Timestamp:   time.Now(),
		Tool:        tool,
		Parameters:  params,
		Approved:    approved,
		Mode:        h.mode,
		Reason:      reason,
		AutoApprove: reason != "user approved" && reason != "user rejected",
	}

	h.history = append(h.history, record)

	// Limit history size
	if len(h.history) > h.historyLimit {
		h.history = h.history[len(h.history)-h.historyLimit:]
	}
}

func (h *InteractiveApprovalHandler) isWriteOperation(tool string) bool {
	writeOps := []string{"write_file", "edit_file", "delete_file", "create_directory", "remove_directory"}
	for _, op := range writeOps {
		if tool == op {
			return true
		}
	}
	return false
}

func (h *InteractiveApprovalHandler) isSafeOperation(tool string) bool {
	safeOps := []string{"read_file", "list_files", "search_files", "get_info"}
	for _, op := range safeOps {
		if tool == op {
			return true
		}
	}
	return false
}

func (h *InteractiveApprovalHandler) getRiskLevel(tool string) string {
	switch tool {
	case "delete_file", "remove_directory":
		return "HIGH - Permanent data loss"
	case "write_file", "edit_file":
		return "MEDIUM - Data modification"
	case "create_directory":
		return "LOW - Filesystem change"
	default:
		return "MINIMAL - Read-only operation"
	}
}

func (h *InteractiveApprovalHandler) getOperationImpact(tool string, params map[string]interface{}) string {
	switch tool {
	case "write_file":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("- Will create or overwrite file: %s", path)
		}
	case "edit_file":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("- Will modify existing file: %s", path)
		}
	case "delete_file":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("- Will permanently delete: %s", path)
		}
	}
	return ""
}

func (h *InteractiveApprovalHandler) extractPath(params map[string]interface{}) string {
	if path, ok := params["file_path"].(string); ok {
		return path
	}
	if path, ok := params["path"].(string); ok {
		return path
	}
	return ""
}

func (h *InteractiveApprovalHandler) matchPath(path, pattern string) bool {
	// Simple pattern matching - can be enhanced with glob patterns
	return strings.HasPrefix(path, pattern)
}

func (h *InteractiveApprovalHandler) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if len(v) > 100 {
			return v[:97] + "..."
		}
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (h *InteractiveApprovalHandler) getSessionID(ctx context.Context) string {
	// Extract session ID from context if available
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		return sessionID
	}
	return ""
}
