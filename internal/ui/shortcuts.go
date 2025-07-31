package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShortcutAction represents an action that can be triggered by a shortcut
type ShortcutAction struct {
	Name        string
	Description string
	Keys        []string
	Category    string
	Action      func() tea.Cmd
	Context     string // "global", "chat", "input", etc.
	Mode        string // "all", "normal", "insert", etc.
}

// ShortcutMacro represents a recorded sequence of actions
type ShortcutMacro struct {
	Name        string
	Description string
	Actions     []ShortcutAction
	CreatedAt   time.Time
	LastUsed    time.Time
	UsageCount  int
}

// ShortcutManager manages keyboard shortcuts and command palette
type ShortcutManager struct {
	shortcuts       map[string]ShortcutAction
	macros          map[string]ShortcutMacro
	history         []string
	keyBindingMgr   *KeyBindingManager
	paletteVisible  bool
	paletteQuery    string
	paletteSelected int
	paletteResults  []ShortcutAction
	recording       bool
	recordingMacro  string
	recordedActions []ShortcutAction
	styles          ShortcutStyles
}

// ShortcutStyles holds styling for the shortcut system
type ShortcutStyles struct {
	Palette       lipgloss.Style
	PaletteTitle  lipgloss.Style
	PaletteItem   lipgloss.Style
	PaletteSelect lipgloss.Style
	PaletteKey    lipgloss.Style
	PaletteDesc   lipgloss.Style
	Category      lipgloss.Style
	MacroIcon     lipgloss.Style
	RecordingIcon lipgloss.Style
}

// DefaultShortcutStyles returns default styling for shortcuts
func DefaultShortcutStyles() ShortcutStyles {
	return ShortcutStyles{
		Palette: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Background(lipgloss.Color("235")).
			Width(60).
			MaxHeight(20),
		PaletteTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true).
			Padding(0, 1),
		PaletteItem: lipgloss.NewStyle().
			Padding(0, 2),
		PaletteSelect: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 2),
		PaletteKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		PaletteDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")),
		Category: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Padding(0, 1),
		MacroIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		RecordingIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Blink(true),
	}
}

// NewShortcutManager creates a new shortcut manager
func NewShortcutManager(keyBindingMgr *KeyBindingManager) *ShortcutManager {
	sm := &ShortcutManager{
		shortcuts:       make(map[string]ShortcutAction),
		macros:          make(map[string]ShortcutMacro),
		history:         make([]string, 0, 100),
		keyBindingMgr:   keyBindingMgr,
		paletteVisible:  false,
		paletteQuery:    "",
		paletteSelected: 0,
		paletteResults:  make([]ShortcutAction, 0),
		recording:       false,
		recordingMacro:  "",
		recordedActions: make([]ShortcutAction, 0),
		styles:          DefaultShortcutStyles(),
	}

	// Register built-in shortcuts
	sm.registerBuiltinShortcuts()

	return sm
}

