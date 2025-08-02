package chat

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/tools"
)

// ChatHandler manages the chat interaction flow
type ChatHandler struct {
	aiClient     ai.Client
	toolManager  *tools.Manager
	session      *SessionManager
	config       *config.Config
	history      *History
	systemPrompt string
	persistence  *FilePersistence
	
	// Streaming state
	streamingTokens int
	streamingMutex  sync.Mutex
}

// ChatResponse represents a response from the chat handler
type ChatResponse struct {
	Content          string
	TokenCount       int           // Total token count (deprecated, use TokenUsage.TotalTokens)
	ToolCalls        []ai.ToolCall
	TokenUsage       *ai.Usage     // Detailed token usage from AI response
	EstimatedPrompt  int           // Estimated prompt tokens (before sending)
}

// NewChatHandler creates a new chat handler
func NewChatHandler(aiClient ai.Client, toolManager *tools.Manager, session *SessionManager, cfg *config.Config, history *History) *ChatHandler {
	handler := &ChatHandler{
		aiClient:    aiClient,
		toolManager: toolManager,
		session:     session,
		config:      cfg,
		history:     history,
		systemPrompt: "You are CODA (CODing Agent), an AI assistant designed to help developers with coding tasks. " +
			"You have access to various tools for file operations and can execute them to assist with programming tasks. " +
			"Always be helpful, accurate, and provide clear explanations for your actions.",
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

// HandleMessage processes a user message and returns the response
func (h *ChatHandler) HandleMessage(ctx context.Context, input string) error {
	// Trim and validate input
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("empty input")
	}

	// Handle special commands
	if strings.HasPrefix(input, "/") {
		return h.handleCommand(ctx, input)
	}

	// Get or create current session
	currentSession := h.session.GetCurrent()
	if currentSession == nil {
		sessionID, err := h.session.CreateSession()
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		currentSession, _ = h.session.GetSession(sessionID)
	}

	// Add user message to session
	userMessage := ai.Message{
		Role:    ai.RoleUser,
		Content: input,
	}

	if err := h.session.AddMessage(currentSession.ID, userMessage); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Build messages for AI request
	messages := h.buildMessages(currentSession)

	// Create chat request
	req := ai.ChatRequest{
		Model:       h.config.AI.Model,
		Messages:    messages,
		Temperature: &h.config.AI.Temperature,
		MaxTokens:   &h.config.AI.MaxTokens,
		Tools:       h.getToolDefinitions(),
		Stream:      true,
	}

	// Send request to AI
	stream, err := h.aiClient.ChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create chat stream: %w", err)
	}
	defer stream.Close()

	// Process streaming response
	return h.processStreamResponse(ctx, stream, currentSession.ID)
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
		Model:       h.config.AI.Model,
		Messages:    messages,
		Temperature: &h.config.AI.Temperature,
		MaxTokens:   &h.config.AI.MaxTokens,
		Tools:       h.getToolDefinitions(),
		Stream:      true, // Enable streaming
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
	
	// Reset streaming tokens at start
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()
	
	// Debug logging
	debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[ChatHandler] Starting streaming response processing\n")
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
				
				// Calculate tokens for current content using tokenizer
				estimatedTokens := 0
				contentStr := fullContent.String()
				
				// Use tokenizer for accurate token counting
				if len(contentStr) > 0 {
					// Get model from config
					model := h.config.AI.Model
					if model == "" {
						model = "gpt-4" // Fallback to default
					}
					
					// Calculate tokens using simple estimation
					// TODO: Once tokenizer is in a separate package, use accurate counting
					runeCount := len([]rune(contentStr))
					estimatedTokens = runeCount / 3 // Rough estimation for mixed content
					
					// Debug logging
					debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
					if debugFile != nil {
						fmt.Fprintf(debugFile, "[ChatHandler] Token estimation: runeCount=%d, estimatedTokens=%d\n", runeCount, estimatedTokens)
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

			// Handle tool calls
			if delta.ToolCalls != nil {
				toolCalls = append(toolCalls, delta.ToolCalls...)
			}
		}
		
		// Note: Usage information is typically not available in streaming chunks
		// It will be estimated after streaming completes
	}

	// Reset streaming tokens after streaming completes
	h.streamingMutex.Lock()
	h.streamingTokens = 0
	h.streamingMutex.Unlock()

	// Create final message
	message := ai.Message{
		Role:      ai.RoleAssistant,
		Content:   fullContent.String(),
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
		toolCallInfo := fmt.Sprintf("\n\n[Tool calls requested: %d]", len(toolCalls))
		message.Content += toolCallInfo
	}

	// If usage wasn't provided in stream, estimate it
	if totalUsage.TotalTokens == 0 {
		// Rough estimation
		totalUsage.CompletionTokens = fullContent.Len() / 4
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

// handleCommand processes special commands
func (h *ChatHandler) handleCommand(ctx context.Context, command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("invalid command")
	}

	switch parts[0] {
	case "/clear":
		return h.handleClearCommand()
	case "/save":
		return h.handleSaveCommand(parts)
	case "/load":
		return h.handleLoadCommand(parts)
	case "/help":
		return h.handleHelpCommand()
	default:
		return fmt.Errorf("unknown command: %s", parts[0])
	}
}

// handleClearCommand clears the current session
func (h *ChatHandler) handleClearCommand() error {
	current := h.session.GetCurrent()
	if current == nil {
		fmt.Println("No active session to clear")
		return nil
	}

	// Create new session
	sessionID, err := h.session.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %w", err)
	}

	h.session.SetCurrent(sessionID)
	fmt.Println("Session cleared. New session started.")
	return nil
}

