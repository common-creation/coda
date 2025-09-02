// Package ai provides types and interfaces for AI service interactions.
package ai

import (
	"encoding/json"
	"time"
)

// Role constants define the different roles in a chat conversation.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleFunction  = "function" // Deprecated: use RoleTool instead
)

// Common model name constants for easy reference.
const (
	ModelGPT4          = "gpt-4"
	ModelGPT4Turbo     = "gpt-4-turbo"
	ModelGPT4Vision    = "gpt-4-vision-preview"
	ModelGPT35Turbo    = "gpt-3.5-turbo"
	ModelGPT35Turbo16k = "gpt-3.5-turbo-16k"
	ModelO3            = "o3"
	ModelGPT5          = "gpt-5"
)

// Default values for various parameters.
const (
	DefaultTemperature      float32 = 0.7
	DefaultMaxTokens        int     = 2048
	DefaultTopP             float32 = 1.0
	DefaultPresencePenalty  float32 = 0.0
	DefaultFrequencyPenalty float32 = 0.0
	DefaultTimeout                  = 30 * time.Second
)

// Message represents a chat message in a conversation.
// Messages can be from the system, user, assistant, or tools.
//
// Example user message:
//
//	Message{
//	    Role:    RoleUser,
//	    Content: "What is the weather like today?",
//	}
//
// Example assistant message with tool calls:
//
//	Message{
//	    Role:    RoleAssistant,
//	    Content: "I'll check the weather for you.",
//	    ToolCalls: []ToolCall{
//	        {
//	            ID:   "call_123",
//	            Type: "function",
//	            Function: FunctionCall{
//	                Name:      "get_weather",
//	                Arguments: `{"location": "Tokyo"}`,
//	            },
//	        },
//	    },
//	}
type Message struct {
	// Role of the message sender (system, user, assistant, tool)
	Role string `json:"role"`

	// Content of the message
	Content string `json:"content"`

	// Name of the message sender (optional)
	Name string `json:"name,omitempty"`

	// Tool calls made by the assistant
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Tool call ID this message is responding to (for tool role messages)
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ChatRequest represents a request to generate a chat completion.
//
// Example:
//
//	req := ChatRequest{
//	    Model: ModelGPT4,
//	    Messages: []Message{
//	        {Role: RoleSystem, Content: "You are a helpful assistant."},
//	        {Role: RoleUser, Content: "Hello!"},
//	    },
//	    Temperature: FloatPtr(0.8),
//	    MaxTokens:   IntPtr(1000),
//	}
type ChatRequest struct {
	// Model ID to use for completion
	Model string `json:"model"`

	// Messages in the conversation
	Messages []Message `json:"messages"`

	// Sampling temperature (0-2)
	Temperature *float32 `json:"temperature,omitempty"`

	// Maximum tokens to generate
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Whether to stream the response
	Stream bool `json:"stream,omitempty"`

	// NOTE: Tools and ToolChoice removed for text-based tool calling
	// Tools are now described in the system prompt instead

	// Response format specification
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Random seed for deterministic generation
	Seed *int `json:"seed,omitempty"`

	// Stop sequences
	Stop []string `json:"stop,omitempty"`

	// Presence penalty (-2.0 to 2.0)
	PresencePenalty *float32 `json:"presence_penalty,omitempty"`

	// Frequency penalty (-2.0 to 2.0)
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`

	// Top-p sampling parameter
	TopP *float32 `json:"top_p,omitempty"`

	// User identifier for tracking
	User string `json:"user,omitempty"`

	// Additional metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// Reasoning effort for GPT-5 models (optional)
	// Valid values: "minimal", "low", "medium", "high"
	ReasoningEffort *string `json:"reasoning_effort,omitempty"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	// Unique identifier for the completion
	ID string `json:"id"`

	// Object type (always "chat.completion")
	Object string `json:"object"`

	// Unix timestamp of creation
	Created int64 `json:"created"`

	// Model used for completion
	Model string `json:"model"`

	// Completion choices
	Choices []Choice `json:"choices"`

	// Token usage statistics
	Usage Usage `json:"usage"`

	// System fingerprint for reproducibility
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	// Index of this choice
	Index int `json:"index"`

	// Generated message
	Message Message `json:"message"`

	// Reason for completion termination
	// Can be: "stop", "length", "tool_calls", "content_filter", "function_call"
	FinishReason string `json:"finish_reason"`

	// Log probabilities (if requested)
	LogProbs interface{} `json:"logprobs,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	// Tokens used in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// Tokens used in the completion
	CompletionTokens int `json:"completion_tokens"`

	// Total tokens used
	TotalTokens int `json:"total_tokens"`
}

// Model represents an available AI model.
type Model struct {
	// Model identifier
	ID string `json:"id"`

	// Object type (always "model")
	Object string `json:"object"`

	// Unix timestamp of model creation
	Created int64 `json:"created"`

	// Organization that owns the model
	OwnedBy string `json:"owned_by"`

	// Permissions for the model
	Permission []string `json:"permission,omitempty"`

	// Root model this was fine-tuned from
	Root string `json:"root,omitempty"`

	// Parent model
	Parent string `json:"parent,omitempty"`
}

// Tool represents a function tool that can be called by the AI.
//
// Example:
//
//	Tool{
//	    Type: "function",
//	    Function: FunctionTool{
//	        Name:        "get_weather",
//	        Description: "Get the current weather for a location",
//	        Parameters: map[string]interface{}{
//	            "type": "object",
//	            "properties": map[string]interface{}{
//	                "location": map[string]interface{}{
//	                    "type":        "string",
//	                    "description": "The city name",
//	                },
//	            },
//	            "required": []string{"location"},
//	        },
//	    },
//	}
type Tool struct {
	// Tool type (currently only "function" is supported)
	Type string `json:"type"`

	// Function definition
	Function FunctionTool `json:"function"`
}

// FunctionTool represents a callable function.
type FunctionTool struct {
	// Function name
	Name string `json:"name"`

	// Human-readable description
	Description string `json:"description"`

	// JSON Schema for function parameters
	Parameters interface{} `json:"parameters"`
}

// ToolCall represents a tool call made by the AI.
type ToolCall struct {
	// Index of the tool call in the list
	Index int `json:"index,omitempty"`

	// Unique identifier for this tool call
	ID string `json:"id"`

	// Type of tool call (currently only "function")
	Type string `json:"type"`

	// Function call details
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a specific function invocation.
type FunctionCall struct {
	// Name of the function to call
	Name string `json:"name"`

	// JSON-encoded arguments for the function
	Arguments string `json:"arguments"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	// Response type: "text", "json_object", or "json_schema"
	Type string `json:"type"`
	// JSONSchema for structured outputs (when Type is "json_schema")
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema defines the structure for Structured Outputs
type JSONSchema struct {
	// Name of the schema (required for Structured Outputs)
	Name string `json:"name"`
	// Description of the schema (optional)
	Description string `json:"description,omitempty"`
	// The JSON Schema definition
	Schema json.RawMessage `json:"schema"`
	// Strict mode ensures the model always follows the schema
	Strict bool `json:"strict"`
}

// StreamReader defines the interface for reading streaming responses.
type StreamReader interface {
	// Read returns the next chunk from the stream.
	// Returns io.EOF when the stream is complete.
	Read() (*StreamChunk, error)

	// Close releases any resources associated with the stream.
	Close() error
}

// StreamChunk represents a chunk of streaming response.
type StreamChunk struct {
	// Unique identifier for the completion
	ID string `json:"id"`

	// Object type (always "chat.completion.chunk")
	Object string `json:"object"`

	// Unix timestamp of creation
	Created int64 `json:"created"`

	// Model used for completion
	Model string `json:"model"`

	// Streaming choices
	Choices []StreamChoice `json:"choices"`

	// System fingerprint for reproducibility
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// StreamChoice represents a streaming choice.
type StreamChoice struct {
	// Index of this choice
	Index int `json:"index"`

	// Delta content
	Delta StreamDelta `json:"delta"`

	// Reason for completion termination (only present in final chunk)
	FinishReason *string `json:"finish_reason,omitempty"`

	// Log probabilities (if requested)
	LogProbs interface{} `json:"logprobs,omitempty"`
}

// StreamDelta represents the delta content in a stream.
type StreamDelta struct {
	// Role of the message (only in first chunk)
	Role string `json:"role,omitempty"`

	// Content delta
	Content string `json:"content,omitempty"`

	// Tool calls delta
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// FunctionDefinition defines a function that can be called by the AI
type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Helper functions for creating pointers to basic types

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int.
func IntPtr(i int) *int {
	return &i
}

// FloatPtr returns a pointer to the given float32.
func FloatPtr(f float32) *float32 {
	return &f
}

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool {
	return &b
}
