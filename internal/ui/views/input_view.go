package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/styles"
)

// InputView manages user text input with advanced features
type InputView struct {
	// Core components
	textInput textinput.Model
	styles    styles.Styles
	logger    *log.Logger

	// Input state
	multiline   bool
	content     string
	lines       []string
	currentLine int
	cursorPos   int

	// History management
	history       []string
	historyIndex  int
	originalInput string

	// Completion and suggestions
	suggestions     []string
	suggestionIndex int
	showSuggestions bool

	// Visual properties
	width       int
	height      int
	placeholder string
	prompt      string
	focused     bool

	// Input mode
	mode         InputMode
	lastActivity time.Time

	// Commands and snippets
	commands []string
	snippets map[string]string
	
	// IME state
	composing bool
}

// InputMode defines the current input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeMultiline
	ModeCommand
	ModeCompletion
)

// InputMessage represents a completed input
type InputMessage struct {
	Content   string
	Mode      InputMode
	Timestamp time.Time
}

// NewInputView creates a new input view
func NewInputView(width, height int, styles styles.Styles, logger *log.Logger) *InputView {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0     // No limit
	ti.Width = width - 4 // Account for padding and borders

	return &InputView{
		textInput:    ti,
		styles:       styles,
		logger:       logger,
		width:        width,
		height:       height,
		placeholder:  "Type your message...",
		prompt:       "> ",
		focused:      true,
		mode:         ModeNormal,
		history:      make([]string, 0, 100), // Max 100 history items
		suggestions:  make([]string, 0),
		lastActivity: time.Now(),
		commands:     []string{"/help", "/clear", "/history", "/exit", "/multiline"},
		snippets: map[string]string{
			"/py":   "```python\n\n```",
			"/js":   "```javascript\n\n```",
			"/go":   "```go\n\n```",
			"/bash": "```bash\n\n```",
		},
	}
}

// SetSize updates the input view dimensions
func (iv *InputView) SetSize(width, height int) {
	iv.width = width
	iv.height = height
	iv.textInput.Width = width - 4
}

// SetPlaceholder sets the placeholder text
func (iv *InputView) SetPlaceholder(placeholder string) {
	iv.placeholder = placeholder
	iv.textInput.Placeholder = placeholder
}

// SetPrompt sets the input prompt
func (iv *InputView) SetPrompt(prompt string) {
	iv.prompt = prompt
}

// Focus gives focus to the input
func (iv *InputView) Focus() {
	iv.focused = true
	iv.textInput.Focus()
}

// Blur removes focus from the input
func (iv *InputView) Blur() {
	iv.focused = false
	iv.textInput.Blur()
}

// IsFocused returns whether the input is focused
func (iv *InputView) IsFocused() bool {
	return iv.focused
}

// SetMode sets the input mode
func (iv *InputView) SetMode(mode InputMode) {
	iv.mode = mode
	switch mode {
	case ModeNormal:
		iv.multiline = false
		iv.textInput.Placeholder = iv.placeholder
	case ModeMultiline:
		iv.multiline = true
		iv.textInput.Placeholder = "Multi-line input (Enter to send, Ctrl+J for newline)"
	case ModeCommand:
		iv.textInput.Placeholder = "Enter command..."
	case ModeCompletion:
		iv.textInput.Placeholder = "Select completion..."
	}
}

// GetContent returns the current input content
func (iv *InputView) GetContent() string {
	if iv.multiline {
		return strings.Join(iv.lines, "\n")
	}
	return iv.textInput.Value()
}

// Clear clears the input content
func (iv *InputView) Clear() {
	iv.textInput.SetValue("")
	iv.content = ""
	iv.lines = make([]string, 0)
	iv.currentLine = 0
	iv.cursorPos = 0
	iv.hideSuggestions()
}

// AddToHistory adds an entry to the input history
func (iv *InputView) AddToHistory(content string) {
	if content == "" {
		return
	}

	// Remove duplicate if exists
	for i, item := range iv.history {
		if item == content {
			iv.history = append(iv.history[:i], iv.history[i+1:]...)
			break
		}
	}

	// Add to beginning
	iv.history = append([]string{content}, iv.history...)

	// Limit history size
	if len(iv.history) > 100 {
		iv.history = iv.history[:100]
	}

	iv.historyIndex = -1
}