// registerBuiltinShortcuts registers the built-in shortcuts
func (sm *ShortcutManager) registerBuiltinShortcuts() {
	// Use alternative keys to avoid conflicts with existing keybindings
	shortcuts := []ShortcutAction{
		{
			Name:        "command_palette",
			Description: "Open command palette",
			Keys:        []string{"ctrl+shift+p"}, // Changed from ctrl+p to avoid conflict
			Category:    "Navigation",
			Context:     "global",
			Mode:        "all",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return ToggleCommandPaletteMsg{}
				}
			},
		},
		{
			Name:        "clear_chat",
			Description: "Clear chat history",
			Keys:        []string{"ctrl+shift+l"}, // Changed from ctrl+l to avoid conflict
			Category:    "Chat",
			Context:     "chat",
			Mode:        "all",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return ClearChatMsg{}
				}
			},
		},
		{
			Name:        "save_session",
			Description: "Save current session",
			Keys:        []string{"ctrl+shift+s"}, // Changed from ctrl+s to avoid conflict
			Category:    "Session",
			Context:     "global",
			Mode:        "all",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return SaveSessionMsg{}
				}
			},
		},
		{
			Name:        "open_session",
			Description: "Open saved session",
			Keys:        []string{"ctrl+o"},
			Category:    "Session",
			Context:     "global",
			Mode:        "all",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return OpenSessionMsg{}
				}
			},
		},
		{
			Name:        "toggle_comment",
			Description: "Toggle comment in input",
			Keys:        []string{"ctrl+/"},
			Category:    "Edit",
			Context:     "input",
			Mode:        "insert",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return ToggleCommentMsg{}
				}
			},
		},
		{
			Name:        "trigger_completion",
			Description: "Trigger auto-completion",
			Keys:        []string{"ctrl+space"},
			Category:    "Edit",
			Context:     "input",
			Mode:        "insert",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return TriggerCompletionMsg{}
				}
			},
		},
		{
			Name:        "submit_without_tools",
			Description: "Submit message without tool usage",
			Keys:        []string{"alt+enter"},
			Category:    "Chat",
			Context:     "input",
			Mode:        "insert",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return SubmitWithoutToolsMsg{}
				}
			},
		},
		{
			Name:        "start_macro_recording",
			Description: "Start recording macro",
			Keys:        []string{"ctrl+shift+r"},
			Category:    "Macro",
			Context:     "global",
			Mode:        "normal",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return StartMacroRecordingMsg{}
				}
			},
		},
		{
			Name:        "stop_macro_recording",
			Description: "Stop recording macro",
			Keys:        []string{"ctrl+shift+e"},
			Category:    "Macro",
			Context:     "global",
			Mode:        "normal",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return StopMacroRecordingMsg{}
				}
			},
		},
		{
			Name:        "replay_last_macro",
			Description: "Replay last recorded macro",
			Keys:        []string{"ctrl+shift+m"},
			Category:    "Macro",
			Context:     "global",
			Mode:        "normal",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return ReplayMacroMsg{Name: "last"}
				}
			},
		},
		{
			Name:        "show_shortcuts",
			Description: "Show all keyboard shortcuts",
			Keys:        []string{"ctrl+shift+h"},
			Category:    "Help",
			Context:     "global",
			Mode:        "all",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return ShowShortcutsMsg{}
				}
			},
		},
	}

	for _, shortcut := range shortcuts {
		sm.shortcuts[shortcut.Name] = shortcut
	}
}

// RegisterShortcut registers a new shortcut
func (sm *ShortcutManager) RegisterShortcut(shortcut ShortcutAction) error {
	// Check for key conflicts
	if sm.keyBindingMgr != nil {
		keymap := sm.keyBindingMgr.GetKeyMap()
		for _, keyStr := range shortcut.Keys {
			// Check against existing shortcuts
			for _, existingShortcut := range sm.shortcuts {
				for _, existingKey := range existingShortcut.Keys {
					if keyStr == existingKey {
						return fmt.Errorf("key '%s' is already bound to '%s'", keyStr, existingShortcut.Name)
					}
				}
			}

			// Check against keybinding system (simplified check)
			if keymap.IsMatch(keyStr, keymap.Quit) ||
				keymap.IsMatch(keyStr, keymap.Help) ||
				keymap.IsMatch(keyStr, keymap.Clear) {
				return fmt.Errorf("key '%s' conflicts with existing keybinding", keyStr)
			}
		}
	}

	sm.shortcuts[shortcut.Name] = shortcut
	return nil
}

// UnregisterShortcut removes a shortcut
func (sm *ShortcutManager) UnregisterShortcut(name string) {
	delete(sm.shortcuts, name)
}

// GetShortcut returns a shortcut by name
func (sm *ShortcutManager) GetShortcut(name string) (ShortcutAction, bool) {
	shortcut, exists := sm.shortcuts[name]
	return shortcut, exists
}

// GetAllShortcuts returns all registered shortcuts
func (sm *ShortcutManager) GetAllShortcuts() map[string]ShortcutAction {
	return sm.shortcuts
}

// ExecuteShortcut executes a shortcut by name
func (sm *ShortcutManager) ExecuteShortcut(name string) tea.Cmd {
	if shortcut, exists := sm.shortcuts[name]; exists {
		// Add to history
		sm.addToHistory(name)

		// Record action if recording macro
		if sm.recording {
			sm.recordedActions = append(sm.recordedActions, shortcut)
		}

		return shortcut.Action()
	}
	return nil
}

// HandleKey processes a key press and executes matching shortcuts
func (sm *ShortcutManager) HandleKey(keyStr string, context string, mode Mode) tea.Cmd {
	for _, shortcut := range sm.shortcuts {
		// Check if key matches
		for _, key := range shortcut.Keys {
			if key == keyStr {
				// Check context and mode
				if sm.matchesContext(shortcut, context, mode) {
					return sm.ExecuteShortcut(shortcut.Name)
				}
			}
		}
	}
	return nil
}

// matchesContext checks if a shortcut matches the current context and mode
func (sm *ShortcutManager) matchesContext(shortcut ShortcutAction, context string, mode Mode) bool {
	// Check context
	if shortcut.Context != "global" && shortcut.Context != context && shortcut.Context != "all" {
		return false
	}

	// Check mode
	modeStr := strings.ToLower(mode.String())
	if shortcut.Mode != "all" && shortcut.Mode != modeStr {
		return false
	}

	return true
}

