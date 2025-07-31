package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ShortcutIntegration integrates shortcuts with the existing UI system
type ShortcutIntegration struct {
	shortcutManager      *ShortcutManager
	contextActionManager *ContextActionManager
	keyBindingManager    *KeyBindingManager

	// Context menu state
	contextMenuVisible  bool
	contextMenuActions  []ContextAction
	contextMenuSelected int
	contextMenuContent  string
	contextMenuLine     int
	contextMenuCol      int
}

// NewShortcutIntegration creates a new shortcut integration
func NewShortcutIntegration(keyBindingMgr *KeyBindingManager) *ShortcutIntegration {
	return &ShortcutIntegration{
		shortcutManager:      NewShortcutManager(keyBindingMgr),
		contextActionManager: NewContextActionManager(),
		keyBindingManager:    keyBindingMgr,
		contextMenuVisible:   false,
		contextMenuActions:   make([]ContextAction, 0),
		contextMenuSelected:  0,
	}
}

// GetShortcutManager returns the shortcut manager
func (si *ShortcutIntegration) GetShortcutManager() *ShortcutManager {
	return si.shortcutManager
}

// GetContextActionManager returns the context action manager
func (si *ShortcutIntegration) GetContextActionManager() *ContextActionManager {
	return si.contextActionManager
}

// HandleKeyPress processes key presses for shortcuts and context actions
func (si *ShortcutIntegration) HandleKeyPress(msg tea.KeyMsg, context string, mode Mode) tea.Cmd {
	keyStr := msg.String()

	// Handle command palette navigation when visible
	if si.shortcutManager.IsCommandPaletteVisible() {
		return si.handleCommandPaletteKey(msg)
	}

	// Handle context menu navigation when visible
	if si.contextMenuVisible {
		return si.handleContextMenuKey(msg)
	}

	// Check for shortcut matches
	if cmd := si.shortcutManager.HandleKey(keyStr, context, mode); cmd != nil {
		return cmd
	}

	// Handle special key combinations for context actions
	if strings.Contains(keyStr, "ctrl+alt+") {
		// Ctrl+Alt combinations for context actions
		if keyStr == "ctrl+alt+c" {
			return si.showContextMenuForCurrentSelection()
		}
	}

	return nil
}

// handleCommandPaletteKey handles key presses when command palette is visible
func (si *ShortcutIntegration) handleCommandPaletteKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		si.shortcutManager.ToggleCommandPalette()
		return nil

	case tea.KeyEnter:
		cmd := si.shortcutManager.ExecuteSelectedPaletteItem()
		return cmd

	case tea.KeyUp:
		si.shortcutManager.MovePaletteSelection(-1)
		return nil

	case tea.KeyDown:
		si.shortcutManager.MovePaletteSelection(1)
		return nil

	default:
		// Update search query
		switch msg.Type {
		case tea.KeyBackspace:
			query := si.shortcutManager.GetPaletteQuery()
			if len(query) > 0 {
				si.shortcutManager.UpdatePaletteQuery(query[:len(query)-1])
			}
			return nil

		case tea.KeyRunes:
			query := si.shortcutManager.GetPaletteQuery() + string(msg.Runes)
			si.shortcutManager.UpdatePaletteQuery(query)
			return nil
		}
	}

	return nil
}

// handleContextMenuKey handles key presses when context menu is visible
func (si *ShortcutIntegration) handleContextMenuKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		si.hideContextMenu()
		return nil

	case tea.KeyEnter:
		return si.executeSelectedContextAction()

	case tea.KeyUp:
		if si.contextMenuSelected > 0 {
			si.contextMenuSelected--
		} else {
			si.contextMenuSelected = len(si.contextMenuActions) - 1
		}
		return nil

	case tea.KeyDown:
		if si.contextMenuSelected < len(si.contextMenuActions)-1 {
			si.contextMenuSelected++
		} else {
			si.contextMenuSelected = 0
		}
		return nil
	}

	return nil
}

