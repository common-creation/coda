package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/chat"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/tools"
	"github.com/common-creation/coda/internal/ui"
)

// E2ETestHelper provides utilities for end-to-end testing
type E2ETestHelper struct {
	t           *testing.T
	app         *ui.App
	testDir     string
	cleanupFns  []func()
	mu          sync.RWMutex
	messages    []ui.Message
	lastMessage *ui.Message
	timeout     time.Duration

	// Mock dependencies
	mockAIClient    *MockAIClient
	mockConfig      *config.Config
	mockChatHandler *MockChatHandler
	mockToolManager *MockToolManager
	mockLogger      *log.Logger

	// Test input/output capture
	inputBuffer  strings.Builder
	outputBuffer strings.Builder
	errorBuffer  strings.Builder
}

// E2ETestOptions contains options for creating a test helper
type E2ETestOptions struct {
	TestDir      string
	Timeout      time.Duration
	WorkspaceDir string
	ConfigFile   string
	EnableMocks  bool
}

// NewE2ETestHelper creates a new E2E test helper
func NewE2ETestHelper(t *testing.T, opts E2ETestOptions) (*E2ETestHelper, error) {
	helper := &E2ETestHelper{
		t:        t,
		timeout:  opts.Timeout,
		messages: make([]ui.Message, 0),
	}

	if helper.timeout == 0 {
		helper.timeout = 30 * time.Second
	}

	// Setup test directory
	if opts.TestDir == "" {
		tempDir, err := os.MkdirTemp("", "coda-e2e-test-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		helper.testDir = tempDir
	} else {
		helper.testDir = opts.TestDir
	}

	// Add cleanup for test directory
	helper.AddCleanup(func() {
		if helper.testDir != "" {
			os.RemoveAll(helper.testDir)
		}
	})

	// Setup workspace directory
	workspaceDir := opts.WorkspaceDir
	if workspaceDir == "" {
		workspaceDir = filepath.Join(helper.testDir, "workspace")
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create workspace dir: %w", err)
		}
	}

	// Setup mock dependencies if enabled
	if opts.EnableMocks {
		if err := helper.setupMocks(workspaceDir, opts.ConfigFile); err != nil {
			return nil, fmt.Errorf("failed to setup mocks: %w", err)
		}
	}

	return helper, nil
}

// setupMocks initializes mock dependencies
func (h *E2ETestHelper) setupMocks(workspaceDir, configFile string) error {
	// Setup mock logger
	h.mockLogger = log.New(os.Stderr)
	h.mockLogger.SetLevel(log.DebugLevel)

	// Setup mock config
	h.mockConfig = &config.Config{
		AI: config.AIConfig{
			Provider: "openai",
			Model:    "gpt-4",
			APIKey:   "test-key",
		},
		UI: config.UIConfig{
			Theme: "default",
		},
	}

	// Setup mock AI client
	h.mockAIClient = NewMockAIClient()

	// Setup mock tool manager
	h.mockToolManager = NewMockToolManager()

	// Setup mock chat handler
	h.mockChatHandler = NewMockChatHandler(h.mockAIClient, h.mockToolManager)

	return nil
}

// StartApp initializes and starts the application for testing
func (h *E2ETestHelper) StartApp() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.app != nil {
		return fmt.Errorf("app already started")
	}

	// Create the app
	app, err := ui.NewApp(ui.AppOptions{
		Config:      h.mockConfig,
		ChatHandler: h.mockChatHandler,
		ToolManager: h.mockToolManager,
		Logger:      h.mockLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	h.app = app

	// Start the app in a goroutine
	go func() {
		if err := h.app.Run(); err != nil {
			h.t.Errorf("app run failed: %v", err)
		}
	}()

	// Wait for app to be ready
	return h.waitForReady()
}

// waitForReady waits for the application to be ready
func (h *E2ETestHelper) waitForReady() error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for app to be ready")
		case <-ticker.C:
			if h.app != nil && h.app.IsRunning() {
				return nil
			}
		}
	}
}

// SendMessage sends a message through the application
func (h *E2ETestHelper) SendMessage(msg tea.Msg) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.app == nil {
		h.t.Fatal("app not started")
		return
	}

	h.app.SendMessage(msg)
}

