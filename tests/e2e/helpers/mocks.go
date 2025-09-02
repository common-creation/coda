package helpers

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/tools"
)

// mockStreamReader implements ai.StreamReader interface
type mockStreamReader struct {
	responses []string
	index     int
	position  int
	latency   time.Duration
	error     bool
	errorMsg  string
	ctx       context.Context
	done      bool
}

func (r *mockStreamReader) Read() (*ai.StreamChunk, error) {
	// Check if already done
	if r.done {
		return nil, io.EOF
	}

	// Simulate latency
	if r.latency > 0 {
		time.Sleep(r.latency)
	}

	// Check for simulated error
	if r.error {
		r.done = true
		return nil, fmt.Errorf("mock streaming error: %s", r.errorMsg)
	}

	// Check if we have responses
	if len(r.responses) == 0 {
		r.done = true
		return nil, fmt.Errorf("no mock responses configured")
	}

	// Get current response
	response := r.responses[r.index%len(r.responses)]

	// Check if we've sent all content
	if r.position >= len(response) {
		r.done = true
		return nil, io.EOF
	}

	// Send next character
	char := string(response[r.position])
	r.position++

	chunk := &ai.StreamChunk{
		ID:      fmt.Sprintf("mock-stream-%d-%d", r.index, r.position),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Choices: []ai.StreamChoice{
			{
				Index: 0,
				Delta: ai.StreamDelta{
					Content: char,
				},
				FinishReason: nil,
			},
		},
	}

	// If this was the last character, mark as finished
	if r.position >= len(response) {
		stop := "stop"
		chunk.Choices[0].FinishReason = &stop
	}

	return chunk, nil
}

func (r *mockStreamReader) Close() error {
	r.done = true
	return nil
}

// MockAIClient is a mock implementation of ai.Client for testing
type MockAIClient struct {
	mu              sync.RWMutex
	responses       []string
	responseIndex   int
	callHistory     []ai.ChatRequest
	simulateLatency time.Duration
	simulateError   bool
	errorMessage    string
}

// NewMockAIClient creates a new mock AI client
func NewMockAIClient() *MockAIClient {
	return &MockAIClient{
		responses:       []string{"Mock AI response"},
		responseIndex:   0,
		callHistory:     make([]ai.ChatRequest, 0),
		simulateLatency: 100 * time.Millisecond,
	}
}

// SetNextResponse sets the next response to return
func (m *MockAIClient) SetNextResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = []string{response}
	m.responseIndex = 0
}

// SetResponses sets multiple responses to cycle through
func (m *MockAIClient) SetResponses(responses []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = responses
	m.responseIndex = 0
}

// SetLatency sets the simulated response latency
func (m *MockAIClient) SetLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateLatency = latency
}

// SetError configures the mock to simulate an error
func (m *MockAIClient) SetError(shouldError bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = shouldError
	m.errorMessage = message
}

// GetCallHistory returns the history of calls made to this client
func (m *MockAIClient) GetCallHistory() []ai.ChatRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ai.ChatRequest(nil), m.callHistory...)
}

// ListModels implements ai.Client interface
func (m *MockAIClient) ListModels(ctx context.Context) ([]ai.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.simulateError {
		return nil, fmt.Errorf(m.errorMessage)
	}

	// Return mock models
	return []ai.Model{
		{ID: "o3"},
		{ID: "gpt-3.5-turbo"},
	}, nil
}

// Ping implements ai.Client interface
func (m *MockAIClient) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.simulateError {
		return fmt.Errorf(m.errorMessage)
	}

	return nil
}

