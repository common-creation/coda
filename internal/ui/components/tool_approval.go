package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/common-creation/coda/internal/styles"
)

// ApprovalResponse represents the user's response to a tool approval request
type ApprovalResponse int

const (
	ApprovalYes ApprovalResponse = iota
	ApprovalNo
	ApprovalAlways
	ApprovalNever
	ApprovalView
	ApprovalHelp
	ApprovalPending
)

// Choice represents a selectable choice in the approval dialog
type Choice struct {
	Label    string
	Key      string
	Value    ApprovalResponse
	Default  bool
	Shortcut rune
}

// ToolApprovalRequest represents a request for tool approval
type ToolApprovalRequest struct {
	ToolName      string
	Parameters    map[string]interface{}
	Description   string
	Risks         []string
	Preview       string
	FilePath      string
	AffectedFiles []string
	Reversible    bool
	Timestamp     time.Time
}

// ToolApprovalDialog handles tool approval UI
type ToolApprovalDialog struct {
	// Request data
	request ToolApprovalRequest

	// UI state
	visible     bool
	selected    int
	choices     []Choice
	showPreview bool
	showHelp    bool
	showDetails bool

	// Display properties
	width  int
	height int
	styles styles.Styles

	// Response tracking
	response       ApprovalResponse
	rememberChoice bool
}

// NewToolApprovalDialog creates a new tool approval dialog
func NewToolApprovalDialog(styles styles.Styles) *ToolApprovalDialog {
	return &ToolApprovalDialog{
		styles:   styles,
		choices:  createDefaultChoices(),
		width:    80,
		height:   20,
		response: ApprovalPending,
	}
}

// createDefaultChoices creates the default choice set
func createDefaultChoices() []Choice {
	return []Choice{
		{Label: "Yes", Key: "y", Value: ApprovalYes, Default: false, Shortcut: 'y'},
		{Label: "No", Key: "n", Value: ApprovalNo, Default: true, Shortcut: 'n'},
		{Label: "Always", Key: "a", Value: ApprovalAlways, Default: false, Shortcut: 'a'},
		{Label: "Never", Key: "e", Value: ApprovalNever, Default: false, Shortcut: 'e'},
		{Label: "View", Key: "v", Value: ApprovalView, Default: false, Shortcut: 'v'},
		{Label: "Help", Key: "?", Value: ApprovalHelp, Default: false, Shortcut: '?'},
	}
}

// Show displays the approval dialog with the given request
func (ta *ToolApprovalDialog) Show(request ToolApprovalRequest) {
	ta.request = request
	ta.visible = true
	ta.response = ApprovalPending
	ta.showPreview = false
	ta.showHelp = false
	ta.showDetails = false

	// Set default selection to "No" for safety
	for i, choice := range ta.choices {
		if choice.Default {
			ta.selected = i
			break
		}
	}
}

// Hide hides the approval dialog
func (ta *ToolApprovalDialog) Hide() {
	ta.visible = false
}

// IsVisible returns whether the dialog is visible
func (ta *ToolApprovalDialog) IsVisible() bool {
	return ta.visible
}

// GetResponse returns the user's response
func (ta *ToolApprovalDialog) GetResponse() ApprovalResponse {
	return ta.response
}

// SetSize updates the dialog dimensions
func (ta *ToolApprovalDialog) SetSize(width, height int) {
	ta.width = width
	ta.height = height
}

// Init implements tea.Model
func (ta *ToolApprovalDialog) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (ta *ToolApprovalDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !ta.visible {
		return ta, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return ta.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		ta.SetSize(msg.Width, msg.Height)
	}

	return ta, nil
}

// handleKeyPress handles keyboard input
func (ta *ToolApprovalDialog) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Check for shortcut keys
	for i, choice := range ta.choices {
		if key == choice.Key || (len(key) == 1 && rune(key[0]) == choice.Shortcut) {
			ta.selected = i
			return ta.selectChoice()
		}
	}

	// Handle navigation keys
	switch key {
	case "up", "k":
		ta.selected = (ta.selected - 1 + len(ta.choices)) % len(ta.choices)
	case "down", "j":
		ta.selected = (ta.selected + 1) % len(ta.choices)
	case "left", "h":
		if ta.selected > 0 {
			ta.selected--
		}
	case "right", "l":
		if ta.selected < len(ta.choices)-1 {
			ta.selected++
		}
	case "enter", " ":
		return ta.selectChoice()
	case "esc":
		ta.response = ApprovalNo
		ta.Hide()
		return ta, func() tea.Msg { return ApprovalResponseMsg{Response: ApprovalNo} }
	case "tab":
		ta.showDetails = !ta.showDetails
	}

	return ta, nil
}

