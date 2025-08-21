package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/mcp"
	"github.com/common-creation/coda/internal/tokenizer"
	"github.com/common-creation/coda/internal/tools"
)

// ChatHandler manages the chat interaction flow
type ChatHandler struct {
	aiClient      ai.Client
	toolManager   *tools.Manager
	mcpManager    mcp.Manager
	session       *SessionManager
	config        *config.Config
	history       *History
	promptBuilder *PromptBuilder
	persistence   *FilePersistence

	// Streaming state
	streamingTokens int
	streamingMutex  sync.Mutex
}

// ChatResponse represents a response from the chat handler
type ChatResponse struct {
	Content         string
	TokenCount      int // Total token count (deprecated, use TokenUsage.TotalTokens)
	ToolCalls       []ai.ToolCall
	TokenUsage      *ai.Usage // Detailed token usage from AI response
	EstimatedPrompt int       // Estimated prompt tokens (before sending)
}

// NewChatHandler creates a new chat handler
func NewChatHandler(aiClient ai.Client, toolManager *tools.Manager, mcpManager mcp.Manager, session *SessionManager, cfg *config.Config, history *History) *ChatHandler {
	// Create a better token counter with the model from config
	betterCounter := NewBetterTokenCounter(cfg.AI.Model)

	// Update session manager to use the better token counter
	session.SetTokenCounter(betterCounter)

	// Initialize prompt builder with better token counter
	promptBuilder := NewPromptBuilder(4000, betterCounter)

	// Add tool information to prompt builder
	if toolManager != nil {
		tools := toolManager.GetAll()
		for _, tool := range tools {
			promptBuilder.AddToolPrompt(tool.Name(), tool.Description())
		}
	}

	// Add MCP tools to prompt builder
	if mcpManager != nil {
		mcpTools, err := mcpManager.ListTools()
		if err == nil {
			for _, tool := range mcpTools {
				promptBuilder.AddToolPrompt(tool.Name, tool.Description)
			}
		}
	}

	handler := &ChatHandler{
		aiClient:      aiClient,
		toolManager:   toolManager,
		mcpManager:    mcpManager,
		session:       session,
		config:        cfg,
		history:       history,
		promptBuilder: promptBuilder,
	}

	// Initialize persistence for auto-save
	sessionPath, err := GetProjectSessionPath()
	if err == nil {
		persistence, err := NewFilePersistence(sessionPath, true, 1*time.Minute)
		if err == nil {
			handler.persistence = persistence
		}
	}

	return handler
}