// HandleShortcutMessage processes shortcut-related messages
func (si *ShortcutIntegration) HandleShortcutMessage(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case ToggleCommandPaletteMsg:
		si.shortcutManager.ToggleCommandPalette()
		return nil

	case StartMacroRecordingMsg:
		// Could prompt for macro name, for now use default
		si.shortcutManager.StartMacroRecording("macro_" + strings.Replace(time.Now().String(), " ", "_", -1))
		return nil

	case StopMacroRecordingMsg:
		si.shortcutManager.StopMacroRecording()
		return nil

	case ReplayMacroMsg:
		return si.shortcutManager.ReplayMacro(msg.Name)

	case ShowContextMenuMsg:
		return si.showContextMenu(msg.Content, msg.Line, msg.Col)

	case HideContextMenuMsg:
		si.hideContextMenu()
		return nil

	// Handle context action results
	case ContextActionResultMsg:
		// Could show notification or status message
		return func() tea.Msg {
			return StatusMessageMsg{
				Message: msg.Message,
				Success: msg.Success,
			}
		}

	case ShowErrorDetailsMsg:
		// Could open error details view
		return func() tea.Msg {
			return OpenErrorDetailsMsg{
				Line:    msg.Line,
				Content: msg.Content,
				Details: msg.Details,
			}
		}

	case SearchDocumentationMsg:
		// Could trigger documentation search
		return func() tea.Msg {
			return OpenDocumentationSearchMsg{
				Term: msg.Term,
			}
		}

	case ExplainFunctionMsg:
		// Could show function explanation
		return func() tea.Msg {
			return OpenFunctionExplanationMsg{
				Function: msg.Function,
			}
		}

	case FormatJSONMsg:
		// Could format and replace JSON
		return func() tea.Msg {
			return ReplaceContentMsg{
				Content: msg.Content, // Would be formatted JSON
			}
		}
	}

	return nil
}

// showContextMenu shows context menu for given content
func (si *ShortcutIntegration) showContextMenu(content string, line int, col int) tea.Cmd {
	actions := si.contextActionManager.GetActionsForContent(content, line, col)

	si.contextMenuVisible = true
	si.contextMenuActions = actions
	si.contextMenuSelected = 0
	si.contextMenuContent = content
	si.contextMenuLine = line
	si.contextMenuCol = col

	return nil
}

// showContextMenuForCurrentSelection shows context menu for current selection
func (si *ShortcutIntegration) showContextMenuForCurrentSelection() tea.Cmd {
	// This would need to be integrated with the actual text selection system
	// For now, return a command to request current selection
	return func() tea.Msg {
		return RequestCurrentSelectionMsg{}
	}
}

// hideContextMenu hides the context menu
func (si *ShortcutIntegration) hideContextMenu() {
	si.contextMenuVisible = false
	si.contextMenuActions = make([]ContextAction, 0)
	si.contextMenuSelected = 0
	si.contextMenuContent = ""
	si.contextMenuLine = 0
	si.contextMenuCol = 0
}

// executeSelectedContextAction executes the selected context action
func (si *ShortcutIntegration) executeSelectedContextAction() tea.Cmd {
	if si.contextMenuSelected >= 0 && si.contextMenuSelected < len(si.contextMenuActions) {
		action := si.contextMenuActions[si.contextMenuSelected]
		si.hideContextMenu()
		return si.contextActionManager.ExecuteAction(action, si.contextMenuContent, si.contextMenuLine, si.contextMenuCol)
	}
	return nil
}

// IsCommandPaletteVisible returns true if command palette is visible
func (si *ShortcutIntegration) IsCommandPaletteVisible() bool {
	return si.shortcutManager.IsCommandPaletteVisible()
}

// IsContextMenuVisible returns true if context menu is visible
func (si *ShortcutIntegration) IsContextMenuVisible() bool {
	return si.contextMenuVisible
}

// IsRecordingMacro returns true if currently recording a macro
func (si *ShortcutIntegration) IsRecordingMacro() bool {
	return si.shortcutManager.IsRecording()
}

