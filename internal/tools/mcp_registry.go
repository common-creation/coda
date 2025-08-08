package tools

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// State represents the state of an MCP server
type State int

const (
	StateStarting State = iota
	StateRunning
	StateError
	StateStopped
)

// String returns the string representation of a State
func (s State) String() string {
	switch s {
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateError:
		return "Error"
	case StateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// ServerStatus represents the current status of an MCP server
type ServerStatus struct {
	Name  string
	State State
	Error error
}

// ToolInfo represents information about an available tool
type ToolInfo struct {
	ServerName  string                 `json:"serverName"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ResourceInfo represents information about an available resource
type ResourceInfo struct {
	ServerName  string `json:"serverName"`
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PromptInfo represents information about an available prompt
type PromptInfo struct {
	ServerName  string `json:"serverName"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// MCPManager defines the interface for MCP client management
type MCPManager interface {
	// Status management
	GetServerStatus(name string) ServerStatus
	GetAllStatuses() map[string]ServerStatus

	// Tool/Resource/Prompt management
	ListTools() ([]ToolInfo, error)
	ListResources() ([]ResourceInfo, error)
	ListPrompts() ([]PromptInfo, error)

	// Tool execution
	ExecuteTool(serverName, toolName string, params map[string]interface{}) (interface{}, error)
}

// MCPRegistry manages dynamic registration of MCP tools to the CODA tool system
type MCPRegistry struct {
	toolManager ToolManager
	mcpManager  MCPManager
	logger      *log.Logger

	// Track registered MCP tools for cleanup
	registeredTools map[string][]string // serverName -> list of tool names
	mu              sync.RWMutex
}

// ToolManager interface for dependency injection
type ToolManager interface {
	Register(tool Tool) error
	Unregister(name string) error
	List() []string
	Get(name string) (Tool, error)
}

// NewMCPRegistry creates a new tool registry instance
func NewMCPRegistry(toolManager ToolManager, mcpManager MCPManager, logger *log.Logger) *MCPRegistry {
	if logger == nil {
		logger = log.New(os.Stderr)
	}

	return &MCPRegistry{
		toolManager:     toolManager,
		mcpManager:      mcpManager,
		logger:          logger,
		registeredTools: make(map[string][]string),
	}
}

// RegisterServerTools registers all available tools from an MCP server
func (tr *MCPRegistry) RegisterServerTools(serverName string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Check if server is running
	status := tr.mcpManager.GetServerStatus(serverName)
	if status.State != StateRunning {
		return fmt.Errorf("server %s is not running (state: %s)", serverName, status.State)
	}

	tr.logger.Debug("Registering tools for MCP server", "server", serverName)

	// Get available tools from MCP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a background context to avoid blocking during tool discovery
	go tr.registerServerToolsAsync(ctx, serverName)

	return nil
}

// UnregisterServerTools removes all tools from a specific MCP server
func (tr *MCPRegistry) UnregisterServerTools(serverName string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	toolNames, exists := tr.registeredTools[serverName]
	if !exists {
		tr.logger.Debug("No tools registered for server", "server", serverName)
		return nil
	}

	var errors []error
	for _, toolName := range toolNames {
		if err := tr.toolManager.Unregister(toolName); err != nil {
			errors = append(errors, fmt.Errorf("failed to unregister tool %s: %w", toolName, err))
			tr.logger.Error("Failed to unregister MCP tool", "tool", toolName, "error", err)
		} else {
			tr.logger.Info("Unregistered MCP tool", "tool", toolName, "server", serverName)
		}
	}

	// Remove from tracking map
	delete(tr.registeredTools, serverName)

	if len(errors) > 0 {
		return fmt.Errorf("failed to unregister some tools from server %s: %v", serverName, errors)
	}

	tr.logger.Info("Unregistered all tools for MCP server", "server", serverName, "count", len(toolNames))
	return nil
}

// RefreshServerTools updates the tool registration for a server
func (tr *MCPRegistry) RefreshServerTools(serverName string) error {
	// First unregister existing tools
	if err := tr.UnregisterServerTools(serverName); err != nil {
		tr.logger.Warn("Error unregistering existing tools during refresh", "server", serverName, "error", err)
	}

	// Then register current tools
	return tr.RegisterServerTools(serverName)
}

// RefreshAllTools refreshes tool registration for all running servers
func (tr *MCPRegistry) RefreshAllTools() error {
	statuses := tr.mcpManager.GetAllStatuses()
	var errors []error

	for serverName, status := range statuses {
		if status.State == StateRunning {
			if err := tr.RefreshServerTools(serverName); err != nil {
				errors = append(errors, fmt.Errorf("failed to refresh tools for server %s: %w", serverName, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to refresh tools for some servers: %v", errors)
	}

	return nil
}

// GetRegisteredMCPTools returns all currently registered MCP tools
func (tr *MCPRegistry) GetRegisteredMCPTools() map[string][]string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string][]string)
	for server, tools := range tr.registeredTools {
		result[server] = make([]string, len(tools))
		copy(result[server], tools)
	}

	return result
}

// GetMCPToolCount returns the total number of registered MCP tools
func (tr *MCPRegistry) GetMCPToolCount() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	count := 0
	for _, toolNames := range tr.registeredTools {
		count += len(toolNames)
	}

	return count
}

// IsToolFromServer checks if a tool belongs to a specific MCP server
func (tr *MCPRegistry) IsToolFromServer(toolName, serverName string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	toolNames, exists := tr.registeredTools[serverName]
	if !exists {
		return false
	}

	for _, registeredTool := range toolNames {
		if registeredTool == toolName {
			return true
		}
	}

	return false
}

// registerServerToolsAsync performs asynchronous tool registration for a server
func (tr *MCPRegistry) registerServerToolsAsync(ctx context.Context, serverName string) {
	// Get tools from MCP manager
	allTools, err := tr.mcpManager.ListTools()
	if err != nil {
		tr.logger.Error("Failed to list MCP tools", "server", serverName, "error", err)
		return
	}

	// Filter tools for this server
	serverTools := []ToolInfo{}
	for _, toolInfo := range allTools {
		if toolInfo.ServerName == serverName {
			serverTools = append(serverTools, toolInfo)
		}
	}

	if len(serverTools) == 0 {
		tr.logger.Debug("No tools available from MCP server", "server", serverName)
		return
	}

	tr.logger.Info("Found MCP tools to register", "server", serverName, "count", len(serverTools))

	// Register each tool
	registeredTools := []string{}
	var registrationErrors []error

	for _, toolInfo := range serverTools {
		mcpTool := NewMCPTool(serverName, toolInfo, tr.mcpManager)
		toolName := mcpTool.Name()

		if err := tr.toolManager.Register(mcpTool); err != nil {
			registrationErrors = append(registrationErrors, fmt.Errorf("failed to register tool %s: %w", toolName, err))
			tr.logger.Error("Failed to register MCP tool", "tool", toolName, "server", serverName, "error", err)
		} else {
			registeredTools = append(registeredTools, toolName)
			tr.logger.Info("Registered MCP tool", "tool", toolName, "server", serverName)
		}
	}

	// Update the tracking map with successfully registered tools
	tr.mu.Lock()
	tr.registeredTools[serverName] = registeredTools
	tr.mu.Unlock()

	if len(registrationErrors) > 0 {
		tr.logger.Warn("Some tools failed to register", "server", serverName, "errors", len(registrationErrors))
	}

	tr.logger.Info("Completed tool registration for MCP server",
		"server", serverName,
		"registered", len(registeredTools),
		"failed", len(registrationErrors))
}

// HandleServerStateChange handles MCP server state changes for tool registration
func (tr *MCPRegistry) HandleServerStateChange(serverName string, oldState, newState State) {
	tr.logger.Debug("Handling server state change",
		"server", serverName,
		"oldState", oldState.String(),
		"newState", newState.String())

	switch newState {
	case StateRunning:
		// Server started running, register its tools
		if err := tr.RegisterServerTools(serverName); err != nil {
			tr.logger.Error("Failed to register tools for started server", "server", serverName, "error", err)
		}

	case StateStopped, StateError:
		// Server stopped or errored, unregister its tools
		if err := tr.UnregisterServerTools(serverName); err != nil {
			tr.logger.Error("Failed to unregister tools for stopped server", "server", serverName, "error", err)
		}
	}
}

// ListMCPToolsByServer returns MCP tools grouped by server
func (tr *MCPRegistry) ListMCPToolsByServer() map[string][]Tool {
	result := make(map[string][]Tool)

	// Get all registered tools from tool manager
	allToolNames := tr.toolManager.List()

	for _, toolName := range allToolNames {
		if !IsMCPTool(toolName) {
			continue
		}

		// Parse the tool name to get server info
		serverName, _, ok := ParseMCPToolName(toolName)
		if !ok {
			continue
		}

		// Get the actual tool instance
		tool, err := tr.toolManager.Get(toolName)
		if err != nil {
			tr.logger.Warn("Failed to get registered MCP tool", "tool", toolName, "error", err)
			continue
		}

		if result[serverName] == nil {
			result[serverName] = []Tool{}
		}
		result[serverName] = append(result[serverName], tool)
	}

	return result
}