// HandleMessageWithResponse processes a user message and returns the response for TUI mode
func (h *ChatHandler) HandleMessageWithResponse(ctx context.Context, input string, tokenCallback func(int)) (*ChatResponse, error) {
	// Trim and validate input
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Handle special commands (TUI should handle these differently)
	if strings.HasPrefix(input, "/") {
		return &ChatResponse{
			Content: fmt.Sprintf("Command %s should be handled by TUI", input),
		}, nil
	}

	// Get or create current session
	currentSession := h.session.GetCurrent()
	if currentSession == nil {
		sessionID, err := h.session.CreateSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		currentSession, _ = h.session.GetSession(sessionID)
	}

	// Add user message to session
	userMessage := ai.Message{
		Role:    ai.RoleUser,
		Content: input,
	}

	if err := h.session.AddMessage(currentSession.ID, userMessage); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// Build messages for AI request
	messages := h.buildMessages(currentSession)

	// Create chat request with streaming enabled
	req := ai.ChatRequest{
		Model:           h.config.AI.Model,
		Messages:        messages,
		Temperature:     &h.config.AI.Temperature,
		MaxTokens:       &h.config.AI.MaxTokens,
		Stream:          true, // Enable streaming
		ReasoningEffort: h.config.AI.ReasoningEffort,
	}
	
	// Enable Structured Outputs if configured
	if h.config.AI.UseStructuredOutputs {
		req.ResponseFormat = &ai.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &ai.JSONSchema{
				Name:        "tool_response",
				Description: "Structured response with optional tool calls",
				Schema:      GetToolCallSchema(),
				Strict:      true,
			},
		}
	}

	// Send request to AI with streaming
	stream, err := h.aiClient.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat stream: %w", err)
	}
	defer stream.Close()

	// Process streaming response
	var fullContent strings.Builder
	var toolCalls []ai.ToolCall
	var totalUsage ai.Usage
	
	// Use structured output parser if enabled, otherwise use text parser
	useStructuredOutputs := h.config.AI.UseStructuredOutputs
	textParser := NewTextToolCallParser() // Still needed as fallback

	// Reset streaming tokens at start
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()

	// Debug logging
	debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[ChatHandler] Starting streaming response processing with text parser\n")
		debugFile.Close()
	}

	chunkCount := 0
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			// Debug logging
			debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[ChatHandler] Stream ended, totalChunks: %d\n", chunkCount)
				debugFile.Close()
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		chunkCount++

		// Process chunk
		if chunk.Choices != nil && len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			// Handle content
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)

				// Parse based on mode
				contentStr := fullContent.String()
				
				if useStructuredOutputs {
					// Try to parse as structured JSON output
					if toolResp, err := ParseStructuredOutput(contentStr); err == nil {
						// Successfully parsed structured output
						if len(toolResp.ToolCalls) > 0 {
							// Convert structured tool calls to AI format
							if aiToolCalls, err := ConvertToAIToolCalls(toolResp.ToolCalls); err == nil {
								toolCalls = aiToolCalls
							}
						}
					}
				} else {
					// Fallback to text-based parsing
					_, parsedToolCalls, err := textParser.ParseMessage(contentStr)
					if err == nil && len(parsedToolCalls) > 0 {
						// Replace any existing tool calls with newly parsed ones
						toolCalls = parsedToolCalls
					}
				}

				// Calculate tokens for current content using tokenizer
				estimatedTokens := 0

				// Use tokenizer for accurate token counting
				if len(contentStr) > 0 {
					// Calculate tokens using the tokenizer package
					tokens, err := tokenizer.EstimateUserMessageTokens(contentStr, h.config.AI.Model)
					if err != nil {
						// Fallback to simple estimation
						runeCount := len([]rune(contentStr))
						estimatedTokens = runeCount / 4
					} else {
						estimatedTokens = tokens
					}

					// Debug logging
					debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
					if debugFile != nil {
						fmt.Fprintf(debugFile, "[ChatHandler] Token estimation: contentLen=%d, estimatedTokens=%d, toolCalls=%d\n", len(contentStr), estimatedTokens, len(toolCalls))
						debugFile.Close()
					}
				}

				// Update ChatHandler's streaming tokens
				h.streamingMutex.Lock()
				h.streamingTokens = estimatedTokens
				h.streamingMutex.Unlock()

				// Debug logging
				debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if debugFile != nil {
					fmt.Fprintf(debugFile, "[ChatHandler] Token update: chunk: %d, deltaContent: %q, totalLen: %d, tokens: %d\n",
						chunkCount, delta.Content, fullContent.Len(), estimatedTokens)
					debugFile.Close()
				}

				// Call the callback if provided
				if tokenCallback != nil {
					tokenCallback(estimatedTokens)
				}
			}

			// Note: delta.ToolCalls will be empty since we're not using structured tool calling
		}

		// Note: Usage information is typically not available in streaming chunks
		// It will be estimated after streaming completes
	}

	// Reset streaming tokens after streaming completes
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()

	// Debug: Log complete response JSON if debug mode is enabled
	if h.config.Logging.Level == "debug" {
		debugFile, err := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil && debugFile != nil {
			defer debugFile.Close()
			
			// Create a complete response structure for debugging
			responseDebug := map[string]interface{}{
				"timestamp":       time.Now().Format(time.RFC3339),
				"model":           h.config.AI.Model,
				"full_content":    fullContent.String(),
				"content_length":  fullContent.Len(),
				"tool_calls_count": len(toolCalls),
				"chunk_count":     chunkCount,
				"usage": map[string]int{
					"prompt_tokens":     totalUsage.PromptTokens,
					"completion_tokens": totalUsage.CompletionTokens,
					"total_tokens":      totalUsage.TotalTokens,
				},
			}
			
			// Add tool calls if present
			if len(toolCalls) > 0 {
				toolCallsDebug := make([]map[string]interface{}, len(toolCalls))
				for i, tc := range toolCalls {
					toolCallsDebug[i] = map[string]interface{}{
						"id":   tc.ID,
						"type": tc.Type,
						"function": map[string]string{
							"name":      tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					}
				}
				responseDebug["tool_calls"] = toolCallsDebug
			}
			
			// Marshal to JSON and write as single line
			if jsonData, err := json.Marshal(responseDebug); err == nil {
				fmt.Fprintf(debugFile, "[ChatHandler] COMPLETE_RESPONSE_JSON: %s\n", string(jsonData))
			}
		}
	}

	// Parse final message based on mode
	var cleanContent string
	contentStr := fullContent.String()
	
	if useStructuredOutputs {
		// Parse structured JSON output
		if toolResp, err := ParseStructuredOutput(contentStr); err == nil {
			// Successfully parsed structured output
			if toolResp.Text != nil {
				cleanContent = *toolResp.Text
			}
			if len(toolResp.ToolCalls) > 0 {
				// Convert structured tool calls to AI format
				if aiToolCalls, err := ConvertToAIToolCalls(toolResp.ToolCalls); err == nil {
					toolCalls = aiToolCalls
				}
			}
		} else {
			// If parsing fails, use raw content
			cleanContent = contentStr
		}
	} else {
		// Use text parser for final extraction
		parsedContent, finalToolCalls, _ := textParser.ParseMessage(contentStr)
		cleanContent = parsedContent
		if len(finalToolCalls) > 0 {
			toolCalls = finalToolCalls
		}
	}

	// Create final message
	message := ai.Message{
		Role:      ai.RoleAssistant,
		Content:   cleanContent,
		ToolCalls: toolCalls,
	}

	// Add assistant message to session
	if err := h.session.AddMessage(currentSession.ID, message); err != nil {
		return nil, fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Auto-save session after each message
	if h.persistence != nil {
		if session := h.session.GetCurrent(); session != nil {
			if err := h.persistence.SaveSession(session); err != nil {
				// Log error but don't fail the operation
				// In TUI mode, we should handle this differently
			}
		}
	}

	// Process tool calls if any (TUI should handle this asynchronously)
	if len(toolCalls) > 0 {
		// For now, just include a note about tool calls
		toolCallInfo := fmt.Sprintf("[Tool calls requested: %d]", len(toolCalls))
		message.Content += toolCallInfo
	}

	// If usage wasn't provided in stream, estimate it
	if totalUsage.TotalTokens == 0 {
		// Use tokenizer for accurate token counting
		tokens, err := tokenizer.EstimateUserMessageTokens(fullContent.String(), h.config.AI.Model)
		if err != nil {
			// Fallback to simple estimation
			totalUsage.CompletionTokens = fullContent.Len() / 4
		} else {
			totalUsage.CompletionTokens = tokens
		}
		totalUsage.TotalTokens = totalUsage.CompletionTokens
	}

	return &ChatResponse{
		Content:    message.Content,
		TokenCount: totalUsage.TotalTokens,
		ToolCalls:  toolCalls,
		TokenUsage: &totalUsage,
		// EstimatedPrompt will be set by the UI layer using tiktoken
	}, nil
}

// buildMessages constructs the message list for the AI request
func (h *ChatHandler) buildMessages(session *Session) []ai.Message {
	messages := make([]ai.Message, 0, len(session.Messages)+1)

	// Build system prompt using PromptBuilder
	systemPrompt, err := h.promptBuilder.Build()
	if err != nil {
		// Fallback to basic prompt if building fails
		systemPrompt = "You are CODA (CODing Agent), an AI assistant designed to help developers with coding tasks."
	}

	// Load workspace-specific prompt from CLAUDE.md if exists
	workspacePrompt := h.loadWorkspacePrompt()
	if workspacePrompt != "" {
		systemPrompt += "\n\n## Workspace-Specific Instructions\n" + workspacePrompt
	}

	// Debug: Log system prompt to file
	debugFile, _ := os.OpenFile("/tmp/coda-system-prompt.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "=== SYSTEM PROMPT ===\n%s\n", systemPrompt)
		debugFile.Close()
	}

	// Add system prompt
	messages = append(messages, ai.Message{
		Role:    ai.RoleSystem,
		Content: systemPrompt,
	})

	// Add conversation history with null content check
	for _, msg := range session.Messages {
		// Ensure content is never null
		if msg.Content == "" {
			msg.Content = "[Empty message]"
		}
		messages = append(messages, msg)
	}

	return messages
}

// loadWorkspacePrompt loads CLAUDE.md from the current workspace root
func (h *ChatHandler) loadWorkspacePrompt() string {
	// Try to find and read CLAUDE.md from the current directory
	claudePath := "CLAUDE.md"
	if content, err := os.ReadFile(claudePath); err == nil {
		return string(content)
	}

	// Try to find CLAUDE.md from the working directory
	if wd, err := os.Getwd(); err == nil {
		claudePath = filepath.Join(wd, "CLAUDE.md")
		if content, err := os.ReadFile(claudePath); err == nil {
			return string(content)
		}
	}

	return ""
}

// NOTE: getToolDefinitions method removed - tool definitions are now included in system prompt

// processToolCalls handles tool execution requests
func (h *ChatHandler) processToolCalls(ctx context.Context, sessionID string, toolCalls []ai.ToolCall) error {
	// This will be implemented in task-033
	// For now, just log the tool calls
	for _, call := range toolCalls {
		fmt.Printf("\n[Tool Call] %s: %s\n", call.Function.Name, call.Function.Arguments)
	}
	return nil
}

// SetSystemPrompt allows updating the system prompt
func (h *ChatHandler) SetSystemPrompt(prompt string) {
	h.promptBuilder.AddCustomPrompt("user_system_prompt", prompt)
}

// GetSystemPrompt returns the current system prompt
func (h *ChatHandler) GetSystemPrompt() string {
	prompt, err := h.promptBuilder.Build()
	if err != nil {
		return "Error building system prompt"
	}
	return prompt
}

// GetStreamingTokens returns the current number of tokens received during streaming
func (h *ChatHandler) GetStreamingTokens() int {
	h.streamingMutex.Lock()
	defer h.streamingMutex.Unlock()

	// Debug logging
	if h.streamingTokens > 0 {
		debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if debugFile != nil {
			fmt.Fprintf(debugFile, "[ChatHandler] GetStreamingTokens called, returning: %d\n", h.streamingTokens)
			debugFile.Close()
		}
	}

	return h.streamingTokens
}

// EstimatePromptTokens estimates the token count for a potential message
func (h *ChatHandler) EstimatePromptTokens(userInput string) (int, error) {
	// Get current session
	currentSession := h.session.GetCurrent()

	// Calculate total content length
	totalContent := ""

	// Add system prompt
	if systemPrompt, err := h.promptBuilder.Build(); err == nil {
		totalContent += systemPrompt + " "
	}

	// Add session messages if available
	if currentSession != nil {
		for _, msg := range currentSession.Messages {
			totalContent += msg.Content + " "
		}
	}

	// Add the potential user message
	totalContent += userInput

	// Use tokenizer for accurate token counting
	tokens, err := tokenizer.EstimateUserMessageTokens(totalContent, h.config.AI.Model)
	if err != nil {
		// Fallback to simple estimation
		runeCount := len([]rune(totalContent))
		estimatedTokens := runeCount / 4
		return estimatedTokens, nil
	}

	return tokens, nil
}

// AddMessageToSession adds a message to the current session
func (h *ChatHandler) AddMessageToSession(message ai.Message) error {
	currentSession := h.session.GetCurrent()
	if currentSession == nil {
		return fmt.Errorf("no active session")
	}
	return h.session.AddMessage(currentSession.ID, message)
}

// GetCurrentSession returns the current session
func (h *ChatHandler) GetCurrentSession() *Session {
	return h.session.GetCurrent()
}

// CreateNewSession creates a new chat session
func (h *ChatHandler) CreateNewSession() error {
	sessionID, err := h.session.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %w", err)
	}

	// Set the new session as current
	if err := h.session.SetCurrent(sessionID); err != nil {
		return fmt.Errorf("failed to set current session: %w", err)
	}

	return nil
}

