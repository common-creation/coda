package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/chat"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/errors"
	"github.com/common-creation/coda/internal/styles"
	"github.com/common-creation/coda/internal/tools"
	"github.com/common-creation/coda/internal/ui/components"
)

// ViewType represents the currently active view
type ViewType int

const (
	ViewChat ViewType = iota
	ViewInput
	ViewStatus
	ViewHelp
)

// Message represents a chat message
type Message struct {
	ID        string
	Content   string
	Role      string // "user", "assistant", "system"
	Timestamp time.Time
	Tokens    int
	Error     error
}

// Removed old KeyMap definition - now using the advanced keybindings system

// Model represents the application state for Bubbletea
type Model struct {
	// UI state
	width  int
	height int
	ready  bool

	// View components (will be implemented later)
	// chatView   ChatView
	// inputView  InputView
	// statusView StatusView
	// helpView   HelpView

	// Application state
	activeView   ViewType
	messages     []Message
	currentInput string
	showHelp     bool
	loading      bool
	error        error

	// Spinner and timing
	spinner spinner.Model

	// Viewport for chat history
	viewport        viewport.Model
	loadingStart    time.Time
	estimatedTokens int       // Estimated tokens for the current request
	userInputTokens int       // Estimated tokens for just the user input
	lastTokenUsage  *ai.Usage // Last response token usage

	// Streaming state
	streamingContent strings.Builder // Buffer for streaming content

	// Styles
	styles styles.Styles

	// Input mode state - Always INSERT mode for IME support
	currentMode   Mode
	previousMode  Mode
	commandBuffer string
	searchBuffer  string
	searchResults []int // indices of matching messages
	currentMatch  int

	// Tool call permit dialog state
	pendingToolCalls     []ai.ToolCall // Tool calls waiting for user approval
	selectedPermitOption int           // Currently selected option (0=reject, 1=approve)
	permitDialogVisible  bool          // Whether permit dialog is currently visible

	// Cursor position management
	cursorPosition int // カーソル位置（rune単位）
	cursorColumn   int // 現在の列位置（上下移動時の列位置保持用）

	// Cursor styles
	cursorStyle      lipgloss.Style // 文字列中のカーソル用（背景色反転）
	blockCursorStyle lipgloss.Style // 行末カーソル用（ブロックシンボル）

	// Dependencies
	config           *config.Config
	chatHandler      *chat.ChatHandler
	toolManager      *tools.Manager
	logger           *log.Logger
	ctx              context.Context
	errorHandler     *errors.ErrorHandler
	errorDisplay     *components.ErrorDisplay
	errorBanner      *components.ErrorBanner
	toast            *components.ToastNotification
	showErrorDetails bool

	// Configuration
	keymap KeyMap

	// Initial message to send on startup
	initialMessage string

	// Ctrl+C double press handling
	lastCtrlCTime time.Time
	ctrlCMessage  string

	// Esc double press handling
	lastEscTime time.Time
	escMessage  string

	// Ctrl+N double press handling
	lastCtrlNTime time.Time
	ctrlNMessage  string
}

// ModelOptions contains options for creating a new Model
type ModelOptions struct {
	Config         *config.Config
	ChatHandler    *chat.ChatHandler
	ToolManager    *tools.Manager
	Logger         *log.Logger
	Context        context.Context
	ErrorHandler   *errors.ErrorHandler
	InitialMessage string // Initial message to send on startup
}

// NewModel creates a new UI model
func NewModel(opts ModelOptions) Model {
	// Initialize styles based on config theme
	themeName := "default"
	if opts.Config != nil && opts.Config.UI.Theme != "" {
		themeName = opts.Config.UI.Theme
	}

	theme := styles.GetTheme(themeName)

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		// Initialize UI state
		width:  80,
		height: 24,
		ready:  false,

		// Initialize application state
		activeView:   ViewChat,
		messages:     make([]Message, 0),
		currentInput: "",
		showHelp:     false,
		loading:      false,
		error:        nil,

		// Initialize spinner and timing
		spinner:         s,
		loadingStart:    time.Time{},
		estimatedTokens: 0,
		userInputTokens: 0,
		lastTokenUsage:  nil,

		// Initialize streaming state
		streamingContent: strings.Builder{},

		// Initialize styles
		styles: theme.GetStyles(),

		// Initialize input mode state - Always INSERT mode for IME support
		currentMode:   ModeInsert, // Always start in Insert mode for IME
		previousMode:  ModeInsert,
		commandBuffer: "",
		searchBuffer:  "",
		searchResults: make([]int, 0),
		currentMatch:  0,

		// Initialize tool call permit dialog state
		pendingToolCalls:     make([]ai.ToolCall, 0),
		selectedPermitOption: 0, // Default to reject (0)
		permitDialogVisible:  false,

		// Initialize cursor position
		cursorPosition: 0,
		cursorColumn:   0,

		// Initialize cursor styles
		cursorStyle:      lipgloss.NewStyle().Reverse(true),
		blockCursorStyle: lipgloss.NewStyle(),

		// Set dependencies
		config:           opts.Config,
		chatHandler:      opts.ChatHandler,
		toolManager:      opts.ToolManager,
		logger:           opts.Logger,
		ctx:              opts.Context,
		errorHandler:     opts.ErrorHandler,
		errorDisplay:     components.NewErrorDisplay(opts.ErrorHandler),
		errorBanner:      components.NewErrorBanner(),
		toast:            nil,
		showErrorDetails: false,

		// Set keymap
		keymap: DefaultKeyMap(),

		// Set initial message
		initialMessage: opts.InitialMessage,

		// Initialize Ctrl+C double press handling
		lastCtrlCTime: time.Time{},
		ctrlCMessage:  "",

		// Initialize Esc double press handling
		lastEscTime: time.Time{},
		escMessage:  "",

		// Initialize Ctrl+N double press handling
		lastCtrlNTime: time.Time{},
		ctrlNMessage:  "",
	}
}

// Init implements tea.Model interface
func (m Model) Init() tea.Cmd {
	m.logger.Debug("Initializing UI model")

	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
		func() tea.Msg {
			return readyMsg{}
		},
	)
}