// selectChoice handles choice selection
func (ta *ToolApprovalDialog) selectChoice() (tea.Model, tea.Cmd) {
	selectedChoice := ta.choices[ta.selected]

	switch selectedChoice.Value {
	case ApprovalView:
		ta.showPreview = !ta.showPreview
		return ta, nil
	case ApprovalHelp:
		ta.showHelp = !ta.showHelp
		return ta, nil
	default:
		ta.response = selectedChoice.Value
		ta.Hide()
		return ta, func() tea.Msg {
			return ApprovalResponseMsg{
				Response: selectedChoice.Value,
				Remember: ta.rememberChoice,
			}
		}
	}
}

// View implements tea.Model
func (ta *ToolApprovalDialog) View() string {
	if !ta.visible {
		return ""
	}

	if ta.showHelp {
		return ta.renderHelp()
	}

	var content strings.Builder

	// Dialog header
	header := ta.renderHeader()
	content.WriteString(header + "\n")

	// Tool information
	toolInfo := ta.renderToolInfo()
	content.WriteString(toolInfo + "\n")

	// Risk information
	if len(ta.request.Risks) > 0 {
		risks := ta.renderRisks()
		content.WriteString(risks + "\n")
	}

	// Preview if requested
	if ta.showPreview && ta.request.Preview != "" {
		preview := ta.renderPreview()
		content.WriteString(preview + "\n")
	}

	// Details if requested
	if ta.showDetails {
		details := ta.renderDetails()
		content.WriteString(details + "\n")
	}

	// Choices
	choices := ta.renderChoices()
	content.WriteString(choices)

	// Wrap in dialog box
	return ta.wrapInDialog(content.String())
}

// renderHeader renders the dialog header
func (ta *ToolApprovalDialog) renderHeader() string {
	title := "Tool Execution Request"
	return ta.styles.Bold.Foreground(ta.styles.Colors.Primary).Render(title)
}

// renderToolInfo renders tool information
func (ta *ToolApprovalDialog) renderToolInfo() string {
	var info strings.Builder

	// Tool name
	toolName := ta.styles.Bold.Render("Tool: ") +
		ta.styles.Code.Render(ta.request.ToolName)
	info.WriteString(toolName + "\n")

	// Description
	if ta.request.Description != "" {
		desc := ta.styles.Bold.Render("Action: ") + ta.request.Description
		info.WriteString(desc + "\n")
	}

	// File path if applicable
	if ta.request.FilePath != "" {
		filePath := ta.styles.Bold.Render("File: ") +
			ta.styles.Link.Render(ta.request.FilePath)
		info.WriteString(filePath + "\n")
	}

	// Parameters (simplified display)
	if len(ta.request.Parameters) > 0 {
		info.WriteString(ta.styles.Bold.Render("Parameters:") + "\n")
		for key, value := range ta.request.Parameters {
			if key == "content" && len(fmt.Sprintf("%v", value)) > 100 {
				// Truncate long content
				content := fmt.Sprintf("%v", value)
				if len(content) > 100 {
					content = content[:97] + "..."
				}
				info.WriteString(fmt.Sprintf("  %s: %s\n", key, content))
			} else {
				info.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
			}
		}
	}

	return info.String()
}

// renderRisks renders risk information
func (ta *ToolApprovalDialog) renderRisks() string {
	var risks strings.Builder

	risks.WriteString(ta.styles.Bold.Render("⚠️  This operation will:") + "\n")

	for _, risk := range ta.request.Risks {
		bullet := ta.styles.StatusError.Render("• ")
		risks.WriteString(bullet + risk + "\n")
	}

	// Reversibility info
	if ta.request.Reversible {
		reversible := ta.styles.StatusActive.Render("✓ This operation can be undone")
		risks.WriteString(reversible + "\n")
	} else {
		irreversible := ta.styles.StatusError.Render("⚠️  This operation cannot be undone")
		risks.WriteString(irreversible + "\n")
	}

	return risks.String()
}

// renderPreview renders content preview
func (ta *ToolApprovalDialog) renderPreview() string {
	var preview strings.Builder

	preview.WriteString(ta.styles.Bold.Render("Preview:") + "\n")

	// Create preview box
	previewContent := ta.request.Preview
	if len(previewContent) > 500 {
		previewContent = previewContent[:497] + "..."
	}

	// Split into lines and limit
	lines := strings.Split(previewContent, "\n")
	if len(lines) > 10 {
		lines = lines[:10]
		lines = append(lines, "...")
	}

	previewBox := ta.styles.Code.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ta.styles.Colors.Border).
		Padding(1).
		Width(ta.width - 10).
		Render(strings.Join(lines, "\n"))

	preview.WriteString(previewBox + "\n")

	return preview.String()
}

// renderDetails renders additional details
func (ta *ToolApprovalDialog) renderDetails() string {
	var details strings.Builder

	details.WriteString(ta.styles.Bold.Render("Details:") + "\n")

	// Affected files
	if len(ta.request.AffectedFiles) > 0 {
		details.WriteString(ta.styles.Muted.Render("Affected files:") + "\n")
		for _, file := range ta.request.AffectedFiles {
			details.WriteString("  " + ta.styles.Link.Render(file) + "\n")
		}
	}

	// Timestamp
	timestamp := ta.request.Timestamp.Format("2006-01-02 15:04:05")
	details.WriteString(ta.styles.Muted.Render("Requested at: "+timestamp) + "\n")

	return details.String()
}