// ToggleCommandPalette toggles the command palette visibility
func (sm *ShortcutManager) ToggleCommandPalette() {
	sm.paletteVisible = !sm.paletteVisible
	if sm.paletteVisible {
		sm.paletteQuery = ""
		sm.paletteSelected = 0
		sm.updatePaletteResults()
	}
}

// IsCommandPaletteVisible returns true if command palette is visible
func (sm *ShortcutManager) IsCommandPaletteVisible() bool {
	return sm.paletteVisible
}

// UpdatePaletteQuery updates the command palette search query
func (sm *ShortcutManager) UpdatePaletteQuery(query string) {
	sm.paletteQuery = query
	sm.paletteSelected = 0
	sm.updatePaletteResults()
}

// GetPaletteQuery returns the current palette query
func (sm *ShortcutManager) GetPaletteQuery() string {
	return sm.paletteQuery
}

// updatePaletteResults updates the filtered results for the command palette
func (sm *ShortcutManager) updatePaletteResults() {
	sm.paletteResults = make([]ShortcutAction, 0)

	query := strings.ToLower(sm.paletteQuery)

	// Collect and score shortcuts
	type scoredShortcut struct {
		shortcut ShortcutAction
		score    int
	}

	var scored []scoredShortcut

	for _, shortcut := range sm.shortcuts {
		score := 0
		name := strings.ToLower(shortcut.Name)
		desc := strings.ToLower(shortcut.Description)

		if query == "" {
			score = 1
		} else {
			// Exact name match gets highest score
			if name == query {
				score = 100
			} else if strings.HasPrefix(name, query) {
				score = 50
			} else if strings.Contains(name, query) {
				score = 25
			} else if strings.Contains(desc, query) {
				score = 10
			}
		}

		// Boost score for recent usage
		for i, historyItem := range sm.history {
			if historyItem == shortcut.Name {
				score += (len(sm.history) - i) / 10
			}
		}

		if score > 0 {
			scored = append(scored, scoredShortcut{shortcut, score})
		}
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Extract shortcuts
	for _, item := range scored {
		sm.paletteResults = append(sm.paletteResults, item.shortcut)
	}

	// Limit results
	if len(sm.paletteResults) > 10 {
		sm.paletteResults = sm.paletteResults[:10]
	}
}

// GetPaletteResults returns the current palette results
func (sm *ShortcutManager) GetPaletteResults() []ShortcutAction {
	return sm.paletteResults
}

// GetPaletteSelected returns the currently selected palette item index
func (sm *ShortcutManager) GetPaletteSelected() int {
	return sm.paletteSelected
}

// SetPaletteSelected sets the selected palette item index
func (sm *ShortcutManager) SetPaletteSelected(index int) {
	if index >= 0 && index < len(sm.paletteResults) {
		sm.paletteSelected = index
	}
}

// MovePaletteSelection moves the palette selection up or down
func (sm *ShortcutManager) MovePaletteSelection(delta int) {
	if len(sm.paletteResults) == 0 {
		return
	}

	newIndex := sm.paletteSelected + delta
	if newIndex < 0 {
		newIndex = len(sm.paletteResults) - 1
	} else if newIndex >= len(sm.paletteResults) {
		newIndex = 0
	}

	sm.paletteSelected = newIndex
}

// ExecuteSelectedPaletteItem executes the currently selected palette item
func (sm *ShortcutManager) ExecuteSelectedPaletteItem() tea.Cmd {
	if sm.paletteSelected >= 0 && sm.paletteSelected < len(sm.paletteResults) {
		selected := sm.paletteResults[sm.paletteSelected]
		sm.paletteVisible = false
		return sm.ExecuteShortcut(selected.Name)
	}
	return nil
}

// addToHistory adds a shortcut name to the usage history
func (sm *ShortcutManager) addToHistory(name string) {
	// Remove existing entry if present
	for i, item := range sm.history {
		if item == name {
			sm.history = append(sm.history[:i], sm.history[i+1:]...)
			break
		}
	}

	// Add to front
	sm.history = append([]string{name}, sm.history...)

	// Limit history size
	if len(sm.history) > 100 {
		sm.history = sm.history[:100]
	}
}

// StartMacroRecording starts recording a macro
func (sm *ShortcutManager) StartMacroRecording(name string) {
	sm.recording = true
	sm.recordingMacro = name
	sm.recordedActions = make([]ShortcutAction, 0)
}

// StopMacroRecording stops recording and saves the macro
func (sm *ShortcutManager) StopMacroRecording() {
	if !sm.recording {
		return
	}

	macro := ShortcutMacro{
		Name:        sm.recordingMacro,
		Description: fmt.Sprintf("Recorded macro with %d actions", len(sm.recordedActions)),
		Actions:     make([]ShortcutAction, len(sm.recordedActions)),
		CreatedAt:   time.Now(),
		LastUsed:    time.Time{},
		UsageCount:  0,
	}

	copy(macro.Actions, sm.recordedActions)
	sm.macros[sm.recordingMacro] = macro

	// Store as "last" macro for quick replay
	sm.macros["last"] = macro

	sm.recording = false
	sm.recordingMacro = ""
	sm.recordedActions = make([]ShortcutAction, 0)
}

// IsRecording returns true if currently recording a macro
func (sm *ShortcutManager) IsRecording() bool {
	return sm.recording
}

// GetRecordingMacroName returns the name of the macro being recorded
func (sm *ShortcutManager) GetRecordingMacroName() string {
	return sm.recordingMacro
}

// ReplayMacro replays a saved macro
func (sm *ShortcutManager) ReplayMacro(name string) tea.Cmd {
	macro, exists := sm.macros[name]
	if !exists {
		return nil
	}

	// Update usage stats
	macro.LastUsed = time.Now()
	macro.UsageCount++
	sm.macros[name] = macro

	// Execute all actions in sequence
	return tea.Sequence(func() []tea.Cmd {
		cmds := make([]tea.Cmd, len(macro.Actions))
		for i, action := range macro.Actions {
			cmds[i] = action.Action()
		}
		return cmds
	}()...)
}

// GetMacros returns all saved macros
func (sm *ShortcutManager) GetMacros() map[string]ShortcutMacro {
	return sm.macros
}

// DeleteMacro deletes a saved macro
func (sm *ShortcutManager) DeleteMacro(name string) {
	delete(sm.macros, name)
}

// GetStyles returns the shortcut styles
func (sm *ShortcutManager) GetStyles() ShortcutStyles {
	return sm.styles
}

// SetStyles sets the shortcut styles
func (sm *ShortcutManager) SetStyles(styles ShortcutStyles) {
	sm.styles = styles
}

// Render renders the command palette
func (sm *ShortcutManager) RenderCommandPalette() string {
	if !sm.paletteVisible {
		return ""
	}

	var content strings.Builder

	// Title
	title := "Command Palette"
	if sm.recording {
		title += " " + sm.styles.RecordingIcon.Render("● REC")
	}
	content.WriteString(sm.styles.PaletteTitle.Render(title))
	content.WriteString("\n\n")

	// Search query
	query := sm.paletteQuery
	if query == "" {
		query = "Type to search..."
	}
	content.WriteString("> " + query)
	content.WriteString("\n\n")

	// Results
	if len(sm.paletteResults) == 0 {
		content.WriteString(sm.styles.PaletteDesc.Render("No matching commands"))
	} else {
		currentCategory := ""
		for i, shortcut := range sm.paletteResults {
			// Category header
			if shortcut.Category != currentCategory {
				if currentCategory != "" {
					content.WriteString("\n")
				}
				content.WriteString(sm.styles.Category.Render(shortcut.Category))
				content.WriteString("\n")
				currentCategory = shortcut.Category
			}

			// Item
			var line strings.Builder

			// Selection indicator
			if i == sm.paletteSelected {
				line.WriteString("► ")
			} else {
				line.WriteString("  ")
			}

			// Name and description
			line.WriteString(shortcut.Description)

			// Keys
			if len(shortcut.Keys) > 0 {
				keyStr := strings.Join(shortcut.Keys, ", ")
				line.WriteString(" ")
				line.WriteString(sm.styles.PaletteKey.Render("[" + keyStr + "]"))
			}

			// Apply style
			if i == sm.paletteSelected {
				content.WriteString(sm.styles.PaletteSelect.Render(line.String()))
			} else {
				content.WriteString(sm.styles.PaletteItem.Render(line.String()))
			}
			content.WriteString("\n")
		}
	}

	return sm.styles.Palette.Render(content.String())
}

// Message types for shortcut actions
type (
	ToggleCommandPaletteMsg struct{}
	ClearChatMsg            struct{}
	SaveSessionMsg          struct{}
	OpenSessionMsg          struct{}
	ToggleCommentMsg        struct{}
	TriggerCompletionMsg    struct{}
	SubmitWithoutToolsMsg   struct{}
	StartMacroRecordingMsg  struct{}
	StopMacroRecordingMsg   struct{}
	ReplayMacroMsg          struct{ Name string }
	ShowShortcutsMsg        struct{}
)