// ContinueConversation continues the conversation without adding a new user message
// This is used after tool execution results have been added to the session
func (h *ChatHandler) ContinueConversation(ctx context.Context, tokenCallback func(int)) (*ChatResponse, error) {
	// Get current session
	currentSession := h.session.GetCurrent()
	if currentSession == nil {
		return nil, fmt.Errorf("no active session")
	}

	// Build messages for AI request (without adding new user message)
	messages := h.buildMessages(currentSession)

	// Create chat request with streaming enabled
	req := ai.ChatRequest{
		Model:           h.config.AI.Model,
		Messages:        messages,
		Temperature:     &h.config.AI.Temperature,
		MaxTokens:       &h.config.AI.MaxTokens,
		Stream:          true, // Enable streaming
		ReasoningEffort: h.config.AI.ReasoningEffort,
	}
	
	// Enable Structured Outputs if configured
	if h.config.AI.UseStructuredOutputs {
		req.ResponseFormat = &ai.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &ai.JSONSchema{
				Name:        "tool_response",
				Description: "Structured response with optional tool calls",
				Schema:      GetToolCallSchema(),
				Strict:      true,
			},
		}
	}

	// Send request to AI with streaming
	stream, err := h.aiClient.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat stream: %w", err)
	}
	defer stream.Close()

	// Process streaming response
	var fullContent strings.Builder
	var toolCalls []ai.ToolCall
	var totalUsage ai.Usage
	
	// Use structured output parser if enabled, otherwise use text parser
	useStructuredOutputs := h.config.AI.UseStructuredOutputs
	textParser := NewTextToolCallParser() // Still needed as fallback

	// Reset streaming tokens at start
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()

	chunkCount := 0
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		chunkCount++

		// Process chunk
		if chunk.Choices != nil && len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			// Handle content
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)

				// Parse based on mode
				contentStr := fullContent.String()
				
				if useStructuredOutputs {
					// Try to parse as structured JSON output
					if toolResp, err := ParseStructuredOutput(contentStr); err == nil {
						// Successfully parsed structured output
						if len(toolResp.ToolCalls) > 0 {
							// Convert structured tool calls to AI format
							if aiToolCalls, err := ConvertToAIToolCalls(toolResp.ToolCalls); err == nil {
								toolCalls = aiToolCalls
							}
						}
					}
				} else {
					// Fallback to text-based parsing
					_, parsedToolCalls, err := textParser.ParseMessage(contentStr)
					if err == nil && len(parsedToolCalls) > 0 {
						// Replace any existing tool calls with newly parsed ones
						toolCalls = parsedToolCalls
					}
				}

				// Calculate tokens for current content using tokenizer
				estimatedTokens := 0

				// Use tokenizer for accurate token counting
				if len(contentStr) > 0 {
					// Calculate tokens using the tokenizer package
					tokens, err := tokenizer.EstimateUserMessageTokens(contentStr, h.config.AI.Model)
					if err != nil {
						// Fallback to simple estimation
						runeCount := len([]rune(contentStr))
						estimatedTokens = runeCount / 4
					} else {
						estimatedTokens = tokens
					}
				}

				// Update ChatHandler's streaming tokens
				h.streamingMutex.Lock()
				h.streamingTokens = estimatedTokens
				h.streamingMutex.Unlock()

				// Call the callback if provided
				if tokenCallback != nil {
					tokenCallback(estimatedTokens)
				}
			}

			// Note: delta.ToolCalls will be empty since we're not using structured tool calling
		}
	}

	// Reset streaming tokens after streaming completes
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()

	// Debug: Log complete response JSON if debug mode is enabled
	if h.config.Logging.Level == "debug" {
		debugFile, err := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil && debugFile != nil {
			defer debugFile.Close()
			
			// Create a complete response structure for debugging
			responseDebug := map[string]interface{}{
				"timestamp":       time.Now().Format(time.RFC3339),
				"model":           h.config.AI.Model,
				"full_content":    fullContent.String(),
				"content_length":  fullContent.Len(),
				"tool_calls_count": len(toolCalls),
				"chunk_count":     chunkCount,
				"usage": map[string]int{
					"prompt_tokens":     totalUsage.PromptTokens,
					"completion_tokens": totalUsage.CompletionTokens,
					"total_tokens":      totalUsage.TotalTokens,
				},
			}
			
			// Add tool calls if present
			if len(toolCalls) > 0 {
				toolCallsDebug := make([]map[string]interface{}, len(toolCalls))
				for i, tc := range toolCalls {
					toolCallsDebug[i] = map[string]interface{}{
						"id":   tc.ID,
						"type": tc.Type,
						"function": map[string]string{
							"name":      tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					}
				}
				responseDebug["tool_calls"] = toolCallsDebug
			}
			
			// Marshal to JSON and write as single line
			if jsonData, err := json.Marshal(responseDebug); err == nil {
				fmt.Fprintf(debugFile, "[ChatHandler] CONTINUE_RESPONSE_JSON: %s\n", string(jsonData))
			}
		}
	}

	// Parse final message based on mode
	var cleanContent string
	contentStr := fullContent.String()
	
	if useStructuredOutputs {
		// Parse structured JSON output
		if toolResp, err := ParseStructuredOutput(contentStr); err == nil {
			// Successfully parsed structured output
			if toolResp.Text != nil {
				cleanContent = *toolResp.Text
			}
			if len(toolResp.ToolCalls) > 0 {
				// Convert structured tool calls to AI format
				if aiToolCalls, err := ConvertToAIToolCalls(toolResp.ToolCalls); err == nil {
					toolCalls = aiToolCalls
				}
			}
		} else {
			// If parsing fails, use raw content
			cleanContent = contentStr
		}
	} else {
		// Use text parser for final extraction
		parsedContent, finalToolCalls, _ := textParser.ParseMessage(contentStr)
		cleanContent = parsedContent
		if len(finalToolCalls) > 0 {
			toolCalls = finalToolCalls
		}
	}

	// Create final message
	message := ai.Message{
		Role:      ai.RoleAssistant,
		Content:   cleanContent,
		ToolCalls: toolCalls,
	}

	// Add assistant message to session
	if err := h.session.AddMessage(currentSession.ID, message); err != nil {
		return nil, fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Auto-save session after each message
	if h.persistence != nil {
		if session := h.session.GetCurrent(); session != nil {
			if err := h.persistence.SaveSession(session); err != nil {
				// Log error but don't fail the operation
			}
		}
	}

	// Process tool calls if any
	if len(toolCalls) > 0 {
		// For now, just include a note about tool calls
		toolCallInfo := fmt.Sprintf("[Tool calls requested: %d]", len(toolCalls))
		message.Content += toolCallInfo
	}

	// If usage wasn't provided in stream, estimate it
	if totalUsage.TotalTokens == 0 {
		// Use tokenizer for accurate token counting
		tokens, err := tokenizer.EstimateUserMessageTokens(fullContent.String(), h.config.AI.Model)
		if err != nil {
			// Fallback to simple estimation
			totalUsage.CompletionTokens = fullContent.Len() / 4
		} else {
			totalUsage.CompletionTokens = tokens
		}
		totalUsage.TotalTokens = totalUsage.CompletionTokens
	}

	return &ChatResponse{
		Content:    message.Content,
		TokenCount: totalUsage.TotalTokens,
		ToolCalls:  toolCalls,
		TokenUsage: &totalUsage,
	}, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
