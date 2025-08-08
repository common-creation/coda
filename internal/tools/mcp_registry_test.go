package tools

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockToolManager is a mock implementation of ToolManager for testing
type MockToolManager struct {
	mock.Mock
	tools map[string]Tool
}

func NewMockToolManager() *MockToolManager {
	return &MockToolManager{
		tools: make(map[string]Tool),
	}
}

func (m *MockToolManager) Register(tool Tool) error {
	args := m.Called(tool)
	if args.Error(0) == nil {
		m.tools[tool.Name()] = tool
	}
	return args.Error(0)
}

func (m *MockToolManager) Unregister(name string) error {
	args := m.Called(name)
	if args.Error(0) == nil {
		delete(m.tools, name)
	}
	return args.Error(0)
}

func (m *MockToolManager) List() []string {
	args := m.Called()
	if len(args) > 0 {
		return args.Get(0).([]string)
	}

	// Default implementation
	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

func (m *MockToolManager) Get(name string) (Tool, error) {
	args := m.Called(name)
	if tool := args.Get(0); tool != nil {
		return tool.(Tool), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestNewMCPRegistry(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	logger := log.New(os.Stderr)

	registry := NewMCPRegistry(toolManager, mcpManager, logger)

	assert.NotNil(t, registry)
	assert.Equal(t, toolManager, registry.toolManager)
	assert.Equal(t, mcpManager, registry.mcpManager)
	assert.NotNil(t, registry.logger)
	assert.NotNil(t, registry.registeredTools)
}

func TestMCPRegistryRegisterServerTools(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "test-server"

	// Mock server status - running
	mcpManager.On("GetServerStatus", serverName).Return(ServerStatus{
		Name:  serverName,
		State: StateRunning,
	})

	// Mock tools list
	toolInfos := []ToolInfo{
		{
			ServerName:  serverName,
			Name:        "tool1",
			Description: "Test tool 1",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			ServerName:  serverName,
			Name:        "tool2",
			Description: "Test tool 2",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	mcpManager.On("ListTools").Return(toolInfos, nil)

	// Mock tool registration - expect Register to be called for each tool
	toolManager.On("Register", mock.Anything).Return(nil).Times(2)

	// Register tools
	err := registry.RegisterServerTools(serverName)
	require.NoError(t, err)

	// Wait a bit for async registration to complete
	time.Sleep(100 * time.Millisecond)

	// Verify that tools were registered
	registeredTools := registry.GetRegisteredMCPTools()
	assert.Contains(t, registeredTools, serverName)
	assert.Len(t, registeredTools[serverName], 2)

	expectedToolNames := []string{"mcp_test-server_tool1", "mcp_test-server_tool2"}
	for _, expectedName := range expectedToolNames {
		found := false
		for _, actualName := range registeredTools[serverName] {
			if actualName == expectedName {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected tool %s to be registered", expectedName)
	}

	mcpManager.AssertExpectations(t)
	toolManager.AssertExpectations(t)
}

func TestMCPRegistryRegisterServerToolsNotRunning(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "stopped-server"

	// Mock server status - not running
	mcpManager.On("GetServerStatus", serverName).Return(ServerStatus{
		Name:  serverName,
		State: StateStopped,
	})

	// Register tools should fail
	err := registry.RegisterServerTools(serverName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not running")

	mcpManager.AssertExpectations(t)
	toolManager.AssertNotCalled(t, "Register")
}

func TestMCPRegistryUnregisterServerTools(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "test-server"
	toolNames := []string{"mcp_test-server_tool1", "mcp_test-server_tool2"}

	// Pre-populate registered tools
	registry.registeredTools[serverName] = toolNames

	// Mock unregistration
	for _, toolName := range toolNames {
		toolManager.On("Unregister", toolName).Return(nil)
	}

	// Unregister tools
	err := registry.UnregisterServerTools(serverName)
	require.NoError(t, err)

	// Verify tools were removed from tracking
	registeredTools := registry.GetRegisteredMCPTools()
	assert.NotContains(t, registeredTools, serverName)

	toolManager.AssertExpectations(t)
}

func TestMCPRegistryUnregisterServerToolsNotRegistered(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "unregistered-server"

	// Unregister tools for non-existent server should not error
	err := registry.UnregisterServerTools(serverName)
	assert.NoError(t, err)

	toolManager.AssertNotCalled(t, "Unregister")
}

func TestMCPRegistryHandleServerStateChange(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "state-change-server"

	tests := []struct {
		name     string
		oldState State
		newState State
		setup    func()
		verify   func()
	}{
		{
			name:     "server starts running",
			oldState: StateStarting,
			newState: StateRunning,
			setup: func() {
				mcpManager.On("GetServerStatus", serverName).Return(ServerStatus{
					Name:  serverName,
					State: StateRunning,
				})
				mcpManager.On("ListTools").Return([]ToolInfo{}, nil)
			},
			verify: func() {
				// Should attempt to register tools (even if empty)
			},
		},
		{
			name:     "server stops",
			oldState: StateRunning,
			newState: StateStopped,
			setup: func() {
				// Pre-populate some tools
				registry.registeredTools[serverName] = []string{"mcp_state-change-server_tool1"}
				toolManager.On("Unregister", "mcp_state-change-server_tool1").Return(nil)
			},
			verify: func() {
				// Should unregister tools
				registeredTools := registry.GetRegisteredMCPTools()
				assert.NotContains(t, registeredTools, serverName)
			},
		},
		{
			name:     "server errors",
			oldState: StateRunning,
			newState: StateError,
			setup: func() {
				// Pre-populate some tools
				registry.registeredTools[serverName] = []string{"mcp_state-change-server_tool1"}
				toolManager.On("Unregister", "mcp_state-change-server_tool1").Return(nil)
			},
			verify: func() {
				// Should unregister tools
				registeredTools := registry.GetRegisteredMCPTools()
				assert.NotContains(t, registeredTools, serverName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous mock calls
			mcpManager.ExpectedCalls = nil
			toolManager.ExpectedCalls = nil

			tt.setup()

			// Handle state change
			registry.HandleServerStateChange(serverName, tt.oldState, tt.newState)

			// Wait for async operations
			time.Sleep(100 * time.Millisecond)

			tt.verify()
		})
	}
}

func TestMCPRegistryGetMCPToolCount(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	// Initially should be 0
	assert.Equal(t, 0, registry.GetMCPToolCount())

	// Add some tools
	registry.registeredTools["server1"] = []string{"tool1", "tool2"}
	registry.registeredTools["server2"] = []string{"tool3"}

	// Should return total count
	assert.Equal(t, 3, registry.GetMCPToolCount())
}

func TestMCPRegistryIsToolFromServer(t *testing.T) {
	toolManager := NewMockToolManager()
	mcpManager := &MockMCPManager{}
	registry := NewMCPRegistry(toolManager, mcpManager, nil)

	serverName := "test-server"
	toolNames := []string{"mcp_test-server_tool1", "mcp_test-server_tool2"}

	registry.registeredTools[serverName] = toolNames

	tests := []struct {
		toolName   string
		serverName string
		expected   bool
	}{
		{"mcp_test-server_tool1", "test-server", true},
		{"mcp_test-server_tool2", "test-server", true},
		{"mcp_test-server_tool3", "test-server", false},
		{"mcp_test-server_tool1", "other-server", false},
		{"regular_tool", "test-server", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName+"_"+tt.serverName, func(t *testing.T) {
			result := registry.IsToolFromServer(tt.toolName, tt.serverName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