// Update implements tea.Model interface
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update viewport only when in scroll mode or for mouse events
	shouldUpdateViewport := false

	// Check if we're handling input-related keys
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()

		// Toggle scroll mode with Ctrl+Y
		if key == "ctrl+y" {
			if m.currentMode == ModeScroll {
				// Return to previous mode
				m.currentMode = m.previousMode
			} else {
				// Enter scroll mode
				m.previousMode = m.currentMode
				m.currentMode = ModeScroll
			}
			return m, nil
		}

		// In scroll mode, allow viewport to handle arrow keys
		if m.currentMode == ModeScroll {
			shouldUpdateViewport = true
		}
	}

	// Always allow mouse events to update viewport
	if _, ok := msg.(tea.MouseMsg); ok {
		shouldUpdateViewport = true
	}

	if shouldUpdateViewport {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		if vpCmd != nil {
			cmds = append(cmds, vpCmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.logger.Debug("Window resized", "width", m.width, "height", m.height)

		// Calculate viewport dimensions
		// Reserve space for input, help line, and margins
		inputHeight := 3  // Input area height
		helpHeight := 1   // Help line height
		marginHeight := 3 // Additional margins

		viewportHeight := m.height - inputHeight - helpHeight - marginHeight
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		// Reserve 1 column for scrollbar
		viewportWidth := m.width - 1
		if viewportWidth < 1 {
			viewportWidth = 1
		}

		// Initialize or update viewport
		if !m.ready {
			m.viewport = viewport.New(viewportWidth, viewportHeight)
			m.viewport.MouseWheelEnabled = true
			m.viewport.MouseWheelDelta = 3
		} else {
			m.viewport.Width = viewportWidth
			m.viewport.Height = viewportHeight
		}

		// Update viewport content
		m.updateViewportContent()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle key events
		return m.handleKeyPress(msg)

	case readyMsg:
		m.ready = true
		m.logger.Debug("UI model ready")

		// Send initial message if provided
		if m.initialMessage != "" {
			m.currentInput = m.initialMessage
			m.initialMessage = "" // Clear to prevent re-sending
			_, cmd := m.sendMessage()
			cmds = append(cmds, cmd)
		}

	case chatResponseMsg:
		// Use completion tokens for assistant message
		assistantTokens := 0
		if msg.TokenUsage != nil {
			assistantTokens = msg.TokenUsage.CompletionTokens
		}

		m.messages = append(m.messages, Message{
			ID:        msg.ID,
			Content:   msg.Content,
			Role:      "assistant",
			Timestamp: time.Now(),
			Tokens:    assistantTokens,
		})
		m.loading = false
		m.lastTokenUsage = msg.TokenUsage
		// Reset streaming state
		m.streamingContent.Reset()
		// Reset user input tokens
		m.userInputTokens = 0
		// Update viewport content with new message
		m.updateViewportContent()

		// Check for tool calls and enter permit mode if needed
		if len(msg.ToolCalls) > 0 {
			m.pendingToolCalls = msg.ToolCalls
			m.permitDialogVisible = true
			m.selectedPermitOption = 0 // Default to reject
			// Store current mode and switch to permit mode
			if m.currentMode != ModePermit {
				m.previousMode = m.currentMode
				m.currentMode = ModePermit
			}
		}

	case errorMsg:
		m.error = msg.error
		m.loading = false

		// Integrate with global error handler
		if m.errorHandler != nil {
			m.errorHandler.HandleWithContext(msg.error, msg.userAction, msg.metadata)
		}

		// Update error display
		if m.errorDisplay != nil {
			m.errorDisplay.SetError(msg.error)
		}

		// Create toast notification for user errors
		if m.errorHandler != nil {
			category := m.errorDisplay.ClassifyError(msg.error)
			if category == errors.UserError {
				userMessage := m.errorHandler.UserMessage(msg.error)
				m.toast = components.NewToastNotification(userMessage, 5*time.Second)
			}
		}

		m.logger.Error("UI error", "error", msg.error)

	case dismissErrorMsg:
		m.error = nil
		if m.errorDisplay != nil {
			m.errorDisplay.SetError(nil)
		}
		m.toast = nil

	case toggleErrorDetailsMsg:
		m.showErrorDetails = !m.showErrorDetails
		if m.errorDisplay != nil {
			m.errorDisplay.ToggleDetails()
		}

	case retryLastActionMsg:
		// Find the last user message and retry
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == "user" {
				m.currentInput = m.messages[i].Content
				_, cmd := m.sendMessage()
				cmds = append(cmds, cmd)
				break
			}
		}

	case clearCtrlCMsg:
		// Clear the Ctrl+C message if it hasn't been cleared already
		if m.ctrlCMessage != "" && time.Since(m.lastCtrlCTime) >= time.Second {
			m.ctrlCMessage = ""
		}

	case clearEscMsg:
		// Clear the Esc message if it hasn't been cleared already
		if m.escMessage != "" && time.Since(m.lastEscTime) >= time.Second {
			m.escMessage = ""
		}

	case clearCtrlNMsg:
		// Clear the Ctrl+N message if it hasn't been cleared already
		if m.ctrlNMessage != "" && time.Since(m.lastCtrlNTime) >= time.Second {
			m.ctrlNMessage = ""
		}

	case toolExecutionMsg:
		// Tool execution completed, send results to LLM
		m.logger.Debug("Tool execution completed", "count", len(msg.results))
		// Convert tool results to messages and send back to LLM
		return m, m.sendToolResults(msg.results)

	case loadingMsg:
		m.loading = msg.loading

	case tokenUpdateMsg:
		// This is a polling tick to update the UI during streaming
		if m.loading {
			// Continue ticking while loading
			cmds = append(cmds, m.tickForTokenUpdates())
			cmds = append(cmds, m.spinner.Tick)
		}
		return m, tea.Batch(cmds...)
	}

	// Update view components (when implemented)
	// m.chatView, cmd = m.chatView.Update(msg)
	// cmds = append(cmds, cmd)

	// Keep spinner ticking while loading
	if m.loading {
		cmds = append(cmds, m.spinner.Tick)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model interface
func (m Model) View() string {
	if !m.ready {
		return "Loading CODA..."
	}

	var view strings.Builder

	// Toast notification (appears at top)
	if m.toast != nil && !m.toast.IsExpired() {
		view.WriteString(m.toast.Render())
		view.WriteString("\n")
	}

	// Error display (if there's an error)
	if m.error != nil && m.errorDisplay != nil {
		errorDisplay := m.errorDisplay.Render(m.width)
		view.WriteString(errorDisplay)
		view.WriteString("\n")
	}

	// Main content
	if m.showHelp {
		view.WriteString(m.renderHelp())
	} else {
		// Render viewport and scrollbar side by side
		chatView := m.renderChat()
		scrollbarView := m.renderScrollbar()

		// Split both views into lines
		chatLines := strings.Split(chatView, "\n")
		scrollbarLines := strings.Split(scrollbarView, "\n")

		// Combine lines horizontally
		var combined []string
		maxLines := len(chatLines)
		if len(scrollbarLines) > maxLines {
			maxLines = len(scrollbarLines)
		}

		for i := 0; i < maxLines; i++ {
			var chatLine, scrollbarLine string

			if i < len(chatLines) {
				chatLine = chatLines[i]
			}
			if i < len(scrollbarLines) {
				scrollbarLine = scrollbarLines[i]
			}

			// Combine the lines
			combined = append(combined, chatLine+scrollbarLine)
		}

		view.WriteString(strings.Join(combined, "\n"))
	}

	// Error banner for less critical errors
	if m.error != nil && m.errorBanner != nil {
		category := m.errorDisplay.ClassifyError(m.error)
		if category == errors.UserError || category == errors.ConfigError {
			userMessage := m.errorHandler.UserMessage(m.error)
			banner := m.errorBanner.Render(userMessage, m.width)
			view.WriteString("\n")
			view.WriteString(banner)
		}
	}

	// Error status (if any)
	if status := m.renderStatus(); status != "" {
		view.WriteString("\n")
		view.WriteString(status)
	}

	// Loading message (above input area)
	if loadingMsg := m.renderLoadingMessage(); loadingMsg != "" {
		view.WriteString("\n")
		view.WriteString(loadingMsg)
	}

	view.WriteString("\n")
	view.WriteString(m.renderInput())

	// Token usage display (right-aligned below input)
	if tokenUsage := m.renderTokenUsage(); tokenUsage != "" {
		view.WriteString("\n")
		view.WriteString(tokenUsage)
	}

	view.WriteString("\n")
	view.WriteString(m.renderHelpLine())

	return view.String()
}

// handleKeyPress handles keyboard input - SIMPLIFIED for IME support
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Debug: Log the actual key event
	m.logger.Debug("Key pressed", "key", key, "runes", msg.Runes, "type", msg.Type)

	// Also write to a debug file for TUI mode
	debugFile, _ := os.OpenFile("/tmp/coda-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] Key pressed: %s, runes: %v, type: %v\n", key, msg.Runes, msg.Type)
		debugFile.Close()
	}

	// Handle Permit mode keys first, before any other processing
	if m.currentMode == ModePermit {
		return m.handlePermitModeKeys(msg)
	}

	// Handle error-specific key bindings first (when error is displayed)
	if m.error != nil {
		switch key {
		case "enter", "esc":
			// Dismiss error
			return m, func() tea.Msg { return dismissErrorMsg{} }
		case "d":
			// Toggle error details
			return m, func() tea.Msg { return toggleErrorDetailsMsg{} }
		case "r":
			// Retry last action (if applicable)
			m.error = nil
			if m.errorDisplay != nil {
				m.errorDisplay.SetError(nil)
			}
			return m, func() tea.Msg { return retryLastActionMsg{} }
		case "q":
			// Quit
			return m, tea.Quit
		}
		// Ignore all other keys when error dialog is shown
		return m, nil
	}

	// Handle global keys
	switch key {
	case "ctrl+c":
		// Check if this is a double press within 1 second
		now := time.Now()
		if !m.lastCtrlCTime.IsZero() && now.Sub(m.lastCtrlCTime) < time.Second {
			// Second press within 1 second, quit
			return m, tea.Quit
		}
		// First press or too much time passed
		m.lastCtrlCTime = now
		m.ctrlCMessage = "終了するにはもう一度 Ctrl+C を押してください"
		// Clear message after 1 second
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return clearCtrlCMsg{}
		})
	case "f1":
		if !m.loading {
			m.showHelp = !m.showHelp
		}
		return m, nil
	case "enter":
		// Enter で送信
		if strings.TrimSpace(m.currentInput) != "" {
			return m.sendMessage()
		}
		return m, nil
	case "ctrl+j":
		// Ctrl+J (Shift+Enter in iTerm2) で改行を挿入
		m.insertTextAtCursor("\n")
		return m, nil
	case "backspace":
		if m.cursorPosition > 0 {
			runes := []rune(m.currentInput)
			m.currentInput = string(append(runes[:m.cursorPosition-1],
				runes[m.cursorPosition:]...))
			m.cursorPosition--
			m.updateCursorColumn()
		}
		return m, nil
	case "delete":
		runes := []rune(m.currentInput)
		if m.cursorPosition < len(runes) {
			m.currentInput = string(append(runes[:m.cursorPosition],
				runes[m.cursorPosition+1:]...))
		}
		return m, nil
	// カーソル移動
	case "left":
		if m.cursorPosition > 0 {
			m.cursorPosition--
			m.updateCursorColumn()
		}
		return m, nil
	case "right":
		runes := []rune(m.currentInput)
		if m.cursorPosition < len(runes) {
			m.cursorPosition++
			m.updateCursorColumn()
		}
		return m, nil
	case "up":
		m.cursorPosition = m.moveCursorUp()
		return m, nil
	case "down":
		m.cursorPosition = m.moveCursorDown()
		return m, nil
	case "home":
		m.cursorPosition = m.moveToLineStart()
		m.cursorColumn = 0
		return m, nil
	case "end":
		m.cursorPosition = m.moveToLineEnd()
		m.updateCursorColumn()
		return m, nil
	case "ctrl+a":
		// 全体の先頭へ
		m.cursorPosition = 0
		m.cursorColumn = 0
		return m, nil
	case "ctrl+e":
		// 全体の末尾へ
		runes := []rune(m.currentInput)
		m.cursorPosition = len(runes)
		m.updateCursorColumn()
		return m, nil
	case "esc":
		// Check if this is a double press within 1 second
		now := time.Now()
		if !m.lastEscTime.IsZero() && now.Sub(m.lastEscTime) < time.Second {
			// Second press within 1 second, clear input
			m.currentInput = ""
			m.cursorPosition = 0
			m.cursorColumn = 0
			m.escMessage = ""
			m.lastEscTime = time.Time{}
			return m, nil
		}
		// First press or too much time passed
		m.lastEscTime = now
		m.escMessage = "Press Esc again to clear textarea"
		// Clear message after 1 second
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return clearEscMsg{}
		})
	case "ctrl+n":
		// Check if this is a double press within 1 second
		now := time.Now()
		if !m.lastCtrlNTime.IsZero() && now.Sub(m.lastCtrlNTime) < time.Second {
			// Second press within 1 second, create new session
			m.messages = make([]Message, 0)
			m.currentInput = ""
			m.cursorPosition = 0
			m.cursorColumn = 0
			m.error = nil
			m.loading = false
			m.streamingContent.Reset()
			m.lastTokenUsage = nil
			m.estimatedTokens = 0
			m.userInputTokens = 0
			m.ctrlNMessage = ""
			m.lastCtrlNTime = time.Time{}
			// Create a new session in chat handler
			if m.chatHandler != nil {
				if err := m.chatHandler.CreateNewSession(); err != nil {
					m.logger.Error("Failed to create new session", "error", err)
				}
			}
			// Update viewport to show welcome message
			m.updateViewportContent()
			return m, nil
		}
		// First press or too much time passed
		m.lastCtrlNTime = now
		m.ctrlNMessage = "Press Ctrl+N again for new session"
		// Clear message after 1 second
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return clearCtrlNMsg{}
		})
	}

	// Handle regular text input (including IME)
	if msg.Runes != nil && len(msg.Runes) > 0 {
		m.insertTextAtCursor(string(msg.Runes))
		return m, nil
	}

	// For single character input
	if len(key) == 1 {
		m.insertTextAtCursor(key)
		return m, nil
	}

	return m, nil
}