// ChatCompletion implements ai.Client interface
func (m *MockAIClient) ChatCompletion(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.callHistory = append(m.callHistory, req)

	// Simulate latency
	if m.simulateLatency > 0 {
		time.Sleep(m.simulateLatency)
	}

	// Check for simulated error
	if m.simulateError {
		return nil, fmt.Errorf("mock AI error: %s", m.errorMessage)
	}

	// Get the current response
	if len(m.responses) == 0 {
		return nil, fmt.Errorf("no mock responses configured")
	}

	response := m.responses[m.responseIndex%len(m.responses)]
	m.responseIndex++

	return &ai.ChatResponse{
		ID:      fmt.Sprintf("mock-response-%d", m.responseIndex),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Choices: []ai.Choice{
			{
				Index: 0,
				Message: ai.Message{
					Role:    ai.RoleAssistant,
					Content: response,
				},
				FinishReason: "stop",
			},
		},
		Usage: ai.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

// StreamChat implements ai.Client interface
func (m *MockAIClient) ChatCompletionStream(ctx context.Context, req ai.ChatRequest) (ai.StreamReader, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.callHistory = append(m.callHistory, req)

	// Create a mock stream reader
	reader := &mockStreamReader{
		responses: m.responses,
		index:     m.responseIndex,
		latency:   m.simulateLatency,
		error:     m.simulateError,
		errorMsg:  m.errorMessage,
		ctx:       ctx,
	}
	m.responseIndex++

	return reader, nil
}

// MockChatHandler is a mock implementation of chat.ChatHandler for testing
type MockChatHandler struct {
	mu          sync.RWMutex
	aiClient    ai.Client
	toolManager *tools.Manager
	sessions    map[string]*MockSession
	responses   []string
	callHistory []string
}

// NewMockChatHandler creates a new mock chat handler
func NewMockChatHandler(aiClient ai.Client, toolManager *tools.Manager) *MockChatHandler {
	return &MockChatHandler{
		aiClient:    aiClient,
		toolManager: toolManager,
		sessions:    make(map[string]*MockSession),
		responses:   []string{"Mock chat response"},
		callHistory: make([]string, 0),
	}
}

// MockSession represents a mock chat session
type MockSession struct {
	ID       string
	Messages []ai.Message
	Active   bool
}

// HandleMessage implements chat.ChatHandler interface
func (m *MockChatHandler) HandleMessage(ctx context.Context, message string, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callHistory = append(m.callHistory, message)

	// Get or create session
	session, exists := m.sessions[sessionID]
	if !exists {
		session = &MockSession{
			ID:       sessionID,
			Messages: make([]ai.Message, 0),
			Active:   true,
		}
		m.sessions[sessionID] = session
	}

	// Add user message
	userMsg := ai.Message{
		Role:    ai.RoleUser,
		Content: message,
	}
	session.Messages = append(session.Messages, userMsg)

	// Generate AI response
	resp, err := m.aiClient.ChatCompletion(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: message},
		},
	})
	if err != nil {
		return fmt.Errorf("AI client error: %w", err)
	}

	// Add assistant message
	if len(resp.Choices) > 0 {
		assistantMsg := resp.Choices[0].Message
		session.Messages = append(session.Messages, assistantMsg)
	}

	return nil
}

// GetSession returns a session by ID
func (m *MockChatHandler) GetSession(sessionID string) (*MockSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[sessionID]
	return session, exists
}

// GetCallHistory returns the history of messages handled
func (m *MockChatHandler) GetCallHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.callHistory...)
}

// MockToolManager is a mock implementation of tools.Manager for testing
type MockToolManager struct {
	mu           sync.RWMutex
	tools        map[string]MockTool
	toolResults  map[string]interface{}
	callHistory  []ToolCall
	approvalMode bool
}

// ToolCall represents a tool execution call
type ToolCall struct {
	ToolName  string
	Arguments map[string]interface{}
	Timestamp time.Time
	Approved  bool
	Result    interface{}
	Error     error
}

// MockTool represents a mock tool
type MockTool struct {
	Name        string
	Description string
	Schema      map[string]interface{}
	Handler     func(args map[string]interface{}) (interface{}, error)
}

// NewMockToolManager creates a new mock tool manager
func NewMockToolManager() *MockToolManager {
	manager := &MockToolManager{
		tools:       make(map[string]MockTool),
		toolResults: make(map[string]interface{}),
		callHistory: make([]ToolCall, 0),
	}

	// Register default mock tools
	manager.registerDefaultTools()

	return manager
}

