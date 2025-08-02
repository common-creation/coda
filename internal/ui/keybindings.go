package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// Mode represents the current input mode (Vim-style)
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeCommand
	ModeSearch
	ModeScroll
	ModePermit
)

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
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

// KeyMap defines all key bindings for the application
type KeyMap struct {
	// Global bindings (work in all modes)
	Quit    key.Binding
	Help    key.Binding
	Clear   key.Binding
	Refresh key.Binding

	// Navigation bindings
	ScrollUp   key.Binding
	ScrollDown key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding
	NextView   key.Binding
	PrevView   key.Binding

	// Edit bindings
	Submit   key.Binding
	Cancel   key.Binding
	Complete key.Binding
	Copy     key.Binding
	Paste    key.Binding
	Cut      key.Binding
	Undo     key.Binding
	Redo     key.Binding

	// Mode-specific bindings
	Normal  NormalModeKeyMap
	Insert  InsertModeKeyMap
	Command CommandModeKeyMap
	Search  SearchModeKeyMap
	Permit  PermitModeKeyMap

	// Custom bindings loaded from config
	Custom map[string]key.Binding
}

// NormalModeKeyMap defines Vim-style normal mode bindings
type NormalModeKeyMap struct {
	// Movement
	MoveUp    key.Binding
	MoveDown  key.Binding
	MoveLeft  key.Binding
	MoveRight key.Binding
	WordNext  key.Binding
	WordPrev  key.Binding
	LineStart key.Binding
	LineEnd   key.Binding

	// Mode transitions
	InsertMode        key.Binding
	InsertModeAppend  key.Binding
	InsertModeNewLine key.Binding
	CommandMode       key.Binding
	SearchMode        key.Binding

	// Actions
	Delete     key.Binding
	DeleteLine key.Binding
	Change     key.Binding
	Yank       key.Binding
	YankLine   key.Binding
	Put        key.Binding
	PutBefore  key.Binding

	// Chat-specific
	SendMessage  key.Binding
	NewChat      key.Binding
	ClearHistory key.Binding
}

// InsertModeKeyMap defines insert mode bindings
type InsertModeKeyMap struct {
	ExitMode  key.Binding
	Backspace key.Binding
	Delete    key.Binding
	Enter     key.Binding
	Tab       key.Binding
	ShiftTab  key.Binding

	// Quick actions
	SaveAndExit key.Binding
	ForceExit   key.Binding
}

// CommandModeKeyMap defines command mode bindings
type CommandModeKeyMap struct {
	ExitMode key.Binding
	Execute  key.Binding
	History  key.Binding
	Complete key.Binding
	Clear    key.Binding
}

// SearchModeKeyMap defines search mode bindings
type SearchModeKeyMap struct {
	ExitMode      key.Binding
	Execute       key.Binding
	NextMatch     key.Binding
	PrevMatch     key.Binding
	CaseSensitive key.Binding
	Regex         key.Binding
}

// PermitModeKeyMap defines permit mode bindings for tool call approval
type PermitModeKeyMap struct {
	ExitMode    key.Binding // Exit permit mode (reject by default)
	Approve     key.Binding // Approve the tool call
	Reject      key.Binding // Reject the tool call
	SelectPrev  key.Binding // Move selection to previous option (left arrow)
	SelectNext  key.Binding // Move selection to next option (right arrow)
}