// handleKeyPress_OLD handles keyboard input based on current mode - DISABLED
func (m Model) handleKeyPress_OLD(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle error-specific key bindings first (when error is displayed)
	if m.error != nil {
		switch key {
		case "enter", "esc":
			// Dismiss error
			return m, func() tea.Msg { return dismissErrorMsg{} }
		case "d":
			// Toggle error details
			return m, func() tea.Msg { return toggleErrorDetailsMsg{} }
		case "r":
			// Retry last action (if applicable)
			m.error = nil
			if m.errorDisplay != nil {
				m.errorDisplay.SetError(nil)
			}
			return m, func() tea.Msg { return retryLastActionMsg{} }
		}
	}

	// Handle global key bindings (work in all modes)
	if m.keymap.IsMatch(key, m.keymap.Quit) {
		return m, tea.Quit
	}

	if m.keymap.IsMatch(key, m.keymap.Help) {
		m.showHelp = !m.showHelp
		return m, nil
	}

	if m.keymap.IsMatch(key, m.keymap.Clear) {
		m.messages = make([]Message, 0)
		return m, nil
	}

	// Handle mode-specific key bindings
	switch m.currentMode {
	case ModeNormal:
		return m.handleNormalModeKeys(msg)
	case ModeInsert:
		return m.handleInsertModeKeys(msg)
	case ModeCommand:
		return m.handleCommandModeKeys(msg)
	case ModeSearch:
		return m.handleSearchModeKeys(msg)
	case ModeScroll:
		return m.handleScrollModeKeys(msg)
	case ModePermit:
		return m.handlePermitModeKeys(msg)
	default:
		return m, nil
	}
}

// handleNormalModeKeys handles keys in normal mode (Vim-style)
func (m Model) handleNormalModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Mode transitions
	if m.keymap.IsMatch(key, m.keymap.Normal.InsertMode) {
		m.previousMode = m.currentMode
		m.currentMode = ModeInsert
		return m, nil
	}

	if m.keymap.IsMatch(key, m.keymap.Normal.CommandMode) {
		m.previousMode = m.currentMode
		m.currentMode = ModeCommand
		m.commandBuffer = ":"
		return m, nil
	}

	if m.keymap.IsMatch(key, m.keymap.Normal.SearchMode) {
		m.previousMode = m.currentMode
		m.currentMode = ModeSearch
		if key == "/" {
			m.searchBuffer = "/"
		} else {
			m.searchBuffer = "?"
		}
		return m, nil
	}

	// Chat actions
	if m.keymap.IsMatch(key, m.keymap.Normal.SendMessage) {
		if m.currentInput != "" {
			return m.sendMessage()
		}
	}

	if m.keymap.IsMatch(key, m.keymap.Normal.NewChat) {
		m.messages = make([]Message, 0)
		m.currentInput = ""
		return m, nil
	}

	if m.keymap.IsMatch(key, m.keymap.Normal.ClearHistory) {
		m.messages = make([]Message, 0)
		return m, nil
	}

	// Navigation
	if m.keymap.IsMatch(key, m.keymap.ScrollUp) {
		// Implement scrolling logic here
		m.logger.Debug("Scroll up in normal mode")
	}

	if m.keymap.IsMatch(key, m.keymap.ScrollDown) {
		// Implement scrolling logic here
		m.logger.Debug("Scroll down in normal mode")
	}

	return m, nil
}

// handleInsertModeKeys handles keys in insert mode
func (m Model) handleInsertModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Exit insert mode
	if m.keymap.IsMatch(key, m.keymap.Insert.ExitMode) {
		m.currentMode = m.previousMode
		return m, nil
	}

	// Submit input
	if m.keymap.IsMatch(key, m.keymap.Insert.Enter) {
		if m.currentInput != "" {
			return m.sendMessage()
		}
		return m, nil
	}

	// Save and exit
	if m.keymap.IsMatch(key, m.keymap.Insert.SaveAndExit) {
		if m.currentInput != "" {
			newModel, cmd := m.sendMessage()
			if model, ok := newModel.(Model); ok {
				model.currentMode = ModeNormal
				return model, cmd
			}
			return newModel, cmd
		}
		m.currentMode = ModeNormal
		return m, nil
	}

	// Backspace
	if m.keymap.IsMatch(key, m.keymap.Insert.Backspace) {
		if len(m.currentInput) > 0 {
			m.currentInput = m.currentInput[:len(m.currentInput)-1]
		}
		return m, nil
	}

	// Delete
	if m.keymap.IsMatch(key, m.keymap.Insert.Delete) {
		// For now, same as backspace
		if len(m.currentInput) > 0 {
			m.currentInput = m.currentInput[:len(m.currentInput)-1]
		}
		return m, nil
	}

	// Force exit
	if m.keymap.IsMatch(key, m.keymap.Insert.ForceExit) {
		m.currentMode = ModeNormal
		return m, nil
	}

	// Add regular characters to input
	if len(key) == 1 && key != "\x00" {
		m.currentInput += key
	}

	return m, nil
}

