package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMCPManager is a mock implementation of Manager for testing
type MockMCPManager struct {
	mock.Mock
}

func (m *MockMCPManager) LoadConfig(paths []string) error {
	args := m.Called(paths)
	return args.Error(0)
}

func (m *MockMCPManager) StartServer(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockMCPManager) StopServer(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockMCPManager) RestartServer(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockMCPManager) StartAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMCPManager) StopAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMCPManager) GetServerStatus(name string) ServerStatus {
	args := m.Called(name)
	return args.Get(0).(ServerStatus)
}

func (m *MockMCPManager) GetAllStatuses() map[string]ServerStatus {
	args := m.Called()
	return args.Get(0).(map[string]ServerStatus)
}

func (m *MockMCPManager) ListTools() ([]ToolInfo, error) {
	args := m.Called()
	return args.Get(0).([]ToolInfo), args.Error(1)
}

func (m *MockMCPManager) ListResources() ([]ResourceInfo, error) {
	args := m.Called()
	return args.Get(0).([]ResourceInfo), args.Error(1)
}

func (m *MockMCPManager) ListPrompts() ([]PromptInfo, error) {
	args := m.Called()
	return args.Get(0).([]PromptInfo), args.Error(1)
}

func (m *MockMCPManager) ExecuteTool(serverName, toolName string, params map[string]interface{}) (interface{}, error) {
	args := m.Called(serverName, toolName, params)
	return args.Get(0), args.Error(1)
}

func TestNewMCPTool(t *testing.T) {
	manager := &MockMCPManager{}

	toolInfo := ToolInfo{
		ServerName:  "test-server",
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
			"required": []interface{}{"param1"},
		},
	}

	tool := NewMCPTool("test-server", toolInfo, manager)

	assert.NotNil(t, tool)
	assert.Equal(t, "test-server", tool.serverName)
	assert.Equal(t, "test-tool", tool.toolName)
	assert.Equal(t, toolInfo, tool.toolInfo)
	assert.Equal(t, manager, tool.manager)
}

func TestMCPToolName(t *testing.T) {
	manager := &MockMCPManager{}

	toolInfo := ToolInfo{
		ServerName: "filesystem",
		Name:       "read_file",
	}

	tool := NewMCPTool("filesystem", toolInfo, manager)

	assert.Equal(t, "mcp_filesystem_read_file", tool.Name())
}

func TestMCPToolDescription(t *testing.T) {
	manager := &MockMCPManager{}

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "with description",
			description: "Reads a file from the filesystem",
			expected:    "[MCP:filesystem] Reads a file from the filesystem",
		},
		{
			name:        "without description",
			description: "",
			expected:    "[MCP:filesystem] MCP tool read_file from server filesystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolInfo := ToolInfo{
				ServerName:  "filesystem",
				Name:        "read_file",
				Description: tt.description,
			}

			tool := NewMCPTool("filesystem", toolInfo, manager)

			assert.Equal(t, tt.expected, tool.Description())
		})
	}
}

func TestMCPToolSchema(t *testing.T) {
	manager := &MockMCPManager{}

	toolInfo := ToolInfo{
		ServerName: "filesystem",
		Name:       "read_file",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file",
				},
				"encoding": map[string]interface{}{
					"type":        "string",
					"description": "File encoding",
					"default":     "utf-8",
					"enum":        []interface{}{"utf-8", "ascii", "base64"},
				},
			},
			"required": []interface{}{"path"},
		},
	}

	tool := NewMCPTool("filesystem", toolInfo, manager)
	schema := tool.Schema()

	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 2)
	assert.Contains(t, schema.Required, "path")
	assert.NotContains(t, schema.Required, "encoding")

	// Check path property
	pathProp := schema.Properties["path"]
	assert.Equal(t, "string", pathProp.Type)
	assert.Equal(t, "Path to the file", pathProp.Description)

	// Check encoding property
	encodingProp := schema.Properties["encoding"]
	assert.Equal(t, "string", encodingProp.Type)
	assert.Equal(t, "File encoding", encodingProp.Description)
	assert.Equal(t, "utf-8", encodingProp.Default)
	assert.Equal(t, []string{"utf-8", "ascii", "base64"}, encodingProp.Enum)
}