// Init implements tea.Model
func (iv *InputView) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (iv *InputView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	iv.lastActivity = time.Now()

	// First, always update textinput to handle IME properly
	iv.textInput, cmd = iv.textInput.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle special keys that don't interfere with text input
		if iv.shouldHandleSpecialKey(msg) {
			model, keyCmd := iv.handleKeyPress(msg)
			if keyCmd != nil {
				cmds = append(cmds, keyCmd)
			}
			return model, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		iv.SetSize(msg.Width, msg.Height)
	}

	// Check for completion triggers
	iv.updateSuggestions()

	return iv, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (iv *InputView) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global key handlers
	switch key {
	case "ctrl+c":
		return iv, tea.Quit

	case "tab":
		if iv.showSuggestions {
			return iv.selectNextSuggestion()
		}
		return iv.triggerCompletion()

	case "shift+tab":
		if iv.showSuggestions {
			return iv.selectPrevSuggestion()
		}
		return iv, nil

	case "esc":
		if iv.showSuggestions {
			iv.hideSuggestions()
			return iv, nil
		}
		if iv.mode == ModeMultiline {
			iv.SetMode(ModeNormal)
			return iv, nil
		}

	case "up":
		if iv.showSuggestions {
			return iv.selectPrevSuggestion()
		}
		return iv.navigateHistory(-1)

	case "down":
		if iv.showSuggestions {
			return iv.selectNextSuggestion()
		}
		return iv.navigateHistory(1)

	case "enter":
		if iv.showSuggestions {
			return iv.applySuggestion()
		}
		// Always submit on Enter (single or multiline mode)
		return iv.submitInput()

	case "ctrl+j":
		// Ctrl+J (Shift+Enter in iTerm2) adds new line
		if !iv.multiline {
			iv.SetMode(ModeMultiline)
		}
		return iv.addNewLine()

	case "alt+enter":
		return iv.addNewLine()

	case "ctrl+l":
		iv.Clear()
		return iv, nil

	case "ctrl+m":
		// Toggle multiline mode
		if iv.mode == ModeMultiline {
			iv.SetMode(ModeNormal)
		} else {
			iv.SetMode(ModeMultiline)
		}
		return iv, nil
	}

	// For other keys, let the Update method handle it
	// to ensure IME events are processed correctly
	return iv, nil
}

// navigateHistory navigates through input history
func (iv *InputView) navigateHistory(direction int) (tea.Model, tea.Cmd) {
	if len(iv.history) == 0 {
		return iv, nil
	}

	// Save current input if navigating for the first time
	if iv.historyIndex == -1 {
		iv.originalInput = iv.textInput.Value()
	}

	// Navigate history
	newIndex := iv.historyIndex + direction
	if newIndex < -1 {
		newIndex = len(iv.history) - 1
	} else if newIndex >= len(iv.history) {
		newIndex = -1
	}

	iv.historyIndex = newIndex

	// Set input value
	if iv.historyIndex == -1 {
		iv.textInput.SetValue(iv.originalInput)
	} else {
		iv.textInput.SetValue(iv.history[iv.historyIndex])
	}

	return iv, nil
}

// updateSuggestions updates the suggestion list based on current input
func (iv *InputView) updateSuggestions() {
	input := iv.textInput.Value()
	iv.suggestions = iv.suggestions[:0] // Clear suggestions

	if len(input) == 0 {
		iv.hideSuggestions()
		return
	}

	// Command completion
	if strings.HasPrefix(input, "/") {
		for _, cmd := range iv.commands {
			if strings.HasPrefix(cmd, input) {
				iv.suggestions = append(iv.suggestions, cmd)
			}
		}
	}

	// Snippet completion
	for snippet, _ := range iv.snippets {
		if strings.HasPrefix(snippet, input) {
			iv.suggestions = append(iv.suggestions, snippet)
		}
	}

	// Show suggestions if we have any
	if len(iv.suggestions) > 0 {
		iv.showSuggestions = true
		iv.suggestionIndex = 0
	} else {
		iv.hideSuggestions()
	}
}

// triggerCompletion triggers completion
func (iv *InputView) triggerCompletion() (tea.Model, tea.Cmd) {
	iv.updateSuggestions()
	if len(iv.suggestions) > 0 {
		iv.showSuggestions = true
		iv.suggestionIndex = 0
	}
	return iv, nil
}

// selectNextSuggestion selects the next suggestion
func (iv *InputView) selectNextSuggestion() (tea.Model, tea.Cmd) {
	if len(iv.suggestions) == 0 {
		return iv, nil
	}
	iv.suggestionIndex = (iv.suggestionIndex + 1) % len(iv.suggestions)
	return iv, nil
}

// selectPrevSuggestion selects the previous suggestion
func (iv *InputView) selectPrevSuggestion() (tea.Model, tea.Cmd) {
	if len(iv.suggestions) == 0 {
		return iv, nil
	}
	iv.suggestionIndex = (iv.suggestionIndex - 1 + len(iv.suggestions)) % len(iv.suggestions)
	return iv, nil
}

// applySuggestion applies the selected suggestion
func (iv *InputView) applySuggestion() (tea.Model, tea.Cmd) {
	if len(iv.suggestions) == 0 {
		return iv, nil
	}

	suggestion := iv.suggestions[iv.suggestionIndex]

	// Check if it's a snippet
	if expansion, exists := iv.snippets[suggestion]; exists {
		iv.textInput.SetValue(expansion)
		// Position cursor between the code block markers
		if strings.Contains(expansion, "\n\n") {
			// This is a simplified cursor positioning
			iv.textInput.SetValue(expansion)
		}
	} else {
		iv.textInput.SetValue(suggestion)
	}

	iv.hideSuggestions()
	return iv, nil
}

