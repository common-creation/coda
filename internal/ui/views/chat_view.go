package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/styles"
)

// ChatView manages the display of chat messages
type ChatView struct {
	// Core components
	viewport viewport.Model
	messages []ChatMessage
	styles   styles.Styles
	logger   *log.Logger

	// Display properties
	width         int
	height        int
	scrollOffset  int
	autoScroll    bool
	showTimestamp bool

	// Rendering cache
	renderedMessages []string
	lastUpdate       time.Time
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	ID        string
	Role      string // "user", "assistant", "system", "error"
	Content   string
	Timestamp time.Time
	IsError   bool
	Tokens    int
	IsStream  bool
}

// NewChatView creates a new chat view instance
func NewChatView(width, height int, styles styles.Styles, logger *log.Logger) *ChatView {
	vp := viewport.New(width, height-2) // Reserve space for borders
	vp.Style = styles.Container

	return &ChatView{
		viewport:         vp,
		messages:         make([]ChatMessage, 0),
		styles:           styles,
		logger:           logger,
		width:            width,
		height:           height,
		autoScroll:       true,
		showTimestamp:    true,
		renderedMessages: make([]string, 0),
	}
}

// SetSize updates the view dimensions
func (cv *ChatView) SetSize(width, height int) {
	cv.width = width
	cv.height = height
	cv.viewport.Width = width
	cv.viewport.Height = height - 2 // Account for borders

	// Invalidate cache
	cv.invalidateCache()
}

// AddMessage adds a new message to the chat
func (cv *ChatView) AddMessage(msg ChatMessage) {
	cv.messages = append(cv.messages, msg)
	cv.invalidateCache()

	if cv.autoScroll {
		cv.scrollToBottom()
	}

	cv.logger.Debug("Message added to chat view",
		"id", msg.ID,
		"role", msg.Role,
		"content_length", len(msg.Content))
}

// UpdateMessage updates an existing message (for streaming)
func (cv *ChatView) UpdateMessage(id string, content string) {
	for i := range cv.messages {
		if cv.messages[i].ID == id {
			cv.messages[i].Content = content
			cv.messages[i].Timestamp = time.Now()
			cv.invalidateCache()

			if cv.autoScroll {
				cv.scrollToBottom()
			}
			return
		}
	}
}

// ClearMessages removes all messages
func (cv *ChatView) ClearMessages() {
	cv.messages = make([]ChatMessage, 0)
	cv.renderedMessages = make([]string, 0)
	cv.viewport.SetContent("")
	cv.logger.Debug("Chat messages cleared")
}

// GetMessages returns all messages
func (cv *ChatView) GetMessages() []ChatMessage {
	return cv.messages
}

// SetAutoScroll enables or disables auto-scrolling
func (cv *ChatView) SetAutoScroll(enabled bool) {
	cv.autoScroll = enabled
}

// SetShowTimestamp enables or disables timestamp display
func (cv *ChatView) SetShowTimestamp(enabled bool) {
	cv.showTimestamp = enabled
	cv.invalidateCache()
}

// Init implements tea.Model
func (cv *ChatView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (cv *ChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle scrolling keys
		switch msg.String() {
		case "up", "k":
			cv.autoScroll = false
			cv.viewport.LineUp(1)
		case "down", "j":
			cv.viewport.LineDown(1)
			// Re-enable auto-scroll if at bottom
			if cv.viewport.AtBottom() {
				cv.autoScroll = true
			}
		case "pgup":
			cv.autoScroll = false
			cv.viewport.HalfViewUp()
		case "pgdown":
			cv.viewport.HalfViewDown()
			if cv.viewport.AtBottom() {
				cv.autoScroll = true
			}
		case "home":
			cv.autoScroll = false
			cv.viewport.GotoTop()
		case "end":
			cv.viewport.GotoBottom()
			cv.autoScroll = true
		}

	case tea.WindowSizeMsg:
		cv.SetSize(msg.Width, msg.Height)
	}

	// Update viewport
	cv.viewport, cmd = cv.viewport.Update(msg)

	// Render content if cache is invalid
	if cv.needsRerender() {
		cv.renderContent()
	}

	return cv, cmd
}

// View implements tea.Model
func (cv *ChatView) View() string {
	if len(cv.messages) == 0 {
		return cv.renderEmptyState()
	}

	// Ensure content is rendered
	if cv.needsRerender() {
		cv.renderContent()
	}

	return cv.viewport.View()
}

// renderContent renders all messages to the viewport
func (cv *ChatView) renderContent() {
	var content strings.Builder

	for i, msg := range cv.messages {
		rendered := cv.renderMessage(msg, i)
		content.WriteString(rendered)
		if i < len(cv.messages)-1 {
			content.WriteString("\n")
		}
	}

	cv.viewport.SetContent(content.String())
	cv.lastUpdate = time.Now()
}

// renderMessage renders a single message
func (cv *ChatView) renderMessage(msg ChatMessage, index int) string {
	var style lipgloss.Style
	var roleDisplay string

	// Choose style based on role
	switch msg.Role {
	case "user":
		style = cv.styles.UserMessage
		roleDisplay = "You"
	case "assistant":
		style = cv.styles.AIMessage
		roleDisplay = "Assistant"
	case "system":
		style = cv.styles.SystemMessage
		roleDisplay = "System"
	case "error":
		style = cv.styles.ErrorMessage
		roleDisplay = "Error"
	default:
		style = cv.styles.ChatMessage
		roleDisplay = strings.Title(msg.Role)
	}

	// Create header
	header := cv.createMessageHeader(roleDisplay, msg.Timestamp, msg.Tokens)

	// Process content
	content := cv.processMessageContent(msg.Content, msg.Role)

	// Create message box
	messageBox := cv.createMessageBox(header, content, style)

	return messageBox
}

