package tools

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages tool registration, discovery, and execution
type Manager struct {
	tools    map[string]Tool
	mu       sync.RWMutex
	security SecurityValidator
	logger   Logger
}

// NewManager creates a new tool manager instance
func NewManager(validator SecurityValidator, logger Logger) *Manager {
	return &Manager{
		tools:    make(map[string]Tool),
		security: validator,
		logger:   logger,
	}
}

// Register adds a new tool to the manager
func (m *Manager) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[name]; exists {
		return fmt.Errorf("tool '%s' is already registered", name)
	}

	m.tools[name] = tool
	if m.logger != nil {
		m.logger.Info("Registered tool", "name", name)
	}

	return nil
}

// Get retrieves a tool by name
func (m *Manager) Get(name string) (Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool, nil
}

// Execute runs a tool with the given parameters
func (m *Manager) Execute(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, err := m.Get(name)
	if err != nil {
		return nil, err
	}

	// Log execution start
	if m.logger != nil {
		m.logger.Debug("Executing tool", "name", name, "params", params)
	}

	// Validate parameters
	if err := tool.Validate(params); err != nil {
		if m.logger != nil {
			m.logger.Error("Tool validation failed", "name", name, "error", err)
		}
		return nil, fmt.Errorf("validation failed for tool '%s': %w", name, err)
	}

	// Execute the tool
	result, err := tool.Execute(ctx, params)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("Tool execution failed", "name", name, "error", err)
		}
		return nil, fmt.Errorf("execution failed for tool '%s': %w", name, err)
	}

	// Log execution success
	if m.logger != nil {
		m.logger.Debug("Tool executed successfully", "name", name)
	}

	return result, nil
}

// List returns all registered tool names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}

	return names
}

// GetSchema returns the schema for a specific tool
func (m *Manager) GetSchema(name string) (ToolSchema, error) {
	tool, err := m.Get(name)
	if err != nil {
		return ToolSchema{}, err
	}

	return tool.Schema(), nil
}

// GetAllSchemas returns schemas for all registered tools
func (m *Manager) GetAllSchemas() map[string]ToolSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()

	schemas := make(map[string]ToolSchema)
	for name, tool := range m.tools {
		schemas[name] = tool.Schema()
	}

	return schemas
}

// Unregister removes a tool from the manager
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(m.tools, name)
	if m.logger != nil {
		m.logger.Info("Unregistered tool", "name", name)
	}

	return nil
}

// SetSecurityValidator updates the security validator
func (m *Manager) SetSecurityValidator(validator SecurityValidator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.security = validator
}

// GetSecurityValidator returns the current security validator
func (m *Manager) GetSecurityValidator() SecurityValidator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.security
}

// GetAll returns all registered tools
func (m *Manager) GetAll() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}

	return tools
}
