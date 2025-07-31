package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/styles"
)

// HelpView displays help information and keybindings
type HelpView struct {
	// Core components
	viewport viewport.Model
	styles   styles.Styles
	logger   *log.Logger

	// Display state
	visible     bool
	width       int
	height      int
	activeTab   int
	searchQuery string
	searching   bool

	// Content
	sections    []HelpSection
	tabs        []string
	allBindings []KeyBinding
}

// HelpSection represents a section of help content
type HelpSection struct {
	Title       string
	Description string
	Bindings    []KeyBinding
	Commands    []Command
	Tips        []string
}

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key         string
	Description string
	Context     string // Global, Chat, Input, etc.
}

// Command represents a command
type Command struct {
	Name        string
	Usage       string
	Description string
	Examples    []string
}

// NewHelpView creates a new help view
func NewHelpView(width, height int, styles styles.Styles, logger *log.Logger) *HelpView {
	vp := viewport.New(width-2, height-4) // Account for borders and title
	vp.Style = styles.Container

	hv := &HelpView{
		viewport: vp,
		styles:   styles,
		logger:   logger,
		width:    width,
		height:   height,
		visible:  false,
		tabs:     []string{"Keys", "Commands", "Tips"},
	}

	hv.initializeContent()
	hv.renderContent()

	return hv
}

// initializeContent initializes the help content
func (hv *HelpView) initializeContent() {
	// Global keybindings
	globalBindings := []KeyBinding{
		{"Ctrl+C", "Exit application", "Global"},
		{"?", "Toggle help", "Global"},
		{"Ctrl+L", "Clear screen", "Global"},
		{"Tab", "Next view/completion", "Global"},
		{"Shift+Tab", "Previous view", "Global"},
		{"Esc", "Cancel/Close", "Global"},
	}

	// Chat keybindings
	chatBindings := []KeyBinding{
		{"Enter", "Send message (single-line)", "Chat"},
		{"Ctrl+Enter", "Send message (multi-line)", "Chat"},
		{"Alt+Enter", "Add new line", "Chat"},
		{"Up/Down", "Navigate history", "Chat"},
		{"Ctrl+U", "Clear input", "Chat"},
		{"Ctrl+M", "Toggle multi-line mode", "Chat"},
	}

	// Navigation keybindings
	navBindings := []KeyBinding{
		{"j/k", "Scroll down/up", "Navigation"},
		{"Down/Up", "Scroll down/up", "Navigation"},
		{"PgDn/PgUp", "Page down/up", "Navigation"},
		{"Home/End", "Go to top/bottom", "Navigation"},
		{"Ctrl+D/U", "Half page down/up", "Navigation"},
	}

	// Commands
	commands := []Command{
		{
			Name:        "/help",
			Usage:       "/help [topic]",
			Description: "Show help information",
			Examples:    []string{"/help", "/help keys", "/help commands"},
		},
		{
			Name:        "/clear",
			Usage:       "/clear",
			Description: "Clear chat history",
			Examples:    []string{"/clear"},
		},
		{
			Name:        "/history",
			Usage:       "/history [count]",
			Description: "Show input history",
			Examples:    []string{"/history", "/history 10"},
		},
		{
			Name:        "/multiline",
			Usage:       "/multiline",
			Description: "Toggle multi-line input mode",
			Examples:    []string{"/multiline"},
		},
		{
			Name:        "/exit",
			Usage:       "/exit",
			Description: "Exit the application",
			Examples:    []string{"/exit"},
		},
	}

	// Tips
	tips := []string{
		"Use Ctrl+Enter to send multi-line messages",
		"Press Tab to auto-complete commands and paths",
		"Use Up/Down arrows to navigate input history",
		"Type / to see available commands",
		"Press Esc to cancel operations or close dialogs",
		"Use Page Up/Down to scroll through long conversations",
		"Multi-line mode preserves formatting and indentation",
		"Commands starting with / have special functions",
	}

	// Create sections
	hv.sections = []HelpSection{
		{
			Title:       "Keyboard Shortcuts",
			Description: "Essential keyboard shortcuts for efficient usage",
			Bindings:    append(append(globalBindings, chatBindings...), navBindings...),
		},
		{
			Title:       "Commands",
			Description: "Available slash commands",
			Commands:    commands,
		},
		{
			Title:       "Tips & Tricks",
			Description: "Helpful tips to improve your CODA experience",
			Tips:        tips,
		},
	}

	// Collect all bindings for search
	hv.allBindings = append(append(globalBindings, chatBindings...), navBindings...)
}

// SetSize updates the help view dimensions
func (hv *HelpView) SetSize(width, height int) {
	hv.width = width
	hv.height = height
	hv.viewport.Width = width - 2
	hv.viewport.Height = height - 4
	hv.renderContent()
}