// renderChoices renders the choice buttons
func (ta *ToolApprovalDialog) renderChoices() string {
	var choices strings.Builder

	// Create choice buttons
	var buttons []string
	for i, choice := range ta.choices {
		var style lipgloss.Style
		if i == ta.selected {
			style = ta.styles.ButtonActive
		} else {
			style = ta.styles.Button
		}

		buttonText := fmt.Sprintf("[%s]%s", choice.Key, choice.Label)
		buttons = append(buttons, style.Render(buttonText))
	}

	choices.WriteString(strings.Join(buttons, " ") + "\n")

	// Instructions
	instructions := ta.styles.Muted.Render(
		"Use arrow keys to navigate, Enter to select. Tab for details, Esc to cancel.")
	choices.WriteString(instructions)

	return choices.String()
}

// renderHelp renders the help screen
func (ta *ToolApprovalDialog) renderHelp() string {
	var help strings.Builder

	help.WriteString(ta.styles.HelpHeader.Render("Tool Approval Help") + "\n\n")

	help.WriteString(ta.styles.Bold.Render("Available Actions:") + "\n")

	actions := [][]string{
		{"Y", "Yes", "Approve this operation once"},
		{"N", "No", "Deny this operation"},
		{"A", "Always", "Approve this tool for the entire session"},
		{"E", "Never", "Deny this tool for the entire session"},
		{"V", "View", "Toggle preview of the operation"},
		{"?", "Help", "Show this help screen"},
	}

	for _, action := range actions {
		key := ta.styles.HelpKey.Render(action[0])
		label := ta.styles.Bold.Render(action[1])
		desc := ta.styles.HelpDesc.Render(action[2])
		help.WriteString(fmt.Sprintf("%s %-8s %s\n", key, label, desc))
	}

	help.WriteString("\n" + ta.styles.Bold.Render("Navigation:") + "\n")
	help.WriteString("Arrow keys: Navigate choices\n")
	help.WriteString("Enter/Space: Select choice\n")
	help.WriteString("Tab: Toggle details\n")
	help.WriteString("Esc: Cancel (same as No)\n")

	help.WriteString("\n" + ta.styles.Bold.Render("Safety Note:") + "\n")
	help.WriteString("Always review the operation details and risks before approving.\n")
	help.WriteString("'Always' and 'Never' choices apply to the current session only.\n")

	help.WriteString("\n" + ta.styles.Muted.Render("Press ? again to return to the approval dialog."))

	return ta.wrapInDialog(help.String())
}

// wrapInDialog wraps content in a dialog box
func (ta *ToolApprovalDialog) wrapInDialog(content string) string {
	// Calculate dimensions
	dialogWidth := min(ta.width-4, 80)

	// Create dialog box
	dialog := ta.styles.Container.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ta.styles.Colors.Primary).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)

	// Center the dialog
	return lipgloss.Place(
		ta.width, ta.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)
}

// ApprovalResponseMsg represents an approval response message
type ApprovalResponseMsg struct {
	Response ApprovalResponse
	Remember bool
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateToolApprovalRequest creates a new tool approval request
func CreateToolApprovalRequest(toolName, description string, parameters map[string]interface{}) ToolApprovalRequest {
	return ToolApprovalRequest{
		ToolName:    toolName,
		Description: description,
		Parameters:  parameters,
		Timestamp:   time.Now(),
		Reversible:  true, // Default to reversible
	}
}

// AddRisk adds a risk to the approval request
func (req *ToolApprovalRequest) AddRisk(risk string) {
	req.Risks = append(req.Risks, risk)
}

// SetPreview sets the preview content
func (req *ToolApprovalRequest) SetPreview(preview string) {
	req.Preview = preview
}

// SetFilePath sets the file path
func (req *ToolApprovalRequest) SetFilePath(path string) {
	req.FilePath = path
}

// AddAffectedFile adds an affected file
func (req *ToolApprovalRequest) AddAffectedFile(file string) {
	req.AffectedFiles = append(req.AffectedFiles, file)
}

// SetReversible sets whether the operation is reversible
func (req *ToolApprovalRequest) SetReversible(reversible bool) {
	req.Reversible = reversible
}

// IsApproved returns whether the response indicates approval
func (response ApprovalResponse) IsApproved() bool {
	return response == ApprovalYes || response == ApprovalAlways
}

// IsDenied returns whether the response indicates denial
func (response ApprovalResponse) IsDenied() bool {
	return response == ApprovalNo || response == ApprovalNever
}

// IsSessionWide returns whether the response applies to the entire session
func (response ApprovalResponse) IsSessionWide() bool {
	return response == ApprovalAlways || response == ApprovalNever
}
