package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Custom message types for the application
type (
	// ChatResponseMsg represents a response from the AI
	ChatResponseMsg struct {
		ID       string
		Content  string
		Role     string
		IsStream bool
		Done     bool
		Tokens   int
		Error    error
	}

	// ToolExecutionMsg represents tool execution status
	ToolExecutionMsg struct {
		ID     string
		Tool   string
		Status string // "started", "progress", "completed", "failed"
		Result interface{}
		Error  error
	}

	// StreamChunkMsg represents a streaming response chunk
	StreamChunkMsg struct {
		ID      string
		Content string
		Done    bool
	}

	// AppStateMsg represents application state changes
	AppStateMsg struct {
		State   AppState
		Message string
	}

	// ProgressMsg represents progress updates
	ProgressMsg struct {
		ID       string
		Progress float64 // 0.0 to 1.0
		Message  string
	}

	// UserInputMsg represents user input
	UserInputMsg struct {
		Input string
	}

	// ErrorMsg represents an error message
	ErrorMsg struct {
		Error   error
		Context string
	}

	// SuccessMsg represents a success message
	SuccessMsg struct {
		Message string
		Context string
	}
)

// AppState represents the overall application state
type AppState int

const (
	StateIdle AppState = iota
	StateLoading
	StateProcessing
	StateWaitingForApproval
	StateError
)

// Enhanced update method with detailed event handling
func (m Model) UpdateDetailed(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle different message types
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd = m.handleResize(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		newModel, keyCmd := m.handleKeyPress(msg)
		if model, ok := newModel.(Model); ok {
			m = model
		}
		cmds = append(cmds, keyCmd)

	case ChatResponseMsg:
		var chatCmd tea.Cmd
		m, chatCmd = m.handleChatResponse(msg)
		cmds = append(cmds, chatCmd)

	case ToolExecutionMsg:
		var toolCmd tea.Cmd
		m, toolCmd = m.handleToolExecution(msg)
		cmds = append(cmds, toolCmd)

	case StreamChunkMsg:
		var streamCmd tea.Cmd
		m, streamCmd = m.handleStreamChunk(msg)
		cmds = append(cmds, streamCmd)

	case AppStateMsg:
		var stateCmd tea.Cmd
		m, stateCmd = m.handleAppState(msg)
		cmds = append(cmds, stateCmd)

	case ProgressMsg:
		var progressCmd tea.Cmd
		m, progressCmd = m.handleProgress(msg)
		cmds = append(cmds, progressCmd)

	case ErrorMsg:
		var errorCmd tea.Cmd
		m, errorCmd = m.handleError(msg)
		cmds = append(cmds, errorCmd)

	case SuccessMsg:
		var successCmd tea.Cmd
		m, successCmd = m.handleSuccess(msg)
		cmds = append(cmds, successCmd)

	case UserInputMsg:
		var inputCmd tea.Cmd
		m, inputCmd = m.handleUserInput(msg)
		cmds = append(cmds, inputCmd)
	}

	// Update active view
	var viewCmd tea.Cmd
	m, viewCmd = m.updateActiveView(msg)
	cmds = append(cmds, viewCmd)

	return m, tea.Batch(cmds...)
}

// handleResize handles window resize events
func (m Model) handleResize(msg tea.WindowSizeMsg) tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height

	m.logger.Debug("Window resized",
		"width", m.width,
		"height", m.height)

	// Notify view components of resize
	// This would be implemented when views are created
	return nil
}

// handleChatResponse handles AI chat responses
func (m Model) handleChatResponse(msg ChatResponseMsg) (Model, tea.Cmd) {
	if msg.IsStream && !msg.Done {
		// Handle streaming response
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			// Update existing message
			lastMsg := &m.messages[len(m.messages)-1]
			lastMsg.Content += msg.Content
		} else {
			// Create new streaming message
			m.messages = append(m.messages, Message{
				ID:        msg.ID,
				Content:   msg.Content,
				Role:      msg.Role,
				Timestamp: time.Now(),
				Tokens:    msg.Tokens,
				Error:     msg.Error,
			})
		}
	} else {
		// Handle complete response
		if msg.Done && len(m.messages) > 0 && m.messages[len(m.messages)-1].ID == msg.ID {
			// Update final message
			lastMsg := &m.messages[len(m.messages)-1]
			lastMsg.Content = msg.Content
			lastMsg.Tokens = msg.Tokens
			lastMsg.Error = msg.Error
		} else {
			// Add new complete message
			m.messages = append(m.messages, Message{
				ID:        msg.ID,
				Content:   msg.Content,
				Role:      msg.Role,
				Timestamp: time.Now(),
				Tokens:    msg.Tokens,
				Error:     msg.Error,
			})
		}
		m.loading = false
	}

	if msg.Error != nil {
		m.error = msg.Error
		m.loading = false
	}

	return m, nil
}

