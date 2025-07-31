package tools

import (
	"context"
	"time"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the tool name
	Name() string
	// Description returns the tool description
	Description() string
	// Schema returns the parameter schema (JSON Schema format)
	Schema() ToolSchema
	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	// Validate checks parameters before execution (optional)
	Validate(params map[string]interface{}) error
}

// ToolSchema defines the JSON Schema structure for tool parameters
type ToolSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

// Property defines a single property in the schema
type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Default     interface{}         `json:"default,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Items       *Property           `json:"items,omitempty"`      // For array types
	Properties  map[string]Property `json:"properties,omitempty"` // For object types
}

// ExecutionContext provides context for tool execution
type ExecutionContext struct {
	WorkingDir string
	User       string
	Timeout    time.Duration
	Logger     Logger
}

// Logger interface for logging tool execution
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Result represents the result of a tool execution
type Result struct {
	Success bool
	Data    interface{}
	Error   error
	Logs    []string
}

// Operation represents a file operation type
type Operation string

const (
	OpRead    Operation = "read"
	OpWrite   Operation = "write"
	OpDelete  Operation = "delete"
	OpExecute Operation = "execute"
	OpList    Operation = "list"
)

// SecurityValidator defines the interface for security validation
type SecurityValidator interface {
	ValidatePath(path string) error
	ValidateOperation(op Operation, path string) error
	IsAllowedExtension(path string) bool
	CheckContent(content []byte) error
}