// DefaultKeyMap returns the default key mappings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Global bindings
		Quit:    key.NewBinding(key.WithKeys("ctrl+c", "ctrl+d")),
		Help:    key.NewBinding(key.WithKeys("?", "F1")),
		Clear:   key.NewBinding(key.WithKeys("ctrl+l")),
		Refresh: key.NewBinding(key.WithKeys("F5", "ctrl+r")),

		// Navigation
		ScrollUp:   key.NewBinding(key.WithKeys("up", "k")),
		ScrollDown: key.NewBinding(key.WithKeys("down", "j")),
		PageUp:     key.NewBinding(key.WithKeys("pgup", "ctrl+b")),
		PageDown:   key.NewBinding(key.WithKeys("pgdown", "ctrl+f")),
		Home:       key.NewBinding(key.WithKeys("home", "ctrl+a")),
		End:        key.NewBinding(key.WithKeys("end", "ctrl+e")),
		NextView:   key.NewBinding(key.WithKeys("tab", "ctrl+n")),
		PrevView:   key.NewBinding(key.WithKeys("shift+tab", "ctrl+p")),

		// Edit bindings
		Submit:   key.NewBinding(key.WithKeys("enter", "ctrl+m")),
		Cancel:   key.NewBinding(key.WithKeys("esc", "ctrl+[")),
		Complete: key.NewBinding(key.WithKeys("ctrl+space", "ctrl+x ctrl+o")),
		Copy:     key.NewBinding(key.WithKeys("ctrl+c", "y y")),
		Paste:    key.NewBinding(key.WithKeys("ctrl+v", "p")),
		Cut:      key.NewBinding(key.WithKeys("ctrl+x", "d d")),
		Undo:     key.NewBinding(key.WithKeys("ctrl+z", "u")),
		Redo:     key.NewBinding(key.WithKeys("ctrl+y", "ctrl+r")),

		// Mode-specific bindings
		Normal:  DefaultNormalModeKeyMap(),
		Insert:  DefaultInsertModeKeyMap(),
		Command: DefaultCommandModeKeyMap(),
		Search:  DefaultSearchModeKeyMap(),
		Permit:  DefaultPermitModeKeyMap(),

		// Custom bindings (empty by default)
		Custom: make(map[string]key.Binding),
	}
}

// VimKeyMap returns Vim-style key mappings
func VimKeyMap() KeyMap {
	keymap := DefaultKeyMap()

	// Override with Vim-specific bindings
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+c", ":q"))
	keymap.ScrollUp = key.NewBinding(key.WithKeys("k"))
	keymap.ScrollDown = key.NewBinding(key.WithKeys("j"))
	keymap.PageUp = key.NewBinding(key.WithKeys("ctrl+b"))
	keymap.PageDown = key.NewBinding(key.WithKeys("ctrl+f"))
	keymap.Home = key.NewBinding(key.WithKeys("0", "^"))
	keymap.End = key.NewBinding(key.WithKeys("$"))

	return keymap
}

// EmacsKeyMap returns Emacs-style key mappings
func EmacsKeyMap() KeyMap {
	keymap := DefaultKeyMap()

	// Override with Emacs-specific bindings
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+x ctrl+c"))
	keymap.ScrollUp = key.NewBinding(key.WithKeys("ctrl+p"))
	keymap.ScrollDown = key.NewBinding(key.WithKeys("ctrl+n"))
	keymap.PageUp = key.NewBinding(key.WithKeys("alt+v"))
	keymap.PageDown = key.NewBinding(key.WithKeys("ctrl+v"))
	keymap.Home = key.NewBinding(key.WithKeys("ctrl+a"))
	keymap.End = key.NewBinding(key.WithKeys("ctrl+e"))

	return keymap
}

// DefaultNormalModeKeyMap returns the default normal mode key mappings
func DefaultNormalModeKeyMap() NormalModeKeyMap {
	return NormalModeKeyMap{
		// Movement (Vim-style)
		MoveUp:    key.NewBinding(key.WithKeys("k", "up")),
		MoveDown:  key.NewBinding(key.WithKeys("j", "down")),
		MoveLeft:  key.NewBinding(key.WithKeys("h", "left")),
		MoveRight: key.NewBinding(key.WithKeys("l", "right")),
		WordNext:  key.NewBinding(key.WithKeys("w", "ctrl+right")),
		WordPrev:  key.NewBinding(key.WithKeys("b", "ctrl+left")),
		LineStart: key.NewBinding(key.WithKeys("0", "^", "home")),
		LineEnd:   key.NewBinding(key.WithKeys("$", "end")),

		// Mode transitions
		InsertMode:        key.NewBinding(key.WithKeys("i")),
		InsertModeAppend:  key.NewBinding(key.WithKeys("a")),
		InsertModeNewLine: key.NewBinding(key.WithKeys("o")),
		CommandMode:       key.NewBinding(key.WithKeys(":")),
		SearchMode:        key.NewBinding(key.WithKeys("/", "?")),

		// Actions
		Delete:     key.NewBinding(key.WithKeys("x", "delete")),
		DeleteLine: key.NewBinding(key.WithKeys("d d")),
		Change:     key.NewBinding(key.WithKeys("c")),
		Yank:       key.NewBinding(key.WithKeys("y")),
		YankLine:   key.NewBinding(key.WithKeys("y y")),
		Put:        key.NewBinding(key.WithKeys("p")),
		PutBefore:  key.NewBinding(key.WithKeys("P")),

		// Chat-specific
		SendMessage:  key.NewBinding(key.WithKeys("enter", "ctrl+m")),
		NewChat:      key.NewBinding(key.WithKeys("ctrl+n")),
		ClearHistory: key.NewBinding(key.WithKeys("ctrl+l")),
	}
}