// hideSuggestions hides the suggestion list
func (iv *InputView) hideSuggestions() {
	iv.showSuggestions = false
	iv.suggestions = iv.suggestions[:0]
	iv.suggestionIndex = 0
}

// addNewLine adds a new line in multiline mode
func (iv *InputView) addNewLine() (tea.Model, tea.Cmd) {
	if !iv.multiline {
		iv.SetMode(ModeMultiline)
	}

	current := iv.textInput.Value()
	iv.lines = append(iv.lines, current)
	iv.textInput.SetValue("")
	iv.currentLine++

	return iv, nil
}

// submitInput submits the current input
func (iv *InputView) submitInput() (tea.Model, tea.Cmd) {
	content := iv.GetContent()
	if content == "" {
		return iv, nil
	}

	// Add to history
	iv.AddToHistory(content)

	// Create input message
	inputMsg := InputMessage{
		Content:   content,
		Mode:      iv.mode,
		Timestamp: time.Now(),
	}

	// Clear input
	iv.Clear()
	iv.SetMode(ModeNormal)

	iv.logger.Debug("Input submitted", "content", content, "mode", iv.mode)

	// Return command to send the message
	return iv, func() tea.Msg {
		return inputMsg
	}
}

// View implements tea.Model
func (iv *InputView) View() string {
	var view strings.Builder

	// Render multiline content if in multiline mode
	if iv.multiline && len(iv.lines) > 0 {
		for i, line := range iv.lines {
			prefix := "│ "
			if i == len(iv.lines)-1 {
				prefix = "└ "
			}
			view.WriteString(iv.styles.Muted.Render(prefix) + line + "\n")
		}
	}

	// Render input prompt and text input
	promptStyle := iv.styles.InputPrompt
	if iv.focused {
		promptStyle = iv.styles.Bold
	}

	inputStyle := iv.styles.InputText
	if iv.focused {
		inputStyle = iv.styles.InputFocused
	}

	// Create input line
	inputLine := promptStyle.Render(iv.prompt) + inputStyle.Render(iv.textInput.View())
	view.WriteString(inputLine)

	// Render suggestions if shown
	if iv.showSuggestions {
		view.WriteString("\n")
		view.WriteString(iv.renderSuggestions())
	}

	// Render input status
	statusLine := iv.renderStatus()
	if statusLine != "" {
		view.WriteString("\n" + statusLine)
	}

	return view.String()
}

// renderSuggestions renders the suggestion list
func (iv *InputView) renderSuggestions() string {
	if len(iv.suggestions) == 0 {
		return ""
	}

	var suggestions strings.Builder
	suggestions.WriteString(iv.styles.Muted.Render("Suggestions:") + "\n")

	for i, suggestion := range iv.suggestions {
		style := iv.styles.Muted
		if i == iv.suggestionIndex {
			style = iv.styles.Highlight
		}

		prefix := "  "
		if i == iv.suggestionIndex {
			prefix = "► "
		}

		suggestions.WriteString(prefix + style.Render(suggestion) + "\n")
	}

	return suggestions.String()
}

// renderStatus renders the input status line
func (iv *InputView) renderStatus() string {
	var status strings.Builder

	// Show mode
	modeText := ""
	switch iv.mode {
	case ModeMultiline:
		modeText = "MULTI"
	case ModeCommand:
		modeText = "CMD"
	case ModeCompletion:
		modeText = "COMP"
	}

	if modeText != "" {
		status.WriteString(iv.styles.StatusActive.Render("[" + modeText + "]"))
		status.WriteString(" ")
	}

	// Show character count if input is long
	content := iv.GetContent()
	if len(content) > 100 {
		status.WriteString(iv.styles.Muted.Render(fmt.Sprintf("(%d chars)", len(content))))
	}

	// Show help for multiline
	if iv.mode == ModeMultiline {
		status.WriteString(" ")
		status.WriteString(iv.styles.Muted.Render("Enter to send, Ctrl+J for newline"))
	}

	return status.String()
}

// GetHistory returns the input history
func (iv *InputView) GetHistory() []string {
	return iv.history
}

// SetHistory sets the input history
func (iv *InputView) SetHistory(history []string) {
	iv.history = make([]string, len(history))
	copy(iv.history, history)
	iv.historyIndex = -1
}

// isComposing returns whether IME is currently composing
func (iv *InputView) isComposing() bool {
	// Check if the textinput model indicates it's in composition mode
	// This is a simple implementation - you might need to track composition events
	// more explicitly depending on the Bubbletea version
	return iv.composing
}

// shouldHandleSpecialKey determines if a key should be handled as a special command
// This helps prevent interference with IME input
func (iv *InputView) shouldHandleSpecialKey(msg tea.KeyMsg) bool {
	key := msg.String()
	
	// These keys should always be handled specially
	specialKeys := []string{
		"ctrl+c", "tab", "shift+tab", "esc",
		"up", "down", "enter", "ctrl+j",
		"alt+enter", "ctrl+l", "ctrl+m",
	}
	
	for _, sk := range specialKeys {
		if key == sk {
			return true
		}
	}
	
	// Don't handle regular character input as special keys
	// This allows IME to work properly
	return false
}