// handleCommandModeKeys handles keys in command mode
func (m Model) handleCommandModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Exit command mode
	if m.keymap.IsMatch(key, m.keymap.Command.ExitMode) {
		m.currentMode = m.previousMode
		m.commandBuffer = ""
		return m, nil
	}

	// Execute command
	if m.keymap.IsMatch(key, m.keymap.Command.Execute) {
		cmd := m.executeCommand(m.commandBuffer[1:]) // Remove the ':'
		m.currentMode = m.previousMode
		m.commandBuffer = ""
		return m, cmd
	}

	// Clear command buffer
	if m.keymap.IsMatch(key, m.keymap.Command.Clear) {
		m.commandBuffer = ":"
		return m, nil
	}

	// Handle backspace
	if key == "backspace" {
		if len(m.commandBuffer) > 1 { // Keep the ':'
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
		return m, nil
	}

	// Add characters to command buffer
	if len(key) == 1 && key != "\x00" {
		m.commandBuffer += key
	}

	return m, nil
}

// handleSearchModeKeys handles keys in search mode
func (m Model) handleSearchModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Exit search mode
	if m.keymap.IsMatch(key, m.keymap.Search.ExitMode) {
		m.currentMode = m.previousMode
		m.searchBuffer = ""
		m.searchResults = make([]int, 0)
		return m, nil
	}

	// Execute search
	if m.keymap.IsMatch(key, m.keymap.Search.Execute) {
		m.performSearch(m.searchBuffer[1:]) // Remove the '/' or '?'
		m.currentMode = m.previousMode
		return m, nil
	}

	// Navigate search results
	if m.keymap.IsMatch(key, m.keymap.Search.NextMatch) {
		if len(m.searchResults) > 0 {
			m.currentMatch = (m.currentMatch + 1) % len(m.searchResults)
		}
		return m, nil
	}

	if m.keymap.IsMatch(key, m.keymap.Search.PrevMatch) {
		if len(m.searchResults) > 0 {
			m.currentMatch = (m.currentMatch - 1 + len(m.searchResults)) % len(m.searchResults)
		}
		return m, nil
	}

	// Handle backspace
	if key == "backspace" {
		if len(m.searchBuffer) > 1 { // Keep the '/' or '?'
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
		}
		return m, nil
	}

	// Add characters to search buffer
	if len(key) == 1 && key != "\x00" {
		m.searchBuffer += key
	}

	return m, nil
}

// handleScrollModeKeys handles keys in scroll mode
func (m Model) handleScrollModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Exit scroll mode with Esc or Ctrl+Y
	if key == "esc" || key == "ctrl+y" {
		m.currentMode = m.previousMode
		return m, nil
	}

	// In scroll mode, let viewport handle arrow keys
	// The actual scrolling is handled by viewport update in Update()
	switch key {
	case "up":
		m.viewport.LineUp(1)
	case "down":
		m.viewport.LineDown(1)
	case "pgup":
		m.viewport.ViewUp()
	case "pgdown":
		m.viewport.ViewDown()
	case "home":
		m.viewport.GotoTop()
	case "end":
		m.viewport.GotoBottom()
	}

	return m, nil
}

// handlePermitModeKeys handles keys in permit mode for tool call approval
func (m Model) handlePermitModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Exit permit mode with rejection
	if m.keymap.IsMatch(key, m.keymap.Permit.ExitMode) {
		return m.exitPermitMode(false) // false = reject
	}

	// Approve tool call
	if m.keymap.IsMatch(key, m.keymap.Permit.Approve) {
		return m.exitPermitMode(true) // true = approve
	}

	// Reject tool call
	if m.keymap.IsMatch(key, m.keymap.Permit.Reject) {
		return m.exitPermitMode(false) // false = reject
	}

	// Move selection left (reject)
	if m.keymap.IsMatch(key, m.keymap.Permit.SelectPrev) {
		m.selectedPermitOption = 0 // 0 = reject
		return m, nil
	}

	// Move selection right (approve)
	if m.keymap.IsMatch(key, m.keymap.Permit.SelectNext) {
		m.selectedPermitOption = 1 // 1 = approve
		return m, nil
	}

	return m, nil
}

// exitPermitMode exits permit mode and handles the tool call decision
func (m *Model) exitPermitMode(approved bool) (tea.Model, tea.Cmd) {
	// Reset permit dialog state
	m.permitDialogVisible = false
	toolCalls := m.pendingToolCalls
	m.pendingToolCalls = make([]ai.ToolCall, 0)
	m.selectedPermitOption = 0

	// Return to previous mode
	m.currentMode = m.previousMode

	if approved {
		m.logger.Debug("Tool calls approved", "count", len(toolCalls))
		// Execute tool calls and send results back to LLM
		return m, m.executeToolCalls(toolCalls)
	} else {
		// Tool calls rejected
		m.logger.Debug("Tool calls rejected", "count", len(toolCalls))
		m.messages = append(m.messages, Message{
			ID:        generateMessageID(),
			Content:   "Tool calls rejected by user",
			Role:      "system",
			Timestamp: time.Now(),
			Tokens:    0,
		})
		// Update viewport with rejection message
		m.updateViewportContent()
	}

	return m, nil
}

// sendMessage sends the current input as a chat message
func (m *Model) sendMessage() (tea.Model, tea.Cmd) {
	// Trim whitespace and check if empty
	trimmedInput := strings.TrimSpace(m.currentInput)
	if trimmedInput == "" {
		return m, nil
	}

	// Estimate tokens for the user message (for display in message list)
	estimatedTokens := 0
	if m.config != nil && m.config.AI.Model != "" {
		if tokens, err := EstimateUserMessageTokens(trimmedInput, m.config.AI.Model); err == nil {
			estimatedTokens = tokens
		} else {
			m.logger.Debug("Failed to estimate user message tokens", "error", err)
		}
	}

	// Save user input tokens for display
	m.userInputTokens = estimatedTokens

	// Estimate total prompt tokens (for display during thinking)
	if m.chatHandler != nil {
		if promptTokens, err := m.chatHandler.EstimatePromptTokens(trimmedInput); err == nil {
			m.estimatedTokens = promptTokens
		} else {
			// Fallback to just user message tokens
			m.estimatedTokens = estimatedTokens
			m.logger.Debug("Failed to estimate prompt tokens", "error", err)
		}
	} else {
		m.estimatedTokens = estimatedTokens
	}

	// Add user message with token count
	userMsg := Message{
		ID:        generateMessageID(),
		Content:   trimmedInput,
		Role:      "user",
		Timestamp: time.Now(),
		Tokens:    estimatedTokens,
	}
	m.messages = append(m.messages, userMsg)
	// Update viewport content with new message
	m.updateViewportContent()

	// Clear input and reset cursor
	m.currentInput = ""
	m.cursorPosition = 0
	m.cursorColumn = 0
	m.loading = true
	m.loadingStart = time.Now()
	m.error = nil
	// Reset streaming state
	m.streamingContent.Reset()

	// Send to chat handler
	return m, tea.Batch(
		m.spinner.Tick,
		m.streamChatResponse(trimmedInput),
		m.tickForTokenUpdates(), // Poll for token updates during streaming
	)
}

// tickForTokenUpdates polls for token updates during streaming
func (m Model) tickForTokenUpdates() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tokenUpdateMsg{receivedTokens: -1} // Special value to trigger a check
	})
}