// registerDefaultTools registers commonly used mock tools
func (m *MockToolManager) registerDefaultTools() {
	// File read tool
	m.tools["read_file"] = MockTool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Schema: map[string]interface{}{
			"path": "string",
		},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			path, ok := args["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path argument required")
			}
			return fmt.Sprintf("Mock file content from: %s", path), nil
		},
	}

	// File write tool
	m.tools["write_file"] = MockTool{
		Name:        "write_file",
		Description: "Write content to a file",
		Schema: map[string]interface{}{
			"path":    "string",
			"content": "string",
		},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			path, _ := args["path"].(string)
			content, _ := args["content"].(string)
			return fmt.Sprintf("Mock wrote %d bytes to %s", len(content), path), nil
		},
	}

	// List files tool
	m.tools["list_files"] = MockTool{
		Name:        "list_files",
		Description: "List files in a directory",
		Schema: map[string]interface{}{
			"path": "string",
		},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			return []string{"file1.txt", "file2.go", "file3.md"}, nil
		},
	}

	// Search tool
	m.tools["search"] = MockTool{
		Name:        "search",
		Description: "Search for text in files",
		Schema: map[string]interface{}{
			"query": "string",
			"path":  "string",
		},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			query, _ := args["query"].(string)
			return fmt.Sprintf("Mock search results for: %s", query), nil
		},
	}
}

// SetToolResult sets a predefined result for a tool
func (m *MockToolManager) SetToolResult(toolName string, result interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolResults[toolName] = result
}

// SetApprovalMode enables/disables approval mode for tool execution
func (m *MockToolManager) SetApprovalMode(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.approvalMode = enabled
}

// ExecuteTool executes a tool with the given arguments
func (m *MockToolManager) ExecuteTool(toolName string, args map[string]interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := ToolCall{
		ToolName:  toolName,
		Arguments: args,
		Timestamp: time.Now(),
		Approved:  !m.approvalMode, // Auto-approve if not in approval mode
	}

	// Check if we have a predefined result
	if result, exists := m.toolResults[toolName]; exists {
		call.Result = result
		m.callHistory = append(m.callHistory, call)
		return result, nil
	}

	// Check if tool exists
	tool, exists := m.tools[toolName]
	if !exists {
		call.Error = fmt.Errorf("tool not found: %s", toolName)
		m.callHistory = append(m.callHistory, call)
		return nil, call.Error
	}

	// Execute tool handler
	result, err := tool.Handler(args)
	call.Result = result
	call.Error = err
	m.callHistory = append(m.callHistory, call)

	return result, err
}

// GetAvailableTools returns the list of available tools
func (m *MockToolManager) GetAvailableTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]string, 0, len(m.tools))
	for name := range m.tools {
		tools = append(tools, name)
	}
	return tools
}

// GetToolSchema returns the schema for a tool
func (m *MockToolManager) GetToolSchema(toolName string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, exists := m.tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	return tool.Schema, nil
}

// GetCallHistory returns the history of tool calls
func (m *MockToolManager) GetCallHistory() []ToolCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ToolCall(nil), m.callHistory...)
}

// RegisterTool registers a new mock tool
func (m *MockToolManager) RegisterTool(tool MockTool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name] = tool
}

// ClearHistory clears the call history
func (m *MockToolManager) ClearHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callHistory = make([]ToolCall, 0)
}

// MockFileSystem provides utilities for creating test files and directories
type MockFileSystem struct {
	BaseDir string
	Files   map[string]string // path -> content
}

// NewMockFileSystem creates a new mock file system
func NewMockFileSystem(baseDir string) *MockFileSystem {
	return &MockFileSystem{
		BaseDir: baseDir,
		Files:   make(map[string]string),
	}
}

// CreateFile creates a mock file with the given content
func (fs *MockFileSystem) CreateFile(path, content string) error {
	fs.Files[path] = content
	return nil
}

// ReadFile reads a mock file
func (fs *MockFileSystem) ReadFile(path string) (string, error) {
	content, exists := fs.Files[path]
	if !exists {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return content, nil
}

// ListFiles returns all files in the mock file system
func (fs *MockFileSystem) ListFiles() []string {
	files := make([]string, 0, len(fs.Files))
	for path := range fs.Files {
		files = append(files, path)
	}
	return files
}