// createMessageHeader creates the message header with role and timestamp
func (cv *ChatView) createMessageHeader(role string, timestamp time.Time, tokens int) string {
	var header strings.Builder

	header.WriteString(cv.styles.Bold.Render(role))

	if cv.showTimestamp {
		timeStr := timestamp.Format("15:04:05")
		header.WriteString(cv.styles.Muted.Render(" (" + timeStr + ")"))
	}

	if tokens > 0 {
		header.WriteString(cv.styles.Muted.Render(fmt.Sprintf(" [%d tokens]", tokens)))
	}

	return header.String()
}

// processMessageContent processes and formats message content
func (cv *ChatView) processMessageContent(content, role string) string {
	// Basic markdown-like processing
	content = cv.processCodeBlocks(content)
	content = cv.processInlineCode(content)
	content = cv.processLinks(content)

	// Word wrap for long lines
	content = cv.wordWrap(content, cv.width-4) // Account for padding

	return content
}

// processCodeBlocks handles code block formatting
func (cv *ChatView) processCodeBlocks(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false
	language := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End code block
				result.WriteString(cv.styles.Code.Render("```"))
				inCodeBlock = false
				language = ""
			} else {
				// Start code block
				language = strings.TrimPrefix(line, "```")
				result.WriteString(cv.styles.Code.Render("```" + language))
				inCodeBlock = true
			}
		} else if inCodeBlock {
			// Code content
			result.WriteString(cv.styles.Code.Render(line))
		} else {
			// Regular content
			result.WriteString(line)
		}

		if line != lines[len(lines)-1] {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// processInlineCode handles inline code formatting
func (cv *ChatView) processInlineCode(content string) string {
	// Simple inline code processing
	parts := strings.Split(content, "`")
	if len(parts) < 3 {
		return content
	}

	var result strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			// Inside backticks
			result.WriteString(cv.styles.Code.Render(part))
		} else {
			result.WriteString(part)
		}
	}

	return result.String()
}

// processLinks handles link formatting
func (cv *ChatView) processLinks(content string) string {
	// Basic link detection and styling
	words := strings.Fields(content)
	for i, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			words[i] = cv.styles.Link.Render(word)
		}
	}
	return strings.Join(words, " ")
}

// wordWrap wraps text to fit within the specified width
func (cv *ChatView) wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var result strings.Builder
	var lineLength int

	for _, word := range words {
		wordLen := lipgloss.Width(word)

		if lineLength+wordLen+1 > width && lineLength > 0 {
			result.WriteString("\n")
			lineLength = 0
		}

		if lineLength > 0 {
			result.WriteString(" ")
			lineLength++
		}

		result.WriteString(word)
		lineLength += wordLen
	}

	return result.String()
}

// createMessageBox creates a bordered message box
func (cv *ChatView) createMessageBox(header, content string, style lipgloss.Style) string {
	// Calculate box width
	boxWidth := cv.width - 2 // Account for side margins

	// Create the box content
	var box strings.Builder
	box.WriteString("┌─ " + header + " ")

	// Fill the rest of the header with dashes
	headerLen := lipgloss.Width("┌─ " + header + " ")
	for i := headerLen; i < boxWidth-1; i++ {
		box.WriteString("─")
	}
	box.WriteString("┐\n")

	// Add content lines
	contentLines := strings.Split(content, "\n")
	for _, line := range contentLines {
		box.WriteString("│ " + line)

		// Pad to box width
		lineLen := lipgloss.Width("│ " + line)
		for i := lineLen; i < boxWidth-1; i++ {
			box.WriteString(" ")
		}
		box.WriteString(" │\n")
	}

	// Add bottom border
	box.WriteString("└")
	for i := 1; i < boxWidth-1; i++ {
		box.WriteString("─")
	}
	box.WriteString("┘")

	return style.Render(box.String())
}

// renderEmptyState renders the empty chat state
func (cv *ChatView) renderEmptyState() string {
	emptyMsg := "No messages yet. Start a conversation!"
	return cv.styles.Muted.
		Width(cv.width).
		Height(cv.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(emptyMsg)
}

// scrollToBottom scrolls to the bottom of the chat
func (cv *ChatView) scrollToBottom() {
	cv.viewport.GotoBottom()
}

// needsRerender checks if the view needs re-rendering
func (cv *ChatView) needsRerender() bool {
	return len(cv.renderedMessages) != len(cv.messages) ||
		time.Since(cv.lastUpdate) > time.Millisecond*100
}

// invalidateCache invalidates the rendering cache
func (cv *ChatView) invalidateCache() {
	cv.renderedMessages = cv.renderedMessages[:0]
}

// GetScrollPercentage returns the current scroll percentage (0-100)
func (cv *ChatView) GetScrollPercentage() float64 {
	if cv.viewport.TotalLineCount() == 0 {
		return 100.0
	}
	return cv.viewport.ScrollPercent() * 100
}

// IsAtBottom returns true if the view is scrolled to the bottom
func (cv *ChatView) IsAtBottom() bool {
	return cv.viewport.AtBottom()
}

// IsAtTop returns true if the view is scrolled to the top
func (cv *ChatView) IsAtTop() bool {
	return cv.viewport.AtTop()
}
