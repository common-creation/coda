package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/common-creation/coda/internal/ai"
)

// TextToolCall represents a tool call extracted from text
type TextToolCall struct {
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}

// TextToolCallParser extracts tool calls from LLM text responses
type TextToolCallParser struct {
	// Pattern to match JSON tool calls in text
	toolCallPattern *regexp.Regexp
}

// NewTextToolCallParser creates a new text tool call parser
func NewTextToolCallParser() *TextToolCallParser {
	// Pattern to match JSON objects that look like tool calls
	// Matches: {"tool": "tool_name", "arguments": {...}}
	pattern := regexp.MustCompile(`\{"tool"\s*:\s*"[^"]+"\s*,\s*"arguments"\s*:\s*\{[^}]*\}\}`)
	
	return &TextToolCallParser{
		toolCallPattern: pattern,
	}
}

// ParseToolCalls extracts tool calls from text content
func (p *TextToolCallParser) ParseToolCalls(content string) ([]ai.ToolCall, error) {
	matches := p.toolCallPattern.FindAllString(content, -1)
	if len(matches) == 0 {
		return nil, nil // No tool calls found
	}

	var toolCalls []ai.ToolCall
	
	for i, match := range matches {
		// Parse the JSON
		var textCall TextToolCall
		if err := json.Unmarshal([]byte(match), &textCall); err != nil {
			// Skip invalid JSON but don't fail the entire parsing
			continue
		}

		// Convert to ai.ToolCall format
		argsJSON, err := json.Marshal(textCall.Arguments)
		if err != nil {
			continue
		}

		toolCall := ai.ToolCall{
			ID:    fmt.Sprintf("call_%d", i+1),
			Type:  "function",
			Index: i,
			Function: ai.FunctionCall{
				Name:      textCall.Tool,
				Arguments: string(argsJSON),
			},
		}

		toolCalls = append(toolCalls, toolCall)
	}

	return toolCalls, nil
}

// ExtractContentWithoutToolCalls removes tool call JSON from content and returns clean text
func (p *TextToolCallParser) ExtractContentWithoutToolCalls(content string) string {
	// Remove tool call JSON patterns from the content
	cleanContent := p.toolCallPattern.ReplaceAllString(content, "")
	
	// Clean up extra whitespace and newlines
	lines := strings.Split(cleanContent, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	return strings.Join(cleanLines, "\n")
}

// SplitMessages splits content by '\n----\n' separator for handling combined messages
func (p *TextToolCallParser) SplitMessages(content string) []string {
	// Split by the separator pattern
	separator := "\n----\n"
	parts := strings.Split(content, separator)
	
	// Trim each part and filter out empty ones
	var messages []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			messages = append(messages, trimmed)
		}
	}
	
	// If no separator was found, return the original content as a single message
	if len(messages) == 0 && strings.TrimSpace(content) != "" {
		messages = append(messages, strings.TrimSpace(content))
	}
	
	return messages
}

// ParseMessage processes a message that might contain both text and tool calls
// Returns the clean text content and any tool calls found
func (p *TextToolCallParser) ParseMessage(content string) (string, []ai.ToolCall, error) {
	// First, check if the message contains the separator
	messages := p.SplitMessages(content)
	
	var allToolCalls []ai.ToolCall
	var cleanTexts []string
	
	for _, msg := range messages {
		// Check if this message part is a tool call
		toolCalls, err := p.ParseToolCalls(msg)
		if err != nil {
			return "", nil, err
		}
		
		if len(toolCalls) > 0 {
			// This part contains tool calls
			allToolCalls = append(allToolCalls, toolCalls...)
		} else {
			// This part is regular text
			cleanTexts = append(cleanTexts, msg)
		}
	}
	
	// Join clean text parts
	cleanContent := strings.Join(cleanTexts, "\n\n")
	
	return cleanContent, allToolCalls, nil
}

// ValidateToolCall checks if a tool call is valid
func (p *TextToolCallParser) ValidateToolCall(toolCall TextToolCall) error {
	if toolCall.Tool == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	
	if toolCall.Arguments == nil {
		return fmt.Errorf("arguments cannot be nil")
	}
	
	// Add more validation as needed
	return nil
}

// ParseStreamingContent handles incremental parsing for streaming responses
type StreamingParser struct {
	*TextToolCallParser
	buffer          strings.Builder
	pendingToolCall *TextToolCall
	isInToolCall    bool
}

// NewStreamingParser creates a new streaming parser
func NewStreamingParser() *StreamingParser {
	return &StreamingParser{
		TextToolCallParser: NewTextToolCallParser(),
		buffer:             strings.Builder{},
	}
}

// AddChunk adds a chunk of text and returns any complete tool calls found
func (sp *StreamingParser) AddChunk(chunk string) ([]ai.ToolCall, string, error) {
	sp.buffer.WriteString(chunk)
	content := sp.buffer.String()
	
	// Try to extract tool calls from current buffer
	toolCalls, err := sp.ParseToolCalls(content)
	if err != nil {
		return nil, "", err
	}
	
	// Extract clean content (without tool calls)
	cleanContent := sp.ExtractContentWithoutToolCalls(content)
	
	// If we found tool calls, we need to determine what part of the buffer to keep
	if len(toolCalls) > 0 {
		// For now, reset the buffer after finding tool calls
		// In a more sophisticated implementation, we might keep partial JSON
		sp.buffer.Reset()
	}
	
	return toolCalls, cleanContent, nil
}

// Reset resets the streaming parser state
func (sp *StreamingParser) Reset() {
	sp.buffer.Reset()
	sp.pendingToolCall = nil
	sp.isInToolCall = false
}

// GetBufferedContent returns the current buffered content
func (sp *StreamingParser) GetBufferedContent() string {
	return sp.buffer.String()
}