// DefaultInsertModeKeyMap returns the default insert mode key mappings
func DefaultInsertModeKeyMap() InsertModeKeyMap {
	return InsertModeKeyMap{
		ExitMode:    key.NewBinding(key.WithKeys("esc", "ctrl+[")),
		Backspace:   key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
		Delete:      key.NewBinding(key.WithKeys("delete", "ctrl+d")),
		Enter:       key.NewBinding(key.WithKeys("enter", "ctrl+m")),
		Tab:         key.NewBinding(key.WithKeys("tab")),
		ShiftTab:    key.NewBinding(key.WithKeys("shift+tab")),
		SaveAndExit: key.NewBinding(key.WithKeys("ctrl+s")),
		ForceExit:   key.NewBinding(key.WithKeys("ctrl+c")),
	}
}

// DefaultCommandModeKeyMap returns the default command mode key mappings
func DefaultCommandModeKeyMap() CommandModeKeyMap {
	return CommandModeKeyMap{
		ExitMode: key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		Execute:  key.NewBinding(key.WithKeys("enter")),
		History:  key.NewBinding(key.WithKeys("up", "down")),
		Complete: key.NewBinding(key.WithKeys("tab")),
		Clear:    key.NewBinding(key.WithKeys("ctrl+u")),
	}
}

// DefaultSearchModeKeyMap returns the default search mode key mappings
func DefaultSearchModeKeyMap() SearchModeKeyMap {
	return SearchModeKeyMap{
		ExitMode:      key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		Execute:       key.NewBinding(key.WithKeys("enter")),
		NextMatch:     key.NewBinding(key.WithKeys("n", "ctrl+n")),
		PrevMatch:     key.NewBinding(key.WithKeys("N", "ctrl+p")),
		CaseSensitive: key.NewBinding(key.WithKeys("ctrl+i")),
		Regex:         key.NewBinding(key.WithKeys("ctrl+r")),
	}
}

// DefaultPermitModeKeyMap returns the default permit mode key mappings
func DefaultPermitModeKeyMap() PermitModeKeyMap {
	return PermitModeKeyMap{
		ExitMode:   key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		Approve:    key.NewBinding(key.WithKeys("enter", "y")),
		Reject:     key.NewBinding(key.WithKeys("n", "esc")),
		SelectPrev: key.NewBinding(key.WithKeys("left", "h")),
		SelectNext: key.NewBinding(key.WithKeys("right", "l")),
	}
}

// KeyBinding represents a customizable key binding
type KeyBinding struct {
	Keys        []string `yaml:"keys" json:"keys"`
	Description string   `yaml:"description" json:"description"`
	Context     string   `yaml:"context" json:"context"`
	Mode        string   `yaml:"mode" json:"mode"`
}

// KeyBindingConfig represents the configuration for key bindings
type KeyBindingConfig struct {
	Style    string                `yaml:"style" json:"style"` // "default", "vim", "emacs"
	Bindings map[string]KeyBinding `yaml:"bindings" json:"bindings"`
}

// LoadFromConfig loads key bindings from configuration
func (km *KeyMap) LoadFromConfig(config KeyBindingConfig) error {
	// Apply style-specific defaults
	switch strings.ToLower(config.Style) {
	case "vim":
		*km = VimKeyMap()
	case "emacs":
		*km = EmacsKeyMap()
	default:
		*km = DefaultKeyMap()
	}

	// Apply custom bindings
	for name, binding := range config.Bindings {
		keyBinding := key.NewBinding(key.WithKeys(binding.Keys...))
		if binding.Description != "" {
			keyBinding = key.NewBinding(
				key.WithKeys(binding.Keys...),
				key.WithHelp(strings.Join(binding.Keys, "/"), binding.Description),
			)
		}

		// Store in custom bindings
		if km.Custom == nil {
			km.Custom = make(map[string]key.Binding)
		}
		km.Custom[name] = keyBinding
	}

	return nil
}

