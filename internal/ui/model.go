package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

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

	// Styles
	styles styles.Styles

	// Input mode state - Always INSERT mode for IME support
	currentMode   Mode
	previousMode  Mode
	commandBuffer string
	searchBuffer  string
	searchResults []int // indices of matching messages
	currentMatch  int

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
}

// ModelOptions contains options for creating a new Model
type ModelOptions struct {
	Config       *config.Config
	ChatHandler  *chat.ChatHandler
	ToolManager  *tools.Manager
	Logger       *log.Logger
	Context      context.Context
	ErrorHandler *errors.ErrorHandler
}

// NewModel creates a new UI model
func NewModel(opts ModelOptions) Model {
	// Initialize styles based on config theme
	themeName := "default"
	if opts.Config != nil && opts.Config.UI.Theme != "" {
		themeName = opts.Config.UI.Theme
	}

	theme := styles.GetTheme(themeName)

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

		// Initialize styles
		styles: theme.GetStyles(),

		// Initialize input mode state - Always INSERT mode for IME support
		currentMode:   ModeInsert, // Always start in Insert mode for IME
		previousMode:  ModeInsert,
		commandBuffer: "",
		searchBuffer:  "",
		searchResults: make([]int, 0),
		currentMatch:  0,

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
	}
}

// Init implements tea.Model interface
func (m Model) Init() tea.Cmd {
	m.logger.Debug("Initializing UI model")
	return tea.Batch(
		tea.EnterAltScreen,
		func() tea.Msg {
			return readyMsg{}
		},
	)
}

// Update implements tea.Model interface
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.logger.Debug("Window resized", "width", m.width, "height", m.height)

	case tea.KeyMsg:
		// Handle key events
		return m.handleKeyPress(msg)

	case readyMsg:
		m.ready = true
		m.logger.Debug("UI model ready")

	case chatResponseMsg:
		m.messages = append(m.messages, Message{
			ID:        msg.ID,
			Content:   msg.Content,
			Role:      "assistant",
			Timestamp: time.Now(),
			Tokens:    msg.Tokens,
		})
		m.loading = false

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

	case loadingMsg:
		m.loading = msg.loading
	}

	// Update view components (when implemented)
	// m.chatView, cmd = m.chatView.Update(msg)
	// cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model interface
func (m Model) View() string {
	if !m.ready {
		return "Loading CODA..."
	}

	var view strings.Builder

	// Header
	view.WriteString(m.renderHeader())
	view.WriteString("\n")

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
		view.WriteString(m.renderChat())
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

	view.WriteString("\n")
	view.WriteString(m.renderInput())
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

	// Handle global keys
	switch key {
	case "ctrl+c":
		return m, tea.Quit
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

// sendMessage sends the current input as a chat message
func (m Model) sendMessage() (tea.Model, tea.Cmd) {
	// Trim whitespace and check if empty
	trimmedInput := strings.TrimSpace(m.currentInput)
	if trimmedInput == "" {
		return m, nil
	}

	// Add user message
	userMsg := Message{
		ID:        generateMessageID(),
		Content:   trimmedInput,
		Role:      "user",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	// Clear input and reset cursor
	m.currentInput = ""
	m.cursorPosition = 0
	m.cursorColumn = 0
	m.loading = true
	m.error = nil

	// Send to chat handler
	return m, func() tea.Msg {
		// Process message through chat handler
		// Get response from chat handler
		response, err := m.chatHandler.HandleMessageWithResponse(m.ctx, trimmedInput)
		if err != nil {
			return errorMsg{
				error:      err,
				userAction: "sending message",
				metadata:   map[string]interface{}{"message": trimmedInput},
			}
		}

		return chatResponseMsg{
			ID:      generateMessageID(),
			Content: response.Content,
			Tokens:  response.TokenCount,
		}
	}
}

// renderChat renders the chat view
func (m Model) renderChat() string {
	if len(m.messages) == 0 {
		return m.renderWelcomeMessage()
	}

	view := ""
	for _, msg := range m.messages {
		view += fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04"),
			msg.Role,
			msg.Content)
	}

	if m.loading {
		view += "\n🤔 Thinking..."
	}

	return view
}

// renderHeader renders the header with border
func (m Model) renderHeader() string {
	// Create header content
	content := " 𝑪𝑶𝑫𝑨 - CODing Agent "

	// Use the same style as input area
	style := m.styles.UserInput

	// Calculate width
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Center the text
	centeredStyle := style.Width(contentWidth).Align(lipgloss.Center)

	return centeredStyle.Render(content)
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
	contentWidth := m.width - 4
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
	return " Enter:send, Ctrl+J:newline, F1:help, Ctrl+C:quit"
}

// renderInput renders the input area
func (m Model) renderInput() string {
	var content string

	switch m.currentMode {
	case ModeCommand:
		content = fmt.Sprintf("%s_", m.commandBuffer)
	case ModeSearch:
		content = fmt.Sprintf("%s_", m.searchBuffer)
	case ModeInsert:
		return m.renderMultilineInput()
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

		// 罫線で囲む
		style := m.styles.UserInput

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

	// 罫線で囲む
	style := m.styles.UserInput

	// ターミナル幅に合わせて調整
	contentWidth := m.width - 4 // ボーダーとパディング分を引く
	if contentWidth < 20 {
		contentWidth = 20 // 最小幅
	}

	return style.Width(contentWidth).Render(result)
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

	help += "Press ? again to return to chat\n"
	return help
}

// SaveState saves the current model state
func (m Model) SaveState() error {
	// This would save the current state to disk
	// For now, just log
	m.logger.Info("Saving model state", "messages", len(m.messages))
	return nil
}

// Message types for Bubbletea
type readyMsg struct{}

type chatResponseMsg struct {
	ID      string
	Content string
	Tokens  int
}

type errorMsg struct {
	error      error
	userAction string
	metadata   map[string]interface{}
}

type dismissErrorMsg struct{}

type toggleErrorDetailsMsg struct{}

type retryLastActionMsg struct{}

type loadingMsg struct {
	loading bool
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
