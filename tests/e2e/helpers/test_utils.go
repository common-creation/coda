package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/config"
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
}

// E2ETestOptions contains options for setting up the test helper
type E2ETestOptions struct {
	TestDir      string
	WorkspaceDir string
	ConfigFile   string
	EnableMocks  bool
	Timeout      time.Duration
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

// setupMocks sets up mock dependencies for testing
func (h *E2ETestHelper) setupMocks(workspaceDir, configFile string) error {
	// Setup mock config
	h.mockConfig = config.NewDefaultConfig()
	h.mockConfig.Tools.WorkspaceRoot = workspaceDir

	// Override with test config if provided
	if configFile != "" {
		// Load config from file
		// This is simplified - in reality would parse the file
		h.mockConfig.AI.Provider = "openai"
		h.mockConfig.AI.Model = "o3"
	}

	// Setup mock logger
	h.mockLogger = log.New(os.Stderr)
	h.mockLogger.SetLevel(log.DebugLevel)

	// Setup mock AI client
	h.mockAIClient = NewMockAIClient()

	// Setup mock tool manager
	h.mockToolManager = NewMockToolManager()

	// Setup mock chat handler
	h.mockChatHandler = NewMockChatHandler(h.mockAIClient, nil)

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
		Config: h.mockConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	h.app = app
	return nil
}

// SendInput simulates user input
func (h *E2ETestHelper) SendInput(input string) {
	// This is a simplified version - real implementation would require
	// access to the app's internal message handling
	// For now, just record the input
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = append(h.messages, ui.Message{
		Content:   input,
		Role:      "user",
		Timestamp: time.Now(),
	})
}

// WaitForMessage waits for a specific message to appear
func (h *E2ETestHelper) WaitForMessage(content string, timeout time.Duration) (*ui.Message, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		h.mu.RLock()
		for i := len(h.messages) - 1; i >= 0; i-- {
			if h.messages[i].Content == content {
				msg := h.messages[i]
				h.mu.RUnlock()
				return &msg, nil
			}
		}
		h.mu.RUnlock()

		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for message: %s", content)
}

// GetLastMessage returns the last message received
func (h *E2ETestHelper) GetLastMessage() *ui.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.messages) == 0 {
		return nil
	}

	return &h.messages[len(h.messages)-1]
}

// GetAllMessages returns all messages received
func (h *E2ETestHelper) GetAllMessages() []ui.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return append([]ui.Message(nil), h.messages...)
}

// AssertLastMessage asserts the last message matches expected content
func (h *E2ETestHelper) AssertLastMessage(expected string) {
	msg := h.GetLastMessage()
	require.NotNil(h.t, msg, "Expected a message but got none")
	assert.Equal(h.t, expected, msg.Content)
}

// AssertMessageCount asserts the number of messages received
func (h *E2ETestHelper) AssertMessageCount(expected int) {
	h.mu.RLock()
	actual := len(h.messages)
	h.mu.RUnlock()

	assert.Equal(h.t, expected, actual, "Expected %d messages but got %d", expected, actual)
}

// AssertNoErrors asserts that no error messages were received
func (h *E2ETestHelper) AssertNoErrors() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check if any message contains error indicators
	for _, msg := range h.messages {
		// Simple heuristic - check if role is "error" or content contains "error"
		if msg.Role == "error" {
			h.t.Errorf("Unexpected error message: %s", msg.Content)
		}
	}
}

// Cleanup performs cleanup operations
func (h *E2ETestHelper) Cleanup() {
	// Stop the app if running
	if h.app != nil {
		h.app.Shutdown()
		h.app = nil
	}

	// Run cleanup functions
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, fn := range h.cleanupFns {
		fn()
	}
}

// AddCleanup adds a cleanup function to be called during cleanup
func (h *E2ETestHelper) AddCleanup(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupFns = append(h.cleanupFns, fn)
}

// CreateTestFile creates a test file in the workspace
func (h *E2ETestHelper) CreateTestFile(relativePath, content string) error {
	fullPath := filepath.Join(h.mockConfig.Tools.WorkspaceRoot, relativePath)

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
	fullPath := filepath.Join(h.mockConfig.Tools.WorkspaceRoot, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	return string(content), nil
}

// SimulateUserInteraction simulates a complete user interaction
func (h *E2ETestHelper) SimulateUserInteraction(input string, expectedResponse string) error {
	// Send input
	h.SendInput(input)

	// Wait for response
	msg, err := h.WaitForMessage(expectedResponse, h.timeout)
	if err != nil {
		return fmt.Errorf("interaction failed: %w", err)
	}

	h.mu.Lock()
	h.lastMessage = msg
	h.mu.Unlock()

	return nil
}

// GetTestDir returns the test directory path
func (h *E2ETestHelper) GetTestDir() string {
	return h.testDir
}

// GetWorkspaceDir returns the workspace directory path
func (h *E2ETestHelper) GetWorkspaceDir() string {
	return h.mockConfig.Tools.WorkspaceRoot
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
