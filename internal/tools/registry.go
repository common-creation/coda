package tools

import (
	"fmt"
	"sync"
)

// ToolFactory is a function that creates a new instance of a Tool
type ToolFactory func() Tool

// Registry manages tool factories and categories
type Registry struct {
	tools      map[string]ToolFactory
	categories map[string][]string
	mu         sync.RWMutex
}

// DefaultRegistry is the global tool registry
var DefaultRegistry = NewRegistry()

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:      make(map[string]ToolFactory),
		categories: make(map[string][]string),
	}
}

// RegisterFactory registers a tool factory with the given name
func (r *Registry) RegisterFactory(name string, factory ToolFactory) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("tool factory cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' is already registered", name)
	}

	r.tools[name] = factory
	return nil
}

// RegisterCategory associates tool names with a category
func (r *Registry) RegisterCategory(category string, toolNames []string) error {
	if category == "" {
		return fmt.Errorf("category name cannot be empty")
	}
	if len(toolNames) == 0 {
		return fmt.Errorf("tool names cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate that all tools exist
	for _, name := range toolNames {
		if _, exists := r.tools[name]; !exists {
			return fmt.Errorf("tool '%s' not found in registry", name)
		}
	}

	r.categories[category] = toolNames
	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(r.tools, name)

	// Remove from categories
	for category, tools := range r.categories {
		filtered := make([]string, 0, len(tools))
		for _, toolName := range tools {
			if toolName != name {
				filtered = append(filtered, toolName)
			}
		}
		if len(filtered) == 0 {
			delete(r.categories, category)
		} else {
			r.categories[category] = filtered
		}
	}

	return nil
}

// GetByCategory returns all tools in the specified category
func (r *Registry) GetByCategory(category string) ([]Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolNames, exists := r.categories[category]
	if !exists {
		return nil, fmt.Errorf("category '%s' not found", category)
	}

	tools := make([]Tool, 0, len(toolNames))
	for _, name := range toolNames {
		factory, exists := r.tools[name]
		if !exists {
			continue // Skip if tool was unregistered
		}
		tools = append(tools, factory())
	}

	return tools, nil
}

// CreateTool creates a new instance of the specified tool
func (r *Registry) CreateTool(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return factory(), nil
}

// ListTools returns all registered tool names
func (r *Registry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// ListCategories returns all registered category names
func (r *Registry) ListCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make([]string, 0, len(r.categories))
	for category := range r.categories {
		categories = append(categories, category)
	}

	return categories
}

// GetToolsInCategory returns the tool names in a specific category
func (r *Registry) GetToolsInCategory(category string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools, exists := r.categories[category]
	if !exists {
		return nil, fmt.Errorf("category '%s' not found", category)
	}

	// Return a copy to prevent external modification
	result := make([]string, len(tools))
	copy(result, tools)

	return result, nil
}

// HasTool checks if a tool is registered
func (r *Registry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// Clear removes all tools and categories from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]ToolFactory)
	r.categories = make(map[string][]string)
}

// RegisterFactoryGlobal registers a tool factory in the default registry
func RegisterFactoryGlobal(name string, factory ToolFactory) error {
	return DefaultRegistry.RegisterFactory(name, factory)
}

// RegisterCategoryGlobal registers a category in the default registry
func RegisterCategoryGlobal(category string, toolNames []string) error {
	return DefaultRegistry.RegisterCategory(category, toolNames)
}