// Show shows the help view
func (hv *HelpView) Show() {
	hv.visible = true
	hv.renderContent()
}

// Hide hides the help view
func (hv *HelpView) Hide() {
	hv.visible = false
}

// Toggle toggles the help view visibility
func (hv *HelpView) Toggle() {
	if hv.visible {
		hv.Hide()
	} else {
		hv.Show()
	}
}

// IsVisible returns whether the help view is visible
func (hv *HelpView) IsVisible() bool {
	return hv.visible
}

// SetActiveTab sets the active tab
func (hv *HelpView) SetActiveTab(tab int) {
	if tab >= 0 && tab < len(hv.tabs) {
		hv.activeTab = tab
		hv.renderContent()
	}
}

// StartSearch starts search mode
func (hv *HelpView) StartSearch() {
	hv.searching = true
	hv.searchQuery = ""
}

// StopSearch stops search mode
func (hv *HelpView) StopSearch() {
	hv.searching = false
	hv.searchQuery = ""
	hv.renderContent()
}

// Init implements tea.Model
func (hv *HelpView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (hv *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !hv.visible {
		return hv, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if hv.searching {
			return hv.handleSearchInput(msg)
		}
		return hv.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		hv.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	hv.viewport, cmd = hv.viewport.Update(msg)
	return hv, cmd
}

// handleKeyPress handles keyboard input in normal mode
func (hv *HelpView) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?":
		hv.Hide()
		return hv, nil

	case "1", "2", "3":
		tab := int(msg.String()[0] - '1')
		hv.SetActiveTab(tab)
		return hv, nil

	case "tab":
		hv.SetActiveTab((hv.activeTab + 1) % len(hv.tabs))
		return hv, nil

	case "shift+tab":
		hv.SetActiveTab((hv.activeTab - 1 + len(hv.tabs)) % len(hv.tabs))
		return hv, nil

	case "/":
		hv.StartSearch()
		return hv, nil

	case "j", "down":
		hv.viewport.LineDown(1)
		return hv, nil

	case "k", "up":
		hv.viewport.LineUp(1)
		return hv, nil

	case "d", "pgdown":
		hv.viewport.HalfViewDown()
		return hv, nil

	case "u", "pgup":
		hv.viewport.HalfViewUp()
		return hv, nil

	case "g", "home":
		hv.viewport.GotoTop()
		return hv, nil

	case "G", "end":
		hv.viewport.GotoBottom()
		return hv, nil
	}

	return hv, nil
}

// handleSearchInput handles keyboard input in search mode
func (hv *HelpView) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		hv.StopSearch()
		return hv, nil

	case "enter":
		hv.performSearch()
		hv.StopSearch()
		return hv, nil

	case "backspace":
		if len(hv.searchQuery) > 0 {
			hv.searchQuery = hv.searchQuery[:len(hv.searchQuery)-1]
		}
		return hv, nil

	default:
		if len(msg.String()) == 1 {
			hv.searchQuery += msg.String()
		}
	}

	return hv, nil
}

// performSearch performs a search and updates the view
func (hv *HelpView) performSearch() {
	if hv.searchQuery == "" {
		hv.renderContent()
		return
	}

	// Search through bindings
	var matchedBindings []KeyBinding
	query := strings.ToLower(hv.searchQuery)

	for _, binding := range hv.allBindings {
		if strings.Contains(strings.ToLower(binding.Key), query) ||
			strings.Contains(strings.ToLower(binding.Description), query) ||
			strings.Contains(strings.ToLower(binding.Context), query) {
			matchedBindings = append(matchedBindings, binding)
		}
	}

	// Render search results
	hv.renderSearchResults(matchedBindings)
}

