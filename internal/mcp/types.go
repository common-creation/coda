package mcp

import (
	"time"
)

// Config represents the MCP configuration structure
type Config struct {
	Servers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig defines configuration for an individual MCP server
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Type    string            `json:"type,omitempty"`    // stdio, http, sse
	URL     string            `json:"url,omitempty"`     // for http/sse
	Headers map[string]string `json:"headers,omitempty"` // for http/sse
}

// Manager defines the interface for MCP client management
type Manager interface {
	// Configuration management
	LoadConfig(paths []string) error

	// Server lifecycle management
	StartServer(name string) error
	StopServer(name string) error
	RestartServer(name string) error
	StartAll() error
	StopAll() error

	// Status management
	GetServerStatus(name string) ServerStatus
	GetAllStatuses() map[string]ServerStatus

	// Tool/resource management
	ListTools() ([]ToolInfo, error)
	ListResources() ([]ResourceInfo, error)
	ListPrompts() ([]PromptInfo, error)

	// Tool execution
	ExecuteTool(serverName, toolName string, params map[string]interface{}) (interface{}, error)
}

// ServerStatus represents the current status of an MCP server
type ServerStatus struct {
	Name         string
	State        State
	Error        error
	StartedAt    time.Time
	Transport    string
	Capabilities ServerCapabilities
}

// State represents the current state of an MCP server
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

// ServerCapabilities represents the capabilities of an MCP server
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability represents tool-related capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability represents resource-related capabilities
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability represents prompt-related capabilities
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
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
	ServerName  string           `json:"serverName"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}