func TestMCPToolSchemaEmpty(t *testing.T) {
	manager := &MockMCPManager{}

	toolInfo := ToolInfo{
		ServerName:  "simple-server",
		Name:        "ping",
		InputSchema: nil,
	}

	tool := NewMCPTool("simple-server", toolInfo, manager)
	schema := tool.Schema()

	assert.Equal(t, "object", schema.Type)
	assert.Empty(t, schema.Properties)
	assert.Empty(t, schema.Required)
}

func TestMCPToolExecute(t *testing.T) {
	manager := &MockMCPManager{}

	// Mock server status check
	manager.On("GetServerStatus", "filesystem").Return(ServerStatus{
		Name:  "filesystem",
		State: StateRunning,
	})

	// Mock tool execution
	expectedResult := map[string]interface{}{
		"content": "file content here",
		"size":    100,
	}
	manager.On("ExecuteTool", "filesystem", "read_file", mock.Anything).Return(expectedResult, nil)

	toolInfo := ToolInfo{
		ServerName: "filesystem",
		Name:       "read_file",
	}

	tool := NewMCPTool("filesystem", toolInfo, manager)

	params := map[string]interface{}{
		"path": "/path/to/file.txt",
	}

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "filesystem", resultMap["server"])
	assert.Equal(t, "read_file", resultMap["tool"])
	assert.Equal(t, expectedResult, resultMap["result"])
	assert.Equal(t, true, resultMap["success"])

	manager.AssertExpectations(t)
}

func TestMCPToolExecuteServerNotRunning(t *testing.T) {
	manager := &MockMCPManager{}

	// Mock server status check - server not running
	manager.On("GetServerStatus", "filesystem").Return(ServerStatus{
		Name:  "filesystem",
		State: StateStopped,
	})

	toolInfo := ToolInfo{
		ServerName: "filesystem",
		Name:       "read_file",
	}

	tool := NewMCPTool("filesystem", toolInfo, manager)

	params := map[string]interface{}{
		"path": "/path/to/file.txt",
	}

	result, err := tool.Execute(context.Background(), params)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "MCP server 'filesystem' is not running")

	manager.AssertExpectations(t)
}

func TestMCPToolValidate(t *testing.T) {
	manager := &MockMCPManager{}

	toolInfo := ToolInfo{
		ServerName: "filesystem",
		Name:       "read_file",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"size": map[string]interface{}{
					"type": "integer",
				},
			},
			"required": []interface{}{"path"},
		},
	}

	tool := NewMCPTool("filesystem", toolInfo, manager)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"path": "/path/to/file.txt",
				"size": 100,
			},
			wantErr: false,
		},
		{
			name: "missing required parameter",
			params: map[string]interface{}{
				"size": 100,
			},
			wantErr: true,
		},
		{
			name: "wrong parameter type",
			params: map[string]interface{}{
				"path": "/path/to/file.txt",
				"size": "not-a-number",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsMCPTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"mcp_filesystem_read_file", true},
		{"mcp_github_create_issue", true},
		{"read_file", false},
		{"write_file", false},
		{"list_files", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsMCPTool(tt.toolName))
		})
	}
}

func TestParseMCPToolName(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		wantServer string
		wantTool   string
		wantOK     bool
	}{
		{
			name:       "valid MCP tool name",
			toolName:   "mcp_filesystem_read_file",
			wantServer: "filesystem",
			wantTool:   "read_file",
			wantOK:     true,
		},
		{
			name:       "MCP tool with underscore in tool name",
			toolName:   "mcp_github_create_pull_request",
			wantServer: "github",
			wantTool:   "create_pull_request",
			wantOK:     true,
		},
		{
			name:     "non-MCP tool",
			toolName: "read_file",
			wantOK:   false,
		},
		{
			name:     "invalid MCP tool format",
			toolName: "mcp_filesystem",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, tool, ok := ParseMCPToolName(tt.toolName)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantServer, server)
				assert.Equal(t, tt.wantTool, tool)
			}
		})
	}
}