// Validate checks for key binding conflicts
func (km KeyMap) Validate() []string {
	var conflicts []string

	// Create a map to track all key combinations
	keyMap := make(map[string][]string)

	// Helper function to add keys to the conflict checker
	addKeys := func(binding key.Binding, context string) {
		if binding.Keys() != nil {
			for _, k := range binding.Keys() {
				keyMap[k] = append(keyMap[k], context)
			}
		}
	}

	// Check global bindings
	addKeys(km.Quit, "global.quit")
	addKeys(km.Help, "global.help")
	addKeys(km.Clear, "global.clear")
	addKeys(km.Refresh, "global.refresh")
	addKeys(km.ScrollUp, "global.scroll_up")
	addKeys(km.ScrollDown, "global.scroll_down")
	addKeys(km.PageUp, "global.page_up")
	addKeys(km.PageDown, "global.page_down")
	addKeys(km.Home, "global.home")
	addKeys(km.End, "global.end")
	addKeys(km.NextView, "global.next_view")
	addKeys(km.PrevView, "global.prev_view")

	// Check normal mode bindings
	addKeys(km.Normal.MoveUp, "normal.move_up")
	addKeys(km.Normal.MoveDown, "normal.move_down")
	addKeys(km.Normal.MoveLeft, "normal.move_left")
	addKeys(km.Normal.MoveRight, "normal.move_right")
	addKeys(km.Normal.InsertMode, "normal.insert_mode")
	addKeys(km.Normal.CommandMode, "normal.command_mode")
	addKeys(km.Normal.SearchMode, "normal.search_mode")

	// Check insert mode bindings
	addKeys(km.Insert.ExitMode, "insert.exit_mode")
	addKeys(km.Insert.Enter, "insert.enter")
	addKeys(km.Insert.Tab, "insert.tab")

	// Check command mode bindings
	addKeys(km.Command.ExitMode, "command.exit_mode")
	addKeys(km.Command.Execute, "command.execute")

	// Check search mode bindings
	addKeys(km.Search.ExitMode, "search.exit_mode")
	addKeys(km.Search.Execute, "search.execute")

	// Check custom bindings
	for name, binding := range km.Custom {
		addKeys(binding, fmt.Sprintf("custom.%s", name))
	}

	// Find conflicts
	for keyCombo, contexts := range keyMap {
		if len(contexts) > 1 {
			conflicts = append(conflicts,
				fmt.Sprintf("Key '%s' is bound to multiple actions: %s",
					keyCombo, strings.Join(contexts, ", ")))
		}
	}

	return conflicts
}

