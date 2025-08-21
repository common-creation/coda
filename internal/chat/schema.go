package chat

import (
	"encoding/json"
	"fmt"
	
	"github.com/common-creation/coda/internal/ai"
)

// ToolCallSchema defines the JSON schema for structured tool calls
// This schema ensures the model always returns a well-formed response
const ToolCallSchemaJSON = `{
	"type": "object",
	"properties": {
		"response_type": {
			"type": "string",
			"description": "Type of response: text for normal responses, tool_call for tool invocations, both for mixed",
			"enum": ["text", "tool_call", "both"]
		},
		"text": {
			"type": ["string", "null"],
			"description": "The text content of the response (null when response_type is tool_call)"
		},
		"tool_calls": {
			"type": "array",
			"description": "List of tool calls to execute",
			"items": {
				"type": "object",
				"properties": {
					"tool": {
						"type": "string",
						"description": "Name of the tool to invoke"
					},
					"arguments": {
						"type": "object",
						"description": "Arguments to pass to the tool",
						"additionalProperties": true
					}
				},
				"required": ["tool", "arguments"],
				"additionalProperties": false
			}
		}
	},
	"required": ["response_type", "text", "tool_calls"],
	"additionalProperties": false
}`

// ToolResponse represents the structured response from the AI model
type ToolResponse struct {
	ResponseType string     `json:"response_type"`
	Text         *string    `json:"text"`
	ToolCalls    []ToolCall `json:"tool_calls"`
}

// ToolCall represents a single tool invocation
type ToolCall struct {
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ParseStructuredOutput parses a JSON string into a ToolResponse
func ParseStructuredOutput(jsonStr string) (*ToolResponse, error) {
	var resp ToolResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ConvertToAIToolCalls converts structured tool calls to AI package format
func ConvertToAIToolCalls(toolCalls []ToolCall) ([]ai.ToolCall, error) {
	var aiToolCalls []ai.ToolCall
	
	for i, tc := range toolCalls {
		argsJSON, err := json.Marshal(tc.Arguments)
		if err != nil {
			return nil, err
		}
		
		aiToolCall := ai.ToolCall{
			ID:    fmt.Sprintf("call_%d", i+1),
			Type:  "function",
			Index: i,
			Function: ai.FunctionCall{
				Name:      tc.Tool,
				Arguments: string(argsJSON),
			},
		}
		aiToolCalls = append(aiToolCalls, aiToolCall)
	}
	
	return aiToolCalls, nil
}

// GetToolCallSchema returns the JSON schema for tool calls as raw message
func GetToolCallSchema() json.RawMessage {
	return json.RawMessage(ToolCallSchemaJSON)
}