// handleToolExecution handles tool execution updates
func (m Model) handleToolExecution(msg ToolExecutionMsg) (Model, tea.Cmd) {
	switch msg.Status {
	case "started":
		m.loading = true
		m.logger.Info("Tool execution started", "tool", msg.Tool, "id", msg.ID)

	case "progress":
		// Update progress if needed
		m.logger.Debug("Tool execution progress", "tool", msg.Tool, "id", msg.ID)

	case "completed":
		m.loading = false
		m.logger.Info("Tool execution completed", "tool", msg.Tool, "id", msg.ID)

		// Add tool result to messages if applicable
		if result, ok := msg.Result.(string); ok && result != "" {
			m.messages = append(m.messages, Message{
				ID:        generateMessageID(),
				Content:   result,
				Role:      "system",
				Timestamp: time.Now(),
			})
		}

	case "failed":
		m.loading = false
		m.error = msg.Error
		m.logger.Error("Tool execution failed", "tool", msg.Tool, "id", msg.ID, "error", msg.Error)
	}

	return m, nil
}

// handleStreamChunk handles streaming response chunks
func (m Model) handleStreamChunk(msg StreamChunkMsg) (Model, tea.Cmd) {
	// Find or create the streaming message
	found := false
	for i := range m.messages {
		if m.messages[i].ID == msg.ID {
			m.messages[i].Content += msg.Content
			if msg.Done {
				m.loading = false
			}
			found = true
			break
		}
	}

	if !found {
		// Create new streaming message
		m.messages = append(m.messages, Message{
			ID:        msg.ID,
			Content:   msg.Content,
			Role:      "assistant",
			Timestamp: time.Now(),
		})
	}

	return m, nil
}

// handleAppState handles application state changes
func (m Model) handleAppState(msg AppStateMsg) (Model, tea.Cmd) {
	switch msg.State {
	case StateIdle:
		m.loading = false
		m.error = nil

	case StateLoading:
		m.loading = true
		m.error = nil

	case StateProcessing:
		m.loading = true

	case StateWaitingForApproval:
		m.loading = false
		// Show approval dialog

	case StateError:
		m.loading = false
		if msg.Message != "" {
			m.error = fmt.Errorf(msg.Message)
		}
	}

	return m, nil
}

// handleProgress handles progress updates
func (m Model) handleProgress(msg ProgressMsg) (Model, tea.Cmd) {
	// This would update a progress indicator
	m.logger.Debug("Progress update",
		"id", msg.ID,
		"progress", msg.Progress,
		"message", msg.Message)

	return m, nil
}

// handleError handles error messages
func (m Model) handleError(msg ErrorMsg) (Model, tea.Cmd) {
	m.error = msg.Error
	m.loading = false

	m.logger.Error("UI error",
		"error", msg.Error,
		"context", msg.Context)

	return m, nil
}

// handleSuccess handles success messages
func (m Model) handleSuccess(msg SuccessMsg) (Model, tea.Cmd) {
	m.error = nil

	m.logger.Info("Success",
		"message", msg.Message,
		"context", msg.Context)

	// Could show a temporary success indicator
	return m, nil
}

// handleUserInput handles user input messages
func (m Model) handleUserInput(msg UserInputMsg) (Model, tea.Cmd) {
	m.currentInput = msg.Input
	return m, nil
}

// updateActiveView updates the currently active view
func (m Model) updateActiveView(msg tea.Msg) (Model, tea.Cmd) {
	// This would delegate to the appropriate view component
	// For now, just return as-is
	switch m.activeView {
	case ViewChat:
		// m.chatView, cmd = m.chatView.Update(msg)
		// return m, cmd

	case ViewInput:
		// m.inputView, cmd = m.inputView.Update(msg)
		// return m, cmd

	case ViewStatus:
		// m.statusView, cmd = m.statusView.Update(msg)
		// return m, cmd

	case ViewHelp:
		// m.helpView, cmd = m.helpView.Update(msg)
		// return m, cmd
	}

	return m, nil
}

// Command generators for async operations

// SendMessageCmd creates a command to send a message to the AI
func (m Model) SendMessageCmd(input string) tea.Cmd {
	return func() tea.Msg {
		// Add user message first
		_ = Message{
			ID:        generateMessageID(),
			Content:   input,
			Role:      "user",
			Timestamp: time.Now(),
		}

		// This would integrate with the actual chat handler
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Simulate async processing
		time.Sleep(100 * time.Millisecond)

		// Return a chat response message
		return ChatResponseMsg{
			ID:       generateMessageID(),
			Content:  "Response to: " + input,
			Role:     "assistant",
			IsStream: false,
			Done:     true,
			Tokens:   10,
		}
	}
}

// ExecuteToolCmd creates a command to execute a tool
func (m Model) ExecuteToolCmd(toolName string, args map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		toolID := generateMessageID()

		// Start execution
		go func() {
			// This would integrate with the actual tool manager
			time.Sleep(500 * time.Millisecond)

			// Send completion message
			// This is a simplified example
		}()

		return ToolExecutionMsg{
			ID:     toolID,
			Tool:   toolName,
			Status: "started",
		}
	}
}

// StreamResponseCmd creates a command for streaming responses
func (m Model) StreamResponseCmd(input string) tea.Cmd {
	return func() tea.Msg {
		// This would set up a streaming response
		// For now, just return a simple response
		return ChatResponseMsg{
			ID:       generateMessageID(),
			Content:  "Streaming response to: " + input,
			Role:     "assistant",
			IsStream: true,
			Done:     true,
			Tokens:   15,
		}
	}
}