// GetHelpText returns help text for all key bindings
func (km KeyMap) GetHelpText(mode Mode) []string {
	var help []string

	// Add global bindings
	help = append(help, "Global Commands:")
	help = append(help, fmt.Sprintf("  %s: Quit application", km.getKeyStrings(km.Quit)))
	help = append(help, fmt.Sprintf("  %s: Show/hide help", km.getKeyStrings(km.Help)))
	help = append(help, fmt.Sprintf("  %s: Clear screen", km.getKeyStrings(km.Clear)))
	help = append(help, fmt.Sprintf("  %s: Refresh view", km.getKeyStrings(km.Refresh)))
	help = append(help, "")

	// Add navigation bindings
	help = append(help, "Navigation:")
	help = append(help, fmt.Sprintf("  %s: Scroll up", km.getKeyStrings(km.ScrollUp)))
	help = append(help, fmt.Sprintf("  %s: Scroll down", km.getKeyStrings(km.ScrollDown)))
	help = append(help, fmt.Sprintf("  %s: Page up", km.getKeyStrings(km.PageUp)))
	help = append(help, fmt.Sprintf("  %s: Page down", km.getKeyStrings(km.PageDown)))
	help = append(help, fmt.Sprintf("  %s: Go to start", km.getKeyStrings(km.Home)))
	help = append(help, fmt.Sprintf("  %s: Go to end", km.getKeyStrings(km.End)))
	help = append(help, "")

	// Add mode-specific bindings
	switch mode {
	case ModeNormal:
		help = append(help, "Normal Mode Commands:")
		help = append(help, fmt.Sprintf("  %s: Enter insert mode", km.getKeyStrings(km.Normal.InsertMode)))
		help = append(help, fmt.Sprintf("  %s: Enter command mode", km.getKeyStrings(km.Normal.CommandMode)))
		help = append(help, fmt.Sprintf("  %s: Enter search mode", km.getKeyStrings(km.Normal.SearchMode)))
		help = append(help, fmt.Sprintf("  %s: Send message", km.getKeyStrings(km.Normal.SendMessage)))
		help = append(help, fmt.Sprintf("  %s: New chat", km.getKeyStrings(km.Normal.NewChat)))
		help = append(help, fmt.Sprintf("  %s: Clear history", km.getKeyStrings(km.Normal.ClearHistory)))

	case ModeInsert:
		help = append(help, "Insert Mode Commands:")
		help = append(help, fmt.Sprintf("  %s: Exit insert mode", km.getKeyStrings(km.Insert.ExitMode)))
		help = append(help, fmt.Sprintf("  %s: Submit input", km.getKeyStrings(km.Insert.Enter)))
		help = append(help, fmt.Sprintf("  %s: Save and exit", km.getKeyStrings(km.Insert.SaveAndExit)))

	case ModeCommand:
		help = append(help, "Command Mode Commands:")
		help = append(help, fmt.Sprintf("  %s: Exit command mode", km.getKeyStrings(km.Command.ExitMode)))
		help = append(help, fmt.Sprintf("  %s: Execute command", km.getKeyStrings(km.Command.Execute)))
		help = append(help, fmt.Sprintf("  %s: Command completion", km.getKeyStrings(km.Command.Complete)))

	case ModeSearch:
		help = append(help, "Search Mode Commands:")
		help = append(help, fmt.Sprintf("  %s: Exit search mode", km.getKeyStrings(km.Search.ExitMode)))
		help = append(help, fmt.Sprintf("  %s: Execute search", km.getKeyStrings(km.Search.Execute)))
		help = append(help, fmt.Sprintf("  %s: Next match", km.getKeyStrings(km.Search.NextMatch)))
		help = append(help, fmt.Sprintf("  %s: Previous match", km.getKeyStrings(km.Search.PrevMatch)))

	case ModePermit:
		help = append(help, "Permit Mode Commands:")
		help = append(help, fmt.Sprintf("  %s: Approve tool call", km.getKeyStrings(km.Permit.Approve)))
		help = append(help, fmt.Sprintf("  %s: Reject tool call", km.getKeyStrings(km.Permit.Reject)))
		help = append(help, fmt.Sprintf("  %s: Select previous option", km.getKeyStrings(km.Permit.SelectPrev)))
		help = append(help, fmt.Sprintf("  %s: Select next option", km.getKeyStrings(km.Permit.SelectNext)))
		help = append(help, fmt.Sprintf("  %s: Exit permit mode", km.getKeyStrings(km.Permit.ExitMode)))
	}

	// Add custom bindings if any
	if len(km.Custom) > 0 {
		help = append(help, "")
		help = append(help, "Custom Commands:")
		for name, binding := range km.Custom {
			help = append(help, fmt.Sprintf("  %s: %s", km.getKeyStrings(binding), name))
		}
	}

	return help
}

// getKeyStrings returns a formatted string of key combinations for a binding
func (km KeyMap) getKeyStrings(binding key.Binding) string {
	if binding.Keys() == nil {
		return "(no keys bound)"
	}
	return strings.Join(binding.Keys(), ", ")
}

// IsMatch checks if a key matches any of the bindings
func (km KeyMap) IsMatch(keyStr string, binding key.Binding) bool {
	if binding.Keys() == nil {
		return false
	}

	for _, k := range binding.Keys() {
		if k == keyStr {
			return true
		}
	}
	return false
}

// Reset resets the key map to defaults
func (km *KeyMap) Reset() {
	*km = DefaultKeyMap()
}