// SendKeyMsg sends a key message to the application
func (h *E2ETestHelper) SendKeyMsg(key string) {
	h.SendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

// SendKeys sends a sequence of keys to the application
func (h *E2ETestHelper) SendKeys(keys ...string) {
	for _, key := range keys {
		h.SendKeyMsg(key)
		time.Sleep(10 * time.Millisecond) // Small delay between keys
	}
}

// TypeText types text character by character
func (h *E2ETestHelper) TypeText(text string) {
	for _, char := range text {
		h.SendKeyMsg(string(char))
		time.Sleep(5 * time.Millisecond)
	}
}

// PressKey sends a special key press
func (h *E2ETestHelper) PressKey(keyType tea.KeyType) {
	h.SendMessage(tea.KeyMsg{Type: keyType})
}

// PressEnter sends an Enter key press
func (h *E2ETestHelper) PressEnter() {
	h.PressKey(tea.KeyEnter)
}

// PressEscape sends an Escape key press
func (h *E2ETestHelper) PressEscape() {
	h.PressKey(tea.KeyEsc)
}

// WaitForResponse waits for a response from the AI
func (h *E2ETestHelper) WaitForResponse(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	initialMessageCount := len(h.GetMessages())

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for response")
		case <-ticker.C:
			messages := h.GetMessages()
			if len(messages) > initialMessageCount {
				// Check if the last message is from assistant
				lastMsg := messages[len(messages)-1]
				if lastMsg.Role == "assistant" {
					return nil
				}
			}
		}
	}
}