// streamChatResponse handles the streaming chat response
func (m *Model) streamChatResponse(input string) tea.Cmd {
	return func() tea.Msg {
		// Call handler without token callback since we're using ChatHandler's internal state
		response, err := m.chatHandler.HandleMessageWithResponse(m.ctx, input, nil)

		if err != nil {
			return errorMsg{
				error:      err,
				userAction: "sending message",
				metadata:   map[string]interface{}{"message": input},
			}
		}

		// Return the complete response
		return chatResponseMsg{
			ID:         generateMessageID(),
			Content:    response.Content,
			Tokens:     response.TokenCount,
			TokenUsage: response.TokenUsage,
			ToolCalls:  response.ToolCalls,
		}
	}
}

// updateViewportContent updates the viewport with chat messages
func (m *Model) updateViewportContent() {
	var content strings.Builder

	// Always show header (CODA figlet + model info) at the top
	content.WriteString(m.renderHeader())
	content.WriteString("\n")

	if len(m.messages) == 0 {
		// Show welcome message if no messages
		content.WriteString(m.renderWelcomeMessage())
		m.viewport.SetContent(content.String())
		return
	}

	// Show chat messages
	for _, msg := range m.messages {
		// Format the message with timestamp and role
		msgLine := fmt.Sprintf("[%s] %s: %s",
			msg.Timestamp.Format("15:04"),
			msg.Role,
			msg.Content)

		content.WriteString(msgLine)
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
	// Auto-scroll to bottom when new content is added
	m.viewport.GotoBottom()
}

// renderChat renders the chat view using viewport
func (m Model) renderChat() string {
	return m.viewport.View()
}

// renderLoadingMessage renders the loading message for display above input
func (m Model) renderLoadingMessage() string {
	if !m.loading {
		return ""
	}

	elapsed := time.Since(m.loadingStart)

	// Determine the status message based on streaming tokens
	statusMsg := "Thinking..."
	if m.chatHandler != nil && m.chatHandler.GetStreamingTokens() >= 1 {
		statusMsg = "Answering..."
	}

	// Build the loading message
	loadingMsg := fmt.Sprintf("%s %s (%s)",
		m.spinner.View(),
		statusMsg,
		formatDuration(elapsed))

	// Add token information if available
	if m.userInputTokens > 0 {
		/// DO NOT CHANGE '≈' TO '~'
		loadingMsg += fmt.Sprintf(" | Send: ≈%d tokens", m.userInputTokens)
	}

	// Add streaming token count if receiving
	if m.chatHandler != nil {
		currentStreamingTokens := m.chatHandler.GetStreamingTokens()

		if currentStreamingTokens > 0 {
			// DO NOT CHANGE '≈' TO '~'
			loadingMsg += fmt.Sprintf(" | Receive: ≈%d tokens", currentStreamingTokens)
		}
	}

	return loadingMsg
}

// renderScrollbar renders a vertical scrollbar for the viewport
func (m Model) renderScrollbar() string {
	height := m.viewport.Height

	// Don't render scrollbar if content fits in viewport
	if m.viewport.TotalLineCount() <= m.viewport.VisibleLineCount() {
		// Return empty scrollbar track
		var bar strings.Builder
		for i := 0; i < height; i++ {
			bar.WriteString(" ")
			if i < height-1 {
				bar.WriteString("\n")
			}
		}
		return bar.String()
	}

	// Calculate scrollbar position and size
	scrollPercent := m.viewport.ScrollPercent()
	totalLines := m.viewport.TotalLineCount()
	visibleLines := m.viewport.VisibleLineCount()

	// Calculate thumb size (minimum 1 line)
	thumbSize := int(float64(height) * float64(visibleLines) / float64(totalLines))
	if thumbSize < 1 {
		thumbSize = 1
	}

	// Calculate thumb position
	thumbPosition := int(float64(height-thumbSize) * scrollPercent)

	// Build scrollbar
	var scrollbar strings.Builder
	scrollbarStyle := m.styles.ScrollbarTrack
	thumbStyle := m.styles.ScrollbarThumb

	for i := 0; i < height; i++ {
		if i >= thumbPosition && i < thumbPosition+thumbSize {
			// Render thumb
			scrollbar.WriteString(thumbStyle.Render("█"))
		} else {
			// Render track
			scrollbar.WriteString(scrollbarStyle.Render("│"))
		}
		if i < height-1 {
			scrollbar.WriteString("\n")
		}
	}

	return scrollbar.String()
}

// renderHeader renders the header with border
func (m Model) renderHeader() string { // Create header content ( DO NOT format below figlet )
	figlet := ` ▄████████  ▄██████▄  ████████▄     ▄████████
███    ███ ███    ███ ███   ▀███   ███    ███
███    █▀  ███    ███ ███    ███   ███    ███
███        ███    ███ ███    ███   ███    ███
███        ███    ███ ███    ███ ▀███████████
███    █▄  ███    ███ ███    ███   ███    ███
███    ███ ███    ███ ███   ▄███   ███    ███
████████▀   ▀██████▀  ████████▀    ███    █▀
`

	// Split figlet into lines
	lines := strings.Split(strings.TrimSpace(figlet), "\n")

	// Define gradient colors from light to dark red
	// Starting color: #ff6b7d (light red)
	// Ending color: #b40028 (corporate color)
	gradientColors := []string{
		"#ff6b7d", // Lightest
		"#f55a6e",
		"#eb495f",
		"#e13850",
		"#d72741",
		"#cd1632",
		"#c30529",
		"#b40028", // Corporate color (darkest)
	}

	// Apply gradient to each line
	var styledLines []string
	for i, line := range lines {
		// Create style for this line with gradient color
		lineStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(gradientColors[i])).
			Bold(true)

		styledLines = append(styledLines, lineStyle.Render(line))
	}

	// Join the styled lines
	content := strings.Join(styledLines, "\n")

	// Apply container style (padding, etc.) but not color
	containerStyle := m.styles.Header.
		Foreground(lipgloss.NoColor{}) // Remove foreground color from container

	return containerStyle.Render(content + "\n")
}

// renderWelcomeMessage renders the welcome message box
func (m Model) renderWelcomeMessage() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	// Get model name from config
	modelName := "unknown"
	if m.config != nil && m.config.AI.Model != "" {
		modelName = m.config.AI.Model
	}

	// Create welcome message content with more space
	lines := []string{
		" ∂ Welcome to 𝑪𝑶𝑫𝑨!",
		"",
		fmt.Sprintf("   model: %s", modelName),
		fmt.Sprintf("   cwd: %s", cwd),
	}
	content := strings.Join(lines, "\n")

	// Use the same style as input area
	style := m.styles.UserInput

	// Calculate width
	contentWidth := len(cwd) + 4 + 10
	if m.width-4 < contentWidth {
		contentWidth = m.width - 4
	}
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Return styled welcome message with padding
	return style.Width(contentWidth).Padding(1, 2).Render(content)
}

// renderStatus renders the status bar
func (m Model) renderStatus() string {
	if m.error != nil {
		return fmt.Sprintf("Error: %s", m.error.Error())
	}
	return ""
}

// renderHelpLine renders the help line
func (m Model) renderHelpLine() string {
	if m.currentMode == ModeScroll {
		return " Arrows:scroll, Home/End:top/bottom, Esc/Ctrl+Y:return to input"
	}
	if m.currentMode == ModePermit {
		return " Left/Right:select, Enter:confirm, Esc:reject"
	}
	if m.ctrlCMessage != "" {
		// Show warning when Ctrl+C was pressed once
		return " Enter:send, Ctrl+J:newline, Ctrl+N:new session, Esc:clear textarea, Ctrl+Y:scroll, F1:help, Press Ctrl+C again to quit"
	}
	if m.escMessage != "" {
		// Show warning when Esc was pressed once
		return " Enter:send, Ctrl+J:newline, Ctrl+N:new session, Press Esc again to clear textarea, Ctrl+Y:scroll, F1:help, Ctrl+C:quit"
	}
	if m.ctrlNMessage != "" {
		// Show warning when Ctrl+N was pressed once
		return " Enter:send, Ctrl+J:newline, Press Ctrl+N again for new session, Esc:clear textarea, Ctrl+Y:scroll, F1:help, Ctrl+C:quit"
	}
	return " Enter:send, Ctrl+J:newline, Ctrl+N:new session, Esc:clear textarea, Ctrl+Y:scroll, F1:help, Ctrl+C:quit"
}

