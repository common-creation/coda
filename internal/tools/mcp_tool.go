package tools

import (
	"context"
	"fmt"
	"strings"
)

// MCPTool wraps an MCP server tool to implement the CODA Tool interface
type MCPTool struct {
	serverName string
	toolName   string
	toolInfo   ToolInfo
	manager    MCPManager
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(serverName string, toolInfo ToolInfo, manager MCPManager) *MCPTool {
	return &MCPTool{
		serverName: serverName,
		toolName:   toolInfo.Name,
		toolInfo:   toolInfo,
		manager:    manager,
	}
}

// Name returns the tool name with MCP server prefix
func (t *MCPTool) Name() string {
	return fmt.Sprintf("mcp_%s_%s", t.serverName, t.toolName)
}

// Description returns the tool description from the MCP server
func (t *MCPTool) Description() string {
	description := t.toolInfo.Description
	if description == "" {
		description = fmt.Sprintf("MCP tool %s from server %s", t.toolName, t.serverName)
	}

	// Add information about the source server
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, description)
}

// Schema converts MCP tool input schema to CODA ToolSchema format
func (t *MCPTool) Schema() ToolSchema {
	schema := ToolSchema{
		Type:       "object",
		Properties: make(map[string]Property),
		Required:   []string{},
	}

	if t.toolInfo.InputSchema == nil {
		// Return empty schema if no input schema provided
		return schema
	}

	// Convert MCP JSON schema to CODA ToolSchema
	if schemaType, ok := t.toolInfo.InputSchema["type"].(string); ok {
		schema.Type = schemaType
	}

	// Convert properties
	if propertiesRaw, ok := t.toolInfo.InputSchema["properties"]; ok {
		if properties, ok := propertiesRaw.(map[string]interface{}); ok {
			for propName, propData := range properties {
				if propMap, ok := propData.(map[string]interface{}); ok {
					property := t.convertProperty(propMap)
					schema.Properties[propName] = property
				}
			}
		}
	}

	// Convert required fields
	if requiredRaw, ok := t.toolInfo.InputSchema["required"]; ok {
		if requiredSlice, ok := requiredRaw.([]interface{}); ok {
			for _, req := range requiredSlice {
				if reqStr, ok := req.(string); ok {
					schema.Required = append(schema.Required, reqStr)
				}
			}
		}
	}

	return schema
}

// Execute runs the MCP tool via the manager
func (t *MCPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate tool is available
	if err := t.validateToolAvailability(); err != nil {
		return nil, fmt.Errorf("MCP tool validation failed: %w", err)
	}

	// Execute the tool via MCP manager
	result, err := t.manager.ExecuteTool(t.serverName, t.toolName, params)
	if err != nil {
		return nil, fmt.Errorf("MCP tool execution failed: %w", err)
	}

	// Wrap result in CODA-compatible format
	return map[string]interface{}{
		"server":  t.serverName,
		"tool":    t.toolName,
		"result":  result,
		"success": true,
	}, nil
}

// Validate checks parameters against the MCP tool schema
func (t *MCPTool) Validate(params map[string]interface{}) error {
	// Basic validation against the schema
	schema := t.Schema()

	// Check required parameters
	for _, required := range schema.Required {
		if _, exists := params[required]; !exists {
			return fmt.Errorf("required parameter '%s' is missing", required)
		}
	}

	// Validate parameter types
	for paramName, paramValue := range params {
		property, exists := schema.Properties[paramName]
		if !exists {
			// Allow unknown parameters for now (MCP servers might be flexible)
			continue
		}

		if err := t.validateParameterType(paramName, paramValue, property); err != nil {
			return fmt.Errorf("parameter validation failed for '%s': %w", paramName, err)
		}
	}

	return nil
}

// convertProperty converts an MCP JSON schema property to CODA Property format
func (t *MCPTool) convertProperty(propMap map[string]interface{}) Property {
	property := Property{}

	if propType, ok := propMap["type"].(string); ok {
		property.Type = propType
	}

	if desc, ok := propMap["description"].(string); ok {
		property.Description = desc
	}

	if defaultVal, ok := propMap["default"]; ok {
		property.Default = defaultVal
	}

	if enumRaw, ok := propMap["enum"]; ok {
		if enumSlice, ok := enumRaw.([]interface{}); ok {
			property.Enum = make([]string, 0, len(enumSlice))
			for _, enumVal := range enumSlice {
				if enumStr, ok := enumVal.(string); ok {
					property.Enum = append(property.Enum, enumStr)
				}
			}
		}
	}

	// Handle array items
	if itemsRaw, ok := propMap["items"]; ok {
		if itemsMap, ok := itemsRaw.(map[string]interface{}); ok {
			items := t.convertProperty(itemsMap)
			property.Items = &items
		}
	}

	// Handle nested object properties
	if propertiesRaw, ok := propMap["properties"]; ok {
		if propertiesMap, ok := propertiesRaw.(map[string]interface{}); ok {
			property.Properties = make(map[string]Property)
			for nestedName, nestedProp := range propertiesMap {
				if nestedMap, ok := nestedProp.(map[string]interface{}); ok {
					property.Properties[nestedName] = t.convertProperty(nestedMap)
				}
			}
		}
	}

	return property
}

// validateParameterType validates a parameter against its expected type
func (t *MCPTool) validateParameterType(name string, value interface{}, property Property) error {
	switch property.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case int, int64, float64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "integer":
		switch value.(type) {
		case int, int64:
			// Valid integer types
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	// Validate enum values
	if len(property.Enum) > 0 {
		valueStr := fmt.Sprintf("%v", value)
		for _, enumVal := range property.Enum {
			if enumVal == valueStr {
				return nil
			}
		}
		return fmt.Errorf("value '%s' is not in allowed enum values: %v", valueStr, property.Enum)
	}

	return nil
}

// validateToolAvailability checks if the MCP tool is still available
func (t *MCPTool) validateToolAvailability() error {
	// Check if server is running
	status := t.manager.GetServerStatus(t.serverName)
	if status.State != StateRunning {
		return fmt.Errorf("MCP server '%s' is not running (state: %s)", t.serverName, status.State)
	}

	// Optionally, we could check if the tool is still available via ListTools,
	// but that might be expensive to call on every execution
	return nil
}

// GetServerName returns the name of the MCP server that provides this tool
func (t *MCPTool) GetServerName() string {
	return t.serverName
}

// GetToolName returns the original MCP tool name (without server prefix)
func (t *MCPTool) GetToolName() string {
	return t.toolName
}

// IsFromMCPServer returns true if this is an MCP tool
func (t *MCPTool) IsFromMCPServer() bool {
	return true
}

// GetMCPToolInfo returns the original MCP tool information
func (t *MCPTool) GetMCPToolInfo() ToolInfo {
	return t.toolInfo
}

// IsMCPTool checks if a tool name represents an MCP tool
func IsMCPTool(toolName string) bool {
	return strings.HasPrefix(toolName, "mcp_")
}

// ParseMCPToolName extracts server name and tool name from an MCP tool name
func ParseMCPToolName(toolName string) (serverName, originalToolName string, ok bool) {
	if !IsMCPTool(toolName) {
		return "", "", false
	}

	// Remove "mcp_" prefix and split by "_"
	trimmed := strings.TrimPrefix(toolName, "mcp_")
	parts := strings.SplitN(trimmed, "_", 2)

	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