// WaitForCondition waits for a condition to be true
func (h *E2ETestHelper) WaitForCondition(condition func() bool, timeout time.Duration, description string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition: %s", description)
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// GetModel returns the current UI model
func (h *E2ETestHelper) GetModel() ui.Model {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.app == nil {
		h.t.Fatal("app not started")
		return ui.Model{}
	}

	return h.app.GetModel()
}

// GetMessages returns the current messages
func (h *E2ETestHelper) GetMessages() []ui.Message {
	model := h.GetModel()
	return model.GetMessages() // This method would need to be added to the Model
}

// GetLastMessage returns the last message
func (h *E2ETestHelper) GetLastMessage() *ui.Message {
	messages := h.GetMessages()
	if len(messages) == 0 {
		return nil
	}
	return &messages[len(messages)-1]
}

// AssertOutput checks if the output contains expected text
func (h *E2ETestHelper) AssertOutput(expected string) {
	model := h.GetModel()
	view := model.View()

	assert.Contains(h.t, view, expected, "Expected output not found in view")
}

// AssertMessageCount checks the number of messages
func (h *E2ETestHelper) AssertMessageCount(expected int) {
	messages := h.GetMessages()
	assert.Len(h.t, messages, expected, "Unexpected number of messages")
}

// AssertLastMessageRole checks the role of the last message
func (h *E2ETestHelper) AssertLastMessageRole(expectedRole string) {
	lastMsg := h.GetLastMessage()
	require.NotNil(h.t, lastMsg, "No messages found")
	assert.Equal(h.t, expectedRole, lastMsg.Role, "Unexpected message role")
}

// AssertLastMessageContains checks if the last message contains expected text
func (h *E2ETestHelper) AssertLastMessageContains(expected string) {
	lastMsg := h.GetLastMessage()
	require.NotNil(h.t, lastMsg, "No messages found")
	assert.Contains(h.t, lastMsg.Content, expected, "Expected text not found in last message")
}

// AssertMode checks the current UI mode
func (h *E2ETestHelper) AssertMode(expectedMode ui.Mode) {
	model := h.GetModel()
	currentMode := model.GetCurrentMode() // This method would need to be added to the Model
	assert.Equal(h.t, expectedMode, currentMode, "Unexpected UI mode")
}

// AssertViewType checks the current view type
func (h *E2ETestHelper) AssertViewType(expectedView ui.ViewType) {
	model := h.GetModel()
	currentView := model.GetActiveView() // This method would need to be added to the Model
	assert.Equal(h.t, expectedView, currentView, "Unexpected view type")
}

// CreateTestFile creates a test file in the workspace
func (h *E2ETestHelper) CreateTestFile(relativePath, content string) error {
	fullPath := filepath.Join(h.mockConfig.Workspace.DefaultPath, relativePath)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// ReadTestFile reads a test file from the workspace
func (h *E2ETestHelper) ReadTestFile(relativePath string) (string, error) {
	fullPath := filepath.Join(h.mockConfig.Workspace.DefaultPath, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	return string(content), nil
}

// SimulateUserInteraction simulates a complete user interaction
func (h *E2ETestHelper) SimulateUserInteraction(scenario UserScenario) error {
	for _, step := range scenario.Steps {
		if err := h.executeStep(step); err != nil {
			return fmt.Errorf("failed to execute step %s: %w", step.Description, err)
		}

		// Wait between steps if specified
		if step.WaitAfter > 0 {
			time.Sleep(step.WaitAfter)
		}
	}
	return nil
}

// executeStep executes a single test step
func (h *E2ETestHelper) executeStep(step TestStep) error {
	switch step.Action {
	case "type":
		h.TypeText(step.Input)
	case "key":
		h.SendKeyMsg(step.Input)
	case "enter":
		h.PressEnter()
	case "escape":
		h.PressEscape()
	case "wait_for_response":
		return h.WaitForResponse(h.timeout)
	case "assert_output":
		h.AssertOutput(step.Expected)
	case "assert_message_count":
		count := 1 // Parse from step.Expected if needed
		h.AssertMessageCount(count)
	case "switch_mode":
		// Implementation depends on the specific mode switching logic
		return h.switchToMode(step.Input)
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
	return nil
}

// switchToMode switches to a specific UI mode
func (h *E2ETestHelper) switchToMode(mode string) error {
	switch mode {
	case "insert":
		h.SendKeyMsg("i")
	case "normal":
		h.PressEscape()
	case "command":
		h.SendKeyMsg(":")
	case "search":
		h.SendKeyMsg("/")
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
	return nil
}

// Shutdown stops the application gracefully
func (h *E2ETestHelper) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.app == nil {
		return nil
	}

	return h.app.Shutdown()
}

// AddCleanup adds a cleanup function
func (h *E2ETestHelper) AddCleanup(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupFns = append(h.cleanupFns, fn)
}

// Cleanup runs all cleanup functions
func (h *E2ETestHelper) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Stop the app first
	if h.app != nil {
		h.app.Shutdown()
		h.app = nil
	}

	// Run cleanup functions in reverse order
	for i := len(h.cleanupFns) - 1; i >= 0; i-- {
		h.cleanupFns[i]()
	}
	h.cleanupFns = nil
}

// UserScenario represents a complete user interaction scenario
type UserScenario struct {
	Name        string
	Description string
	Steps       []TestStep
}

// TestStep represents a single step in a test scenario
type TestStep struct {
	Description string
	Action      string        // "type", "key", "enter", "escape", "wait_for_response", "assert_output", etc.
	Input       string        // Input for the action
	Expected    string        // Expected result for assertions
	WaitAfter   time.Duration // Wait time after the step
}

// GetTestDir returns the test directory path
func (h *E2ETestHelper) GetTestDir() string {
	return h.testDir
}

// GetWorkspaceDir returns the workspace directory path
func (h *E2ETestHelper) GetWorkspaceDir() string {
	return h.mockConfig.Workspace.DefaultPath
}

// SetAIResponse sets the next AI response for testing
func (h *E2ETestHelper) SetAIResponse(response string) {
	if h.mockAIClient != nil {
		h.mockAIClient.SetNextResponse(response)
	}
}

// SetToolResult sets the next tool execution result
func (h *E2ETestHelper) SetToolResult(toolName string, result interface{}) {
	if h.mockToolManager != nil {
		h.mockToolManager.SetToolResult(toolName, result)
	}
}

// GetMockAIClient returns the mock AI client for test configuration
func (h *E2ETestHelper) GetMockAIClient() *MockAIClient {
	return h.mockAIClient
}

// GetMockToolManager returns the mock tool manager for test configuration
func (h *E2ETestHelper) GetMockToolManager() *MockToolManager {
	return h.mockToolManager
}

// GetMockChatHandler returns the mock chat handler for test configuration
func (h *E2ETestHelper) GetMockChatHandler() *MockChatHandler {
	return h.mockChatHandler
}

// GetMockConfig returns the mock configuration for test access
func (h *E2ETestHelper) GetMockConfig() *config.Config {
	return h.mockConfig
}