// renderTokenUsage renders the token usage indicator
func (m Model) renderTokenUsage() string {
	if m.config == nil || m.config.AI.Model == "" {
		return ""
	}

	modelName := m.config.AI.Model
	tokenLimit := getModelTokenLimit(modelName)
	usedTokens := m.calculateSessionTokens()

	// Calculate usage percentage
	usagePercent := float64(usedTokens) / float64(tokenLimit) * 100

	// Format the usage string
	// DO NOT CHANGE '≈' TO '~'
	usageStr := fmt.Sprintf("Context usage: ≈%d / %d (%.1f%%)", usedTokens, tokenLimit, usagePercent)

	// Apply color based on usage
	var style lipgloss.Style
	if usagePercent >= 90 {
		// Red for high usage
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	} else if usagePercent >= 70 {
		// Yellow for medium usage
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	} else {
		// Green for low usage
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	}

	// Right-align the usage display
	totalWidth := m.width - 2
	if totalWidth < len(usageStr) {
		return style.Render(usageStr)
	}

	padding := totalWidth - len(usageStr)
	return strings.Repeat(" ", padding) + style.Render(usageStr)
}

// renderInput renders the input area
func (m Model) renderInput() string {
	var content string

	switch m.currentMode {
	case ModeCommand:
		content = fmt.Sprintf("%s_", m.commandBuffer)
	case ModeSearch:
		content = fmt.Sprintf("%s_", m.searchBuffer)
	case ModeInsert, ModeScroll:
		return m.renderMultilineInput()
	case ModePermit:
		return m.renderPermitDialog()
	case ModeNormal:
		if m.currentInput != "" {
			content = fmt.Sprintf("> %s", m.currentInput)
		} else {
			content = "Press 'i' to enter insert mode, ':' for commands, '/' to search"
		}
	default:
		return m.renderMultilineInput()
	}

	// ModeCommand, ModeSearch, ModeNormalの場合は罫線で囲む
	style := m.styles.UserInput

	// ターミナル幅に合わせて調整
	contentWidth := m.width - 4 // ボーダーとパディング分を引く
	if contentWidth < 20 {
		contentWidth = 20 // 最小幅
	}

	return style.Width(contentWidth).Render(content)
}

// renderMultilineInput renders the input area with multiline support
func (m Model) renderMultilineInput() string {
	lines := strings.Split(m.currentInput, "\n")

	// カーソル位置を行と列に変換
	cursorLine, cursorCol := m.getCursorLineAndColumn()

	// 設定から表示行数を取得（0の場合は無制限）
	displayLimit := m.config.UI.InputDisplayLines

	// 入力内容を構築
	var content string

	// 単一行の場合の特別処理
	if len(lines) == 1 {
		lineRunes := []rune(lines[0])
		if cursorCol < len(lineRunes) {
			// カーソルが文字列の途中にある場合
			before := string(lineRunes[:cursorCol])
			cursorChar := string(lineRunes[cursorCol])
			after := string(lineRunes[cursorCol+1:])
			// カーソル位置の文字を背景色反転で表示
			content = fmt.Sprintf("> %s%s%s", before, m.cursorStyle.Render(cursorChar), after)
		} else {
			// カーソルが行末にある場合
			content = fmt.Sprintf("> %s▉", lines[0])
		}

		// 罫線で囲む（モードに応じて色を変更）
		style := m.styles.UserInput
		if m.currentMode == ModeInsert {
			// Inputモードの場合はコーポレートカラー
			style = style.BorderForeground(lipgloss.Color("#b40028"))
		}
		// Scrollモードその他の場合はデフォルトのグレー

		// ターミナル幅に合わせて調整（左右のパディングとボーダーを考慮）
		contentWidth := m.width - 4 // ボーダーとパディング分を引く
		if contentWidth < 20 {
			contentWidth = 20 // 最小幅
		}

		return style.Width(contentWidth).Render(content)
	}

	// 表示する行の範囲を決定
	displayLines := lines
	startLine := 0

	if displayLimit > 0 && len(lines) > displayLimit {
		// カーソルが表示範囲内に収まるように調整
		if cursorLine >= displayLimit {
			startLine = cursorLine - displayLimit + 1
		}
		displayLines = lines[startLine : startLine+displayLimit]
	}

	result := ""
	for i, line := range displayLines {
		actualLine := startLine + i
		prefix := "  "
		if i == len(displayLines)-1 && actualLine == len(lines)-1 {
			prefix = "> "
		}

		if actualLine == cursorLine {
			// カーソルがある行
			lineRunes := []rune(line)
			if cursorCol < len(lineRunes) {
				// カーソルが文字列の途中にある場合
				before := string(lineRunes[:cursorCol])
				cursorChar := string(lineRunes[cursorCol])
				after := string(lineRunes[cursorCol+1:])
				// カーソル位置の文字を背景色反転で表示
				result += fmt.Sprintf("%s%s%s%s\n", prefix, before, m.cursorStyle.Render(cursorChar), after)
			} else {
				// カーソルが行末にある場合
				result += fmt.Sprintf("%s%s▉\n", prefix, line)
			}
		} else {
			result += fmt.Sprintf("%s%s\n", prefix, line)
		}
	}

	// 最後の改行を削除
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	// 省略された行がある場合は表示
	if startLine > 0 {
		result = fmt.Sprintf("  ... (%d more lines above)\n%s", startLine, result)
	}
	if displayLimit > 0 && len(lines) > startLine+displayLimit {
		result = fmt.Sprintf("%s\n  ... (%d more lines below)", result,
			len(lines)-startLine-displayLimit)
	}

	// 罫線で囲む（モードに応じて色を変更）
	style := m.styles.UserInput
	if m.currentMode == ModeInsert {
		// Inputモードの場合はコーポレートカラー
		style = style.BorderForeground(lipgloss.Color("#b40028"))
	}
	// Scrollモードその他の場合はデフォルトのグレー

	// ターミナル幅に合わせて調整
	contentWidth := m.width - 4 // ボーダーとパディング分を引く
	if contentWidth < 20 {
		contentWidth = 20 // 最小幅
	}

	return style.Width(contentWidth).Render(result)
}

// renderPermitDialog renders the tool call permission dialog
func (m Model) renderPermitDialog() string {
	if !m.permitDialogVisible || len(m.pendingToolCalls) == 0 {
		return m.renderMultilineInput() // Fallback to normal input
	}

	var dialogContent strings.Builder

	// Dialog title
	dialogContent.WriteString("🔧 Tool Call Permission Required\n\n")

	// Show tool details
	for i, toolCall := range m.pendingToolCalls {
		if i > 0 {
			dialogContent.WriteString("\n")
		}
		dialogContent.WriteString(fmt.Sprintf("Tool %d: %s\n", i+1, toolCall.Function.Name))

		// Format and show arguments
		formattedArgs := m.formatToolArguments(toolCall.Function.Arguments)
		dialogContent.WriteString(fmt.Sprintf("Arguments:\n%s\n", formattedArgs))
	}

	dialogContent.WriteString("\n")

	// Render selection buttons
	rejectStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241"))

	approveStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241"))

	// Highlight selected option
	if m.selectedPermitOption == 0 {
		// Reject is selected
		rejectStyle = rejectStyle.
			BorderForeground(lipgloss.Color("9")).
			Foreground(lipgloss.Color("9")).
			Bold(true)
	} else {
		// Approve is selected
		approveStyle = approveStyle.
			BorderForeground(lipgloss.Color("10")).
			Foreground(lipgloss.Color("10")).
			Bold(true)
	}

	rejectButton := rejectStyle.Render("Deny")
	approveButton := approveStyle.Render("Allow")

	// Combine buttons horizontally
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, rejectButton, "  ", approveButton)
	dialogContent.WriteString(buttons)

	// Apply dialog styling
	dialogStyle := m.styles.UserInput.
		BorderForeground(lipgloss.Color("#b40028")). // Corporate color for attention
		Padding(1, 2)

	// Calculate content width
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	return dialogStyle.Width(contentWidth).Render(dialogContent.String())
}