// View implements tea.Model
func (hv *HelpView) View() string {
	if !hv.visible {
		return ""
	}

	// Create header with tabs
	header := hv.renderTabs()

	// Create search bar if searching
	var searchBar string
	if hv.searching {
		searchBar = hv.renderSearchBar()
	}

	// Create help content
	content := hv.viewport.View()

	// Create footer
	footer := hv.renderFooter()

	// Combine all parts
	var view strings.Builder
	view.WriteString(header + "\n")
	if searchBar != "" {
		view.WriteString(searchBar + "\n")
	}
	view.WriteString(content + "\n")
	view.WriteString(footer)

	// Apply container style
	return hv.styles.Container.
		Width(hv.width).
		Height(hv.height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(hv.styles.Colors.Border).
		Render(view.String())
}

// renderTabs renders the tab navigation
func (hv *HelpView) renderTabs() string {
	var tabs []string

	for i, tab := range hv.tabs {
		style := hv.styles.Muted
		if i == hv.activeTab {
			style = hv.styles.Bold.Foreground(hv.styles.Colors.Primary)
		}

		tabText := fmt.Sprintf("%d. %s", i+1, tab)
		tabs = append(tabs, style.Render(tabText))
	}

	title := hv.styles.HelpHeader.Render("CODA Help")
	tabBar := strings.Join(tabs, " │ ")

	return title + "\n" + tabBar
}

// renderSearchBar renders the search input bar
func (hv *HelpView) renderSearchBar() string {
	prompt := hv.styles.Bold.Render("Search: ")
	query := hv.styles.InputText.Render(hv.searchQuery + "_")

	return prompt + query + hv.styles.Muted.Render(" (Enter to search, Esc to cancel)")
}

// renderFooter renders the footer with navigation help
func (hv *HelpView) renderFooter() string {
	shortcuts := []string{
		"1-3: Switch tabs",
		"Tab: Next tab",
		"/: Search",
		"j/k: Scroll",
		"q/?: Close",
	}

	footer := strings.Join(shortcuts, " • ")
	return hv.styles.Muted.Render(footer)
}

// renderContent renders the main help content
func (hv *HelpView) renderContent() {
	if hv.activeTab >= len(hv.sections) {
		return
	}

	section := hv.sections[hv.activeTab]
	var content strings.Builder

	// Section title and description
	content.WriteString(hv.styles.HelpHeader.Render(section.Title) + "\n")
	if section.Description != "" {
		content.WriteString(hv.styles.Muted.Render(section.Description) + "\n\n")
	}

	// Render content based on section type
	switch hv.activeTab {
	case 0: // Keybindings
		content.WriteString(hv.renderKeybindings(section.Bindings))
	case 1: // Commands
		content.WriteString(hv.renderCommands(section.Commands))
	case 2: // Tips
		content.WriteString(hv.renderTips(section.Tips))
	}

	hv.viewport.SetContent(content.String())
}

// renderKeybindings renders keyboard shortcuts
func (hv *HelpView) renderKeybindings(bindings []KeyBinding) string {
	var content strings.Builder

	// Group by context
	contexts := make(map[string][]KeyBinding)
	for _, binding := range bindings {
		contexts[binding.Context] = append(contexts[binding.Context], binding)
	}

	for context, contextBindings := range contexts {
		// Context header
		content.WriteString(hv.styles.Bold.Render(context+" Commands") + "\n")
		content.WriteString(strings.Repeat("─", len(context)+9) + "\n")

		// Bindings
		for _, binding := range contextBindings {
			key := hv.styles.HelpKey.Width(12).Render(binding.Key)
			desc := hv.styles.HelpDesc.Render(binding.Description)
			content.WriteString(key + " " + desc + "\n")
		}
		content.WriteString("\n")
	}

	return content.String()
}

// renderCommands renders available commands
func (hv *HelpView) renderCommands(commands []Command) string {
	var content strings.Builder

	for i, cmd := range commands {
		// Command name and usage
		name := hv.styles.HelpKey.Render(cmd.Name)
		usage := hv.styles.Code.Render(cmd.Usage)
		content.WriteString(name + " - " + usage + "\n")

		// Description
		content.WriteString(hv.styles.HelpDesc.Render("  "+cmd.Description) + "\n")

		// Examples
		if len(cmd.Examples) > 0 {
			content.WriteString(hv.styles.Muted.Render("  Examples:") + "\n")
			for _, example := range cmd.Examples {
				content.WriteString(hv.styles.Code.Render("    "+example) + "\n")
			}
		}

		if i < len(commands)-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderTips renders tips and tricks
func (hv *HelpView) renderTips(tips []string) string {
	var content strings.Builder

	for i, tip := range tips {
		bullet := hv.styles.Primary.Render("• ")
		content.WriteString(bullet + hv.styles.HelpDesc.Render(tip) + "\n")

		if i < len(tips)-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderSearchResults renders search results
func (hv *HelpView) renderSearchResults(results []KeyBinding) {
	var content strings.Builder

	content.WriteString(hv.styles.HelpHeader.Render(fmt.Sprintf("Search Results (%d found)", len(results))) + "\n")
	content.WriteString(hv.styles.Muted.Render("Query: \""+hv.searchQuery+"\"") + "\n\n")

	if len(results) == 0 {
		content.WriteString(hv.styles.Muted.Render("No results found.") + "\n")
	} else {
		for _, result := range results {
			key := hv.styles.HelpKey.Width(12).Render(result.Key)
			desc := hv.styles.HelpDesc.Render(result.Description)
			context := hv.styles.Muted.Render("(" + result.Context + ")")
			content.WriteString(key + " " + desc + " " + context + "\n")
		}
	}

	hv.viewport.SetContent(content.String())
}