// GetRecordingMacroName returns the name of the macro being recorded
func (si *ShortcutIntegration) GetRecordingMacroName() string {
	return si.shortcutManager.GetRecordingMacroName()
}

// RenderCommandPalette renders the command palette
func (si *ShortcutIntegration) RenderCommandPalette() string {
	return si.shortcutManager.RenderCommandPalette()
}

// RenderContextMenu renders the context menu
func (si *ShortcutIntegration) RenderContextMenu() string {
	if !si.contextMenuVisible {
		return ""
	}

	return si.contextActionManager.RenderContextMenu(si.contextMenuActions, si.contextMenuSelected)
}

// RenderShortcutIndicators renders status indicators for shortcuts
func (si *ShortcutIntegration) RenderShortcutIndicators() string {
	var indicators []string

	if si.shortcutManager.IsRecording() {
		style := si.shortcutManager.GetStyles().RecordingIcon
		indicators = append(indicators, style.Render("● RECORDING: "+si.shortcutManager.GetRecordingMacroName()))
	}

	if si.contextMenuVisible {
		indicators = append(indicators, "Context Menu Active")
	}

	if si.shortcutManager.IsCommandPaletteVisible() {
		indicators = append(indicators, "Command Palette")
	}

	return strings.Join(indicators, " | ")
}

// GetShortcutHelpText returns help text for shortcuts
func (si *ShortcutIntegration) GetShortcutHelpText() []string {
	var help []string

	help = append(help, "Shortcut System:")
	help = append(help, "  Ctrl+Shift+P: Open command palette")
	help = append(help, "  Ctrl+Shift+L: Clear chat")
	help = append(help, "  Ctrl+Shift+S: Save session")
	help = append(help, "  Ctrl+O: Open session")
	help = append(help, "  Ctrl+/: Toggle comment")
	help = append(help, "  Ctrl+Space: Trigger completion")
	help = append(help, "  Alt+Enter: Submit without tools")
	help = append(help, "")
	help = append(help, "Macro System:")
	help = append(help, "  Ctrl+Shift+R: Start recording macro")
	help = append(help, "  Ctrl+Shift+E: Stop recording macro")
	help = append(help, "  Ctrl+Shift+M: Replay last macro")
	help = append(help, "")
	help = append(help, "Context Actions:")
	help = append(help, "  Ctrl+Alt+C: Show context menu")
	help = append(help, "  Ctrl+Click: Context action (when implemented)")
	help = append(help, "")

	return help
}

// RegisterCustomShortcut registers a custom shortcut
func (si *ShortcutIntegration) RegisterCustomShortcut(shortcut ShortcutAction) error {
	return si.shortcutManager.RegisterShortcut(shortcut)
}

// RegisterCustomContextAction registers a custom context action
func (si *ShortcutIntegration) RegisterCustomContextAction(action ContextAction) {
	si.contextActionManager.RegisterAction(action)
}

// GetAllShortcuts returns all registered shortcuts
func (si *ShortcutIntegration) GetAllShortcuts() map[string]ShortcutAction {
	return si.shortcutManager.GetAllShortcuts()
}

// GetAllMacros returns all saved macros
func (si *ShortcutIntegration) GetAllMacros() map[string]ShortcutMacro {
	return si.shortcutManager.GetMacros()
}

// DetectContextAtPosition detects context actions at a specific position
func (si *ShortcutIntegration) DetectContextAtPosition(content string, line int, col int) []ContextAction {
	return si.contextActionManager.GetActionsForContent(content, line, col)
}

// Additional message types for integration
type (
	ShowContextMenuMsg struct {
		Content string
		Line    int
		Col     int
	}
	HideContextMenuMsg         struct{}
	RequestCurrentSelectionMsg struct{}
	StatusMessageMsg           struct {
		Message string
		Success bool
	}
	OpenErrorDetailsMsg struct {
		Line    int
		Content string
		Details string
	}
	OpenDocumentationSearchMsg struct {
		Term string
	}
	OpenFunctionExplanationMsg struct {
		Function string
	}
	ReplaceContentMsg struct {
		Content string
	}
)