// formatToolArguments formats JSON arguments in a readable key-value format
func (m Model) formatToolArguments(args string) string {
	if args == "" {
		return "  (no arguments)"
	}

	// Try to parse as JSON and format nicely
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(args), &jsonData); err != nil {
		// If not valid JSON, just return the raw string (truncated if too long)
		if len(args) > 200 {
			return fmt.Sprintf("  %s...", args[:200])
		}
		return fmt.Sprintf("  %s", args)
	}

	// Format as key-value pairs
	var formatted strings.Builder
	for key, value := range jsonData {
		// Format the value based on its type
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = fmt.Sprintf("\"%s\"", v)
		case bool:
			valueStr = fmt.Sprintf("%t", v)
		case float64:
			// Check if it's actually an integer
			if v == float64(int64(v)) {
				valueStr = fmt.Sprintf("%.0f", v)
			} else {
				valueStr = fmt.Sprintf("%g", v)
			}
		default:
			// For complex types, use JSON marshaling
			if jsonBytes, err := json.Marshal(v); err == nil {
				valueStr = string(jsonBytes)
			} else {
				valueStr = fmt.Sprintf("%v", v)
			}
		}

		formatted.WriteString(fmt.Sprintf("  %s: %s\n", key, valueStr))
	}

	result := formatted.String()
	if len(result) > 0 {
		// Remove the last newline
		result = result[:len(result)-1]
	}

	return result
}

// renderHelp renders the help view
func (m Model) renderHelp() string {
	help := "CODA Help - Advanced Key Bindings\n"
	help += "==================================\n\n"

	// Get help text from keymap
	helpLines := m.keymap.GetHelpText(m.currentMode)
	for _, line := range helpLines {
		help += line + "\n"
	}

	help += "\nAdvanced Features:\n"
	help += "- Vim-style modes: Normal, Insert, Command, Search\n"
	help += "- Customizable key bindings via configuration\n"
	help += "- Context-sensitive help based on current mode\n"
	help += "- Search through chat history with highlighting\n"
	help += "- Command mode for advanced operations\n\n"

	help += "Configuration:\n"
	help += "- Supports Vim, Emacs, and Default key binding styles\n"
	help += "- Custom key bindings can be defined in config file\n"
	help += "- Key conflict detection and validation\n\n"

	help += "Press F1 again to return to chat\n"
	return help
}

// SaveState saves the current model state
func (m Model) SaveState() error {
	// This would save the current state to disk
	// For now, just log
	m.logger.Info("Saving model state", "messages", len(m.messages))
	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// Message types for Bubbletea
type readyMsg struct{}

type chatResponseMsg struct {
	ID         string
	Content    string
	Tokens     int           // Total tokens (deprecated)
	TokenUsage *ai.Usage     // Detailed token usage
	ToolCalls  []ai.ToolCall // Tool calls requested by AI
}

type errorMsg struct {
	error      error
	userAction string
	metadata   map[string]interface{}
}

type dismissErrorMsg struct{}

// tokenUpdateMsg is sent during streaming to update token count
type tokenUpdateMsg struct {
	receivedTokens int // Current number of tokens received
}

type toggleErrorDetailsMsg struct{}

type retryLastActionMsg struct{}

type loadingMsg struct {
	loading bool
}

// clearCtrlCMsg is sent to clear the Ctrl+C warning message
type clearCtrlCMsg struct{}

// clearEscMsg is sent to clear the Esc warning message
type clearEscMsg struct{}

// clearCtrlNMsg is sent to clear the Ctrl+N warning message
type clearCtrlNMsg struct{}

// toolExecutionMsg is sent when tool execution is complete
type toolExecutionMsg struct {
	results []chat.ToolResult
}

// executeCommand executes a command mode command
func (m *Model) executeCommand(command string) tea.Cmd {
	m.logger.Debug("Executing command", "command", command)

	switch command {
	case "q", "quit":
		return tea.Quit
	case "h", "help":
		m.showHelp = !m.showHelp
	case "clear":
		m.messages = make([]Message, 0)
	case "new":
		m.messages = make([]Message, 0)
		m.currentInput = ""
	default:
		m.error = fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

// executeToolCalls executes the approved tool calls and returns a command to send results back to LLM
func (m *Model) executeToolCalls(toolCalls []ai.ToolCall) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		results := make([]chat.ToolResult, 0, len(toolCalls))

		for _, toolCall := range toolCalls {
			startTime := time.Now()

			// Parse tool call arguments
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
				// Failed to parse arguments
				results = append(results, chat.ToolResult{
					ToolCallID: toolCall.ID,
					ToolName:   toolCall.Function.Name,
					Error:      fmt.Errorf("failed to parse tool arguments: %w", err),
					ExecutedAt: time.Now(),
					Duration:   time.Since(startTime),
				})
				continue
			}

			// Execute the tool
			result, err := m.toolManager.Execute(m.ctx, toolCall.Function.Name, params)
			results = append(results, chat.ToolResult{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Function.Name,
				Result:     result,
				Error:      err,
				ExecutedAt: time.Now(),
				Duration:   time.Since(startTime),
			})
		}

		return toolExecutionMsg{results: results}
	})
}

// sendToolResults sends tool execution results back to the LLM
func (m *Model) sendToolResults(results []chat.ToolResult) tea.Cmd {
	// Add tool results as messages to the session
	for _, result := range results {
		content := ""
		if result.Error != nil {
			content = fmt.Sprintf("Tool execution failed: %v", result.Error)
		} else if result.Result == nil {
			// Handle nil result explicitly
			content = "Tool executed successfully"
		} else {
			// Convert result to string
			switch v := result.Result.(type) {
			case string:
				content = v
			case []byte:
				content = string(v)
			default:
				if data, err := json.Marshal(v); err == nil {
					content = string(data)
				} else {
					content = fmt.Sprintf("%v", v)
				}
			}
		}

		// Ensure content is never empty
		if content == "" {
			content = "Tool executed successfully with empty result"
		}

		// Add tool result as user message with special formatting (text-based approach)
		toolResultText := fmt.Sprintf("TOOL_RESULT[%s]: %s", result.ToolName, content)
		message := ai.Message{
			Role:    ai.RoleUser,
			Content: toolResultText,
		}

		// Add message to current session
		if err := m.chatHandler.AddMessageToSession(message); err != nil {
			m.logger.Error("Failed to add tool result message", "error", err)
		}

		// Add to UI messages for display with brief summary
		briefSummary := m.getToolResultSummary(result)
		m.messages = append(m.messages, Message{
			ID:        generateMessageID(),
			Content:   briefSummary,
			Role:      "tool",
			Timestamp: result.ExecutedAt,
			Tokens:    0,
		})
	}

	// Update viewport with new messages
	m.updateViewportContent()

	// Set loading state for LLM response
	m.loading = true
	m.loadingStart = time.Now()
	m.streamingContent.Reset()

	// Send continuation request to LLM without adding new user message
	return tea.Cmd(func() tea.Msg {
		// Use ContinueConversation to continue with tool results
		response, err := m.chatHandler.ContinueConversation(m.ctx, nil)
		if err != nil {
			return errorMsg{
				error:      err,
				userAction: "send tool results",
			}
		}

		return chatResponseMsg{
			ID:         generateMessageID(),
			Content:    response.Content,
			Tokens:     response.TokenCount,
			TokenUsage: response.TokenUsage,
			ToolCalls:  response.ToolCalls,
		}
	})
}

// performSearch performs a search in the chat history
func (m *Model) performSearch(query string) {
	m.searchResults = make([]int, 0)
	m.currentMatch = 0

	if query == "" {
		return
	}

	// Search through messages
	for i, message := range m.messages {
		if strings.Contains(strings.ToLower(message.Content), strings.ToLower(query)) {
			m.searchResults = append(m.searchResults, i)
		}
	}

	m.logger.Debug("Search completed", "query", query, "results", len(m.searchResults))
}

// getCurrentModeString returns a string representation of the current mode for display
func (m Model) getCurrentModeString() string {
	switch m.currentMode {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	case ModeSearch:
		return "SEARCH"
	case ModeScroll:
		return "SCROLL"
	case ModePermit:
		return "PERMIT"
	default:
		return "UNKNOWN"
	}
}

// Testing support methods - these methods are used by E2E tests

// GetMessages returns the current messages (for testing)
func (m Model) GetMessages() []Message {
	return m.messages
}