// handleSaveCommand saves the current session
func (h *ChatHandler) handleSaveCommand(parts []string) error {
	current := h.session.GetCurrent()
	if current == nil {
		fmt.Println("No active session to save")
		return nil
	}

	// Generate title if not provided
	title := "Chat Session"
	if len(parts) > 1 {
		title = strings.Join(parts[1:], " ")
	} else {
		// Try to generate title from first user message
		for _, msg := range current.Messages {
			if msg.Role == ai.RoleUser {
				title = truncateString(msg.Content, 50)
				break
			}
		}
	}

	// Save session
	if err := h.history.Save(current); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Printf("Session saved: %s\n", title)
	return nil
}

// handleLoadCommand loads a previous session
func (h *ChatHandler) handleLoadCommand(parts []string) error {
	if len(parts) < 2 {
		// List available sessions
		sessions := h.history.GetRecent(10)
		if len(sessions) == 0 {
			fmt.Println("No saved sessions found")
			return nil
		}

		fmt.Println("Available sessions:")
		for i, s := range sessions {
			fmt.Printf("%d. %s (%s) - %d messages\n",
				i+1, s.Title, s.StartTime.Format("2006-01-02 15:04"), s.Messages)
		}
		fmt.Println("\nUse '/load <number>' or '/load <session-id>' to load a session")
		return nil
	}

	// Load session by ID or index
	sessionID := parts[1]

	// Try to parse as index first
	sessions := h.history.GetRecent(10)
	for i, s := range sessions {
		if fmt.Sprintf("%d", i+1) == sessionID {
			sessionID = s.ID
			break
		}
	}

	// Load session
	loadedSession, err := h.history.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Set as current session
	h.session.sessions[loadedSession.ID] = loadedSession
	h.session.SetCurrent(loadedSession.ID)

	fmt.Printf("Loaded session: %s\n", sessionID)
	return nil
}

// handleHelpCommand displays help information
func (h *ChatHandler) handleHelpCommand() error {
	help := `
CODA Chat Commands:

/clear          - Clear the current session and start fresh
/save [title]   - Save the current session with an optional title
/load [id]      - Load a previous session by ID or list available sessions
/help           - Show this help message

During chat:
- Type your message and press Enter to send
- The AI can use various tools to help with coding tasks
- Tool executions require your approval for safety
`
	fmt.Println(help)
	return nil
}

// buildMessages constructs the message list for the AI request
func (h *ChatHandler) buildMessages(session *Session) []ai.Message {
	messages := make([]ai.Message, 0, len(session.Messages)+1)

	// Add system prompt
	messages = append(messages, ai.Message{
		Role:    ai.RoleSystem,
		Content: h.systemPrompt,
	})

	// Add conversation history
	messages = append(messages, session.Messages...)

	return messages
}

// getToolDefinitions returns the tool definitions for the AI
func (h *ChatHandler) getToolDefinitions() []ai.Tool {
	tools := h.toolManager.GetAll()
	definitions := make([]ai.Tool, 0, len(tools))

	for _, tool := range tools {
		definitions = append(definitions, ai.Tool{
			Type: "function",
			Function: ai.FunctionTool{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters: map[string]interface{}{
					"type":       tool.Schema().Type,
					"properties": tool.Schema().Properties,
					"required":   tool.Schema().Required,
				},
			},
		})
	}

	return definitions
}

// processStreamResponse handles the streaming response from the AI
func (h *ChatHandler) processStreamResponse(ctx context.Context, stream ai.StreamReader, sessionID string) error {
	var fullContent strings.Builder
	var toolCalls []ai.ToolCall

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stream: %w", err)
		}

		// Process chunk
		if chunk.Choices != nil && len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			// Handle content
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				fmt.Print(delta.Content)
			}

			// Handle tool calls
			if delta.ToolCalls != nil {
				toolCalls = append(toolCalls, delta.ToolCalls...)
			}
		}
	}

	// Add assistant message to session
	assistantMessage := ai.Message{
		Role:      ai.RoleAssistant,
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	}

	if err := h.session.AddMessage(sessionID, assistantMessage); err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Auto-save session after each message
	if h.persistence != nil {
		if session := h.session.GetCurrent(); session != nil {
			if err := h.persistence.SaveSession(session); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Warning: failed to auto-save session: %v\n", err)
			}
		}
	}

	// Process tool calls if any
	if len(toolCalls) > 0 {
		return h.processToolCalls(ctx, sessionID, toolCalls)
	}

	fmt.Println() // New line after response
	return nil
}

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
	h.systemPrompt = prompt
}

// GetSystemPrompt returns the current system prompt
func (h *ChatHandler) GetSystemPrompt() string {
	return h.systemPrompt
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
	totalContent += h.systemPrompt + " "
	
	// Add session messages if available
	if currentSession != nil {
		for _, msg := range currentSession.Messages {
			totalContent += msg.Content + " "
		}
	}
	
	// Add the potential user message
	totalContent += userInput
	
	// Simple token estimation
	// TODO: Once tokenizer is in a separate package, use accurate counting
	runeCount := len([]rune(totalContent))
	estimatedTokens := runeCount / 3 // Rough estimation for mixed content
	
	return estimatedTokens, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