// GetCurrentMode returns the current input mode (for testing)
func (m Model) GetCurrentMode() Mode {
	return m.currentMode
}

// GetActiveView returns the currently active view (for testing)
func (m Model) GetActiveView() ViewType {
	return m.activeView
}

// GetCurrentInput returns the current input text (for testing)
func (m Model) GetCurrentInput() string {
	return m.currentInput
}

// IsLoading returns whether the UI is in loading state (for testing)
func (m Model) IsLoading() bool {
	return m.loading
}

// getToolResultSummary returns a brief summary of tool execution result
func (m *Model) getToolResultSummary(result chat.ToolResult) string {
	toolName := result.ToolName

	// Handle error case
	if result.Error != nil {
		return fmt.Sprintf("[%s] ❌ Failed: %v", toolName, result.Error)
	}

	// Generate brief summary based on tool type
	switch toolName {
	case "read_file":
		// Extract filename from parameters if available
		if result.ToolCallID != "" {
			return fmt.Sprintf("[%s] ✅ File read successfully", toolName)
		}
		return fmt.Sprintf("[%s] ✅ Completed", toolName)

	case "write_file", "edit_file":
		return fmt.Sprintf("[%s] ✅ File modified successfully", toolName)

	case "list_files":
		// Try to count files if result is a slice
		if files, ok := result.Result.([]interface{}); ok {
			return fmt.Sprintf("[%s] ✅ Found %d items", toolName, len(files))
		}
		return fmt.Sprintf("[%s] ✅ Directory listed", toolName)

	case "search_files":
		// Try to count search results
		if results, ok := result.Result.(map[string]interface{}); ok {
			return fmt.Sprintf("[%s] ✅ Found matches in %d files", toolName, len(results))
		}
		return fmt.Sprintf("[%s] ✅ Search completed", toolName)

	default:
		return fmt.Sprintf("[%s] ✅ Completed", toolName)
	}
}

// GetError returns the current error state (for testing)
func (m Model) GetError() error {
	return m.error
}

// GetSearchResults returns current search results (for testing)
func (m Model) GetSearchResults() []int {
	return m.searchResults
}

// GetCommandBuffer returns the current command buffer (for testing)
func (m Model) GetCommandBuffer() string {
	return m.commandBuffer
}

// GetSearchBuffer returns the current search buffer (for testing)
func (m Model) GetSearchBuffer() string {
	return m.searchBuffer
}

// IsHelpVisible returns whether help is currently visible (for testing)
func (m Model) IsHelpVisible() bool {
	return m.showHelp
}

// Helper functions
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// insertTextAtCursor inserts text at current cursor position
func (m *Model) insertTextAtCursor(text string) {
	runes := []rune(m.currentInput)
	textRunes := []rune(text)

	// カーソル位置に挿入
	newRunes := make([]rune, 0, len(runes)+len(textRunes))
	newRunes = append(newRunes, runes[:m.cursorPosition]...)
	newRunes = append(newRunes, textRunes...)
	newRunes = append(newRunes, runes[m.cursorPosition:]...)

	m.currentInput = string(newRunes)
	m.cursorPosition += len(textRunes)
	m.updateCursorColumn()
}

// updateCursorColumn updates the cursor column based on current position
func (m *Model) updateCursorColumn() {
	runes := []rune(m.currentInput)
	col := 0
	for i := 0; i < m.cursorPosition && i < len(runes); i++ {
		if runes[i] == '\n' {
			col = 0
		} else {
			col++
		}
	}
	m.cursorColumn = col
}

// moveToLineStart moves cursor to the start of current line
func (m Model) moveToLineStart() int {
	runes := []rune(m.currentInput)
	pos := m.cursorPosition

	// 現在位置から逆方向に改行を探す
	for pos > 0 && pos <= len(runes) {
		if pos > 0 && runes[pos-1] == '\n' {
			break
		}
		pos--
	}

	return pos
}

// moveToLineEnd moves cursor to the end of current line
func (m Model) moveToLineEnd() int {
	runes := []rune(m.currentInput)
	pos := m.cursorPosition

	// 現在位置から順方向に改行を探す
	for pos < len(runes) && runes[pos] != '\n' {
		pos++
	}

	return pos
}

// moveCursorUp moves cursor up one line
func (m Model) moveCursorUp() int {
	runes := []rune(m.currentInput)

	// 現在の行の先頭を見つける
	lineStart := m.moveToLineStart()

	// 既に最初の行にいる場合
	if lineStart == 0 {
		return 0
	}

	// 前の行の先頭を見つける
	prevLineEnd := lineStart - 1
	prevLineStart := prevLineEnd
	for prevLineStart > 0 && runes[prevLineStart-1] != '\n' {
		prevLineStart--
	}

	// 前の行での同じ列位置を計算
	prevLineLength := prevLineEnd - prevLineStart
	targetCol := m.cursorColumn
	if targetCol > prevLineLength {
		targetCol = prevLineLength
	}

	return prevLineStart + targetCol
}

// moveCursorDown moves cursor down one line
func (m Model) moveCursorDown() int {
	runes := []rune(m.currentInput)

	// 現在の行の末尾を見つける
	lineEnd := m.moveToLineEnd()

	// 既に最後の行にいる場合
	if lineEnd >= len(runes) {
		return m.cursorPosition
	}

	// 次の行の先頭
	nextLineStart := lineEnd + 1

	// 次の行での同じ列位置を計算
	targetCol := m.cursorColumn
	pos := nextLineStart
	col := 0

	for pos < len(runes) && runes[pos] != '\n' && col < targetCol {
		pos++
		col++
	}

	return pos
}

// getCursorLineAndColumn converts cursor position to line and column
func (m Model) getCursorLineAndColumn() (int, int) {
	runes := []rune(m.currentInput)
	line := 0
	col := 0

	for i := 0; i < m.cursorPosition && i < len(runes); i++ {
		if runes[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}

	return line, col
}

// getModelTokenLimit returns the token limit for the given model
func getModelTokenLimit(model string) int {
	// o-series models (o1, o3, etc.) have 200k context
	if strings.HasPrefix(model, "o") {
		return 200000
	}

	// GPT-4.1 models (gpt-4.1, gpt-4.1 mini) have 1M context
	if strings.HasPrefix(model, "gpt-4.1") {
		return 1000000
	}

	// GPT-4 Turbo and newer models or 4-omni
	if strings.Contains(model, "gpt-4-turbo") || strings.Contains(model, "gpt-4") || strings.HasPrefix(model, "gpt-4o") {
		return 128000
	}

	// GPT-4 (older versions)
	if strings.Contains(model, "gpt-4-32k") {
		return 32768
	}
	if strings.Contains(model, "gpt-4") {
		return 8192
	}

	// GPT-3.5 Turbo
	if strings.Contains(model, "gpt-3.5-turbo-16k") {
		return 16384
	}
	if strings.Contains(model, "gpt-3.5-turbo") {
		return 4096
	}

	// Default for unknown models
	return 8192
}

// calculateSessionTokens calculates the total token usage for the current session
func (m Model) calculateSessionTokens() int {
	totalTokens := 0

	// Calculate actual system prompt tokens
	if m.chatHandler != nil {
		systemPrompt := m.chatHandler.GetSystemPrompt()
		if systemPrompt != "" && m.config != nil && m.config.AI.Model != "" {
			// Use tokenizer for accurate system prompt token count
			systemTokens, err := EstimateUserMessageTokens(systemPrompt, m.config.AI.Model)
			if err != nil {
				panic(err)
				// Fallback to rough estimate on error
				systemTokens = 800
			}
			totalTokens += systemTokens
		} else {
			// Fallback if no handler or config available
			totalTokens += 800
		}
	} else {
		// Fallback if no handler available
		totalTokens += 800
	}

	// Add up tokens from all messages
	for _, msg := range m.messages {
		if msg.Tokens > 0 {
			totalTokens += msg.Tokens
		}
	}

	// Add the current estimated tokens if loading
	if m.loading && m.estimatedTokens > 0 {
		totalTokens += m.estimatedTokens
	}

	// Add streaming tokens if available
	if m.loading && m.chatHandler != nil {
		streamingTokens := m.chatHandler.GetStreamingTokens()
		if streamingTokens > 0 {
			totalTokens += streamingTokens
		}
	}

	return totalTokens
}
